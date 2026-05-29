package api

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/types"
)

func respondError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, types.ErrorResponse{Error: types.ErrorBody{Code: code, Message: msg}})
}

var wechatErrcodeRe = regexp.MustCompile(`errcode=(\d+)`)

// parseWechatErrcode 从 silenceper 返回的错误字符串中提取数字 errcode（形如 "... errcode=40007 ..."）。
// 提取不到时返回 0。
func parseWechatErrcode(msg string) int64 {
	m := wechatErrcodeRe.FindStringSubmatch(msg)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.ParseInt(m[1], 10, 64)
	return n
}

// respondWechatError 以 502 透传微信调用错误，并尽量从错误文本中解析出 wechat_errcode。
func respondWechatError(c *gin.Context, err error) {
	msg := err.Error()
	c.JSON(http.StatusBadGateway, types.ErrorResponse{Error: types.ErrorBody{
		Code:          "wechat_error",
		Message:       msg,
		WechatErrcode: parseWechatErrcode(msg),
	}})
}
