package api

import (
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
		respondWechatError(c, err.Error(), 0)
		return
	}
	c.JSON(200, types.MaterialResponse{MediaID: mediaID, URL: url})
}
