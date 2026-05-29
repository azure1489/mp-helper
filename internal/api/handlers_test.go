package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

// fakeWechat 实现 wechat.Service，用于 handler 测试。
type fakeWechat struct {
	uploadMediaID string
	uploadURL     string
	uploadErr     error
	draftMediaID  string
	draftErr      error
	gotType       string
	gotArticles   []types.Article
	invalidated   []string
}

func (f *fakeWechat) UploadMaterial(appID, appSecret, mediaType, filename string, r io.Reader) (string, string, error) {
	f.gotType = mediaType
	_, _ = io.Copy(io.Discard, r)
	return f.uploadMediaID, f.uploadURL, f.uploadErr
}
func (f *fakeWechat) AddDraft(appID, appSecret string, articles []types.Article) (string, error) {
	f.gotArticles = articles
	return f.draftMediaID, f.draftErr
}
func (f *fakeWechat) Invalidate(appID string) { f.invalidated = append(f.invalidated, appID) }

// withAccount 注入一个已鉴权账号（绕过 DataAuth，专注 handler 逻辑）。
func withAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ctxAccountKey, store.Account{ID: 1, AppID: "appidA", AppSecret: "secretA"})
		c.Next()
	}
}

func TestUploadMaterial(t *testing.T) {
	fw := &fakeWechat{uploadMediaID: "media123", uploadURL: "http://img/x.png"}
	s := &Server{wechat: fw}
	r := gin.New()
	r.POST("/api/v1/materials", withAccount(), s.handleUploadMaterial)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fwpart, _ := mw.CreateFormFile("file", "x.png")
	_, _ = fwpart.Write([]byte("PNGDATA"))
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/materials", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var mr types.MaterialResponse
	_ = json.Unmarshal(w.Body.Bytes(), &mr)
	if mr.MediaID != "media123" || mr.URL != "http://img/x.png" {
		t.Fatalf("resp = %+v", mr)
	}
	if fw.gotType != "image" {
		t.Fatalf("default type = %q want image", fw.gotType)
	}
}

func TestUploadMaterialMissingFile(t *testing.T) {
	s := &Server{wechat: &fakeWechat{}}
	r := gin.New()
	r.POST("/api/v1/materials", withAccount(), s.handleUploadMaterial)

	req := httptest.NewRequest("POST", "/api/v1/materials", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=zzz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d want 400", w.Code)
	}
}

func TestUploadMaterialBadType(t *testing.T) {
	s := &Server{wechat: &fakeWechat{}}
	r := gin.New()
	r.POST("/api/v1/materials", withAccount(), s.handleUploadMaterial)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("type", "video")
	fwpart, _ := mw.CreateFormFile("file", "x.mp4")
	_, _ = fwpart.Write([]byte("DATA"))
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/materials", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("unsupported type status = %d want 400", w.Code)
	}
}

func TestCreateDraft(t *testing.T) {
	fw := &fakeWechat{draftMediaID: "draft987"}
	s := &Server{wechat: fw}
	r := gin.New()
	r.POST("/api/v1/drafts", withAccount(), s.handleCreateDraft)

	body, _ := json.Marshal(types.DraftRequest{Articles: []types.Article{
		{Title: "T", Content: "<p>c</p>", ThumbMediaID: "m1"},
	}})
	req := httptest.NewRequest("POST", "/api/v1/drafts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var dr types.DraftResponse
	_ = json.Unmarshal(w.Body.Bytes(), &dr)
	if dr.MediaID != "draft987" {
		t.Fatalf("resp = %+v", dr)
	}
	if len(fw.gotArticles) != 1 || fw.gotArticles[0].Title != "T" {
		t.Fatalf("forwarded articles = %+v", fw.gotArticles)
	}
}

func TestCreateDraftValidation(t *testing.T) {
	s := &Server{wechat: &fakeWechat{}}
	r := gin.New()
	r.POST("/api/v1/drafts", withAccount(), s.handleCreateDraft)

	// 空 articles
	body, _ := json.Marshal(types.DraftRequest{})
	req := httptest.NewRequest("POST", "/api/v1/drafts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("empty articles status = %d want 400", w.Code)
	}

	// 缺 thumb_media_id
	body, _ = json.Marshal(types.DraftRequest{Articles: []types.Article{{Title: "t", Content: "c"}}})
	req = httptest.NewRequest("POST", "/api/v1/drafts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("missing thumb status = %d want 400", w.Code)
	}
}
