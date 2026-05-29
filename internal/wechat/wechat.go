// Package wechat 在 silenceper/wechat v2 之上封装多账号的素材/草稿操作。
package wechat

import (
	"io"
	"sync"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/officialaccount/material"

	"github.com/azure1489/mp-helper/internal/types"
)

// Service 是 handler 依赖的微信能力抽象，便于测试 mock。
type Service interface {
	UploadMaterial(appID, appSecret, mediaType, filename string, r io.Reader) (mediaID, url string, err error)
	AddDraft(appID, appSecret string, articles []types.Article) (mediaID string, err error)
	Invalidate(appID string)
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

func (m *Manager) UploadMaterial(appID, appSecret, mediaType, filename string, r io.Reader) (string, string, error) {
	mat := m.get(appID, appSecret).GetMaterial()
	return mat.AddMaterialFromReader(material.MediaType(mediaType), filename, r)
}

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

// 编译期断言 Manager 实现 Service。
var _ Service = (*Manager)(nil)
