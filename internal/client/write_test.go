package client

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/types"
)

const fullUpdateArticle = `{"title":"T","author":"","digest":"","content":"<p>c</p>","content_source_url":"","thumb_media_id":"same-cover","show_cover_pic":0,"need_open_comment":0,"only_fans_can_comment":0}`

func TestWriteClientOriginalBytesAndExactUpdate(t *testing.T) {
	var pngBytes bytes.Buffer
	png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	path := filepath.Join(t.TempDir(), "x.png")
	os.WriteFile(path, pngBytes.Bytes(), 0600)
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fake-key" {
			t.Error("missing business key")
		}
		switch r.URL.Path {
		case "/api/v1/body-images":
			f, _, err := r.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			got, _ := io.ReadAll(f)
			if !bytes.Equal(got, pngBytes.Bytes()) {
				t.Error("modified original bytes")
			}
			io.WriteString(w, `{"url":"https://img/body"}`)
		case "/api/v1/drafts":
			if r.Method != "PUT" {
				t.Error("create instead of update")
			}
			var req types.DraftUpdateRequest
			json.NewDecoder(r.Body).Decode(&req)
			a, err := req.Validate()
			if err != nil || req.MediaID != "same-ID" || *req.Index != 0 || a.ThumbMediaID != "same-cover" {
				t.Errorf("unexpected update %+v %v", req, err)
			}
			io.WriteString(w, `{"media_id":"same-ID"}`)
		default:
			t.Errorf("forbidden path %s", r.URL.Path)
		}
	}))
	defer ts.Close()
	c := New(ts.URL, "fake-key")
	if _, err := c.UploadBodyImage(path); err != nil {
		t.Fatal(err)
	}
	index := 0
	if _, err := c.UpdateDraft(types.DraftUpdateRequest{MediaID: "same-ID", Index: &index, Articles: json.RawMessage(fullUpdateArticle)}); err != nil {
		t.Fatal(err)
	}
	for _, req := range []types.DraftUpdateRequest{{}, {MediaID: "id", Articles: json.RawMessage(fullUpdateArticle)}, {MediaID: "id", Index: &index, Articles: json.RawMessage(`{"title":"t"}`)}} {
		if _, err := c.UpdateDraft(req); err == nil {
			t.Fatal("accepted invalid input")
		}
	}
	for _, data := range [][]byte{[]byte("GIF89a"), bytes.Repeat([]byte("x"), types.BodyImageLimit)} {
		os.WriteFile(path, data, 0600)
		if _, err := c.UploadBodyImage(path); err == nil {
			t.Fatal("accepted invalid image")
		}
	}
	if calls != 2 {
		t.Fatalf("invalid inputs called upstream: %d", calls)
	}
}

func TestWriteClientErrorCodeAndNoRedirectRetry(t *testing.T) {
	for _, status := range []int{502, 504, 307} {
		calls := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Location", "/fallback")
			w.WriteHeader(status)
			io.WriteString(w, `{"error":{"code":"wechat_error","wechat_errcode":40007,"message":"access_token=FAKE_TOKEN AppSecret=FAKE_SECRET Bearer FAKE_KEY"}}`)
		}))
		index := 0
		c := New(ts.URL, "FAKE_KEY")
		_, err := c.UpdateDraft(types.DraftUpdateRequest{MediaID: "same-ID", Index: &index, Articles: json.RawMessage(fullUpdateArticle)})
		ts.Close()
		if err == nil || !strings.Contains(err.Error(), "40007") || strings.Contains(err.Error(), "FAKE_") || !strings.Contains(err.Error(), "no automatic retry") || calls != 1 {
			t.Fatalf("status=%d calls=%d err=%v", status, calls, err)
		}
	}
}
