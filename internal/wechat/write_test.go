package wechat

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/types"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/util"
)

// All SDK requests (including token fetch) are redirected to local httptest.
// No live WeChat credentials or calls are used.
func TestSDKBodyImageAndSameIDDraftUpdate(t *testing.T) {
	var imageBytes bytes.Buffer
	png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	original := imageBytes.Bytes()
	if sample := os.Getenv("MP_HELPER_TEST_IMAGE"); sample != "" {
		data, err := os.ReadFile(sample)
		if err != nil {
			t.Fatal(err)
		}
		original = data
		t.Logf("local original-byte sample: %d bytes", len(original))
	}
	var paths []string
	var sent []byte
	var got struct {
		MediaID  string                 `json:"media_id"`
		Index    uint                   `json:"index"`
		Articles map[string]interface{} `json:"articles"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/cgi-bin/token":
			if r.URL.Query().Get("appid") != "fake-app" || r.URL.Query().Get("secret") != "fake-secret" {
				t.Error("wrong account binding")
			}
			io.WriteString(w, `{"access_token":"fake-token","expires_in":7200}`)
		case "/cgi-bin/media/uploadimg":
			if r.URL.Query().Get("access_token") != "fake-token" {
				t.Error("missing server token")
			}
			file, _, err := r.FormFile("media")
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()
			sent, _ = io.ReadAll(file)
			io.WriteString(w, `{"url":"https://img.example/body.png"}`)
		case "/cgi-bin/draft/update":
			if r.URL.Query().Get("access_token") != "fake-token" {
				t.Error("missing server token")
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
		default:
			t.Errorf("forbidden upstream path %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer ts.Close()
	util.SetURIModifier(func(uri string) string {
		u, err := url.Parse(uri)
		if err != nil {
			t.Fatal(err)
		}
		return ts.URL + u.RequestURI()
	})
	defer util.SetURIModifier(nil)
	m := NewManager(cache.NewMemory())
	bodyURL, err := m.UploadBodyImage("fake-app", "fake-secret", "sample.png", bytes.NewReader(original))
	if err != nil || bodyURL != "https://img.example/body.png" || !bytes.Equal(original, sent) {
		t.Fatalf("upload original bytes: url=%s err=%v equal=%v", bodyURL, err, bytes.Equal(original, sent))
	}
	a := types.Article{Title: "T", Content: "<p>unchanged</p>", ThumbMediaID: "original-cover"}
	if err = m.UpdateDraft("fake-app", "fake-secret", "same-ID", 0, a); err != nil {
		t.Fatal(err)
	}
	if got.MediaID != "same-ID" || got.Index != 0 || len(got.Articles) != 9 {
		t.Fatalf("update payload %+v", got)
	}
	if got.Articles["thumb_media_id"] != "original-cover" || got.Articles["content"] != a.Content || got.Articles["author"] != "" || got.Articles["show_cover_pic"] != float64(0) {
		t.Fatalf("changed explicit values %+v", got.Articles)
	}
	if strings.Join(paths, ",") != "/cgi-bin/token,/cgi-bin/media/uploadimg,/cgi-bin/draft/update" {
		t.Fatalf("unexpected calls %v", paths)
	}
}

func TestSDKWriteWechatErrorNoFallback(t *testing.T) {
	var writes int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/token" {
			io.WriteString(w, `{"access_token":"fake-token","expires_in":7200}`)
			return
		}
		if r.URL.Path != "/cgi-bin/media/uploadimg" && r.URL.Path != "/cgi-bin/draft/update" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		writes++
		io.WriteString(w, `{"errcode":40007,"errmsg":"invalid media_id"}`)
	}))
	defer ts.Close()
	util.SetURIModifier(func(uri string) string { u, _ := url.Parse(uri); return ts.URL + u.RequestURI() })
	defer util.SetURIModifier(nil)
	m := NewManager(cache.NewMemory())
	_, err := m.UploadBodyImage("fake-app", "fake-secret", "x.png", strings.NewReader("unchanged"))
	if err == nil || !strings.Contains(err.Error(), "40007") {
		t.Fatalf("upload error %v", err)
	}
	err = m.UpdateDraft("fake-app", "fake-secret", "same-ID", 0, types.Article{})
	if err == nil || !strings.Contains(err.Error(), "40007") {
		t.Fatalf("update error %v", err)
	}
	if writes != 2 {
		t.Fatalf("retry/fallback writes=%d", writes)
	}
}
