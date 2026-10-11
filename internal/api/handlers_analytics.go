package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/azure1489/mp-helper/internal/analytics"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

// 数据分析：把新版统计接口整理成单篇表现与每日粉丝增减；文章补录信息（推送时刻、分类）存本地库。

// storeCache 把 SQLite 缓存表适配为 analytics.Cache（按账号隔离）；读写失败只当未命中。
type storeCache struct {
	st        *store.Store
	accountID int64
}

func (c storeCache) Get(api, date string) ([]byte, time.Time, bool) {
	body, at, err := c.st.GetCachedDataCube(c.accountID, api, date)
	return body, at, err == nil
}

func (c storeCache) Put(api, date string, body []byte, at time.Time) {
	_ = c.st.PutCachedDataCube(c.accountID, api, date, body, at)
}

// builderFor 组装某账号的分析器（含补录信息与 SQLite 缓存）。
func (s *Server) builderFor(acc store.Account) (*analytics.Builder, error) {
	meta, err := s.store.ListArticleMeta(acc.ID)
	if err != nil {
		return nil, err
	}
	return &analytics.Builder{
		DataCube: func(api, begin, end string) (json.RawMessage, error) {
			return s.wechat.DataCube(acc.AppID, acc.AppSecret, api, begin, end)
		},
		ListPublished: func(offset, count int) (json.RawMessage, error) {
			return s.wechat.ListPublished(acc.AppID, acc.AppSecret, offset, count, true)
		},
		Meta:  meta,
		Cache: storeCache{s.store, acc.ID},
		Now:   time.Now(),
	}, nil
}

func (s *Server) analyticsBuilder(c *gin.Context) (*analytics.Builder, bool) {
	b, err := s.builderFor(accountFromContext(c))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", "load article meta failed")
		return nil, false
	}
	return b, true
}

func analyticsRange(c *gin.Context) (time.Time, time.Time, bool) {
	b, e, err := analytics.ValidateRange(c.Query("begin"), c.Query("end"), time.Now())
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return b, e, false
	}
	return b, e, true
}

func (s *Server) handleAnalyticsArticles(c *gin.Context) {
	begin, end, ok := analyticsRange(c)
	if !ok {
		return
	}
	b, ok := s.analyticsBuilder(c)
	if !ok {
		return
	}
	series := c.Query("series")
	rep, err := b.Articles(begin, end, series == "1" || series == "true")
	if err != nil {
		respondWechatError(c, err)
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (s *Server) handleAnalyticsDaily(c *gin.Context) {
	begin, end, ok := analyticsRange(c)
	if !ok {
		return
	}
	b, ok := s.analyticsBuilder(c)
	if !ok {
		return
	}
	rep, err := b.Daily(begin, end)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	c.JSON(http.StatusOK, rep)
}

var msgIDRe = regexp.MustCompile(`^\d+_\d+$`)

const maxMetaText = 200

func (s *Server) handlePutArticleMeta(c *gin.Context) {
	msgid := c.Param("msgid")
	if !msgIDRe.MatchString(msgid) {
		respondError(c, http.StatusBadRequest, "bad_request", "msgid must look like <msg_data_id>_<index>, e.g. 2247549594_1")
		return
	}
	var req types.ArticleMetaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	if req.PublishedAt == nil && req.Category == nil && req.Note == nil {
		respondError(c, http.StatusBadRequest, "bad_request", "nothing to update: set published_at, category or note")
		return
	}
	if req.PublishedAt != nil {
		t, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(*req.PublishedAt), analytics.CST)
		if err != nil {
			respondError(c, http.StatusBadRequest, "bad_request", `published_at must be "YYYY-MM-DD HH:MM" (Beijing time)`)
			return
		}
		norm := t.Format("2006-01-02 15:04")
		req.PublishedAt = &norm
	}
	for _, p := range []*string{req.Category, req.Note} {
		if p != nil && utf8.RuneCountInString(*p) > maxMetaText {
			respondError(c, http.StatusBadRequest, "bad_request", "category and note must be at most 200 characters")
			return
		}
	}
	acc := accountFromContext(c)
	if err := s.store.UpsertArticleMeta(acc.ID, msgid, req.PublishedAt, req.Category, req.Note); err != nil {
		respondError(c, http.StatusInternalServerError, "internal", "save article meta failed")
		return
	}
	metas, err := s.store.ListArticleMeta(acc.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", "load article meta failed")
		return
	}
	for _, m := range metas {
		if m.MsgID == msgid {
			c.JSON(http.StatusOK, toMetaDTO(m))
			return
		}
	}
	respondError(c, http.StatusInternalServerError, "internal", "article meta not found after save")
}

func (s *Server) handleListArticleMeta(c *gin.Context) {
	metas, err := s.store.ListArticleMeta(accountFromContext(c).ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", "load article meta failed")
		return
	}
	out := make([]types.ArticleMeta, 0, len(metas))
	for _, m := range metas {
		out = append(out, toMetaDTO(m))
	}
	c.JSON(http.StatusOK, out)
}

func toMetaDTO(m store.ArticleMeta) types.ArticleMeta {
	return types.ArticleMeta{MsgID: m.MsgID, PublishedAt: m.PublishedAt, Category: m.Category, Note: m.Note, UpdatedAt: m.UpdatedAt}
}
