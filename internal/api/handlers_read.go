package api

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 只读查询：已发布列表与单篇、草稿读取、数据统计。响应体为微信原始返回（原样透传，不改字段名）。

// dataCubeAPIs 允许透传的统计接口；其余一律拒绝，避免本服务成为任意微信接口的代理。
var dataCubeAPIs = map[string]bool{
	"getarticletotaldetail": true, // 单篇图文逐日明细（含「发布」与「群发」）
	"getbizsummary":         true, // 账号每日阅读汇总
	"getusersummary":        true, // 用户增减
	"getusercumulate":       true, // 累计用户
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// pageParams 解析 offset / count / no_content：count 为 1–20（微信上限），no_content 缺省为 1（不返回正文）。
func pageParams(c *gin.Context) (offset, count int, noContent, ok bool) {
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		respondError(c, http.StatusBadRequest, "bad_request", "offset must be a non-negative integer")
		return 0, 0, false, false
	}
	count, err = strconv.Atoi(c.DefaultQuery("count", "20"))
	if err != nil || count < 1 || count > 20 {
		respondError(c, http.StatusBadRequest, "bad_request", "count must be between 1 and 20")
		return 0, 0, false, false
	}
	nc := c.DefaultQuery("no_content", "1")
	return offset, count, nc != "0" && nc != "false", true
}

func writeRaw(c *gin.Context, raw []byte) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (s *Server) handleListPublished(c *gin.Context) {
	acc := accountFromContext(c)
	offset, count, noContent, ok := pageParams(c)
	if !ok {
		return
	}
	raw, err := s.wechat.ListPublished(acc.AppID, acc.AppSecret, offset, count, noContent)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	writeRaw(c, raw)
}

func (s *Server) handleGetPublished(c *gin.Context) {
	acc := accountFromContext(c)
	raw, err := s.wechat.GetPublished(acc.AppID, acc.AppSecret, c.Param("article_id"))
	if err != nil {
		respondWechatError(c, err)
		return
	}
	writeRaw(c, raw)
}

func (s *Server) handleListDrafts(c *gin.Context) {
	acc := accountFromContext(c)
	offset, count, noContent, ok := pageParams(c)
	if !ok {
		return
	}
	raw, err := s.wechat.ListDrafts(acc.AppID, acc.AppSecret, offset, count, noContent)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	writeRaw(c, raw)
}

func (s *Server) handleGetDraft(c *gin.Context) {
	acc := accountFromContext(c)
	raw, err := s.wechat.GetDraft(acc.AppID, acc.AppSecret, c.Param("media_id"))
	if err != nil {
		respondWechatError(c, err)
		return
	}
	writeRaw(c, raw)
}

func (s *Server) handleDataCube(c *gin.Context) {
	acc := accountFromContext(c)
	api := c.Param("api")
	if !dataCubeAPIs[api] {
		respondError(c, http.StatusBadRequest, "bad_request", "unsupported datacube api: "+api)
		return
	}
	begin, end := c.Query("begin"), c.Query("end")
	if !dateRe.MatchString(begin) || !dateRe.MatchString(end) {
		respondError(c, http.StatusBadRequest, "bad_request", "begin and end must be YYYY-MM-DD")
		return
	}
	b, err1 := time.Parse("2006-01-02", begin)
	e, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil || e.Before(b) {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid date range")
		return
	}
	raw, err := s.wechat.DataCube(acc.AppID, acc.AppSecret, api, begin, end)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	writeRaw(c, raw)
}
