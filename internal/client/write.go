package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/azure1489/mp-helper/internal/types"
)

// Single write, bounded wait, no retry or create fallback. Disable redirects
// (307/308 could replay a write or forward the business key to another host).
func (c *Client) doWrite(req *http.Request, out interface{}) error {
	copyClient := *c.http
	copyClient.Timeout = 60 * time.Second
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := copyClient.Do(req)
	if err != nil {
		var n net.Error
		if errors.As(err, &n) && n.Timeout() {
			return fmt.Errorf("write timeout: outcome uncertain; no automatic retry")
		}
		return fmt.Errorf("write transport failure: outcome uncertain; no automatic retry")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var er types.ErrorResponse
		_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&er)
		// Never reflect arbitrary upstream/server text or URLs in CLI output.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// 4xx 是服务端在调用微信之前的本地拒绝（参数、大小、鉴权），没有写入。
			return fmt.Errorf("write rejected (%d) code=%s, wechat_errcode=%d; nothing was written", resp.StatusCode, er.Error.Code, er.Error.WechatErrcode)
		}
		return fmt.Errorf("write failed (%d) code=%s, wechat_errcode=%d; outcome may be uncertain; no automatic retry", resp.StatusCode, er.Error.Code, er.Error.WechatErrcode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(out); err != nil {
		return fmt.Errorf("write response unreadable: outcome uncertain; no automatic retry")
	}
	return nil
}

func (c *Client) UploadBodyImage(path string) (types.BodyImageResponse, error) {
	var out types.BodyImageResponse
	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, types.BodyImageLimit))
	if err != nil {
		return out, err
	}
	if err := types.ValidateBodyImage(data); err != nil {
		return out, err
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return out, err
	}
	if _, err = part.Write(data); err != nil {
		return out, err
	}
	if err = writer.Close(); err != nil {
		return out, err
	}
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/body-images", &buf)
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return out, c.doWrite(req, &out)
}

func (c *Client) UpdateDraft(in types.DraftUpdateRequest) (types.DraftResponse, error) {
	var out types.DraftResponse
	if _, err := in.Validate(); err != nil {
		return out, err
	}
	data, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest("PUT", c.baseURL+"/api/v1/drafts", bytes.NewReader(data))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	return out, c.doWrite(req, &out)
}
