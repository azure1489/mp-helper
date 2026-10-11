package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/azure1489/mp-helper/internal/analytics"
	"github.com/azure1489/mp-helper/internal/config"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

func TestNextArchiveAt(t *testing.T) {
	at := func(s string) time.Time {
		v, _ := time.ParseInLocation("2006-01-02 15:04", s, analytics.CST)
		return v
	}
	cases := [][2]string{
		{"2026-10-12 08:00", "2026-10-12 09:30"},
		{"2026-10-12 09:30", "2026-10-13 09:30"},
		{"2026-10-12 23:59", "2026-10-13 09:30"},
	}
	for _, c := range cases {
		if got := nextArchiveAt(at(c[0]), "09:30"); !got.Equal(at(c[1])) {
			t.Fatalf("%s: got %s want %s", c[0], got, c[1])
		}
	}
}

func TestArchiveEndpoints(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.CreateAccount("a", "appidA", "secretA"); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AdminToken: "adm", Archive: config.ArchiveConfig{Enabled: true, Time: "09:30", MaxCalls: 5}}
	fw := &fakeWechat{readRaw: json.RawMessage(`{"list":[]}`)}
	r := NewServer(cfg, st, fw).Router()
	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer adm")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := do("POST", "/admin/archive/run"); w.Code != 202 {
		t.Fatalf("run: %d %s", w.Code, w.Body.String())
	}
	var stt types.ArchiveStatus
	for i := 0; i < 100; i++ {
		_ = json.Unmarshal(do("GET", "/admin/archive").Body.Bytes(), &stt)
		if !stt.Running && stt.LastFinished != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if stt.Running || len(stt.Results) != 1 || stt.Results[0].Calls != 5 || stt.Results[0].Stopped == "" {
		t.Fatalf("status: %+v", stt)
	}
	// 存档写进了 SQLite 缓存。
	if _, _, err := st.GetCachedDataCube(1, "getusersummary", "2025-11-01"); err != nil {
		t.Fatalf("archived row missing: %v", err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/admin/archive/run", nil))
	if w.Code != 401 {
		t.Fatalf("archive run without token: %d", w.Code)
	}
}
