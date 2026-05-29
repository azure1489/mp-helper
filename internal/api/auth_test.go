package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/store"
)

func seedStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	acc, _ := s.CreateAccount("gz", "appidA", "secretA")
	_, key, _, _ := s.CreateKey(acc.ID, "")
	return s, key
}

func TestDataAuth(t *testing.T) {
	s, key := seedStore(t)
	r := gin.New()
	r.GET("/x", DataAuth(s), func(c *gin.Context) {
		c.String(200, accountFromContext(c).AppID)
	})

	cases := []struct {
		name   string
		header string
		status int
		body   string
	}{
		{"no header", "", 401, ""},
		{"bad key", "Bearer mpk_nope", 401, ""},
		{"valid", "Bearer " + key, 200, "appidA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status = %d want %d", w.Code, tc.status)
			}
			if tc.body != "" && w.Body.String() != tc.body {
				t.Fatalf("body = %q want %q", w.Body.String(), tc.body)
			}
		})
	}
}

func TestAdminAuth(t *testing.T) {
	r := gin.New()
	r.GET("/a", AdminAuth("topsecret"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/a", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/a", nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status = %d want 401", w.Code)
	}

	req = httptest.NewRequest("GET", "/a", nil)
	req.Header.Set("Authorization", "Bearer topsecret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("valid token status = %d", w.Code)
	}
}
