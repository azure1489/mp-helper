package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/config"
	"github.com/azure1489/mp-helper/internal/types"
	"github.com/silenceper/wechat/v2/util"
)

func (f *fakeWechat) UploadBodyImage(appID, secret, filename string, r io.Reader) (string, error) {
	f.bodyCalls++
	f.gotAppID = appID
	f.gotSecret = secret
	f.gotBytes, _ = io.ReadAll(r)
	return f.uploadURL, f.uploadErr
}
func (f *fakeWechat) UpdateDraft(appID, secret, id string, index int, a types.Article) error {
	f.updateCalls++
	f.gotAppID = appID
	f.gotSecret = secret
	f.gotMediaID = id
	f.gotIndex = index
	f.gotArticle = a
	return f.draftErr
}

const fullArticle = `{"title":"T","author":"","digest":"","content":"<p>c</p>","content_source_url":"","thumb_media_id":"original-cover","show_cover_pic":0,"need_open_comment":0,"only_fans_can_comment":0}`

func imageRequest(data []byte) *http.Request {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	p, _ := mw.CreateFormFile("file", "x.png")
	p.Write(data)
	mw.Close()
	r := httptest.NewRequest("POST", "/api/v1/body-images", &b)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}
func TestNewWritesBoundToAccount(t *testing.T) {
	st, keyA := seedStore(t)
	accB, err := st.CreateAccount("B", "appidB", "secretB")
	if err != nil {
		t.Fatal(err)
	}
	_, keyB, _, err := st.CreateKey(accB.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeWechat{uploadURL: "https://img/body"}
	r := NewServer(&config.Config{}, st, f).Router()
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	for _, tc := range []struct{ key, app, secret string }{{keyA, "appidA", "secretA"}, {keyB, "appidB", "secretB"}} {
		req := imageRequest(img.Bytes())
		req.Header.Set("Authorization", "Bearer "+tc.key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 || f.gotAppID != tc.app || f.gotSecret != tc.secret || !bytes.Equal(img.Bytes(), f.gotBytes) {
			t.Fatalf("body upload failed: %d %s", w.Code, w.Body.String())
		}
		req = httptest.NewRequest("PUT", "/api/v1/drafts", strings.NewReader(`{"media_id":"same-ID","index":0,"articles":`+fullArticle+`}`))
		req.Header.Set("Authorization", "Bearer "+tc.key)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 || f.gotAppID != tc.app || f.gotMediaID != "same-ID" || f.gotIndex != 0 || f.gotArticle.ThumbMediaID != "original-cover" {
			t.Fatalf("update failed: %d %s", w.Code, w.Body.String())
		}
	}
	if f.gotType != "" || f.gotArticles != nil {
		t.Fatal("called permanent material or create")
	}
	for _, path := range []string{"/api/v1/body-images", "/api/v1/drafts"} {
		method := "POST"
		if path == "/api/v1/drafts" {
			method = "PUT"
		}
		for _, key := range []string{"", "fake-invalid-key"} {
			req := httptest.NewRequest(method, path, nil)
			req.Header.Set("Authorization", "Bearer "+key)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 401 {
				t.Fatalf("unauthorized %s: %d", path, w.Code)
			}
		}
	}
	if f.bodyCalls != 2 || f.updateCalls != 2 {
		t.Fatal("unauthorized request called upstream")
	}
}

func TestNewWritesInvalidInputsNoUpstream(t *testing.T) {
	st, key := seedStore(t)
	f := &fakeWechat{}
	r := NewServer(&config.Config{}, st, f).Router()
	for _, data := range [][]byte{nil, []byte("GIF89a"), bytes.Repeat([]byte("x"), types.BodyImageLimit), bytes.Repeat([]byte("x"), types.BodyImageLimit+1)} {
		req := imageRequest(data)
		req.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := 400
		if len(data) >= types.BodyImageLimit {
			want = 413
		}
		if w.Code != want {
			t.Fatalf("image size=%d: %d want %d", len(data), w.Code, want)
		}
	}
	inputs := []string{`{}`, `{"media_id":"","index":0,"articles":` + fullArticle + `}`, `{"media_id":"id","articles":` + fullArticle + `}`, `{"media_id":"id","index":-1,"articles":` + fullArticle + `}`, `{"media_id":"id","index":0,"articles":{"title":"t"}}`}
	for _, field := range []string{"title", "author", "digest", "content", "content_source_url", "thumb_media_id", "show_cover_pic", "need_open_comment", "only_fans_can_comment"} {
		var a map[string]interface{}
		json.Unmarshal([]byte(fullArticle), &a)
		delete(a, field)
		b, _ := json.Marshal(a)
		inputs = append(inputs, `{"media_id":"id","index":0,"articles":`+string(b)+`}`)
	}
	inputs = append(inputs, `{"media_id":"id","index":0,"articles":`+strings.Replace(fullArticle, `"T"`, `"`+strings.Repeat("中", 65)+`"`, 1)+`}`, `{"media_id":"id","index":0,"articles":`+strings.Replace(fullArticle, `<p>c</p>`, strings.Repeat("c", 20000), 1)+`}`)
	for _, body := range inputs {
		req := httptest.NewRequest("PUT", "/api/v1/drafts", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("invalid update status %d: %s", w.Code, w.Body.String())
		}
	}
	if f.bodyCalls != 0 || f.updateCalls != 0 || f.gotType != "" || f.gotArticles != nil {
		t.Fatal("invalid input called upstream")
	}
}

func TestNewWritesErrorsAreSecretFreeAndNeverFallback(t *testing.T) {
	st, key := seedStore(t)
	for _, upstream := range []error{
		util.NewCommonError("UpdateDraft", 40007, "access_token=FAKE_TOKEN AppSecret=FAKE_SECRET Bearer FAKE_BEARER https://host/?secret=FAKE_QUERY"),
		&url.Error{Op: "Post", URL: "https://host/?access_token=FAKE_TOKEN&secret=FAKE_QUERY", Err: &net.DNSError{Err: "AppSecret=FAKE_SECRET Bearer FAKE_BEARER", IsTimeout: true}},
		&url.Error{Op: "Post", URL: "https://host/?access_token=FAKE_TOKEN", Err: fmt.Errorf("disconnect FAKE_SECRET")},
	} {
		f := &fakeWechat{draftErr: upstream, uploadErr: upstream}
		r := NewServer(&config.Config{}, st, f).Router()
		var img bytes.Buffer
		png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 1, 1)))
		for _, req := range []*http.Request{imageRequest(img.Bytes()), httptest.NewRequest("PUT", "/api/v1/drafts", strings.NewReader(`{"media_id":"same-ID","index":0,"articles":`+fullArticle+`}`))} {
			req.Header.Set("Authorization", "Bearer "+key)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 502 && w.Code != 504 {
				t.Fatalf("status=%d", w.Code)
			}
			if strings.Contains(w.Body.String(), "FAKE_") || strings.Contains(w.Body.String(), "access_token") || strings.Contains(w.Body.String(), "Bearer") {
				t.Fatal("leaked secret", w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "no automatic retry") {
				t.Fatal("missing retry warning")
			}
			var er types.ErrorResponse
			json.Unmarshal(w.Body.Bytes(), &er)
			if common, ok := upstream.(*util.CommonError); ok && er.Error.WechatErrcode != common.ErrCode {
				t.Fatal("lost errcode")
			}
			if _, ok := upstream.(*url.Error); ok && !strings.Contains(er.Error.Message, "uncertain") {
				t.Fatal("lost uncertain classification")
			}
		}
		if f.updateCalls != 1 || f.bodyCalls != 1 || f.gotArticles != nil || f.gotType != "" {
			t.Fatal("retry/fallback detected")
		}
	}
}
