package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/azure1489/mp-helper/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2/util"
)

// Do not reflect upstream error text: SDK/network errors may contain token URLs,
// AppSecret or bearer credentials. Only numeric WeChat codes leave the server.
func respondWriteError(c *gin.Context, err error) {
	code := "wechat_error"
	message := "upstream write failed; outcome may be uncertain; no automatic retry"
	status := http.StatusBadGateway
	var network net.Error
	if errors.As(err, &network) {
		code = "wechat_write_uncertain"
	}
	if errors.As(err, &network) && network.Timeout() {
		status = http.StatusGatewayTimeout
		code = "wechat_timeout_uncertain"
		message = "upstream write timed out; outcome uncertain; no automatic retry"
	}
	n := parseWechatErrcode(err.Error())
	var common *util.CommonError
	if errors.As(err, &common) {
		n = common.ErrCode
	}
	c.JSON(status, types.ErrorResponse{Error: types.ErrorBody{Code: code, Message: message, WechatErrcode: n}})
}

func (s *Server) handleUploadBodyImage(c *gin.Context) {
	// Bound the entire multipart request before parsing. Read parts directly to
	// avoid gin's multipart temporary files and preserve the original file bytes.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, types.BodyImageLimit+(64<<10))
	mr, err := c.Request.MultipartReader()
	if err != nil {
		respondError(c, 400, "bad_request", "invalid multipart body")
		return
	}
	var data []byte
	var filename string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				respondError(c, 413, "bad_request", "oversized multipart body")
			} else {
				respondError(c, 400, "bad_request", "invalid multipart body")
			}
			return
		}
		if part.FormName() != "file" || part.FileName() == "" || data != nil {
			part.Close()
			respondError(c, 400, "bad_request", "exactly one file field is required")
			return
		}
		filename = part.FileName()
		data, err = io.ReadAll(io.LimitReader(part, types.BodyImageLimit))
		part.Close()
		if len(data) >= types.BodyImageLimit {
			respondError(c, 413, "bad_request", "body image must be smaller than 1 MB")
			return
		}
		if err != nil {
			respondError(c, 400, "bad_request", "invalid file body")
			return
		}
	}
	if err := types.ValidateBodyImage(data); err != nil {
		respondError(c, 400, "bad_request", err.Error())
		return
	}
	acc := accountFromContext(c)
	url, err := s.wechat.UploadBodyImage(acc.AppID, acc.AppSecret, filename, bytes.NewReader(data))
	if err != nil {
		respondWriteError(c, err)
		return
	}
	c.JSON(200, types.BodyImageResponse{URL: url})
}

func (s *Server) handleUpdateDraft(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
	var req types.DraftUpdateRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(c, 400, "bad_request", "invalid json body")
		return
	}
	var extra interface{}
	if decoder.Decode(&extra) != io.EOF {
		respondError(c, 400, "bad_request", "invalid trailing json")
		return
	}
	article, err := req.Validate()
	if err != nil {
		respondError(c, 400, "bad_request", err.Error())
		return
	}
	acc := accountFromContext(c)
	if err := s.wechat.UpdateDraft(acc.AppID, acc.AppSecret, req.MediaID, *req.Index, article); err != nil {
		respondWriteError(c, err)
		return
	}
	c.JSON(200, types.DraftResponse{MediaID: req.MediaID})
}
