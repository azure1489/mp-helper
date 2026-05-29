package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/types"
)

func TestHealth(t *testing.T) {
	s := &Server{}
	r := gin.New()
	r.GET("/healthz", s.handleHealth)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var hr types.HealthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &hr)
	if hr.Status != "ok" {
		t.Fatalf("status field = %q", hr.Status)
	}
}
