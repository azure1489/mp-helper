package api

import (
	"encoding/json"
	"errors"
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
	respondWechatError(c, errors.New("AddDraft Error , errcode=40007 , errmsg=invalid media_id"))

	if w.Code != 502 {
		t.Fatalf("status = %d", w.Code)
	}
	var er types.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &er)
	if er.Error.Code != "wechat_error" || er.Error.WechatErrcode != 40007 {
		t.Fatalf("body = %+v", er)
	}
}

func TestParseWechatErrcode(t *testing.T) {
	cases := map[string]int64{
		"AddDraft Error , errcode=40007 , errmsg=x": 40007,
		"some failure errcode=0 ok":                 0,
		"no code here":                              0,
		"errcode=45009 rate limited":                45009,
	}
	for in, want := range cases {
		if got := parseWechatErrcode(in); got != want {
			t.Errorf("parseWechatErrcode(%q) = %d want %d", in, got, want)
		}
	}
}
