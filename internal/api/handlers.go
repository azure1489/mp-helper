package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/azure1489/mp-helper/internal/types"
)

func (s *Server) handleUploadMaterial(c *gin.Context) {
	acc := accountFromContext(c)

	mtype := c.PostForm("type")
	if mtype == "" {
		mtype = "image"
	}
	if mtype != "image" && mtype != "thumb" {
		respondError(c, http.StatusBadRequest, "bad_request", "unsupported type: "+mtype)
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "missing file field")
		return
	}
	f, err := fh.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "cannot open uploaded file")
		return
	}
	defer f.Close()

	mediaID, url, err := s.wechat.UploadMaterial(acc.AppID, acc.AppSecret, mtype, fh.Filename, f)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	c.JSON(200, types.MaterialResponse{MediaID: mediaID, URL: url})
}

func (s *Server) handleCreateDraft(c *gin.Context) {
	acc := accountFromContext(c)

	var req types.DraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}
	if len(req.Articles) == 0 {
		respondError(c, http.StatusBadRequest, "bad_request", "articles must not be empty")
		return
	}
	for i, a := range req.Articles {
		if a.Title == "" || a.Content == "" || a.ThumbMediaID == "" {
			respondError(c, http.StatusBadRequest, "bad_request",
				fmt.Sprintf("article[%d] requires title, content and thumb_media_id", i))
			return
		}
	}

	mediaID, err := s.wechat.AddDraft(acc.AppID, acc.AppSecret, req.Articles)
	if err != nil {
		respondWechatError(c, err)
		return
	}
	c.JSON(200, types.DraftResponse{MediaID: mediaID})
}
