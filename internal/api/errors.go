package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/types"
)

func respondError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, types.ErrorResponse{Error: types.ErrorBody{Code: code, Message: msg}})
}

func respondWechatError(c *gin.Context, msg string, errcode int64) {
	c.JSON(http.StatusBadGateway, types.ErrorResponse{Error: types.ErrorBody{
		Code:          "wechat_error",
		Message:       msg,
		WechatErrcode: errcode,
	}})
}
