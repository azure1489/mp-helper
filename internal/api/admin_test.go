package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

func adminServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	s := &Server{store: st, wechat: &fakeWechat{}}
	r := gin.New()
	// 测试直接挂载 admin handler（鉴权在 Task 14 的 Router 测试里验证）。
	r.POST("/admin/accounts", s.handleCreateAccount)
	r.GET("/admin/accounts", s.handleListAccounts)
	r.GET("/admin/accounts/:id", s.handleGetAccount)
	r.PUT("/admin/accounts/:id", s.handleUpdateAccount)
	r.DELETE("/admin/accounts/:id", s.handleDeleteAccount)
	return s, r
}

func TestAdminAccountCRUD(t *testing.T) {
	_, r := adminServer(t)

	// create
	body, _ := json.Marshal(types.CreateAccountRequest{Name: "gz", AppID: "app1", AppSecret: "sec1"})
	req := httptest.NewRequest("POST", "/admin/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create status = %d body=%s", w.Code, w.Body.String())
	}
	var ar types.AccountResponse
	_ = json.Unmarshal(w.Body.Bytes(), &ar)
	if ar.ID == 0 || ar.AppID != "app1" {
		t.Fatalf("create resp = %+v", ar)
	}

	// list
	req = httptest.NewRequest("GET", "/admin/accounts", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var list []types.AccountResponse
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("list len = %d", len(list))
	}

	// get 404
	req = httptest.NewRequest("GET", "/admin/accounts/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("get missing status = %d", w.Code)
	}

	// delete
	req = httptest.NewRequest("DELETE", "/admin/accounts/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("delete status = %d", w.Code)
	}
}
