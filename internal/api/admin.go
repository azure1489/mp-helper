package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid id")
		return 0, false
	}
	return id, true
}

func (s *Server) handleCreateAccount(c *gin.Context) {
	var req types.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	if req.Name == "" || req.AppID == "" || req.AppSecret == "" {
		respondError(c, http.StatusBadRequest, "bad_request", "name, appid and app_secret are required")
		return
	}
	acc, err := s.store.CreateAccount(req.Name, req.AppID, req.AppSecret)
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	c.JSON(http.StatusCreated, types.AccountResponse{ID: acc.ID, Name: acc.Name, AppID: acc.AppID})
}

func (s *Server) handleListAccounts(c *gin.Context) {
	accs, err := s.store.ListAccounts()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]types.AccountResponse, 0, len(accs))
	for _, a := range accs {
		out = append(out, types.AccountResponse{ID: a.ID, Name: a.Name, AppID: a.AppID})
	}
	c.JSON(200, out)
}

func (s *Server) handleGetAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := s.store.GetAccount(id)
	if errors.Is(err, store.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	c.JSON(200, types.AccountResponse{ID: a.ID, Name: a.Name, AppID: a.AppID})
}

func (s *Server) handleUpdateAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req types.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	err := s.store.UpdateAccount(id, req.Name, req.AppID, req.AppSecret)
	if errors.Is(err, store.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	a, _ := s.store.GetAccount(id)
	s.wechat.Invalidate(a.AppID)
	c.JSON(200, types.AccountResponse{ID: a.ID, Name: a.Name, AppID: a.AppID})
}

func (s *Server) handleDeleteAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := s.store.GetAccount(id)
	if errors.Is(err, store.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if err := s.store.DeleteAccount(id); err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	s.wechat.Invalidate(a.AppID)
	c.Status(http.StatusNoContent)
}

func (s *Server) handleCreateKey(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req types.CreateKeyRequest
	_ = c.ShouldBindJSON(&req) // label 可选，忽略解析错误
	keyID, plaintext, prefix, err := s.store.CreateKey(id, req.Label)
	if errors.Is(err, store.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "account not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	c.JSON(http.StatusCreated, types.CreateKeyResponse{ID: keyID, Key: plaintext, Prefix: prefix})
}

func (s *Server) handleListKeys(c *gin.Context) {
	keys, err := s.store.ListKeys()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]types.KeyResponse, 0, len(keys))
	for _, k := range keys {
		kr := types.KeyResponse{
			ID: k.ID, AccountID: k.AccountID, Prefix: k.Prefix,
			Label: k.Label, CreatedAt: k.CreatedAt,
		}
		if k.RevokedAt.Valid {
			kr.RevokedAt = k.RevokedAt.String
		}
		out = append(out, kr)
	}
	c.JSON(200, out)
}

func (s *Server) handleRevokeKey(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := s.store.RevokeKey(id)
	if errors.Is(err, store.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "key not found or already revoked")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
