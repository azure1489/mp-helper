// Package wechat 在 silenceper/wechat v2 之上封装多账号的素材/草稿操作。
package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/officialaccount/material"
	"github.com/silenceper/wechat/v2/util"

	"github.com/azure1489/mp-helper/internal/types"
)

// Service 是 handler 依赖的微信能力抽象，便于测试 mock。
type Service interface {
	UploadMaterial(appID, appSecret, mediaType, filename string, r io.Reader) (mediaID, url string, err error)
	AddDraft(appID, appSecret string, articles []types.Article) (mediaID string, err error)
	Invalidate(appID string)

	// 只读查询：返回微信原始响应体（原样透传，不改字段名）。
	ListPublished(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error)
	GetPublished(appID, appSecret, articleID string) (json.RawMessage, error)
	ListDrafts(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error)
	GetDraft(appID, appSecret, mediaID string) (json.RawMessage, error)
	DataCube(appID, appSecret, api, begin, end string) (json.RawMessage, error)
}

// Manager 按 appid 缓存 OfficialAccount 实例，共享一个 token cache。
type Manager struct {
	cache    cache.Cache
	mu       sync.RWMutex
	accounts map[string]*officialaccount.OfficialAccount
}

func NewManager(c cache.Cache) *Manager {
	return &Manager{
		cache:    c,
		accounts: make(map[string]*officialaccount.OfficialAccount),
	}
}

// get 取（或惰性构造并缓存）某 appid 的 OfficialAccount。构造不触网，token 在首次调用时按需获取。
func (m *Manager) get(appID, appSecret string) *officialaccount.OfficialAccount {
	m.mu.RLock()
	oa, ok := m.accounts[appID]
	m.mu.RUnlock()
	if ok {
		return oa
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if oa, ok = m.accounts[appID]; ok {
		return oa
	}
	oa = officialaccount.NewOfficialAccount(&offConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     m.cache,
	})
	m.accounts[appID] = oa
	return oa
}

// Invalidate 在账号被更新/删除时清掉缓存实例。
func (m *Manager) Invalidate(appID string) {
	m.mu.Lock()
	delete(m.accounts, appID)
	m.mu.Unlock()
}

// UploadMaterial 上传永久素材，返回 media_id 与（图片类型的）url。委托给 silenceper 的 material.AddMaterialFromReader。
func (m *Manager) UploadMaterial(appID, appSecret, mediaType, filename string, r io.Reader) (string, string, error) {
	mat := m.get(appID, appSecret).GetMaterial()
	return mat.AddMaterialFromReader(material.MediaType(mediaType), filename, r)
}

// AddDraft 把 types.Article 列表转换为 silenceper 的 draft.Article 并创建草稿，返回草稿 media_id。
func (m *Manager) AddDraft(appID, appSecret string, articles []types.Article) (string, error) {
	d := m.get(appID, appSecret).GetDraft()
	arts := make([]*draft.Article, 0, len(articles))
	for _, a := range articles {
		arts = append(arts, &draft.Article{
			Title:              a.Title,
			Author:             a.Author,
			Digest:             a.Digest,
			Content:            a.Content,
			ContentSourceURL:   a.ContentSourceURL,
			ThumbMediaID:       a.ThumbMediaID,
			ShowCoverPic:       a.ShowCoverPic,
			NeedOpenComment:    a.NeedOpenComment,
			OnlyFansCanComment: a.OnlyFansCanComment,
		})
	}
	return d.AddDraft(arts)
}

// ---- 只读查询（已发布列表与单篇、草稿读取、数据统计）----
// 不做发布、群发、删除；统计接口的白名单在 api 层。

const apiBase = "https://api.weixin.qq.com"

// postRaw 以该账号的 access_token 调用微信接口：errcode 非 0 时返回错误（文本含 errcode=N，供 respondWechatError 解析），
// 否则原样返回响应体。
func (m *Manager) postRaw(appID, appSecret, apiName, path string, body interface{}) (json.RawMessage, error) {
	token, err := m.get(appID, appSecret).GetContext().GetAccessToken()
	if err != nil {
		return nil, err
	}
	resp, err := util.PostJSON(apiBase+path+"?access_token="+token, body)
	if err != nil {
		return nil, err
	}
	var ce struct {
		ErrCode int64  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp, &ce); err != nil {
		return nil, fmt.Errorf("%s: response is not JSON: %w", apiName, err)
	}
	if ce.ErrCode != 0 {
		return nil, fmt.Errorf("%s Error , errcode=%d , errmsg=%s", apiName, ce.ErrCode, ce.ErrMsg)
	}
	return json.RawMessage(resp), nil
}

type pageRequest struct {
	Offset    int `json:"offset"`
	Count     int `json:"count"`
	NoContent int `json:"no_content"`
}

func newPageRequest(offset, count int, noContent bool) pageRequest {
	p := pageRequest{Offset: offset, Count: count}
	if noContent {
		p.NoContent = 1
	}
	return p
}

// ListPublished 分页列出已发布图文（freepublish/batchget）。「群发」的图文不在其中。
func (m *Manager) ListPublished(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error) {
	return m.postRaw(appID, appSecret, "ListPublished", "/cgi-bin/freepublish/batchget", newPageRequest(offset, count, noContent))
}

// GetPublished 读取一篇已发布图文（freepublish/getarticle）。
func (m *Manager) GetPublished(appID, appSecret, articleID string) (json.RawMessage, error) {
	return m.postRaw(appID, appSecret, "GetPublished", "/cgi-bin/freepublish/getarticle", map[string]string{"article_id": articleID})
}

// ListDrafts 分页列出草稿（draft/batchget）。
func (m *Manager) ListDrafts(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error) {
	return m.postRaw(appID, appSecret, "ListDrafts", "/cgi-bin/draft/batchget", newPageRequest(offset, count, noContent))
}

// GetDraft 读取一篇草稿（draft/get），供保存后回读核对。
func (m *Manager) GetDraft(appID, appSecret, mediaID string) (json.RawMessage, error) {
	return m.postRaw(appID, appSecret, "GetDraft", "/cgi-bin/draft/get", map[string]string{"media_id": mediaID})
}

// DataCube 调用数据统计接口（/datacube/<api>），日期为 YYYY-MM-DD。
func (m *Manager) DataCube(appID, appSecret, api, begin, end string) (json.RawMessage, error) {
	return m.postRaw(appID, appSecret, "DataCube."+api, "/datacube/"+api, map[string]string{"begin_date": begin, "end_date": end})
}

// 编译期断言 Manager 实现 Service。
var _ Service = (*Manager)(nil)
