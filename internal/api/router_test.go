package api

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/azure1489/mp-helper/internal/config"
	"github.com/azure1489/mp-helper/internal/store"
)

func TestRouterWiring(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "router.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	cfg := &config.Config{AdminToken: "adm"}
	s := NewServer(cfg, st, &fakeWechat{})
	r := s.Router()

	// health 开放
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("health status = %d", w.Code)
	}

	// 数据面无 key → 401
	req = httptest.NewRequest("POST", "/api/v1/drafts", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("drafts without key status = %d want 401", w.Code)
	}

	// 管理面无 token → 401
	req = httptest.NewRequest("GET", "/admin/accounts", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("admin without token status = %d want 401", w.Code)
	}

	// 管理面带 token → 200
	req = httptest.NewRequest("GET", "/admin/accounts", nil)
	req.Header.Set("Authorization", "Bearer adm")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("admin with token status = %d want 200", w.Code)
	}
}
