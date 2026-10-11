package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/azure1489/mp-helper/internal/analytics"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

func analyticsRouter(t *testing.T, fw *fakeWechat) (*gin.Engine, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "analytics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.CreateAccount("a", "appidA", "secretA"); err != nil { // ID 1，与 withAccount 一致
		t.Fatal(err)
	}
	s := &Server{store: st, wechat: fw}
	r := gin.New()
	g := r.Group("/api/v1", withAccount())
	g.GET("/analytics/articles", s.handleAnalyticsArticles)
	g.GET("/analytics/daily", s.handleAnalyticsDaily)
	g.GET("/articles/meta", s.handleListArticleMeta)
	g.PUT("/articles/:msgid/meta", s.handlePutArticleMeta)
	return r, st
}

func yesterday() string {
	return time.Now().In(analytics.CST).AddDate(0, 0, -1).Format("2006-01-02")
}

func TestAnalyticsEndpoints(t *testing.T) {
	fw := &fakeWechat{readRaw: json.RawMessage(`{"list":[],"is_delay":false}`)}
	r, _ := analyticsRouter(t, fw)
	y := yesterday()

	w := doGet(r, "/api/v1/analytics/articles?begin="+y+"&end="+y)
	var ar types.ArticlesReport
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &ar) != nil || ar.Articles == nil || ar.Begin != y {
		t.Fatalf("articles: %d %s", w.Code, w.Body.String())
	}
	if len(fw.gotRead) != 1 || fw.gotRead[0] != "DataCube getarticletotaldetail "+y+" "+y {
		t.Fatalf("forwarded %v", fw.gotRead)
	}
	// 第二次命中缓存，不再调微信。
	doGet(r, "/api/v1/analytics/articles?begin="+y+"&end="+y)
	if len(fw.gotRead) != 1 {
		t.Fatalf("cache miss: %v", fw.gotRead)
	}

	w = doGet(r, "/api/v1/analytics/daily?begin="+y+"&end="+y)
	var dr types.DailyReport
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &dr) != nil || len(dr.Days) != 1 || len(dr.Summary) != 3 {
		t.Fatalf("daily: %d %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsValidationAndWechatError(t *testing.T) {
	r, _ := analyticsRouter(t, &fakeWechat{readErr: errors.New("DataCube Error , errcode=48001 , errmsg=api unauthorized")})
	today := time.Now().In(analytics.CST).Format("2006-01-02")
	for _, p := range []string{
		"/api/v1/analytics/articles",
		"/api/v1/analytics/articles?begin=" + today + "&end=" + today,
		"/api/v1/analytics/daily?begin=2025-10-01&end=2025-10-02",
	} {
		if w := doGet(r, p); w.Code != 400 {
			t.Fatalf("%s: status %d want 400", p, w.Code)
		}
	}
	y := yesterday()
	w := doGet(r, "/api/v1/analytics/daily?begin="+y+"&end="+y)
	if w.Code != 502 || !strings.Contains(w.Body.String(), `"wechat_errcode":48001`) {
		t.Fatalf("all calls failed: %d %s", w.Code, w.Body.String())
	}
}

func putMeta(r *gin.Engine, msgid, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/articles/"+msgid+"/meta", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestArticleMetaEndpoints(t *testing.T) {
	r, _ := analyticsRouter(t, &fakeWechat{})
	if w := putMeta(r, "2247549594_1", `{"published_at":"2026-10-05 21:00"}`); w.Code != 200 {
		t.Fatalf("put: %d %s", w.Code, w.Body.String())
	}
	w := putMeta(r, "2247549594_1", `{"category":"攻略"}`)
	var m types.ArticleMeta
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &m) != nil || m.PublishedAt != "2026-10-05 21:00" || m.Category != "攻略" {
		t.Fatalf("partial put: %d %s", w.Code, w.Body.String())
	}
	for _, c := range [][2]string{
		{"abc", `{"note":"x"}`},
		{"1_1", `{}`},
		{"1_1", `{"published_at":"2026-10-05T21:00:00Z"}`},
		{"1_1", `{"note":"` + strings.Repeat("长", 201) + `"}`},
	} {
		if w := putMeta(r, c[0], c[1]); w.Code != 400 {
			t.Fatalf("%v: status %d want 400", c, w.Code)
		}
	}
	w = doGet(r, "/api/v1/articles/meta")
	var list []types.ArticleMeta
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &list) != nil || len(list) != 1 || list[0].MsgID != "2247549594_1" {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
}
