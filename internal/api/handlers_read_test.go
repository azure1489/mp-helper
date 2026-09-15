package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/azure1489/mp-helper/internal/types"
)

func (f *fakeWechat) ListPublished(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error) {
	f.gotRead = append(f.gotRead, fmt.Sprintf("ListPublished %d %d %v", offset, count, noContent))
	return f.readRaw, f.readErr
}
func (f *fakeWechat) GetPublished(appID, appSecret, articleID string) (json.RawMessage, error) {
	f.gotRead = append(f.gotRead, "GetPublished "+articleID)
	return f.readRaw, f.readErr
}
func (f *fakeWechat) ListDrafts(appID, appSecret string, offset, count int, noContent bool) (json.RawMessage, error) {
	f.gotRead = append(f.gotRead, fmt.Sprintf("ListDrafts %d %d %v", offset, count, noContent))
	return f.readRaw, f.readErr
}
func (f *fakeWechat) GetDraft(appID, appSecret, mediaID string) (json.RawMessage, error) {
	f.gotRead = append(f.gotRead, "GetDraft "+mediaID)
	return f.readRaw, f.readErr
}
func (f *fakeWechat) DataCube(appID, appSecret, api, begin, end string) (json.RawMessage, error) {
	f.gotRead = append(f.gotRead, fmt.Sprintf("DataCube %s %s %s", api, begin, end))
	return f.readRaw, f.readErr
}

func readRouter(fw *fakeWechat) *gin.Engine {
	s := &Server{wechat: fw}
	r := gin.New()
	g := r.Group("/api/v1", withAccount())
	g.GET("/published", s.handleListPublished)
	g.GET("/published/:article_id", s.handleGetPublished)
	g.GET("/drafts", s.handleListDrafts)
	g.GET("/drafts/:media_id", s.handleGetDraft)
	g.GET("/datacube/:api", s.handleDataCube)
	return r
}

func doGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

func TestReadPassthroughAndParams(t *testing.T) {
	const body = `{"total_count":1,"item":[]}`
	cases := []struct{ path, want string }{
		{"/api/v1/published", "ListPublished 0 20 true"},
		{"/api/v1/published?offset=20&count=5&no_content=0", "ListPublished 20 5 false"},
		{"/api/v1/published/art1", "GetPublished art1"},
		{"/api/v1/drafts?count=3", "ListDrafts 0 3 true"},
		{"/api/v1/drafts/m1", "GetDraft m1"},
		{"/api/v1/datacube/getusersummary?begin=2026-09-08&end=2026-09-14", "DataCube getusersummary 2026-09-08 2026-09-14"},
	}
	for _, tc := range cases {
		fw := &fakeWechat{readRaw: json.RawMessage(body)}
		w := doGet(readRouter(fw), tc.path)
		if w.Code != 200 || w.Body.String() != body {
			t.Fatalf("%s: status=%d body=%s", tc.path, w.Code, w.Body.String())
		}
		if len(fw.gotRead) != 1 || fw.gotRead[0] != tc.want {
			t.Fatalf("%s: forwarded %v want %q", tc.path, fw.gotRead, tc.want)
		}
	}
}

func TestReadValidation(t *testing.T) {
	for _, p := range []string{
		"/api/v1/published?count=0",
		"/api/v1/published?count=21",
		"/api/v1/drafts?offset=-1",
		"/api/v1/datacube/getarticletotal?begin=2026-09-10&end=2026-09-10",
		"/api/v1/datacube/getusersummary?begin=2026-9-8&end=2026-09-14",
		"/api/v1/datacube/getusersummary?begin=2026-09-14&end=2026-09-08",
		"/api/v1/datacube/getusersummary",
	} {
		fw := &fakeWechat{}
		w := doGet(readRouter(fw), p)
		if w.Code != 400 {
			t.Fatalf("%s: status=%d want 400", p, w.Code)
		}
		if len(fw.gotRead) != 0 {
			t.Fatalf("%s: should not call wechat, got %v", p, fw.gotRead)
		}
	}
}

func TestReadWechatError(t *testing.T) {
	fw := &fakeWechat{readErr: errors.New("GetPublished Error , errcode=48001 , errmsg=api unauthorized")}
	w := doGet(readRouter(fw), "/api/v1/published/art1")
	if w.Code != 502 {
		t.Fatalf("status=%d want 502", w.Code)
	}
	var er types.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &er)
	if er.Error.WechatErrcode != 48001 {
		t.Fatalf("resp = %+v", er)
	}
}
