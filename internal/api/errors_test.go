package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/types"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	respondError(c, 400, "bad_request", "boom")

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	var er types.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if er.Error.Code != "bad_request" || er.Error.Message != "boom" {
		t.Fatalf("body = %+v", er)
	}
}

func TestRespondWechatError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	respondWechatError(c, "weixin failed", 40007)

	if w.Code != 502 {
		t.Fatalf("status = %d", w.Code)
	}
	var er types.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &er)
	if er.Error.Code != "wechat_error" || er.Error.WechatErrcode != 40007 {
		t.Fatalf("body = %+v", er)
	}
}
