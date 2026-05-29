package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/types"
)

func TestHealth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	}))
	defer ts.Close()

	c := New(ts.URL, "")
	res, err := c.Health()
	if err != nil || res.Status != "ok" {
		t.Fatalf("res = %+v err = %v", res, err)
	}
}

func TestUploadMaterialSendsMultipartAndAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer k1" {
			t.Errorf("auth = %q", got)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q", ct)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Errorf("file field missing: %v", err)
		}
		_ = json.NewEncoder(w).Encode(types.MaterialResponse{MediaID: "m1", URL: "u1"})
	}))
	defer ts.Close()

	p := filepath.Join(t.TempDir(), "a.png")
	_ = os.WriteFile(p, []byte("DATA"), 0o600)

	c := New(ts.URL, "k1")
	res, err := c.UploadMaterial(p, "image")
	if err != nil || res.MediaID != "m1" {
		t.Fatalf("res = %+v err = %v", res, err)
	}
}

func TestCreateDraftSendsJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.DraftRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Articles) != 1 || req.Articles[0].Title != "T" {
			t.Errorf("body articles = %+v", req.Articles)
		}
		_ = json.NewEncoder(w).Encode(types.DraftResponse{MediaID: "d1"})
	}))
	defer ts.Close()

	c := New(ts.URL, "k1")
	res, err := c.CreateDraft(types.DraftRequest{Articles: []types.Article{{Title: "T", Content: "c", ThumbMediaID: "m"}}})
	if err != nil || res.MediaID != "d1" {
		t.Fatalf("res = %+v err = %v", res, err)
	}
}

func TestErrorResponseParsed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(types.ErrorResponse{Error: types.ErrorBody{Code: "unauthorized", Message: "bad key"}})
	}))
	defer ts.Close()

	c := New(ts.URL, "x")
	if _, err := c.Health(); err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
