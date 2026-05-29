package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/store"
)

const ctxAccountKey = "account"

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// DataAuth 校验业务 API Key，成功后把对应账号放入 context。
func DataAuth(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearerToken(c)
		if tok == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			c.Abort()
			return
		}
		acc, err := s.ResolveAccountByKey(tok)
		if errors.Is(err, store.ErrNotFound) {
			respondError(c, http.StatusUnauthorized, "unauthorized", "invalid or revoked api key")
			c.Abort()
			return
		}
		if err != nil {
			respondError(c, http.StatusInternalServerError, "internal", "auth lookup failed")
			c.Abort()
			return
		}
		c.Set(ctxAccountKey, *acc)
		c.Next()
	}
}

func accountFromContext(c *gin.Context) store.Account {
	return c.MustGet(ctxAccountKey).(store.Account)
}

// AdminAuth 用固定 admin token 做常量时间比较。
func AdminAuth(token string) gin.HandlerFunc {
	want := []byte(token)
	return func(c *gin.Context) {
		got := []byte(bearerToken(c))
		if len(got) == 0 || subtle.ConstantTimeCompare(got, want) != 1 {
			respondError(c, http.StatusUnauthorized, "unauthorized", "invalid admin token")
			c.Abort()
			return
		}
		c.Next()
	}
}
