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

// Router 装配全部路由与中间件。
func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 16 << 20 // 16 MiB

	r.GET("/healthz", s.handleHealth)

	data := r.Group("/api/v1", DataAuth(s.store))
	data.POST("/materials", s.handleUploadMaterial)
	data.POST("/drafts", s.handleCreateDraft)

	admin := r.Group("/admin", AdminAuth(s.cfg.AdminToken))
	admin.POST("/accounts", s.handleCreateAccount)
	admin.GET("/accounts", s.handleListAccounts)
	admin.GET("/accounts/:id", s.handleGetAccount)
	admin.PUT("/accounts/:id", s.handleUpdateAccount)
	admin.DELETE("/accounts/:id", s.handleDeleteAccount)
	admin.POST("/accounts/:id/keys", s.handleCreateKey)
	admin.GET("/keys", s.handleListKeys)
	admin.DELETE("/keys/:id", s.handleRevokeKey)

	return r
}
