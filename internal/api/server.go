package api

import (
	"github.com/gin-gonic/gin"

	"github.com/azure1489/mp-helper/internal/config"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
	"github.com/azure1489/mp-helper/internal/wechat"
)

// Server 持有所有 handler 依赖。
type Server struct {
	cfg    *config.Config
	store  *store.Store
	wechat wechat.Service
}

func NewServer(cfg *config.Config, s *store.Store, w wechat.Service) *Server {
	return &Server{cfg: cfg, store: s, wechat: w}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(200, types.HealthResponse{Status: "ok"})
}
