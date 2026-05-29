package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/types"
)

func run(t *testing.T, baseURL string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("MP_HELPER_API_URL", baseURL)
	t.Setenv("MP_HELPER_API_KEY", "k1")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestHealthCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	}))
	defer ts.Close()

	out, err := run(t, ts.URL, "health")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"status": "ok"`) {
		t.Fatalf("out = %s", out)
	}
}

func TestMaterialUploadCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(types.MaterialResponse{MediaID: "m1", URL: "u1"})
	}))
	defer ts.Close()

	p := filepath.Join(t.TempDir(), "x.png")
	_ = os.WriteFile(p, []byte("D"), 0o600)

	out, err := run(t, ts.URL, "material", "upload", p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"media_id": "m1"`) {
		t.Fatalf("out = %s", out)
	}
}

func TestDraftCreateCmdWithCoverFile(t *testing.T) {
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/materials":
			_ = json.NewEncoder(w).Encode(types.MaterialResponse{MediaID: "coverMedia", URL: "u"})
		case "/api/v1/drafts":
			var req types.DraftRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Articles[0].ThumbMediaID != "coverMedia" {
				t.Errorf("thumb = %q want coverMedia", req.Articles[0].ThumbMediaID)
			}
			_ = json.NewEncoder(w).Encode(types.DraftResponse{MediaID: "d1"})
		}
	}))
	defer ts.Close()

	dir := t.TempDir()
	content := filepath.Join(dir, "a.html")
	_ = os.WriteFile(content, []byte("<p>hi</p>"), 0o600)
	cover := filepath.Join(dir, "cover.png")
	_ = os.WriteFile(cover, []byte("IMG"), 0o600)

	out, err := run(t, ts.URL, "draft", "create", "--title", "T", "--content-file", content, "--cover", cover)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"media_id": "d1"`) {
		t.Fatalf("out = %s", out)
	}
	if len(paths) != 2 {
		t.Fatalf("expected upload then draft, got %v", paths)
	}
}

func TestDraftCreateCmdWithCoverMediaID(t *testing.T) {
	var uploadCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/materials" {
			uploadCalled = true
		}
		var req types.DraftRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Articles[0].ThumbMediaID != "existingMedia" {
			t.Errorf("thumb = %q", req.Articles[0].ThumbMediaID)
		}
		_ = json.NewEncoder(w).Encode(types.DraftResponse{MediaID: "d2"})
	}))
	defer ts.Close()

	content := filepath.Join(t.TempDir(), "a.html")
	_ = os.WriteFile(content, []byte("<p>x</p>"), 0o600)

	_, err := run(t, ts.URL, "draft", "create", "--title", "T", "--content-file", content, "--cover", "existingMedia")
	if err != nil {
		t.Fatal(err)
	}
	if uploadCalled {
		t.Fatal("cover is a media_id, should not upload")
	}
}
