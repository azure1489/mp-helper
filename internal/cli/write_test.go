package cli

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

func TestBodyImageAndDraftUpdateCommands(t *testing.T) {
	const article = `{"title":"T","author":"","digest":"","content":"<p>c</p>","content_source_url":"","thumb_media_id":"same-cover","show_cover_pic":0,"need_open_comment":0,"only_fans_can_comment":0}`
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.json")
	os.WriteFile(articlePath, []byte(article), 0600)
	var data bytes.Buffer
	png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	imagePath := filepath.Join(dir, "image.png")
	os.WriteFile(imagePath, data.Bytes(), 0600)
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/api/v1/body-images":
			f, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				return
			}
			defer f.Close()
			got, _ := io.ReadAll(f)
			if !bytes.Equal(got, data.Bytes()) {
				t.Error("changed image")
			}
			io.WriteString(w, `{"url":"https://img/body"}`)
		case "/api/v1/drafts":
			if r.Method != "PUT" {
				t.Error("create fallback")
			}
			var req types.DraftUpdateRequest
			json.NewDecoder(r.Body).Decode(&req)
			a, err := req.Validate()
			if err != nil || req.Index == nil || *req.Index != 0 || req.MediaID != "same-ID" || a.ThumbMediaID != "same-cover" {
				t.Errorf("wrong update %+v %v", req, err)
			}
			io.WriteString(w, `{"media_id":"same-ID"}`)
		default:
			t.Errorf("forbidden path %s", r.URL.Path)
		}
	}))
	defer ts.Close()
	out, err := run(t, ts.URL, "body-image", "upload", imagePath)
	if err != nil || !strings.Contains(out, `"url"`) {
		t.Fatalf("%s %v", out, err)
	}
	out, err = run(t, ts.URL, "draft", "update", "--media-id", "same-ID", "--index", "0", "--article-file", articlePath)
	if err != nil || !strings.Contains(out, "same-ID") {
		t.Fatalf("%s %v", out, err)
	}
	for _, args := range [][]string{
		{"draft", "update", "--media-id", "same-ID", "--article-file", articlePath},
		{"draft", "update", "--media-id", "same-ID", "--index", "-1", "--article-file", articlePath},
		{"draft", "update", "--index", "0", "--article-file", articlePath},
	} {
		if _, err := run(t, ts.URL, args...); err == nil {
			t.Fatal("accepted missing/invalid arguments")
		}
	}
	os.WriteFile(articlePath, []byte(`{"title":"T"}`), 0600)
	if _, err := run(t, ts.URL, "draft", "update", "--media-id", "same-ID", "--index", "0", "--article-file", articlePath); err == nil {
		t.Fatal("accepted partial article")
	}
	if calls != 2 {
		t.Fatalf("invalid calls reached server: %d", calls)
	}
}
