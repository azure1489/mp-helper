// Package client 是 mp-cli 调用 mp-server 的 HTTP 客户端。
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/azure1489/mp-helper/internal/types"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

func (c *Client) do(req *http.Request, out interface{}) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var er types.ErrorResponse
		if json.Unmarshal(body, &er) == nil && er.Error.Message != "" {
			return fmt.Errorf("server error (%d) %s: %s", resp.StatusCode, er.Error.Code, er.Error.Message)
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

func (c *Client) Health() (types.HealthResponse, error) {
	var out types.HealthResponse
	req, err := http.NewRequest("GET", c.baseURL+"/healthz", nil)
	if err != nil {
		return out, err
	}
	return out, c.do(req, &out)
}

func (c *Client) UploadMaterial(path, mediaType string) (types.MaterialResponse, error) {
	var out types.MaterialResponse
	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if mediaType != "" {
		_ = mw.WriteField("type", mediaType)
	}
	fw, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return out, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return out, err
	}
	if err := mw.Close(); err != nil {
		return out, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/materials", &buf)
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return out, c.do(req, &out)
}

func (c *Client) CreateDraft(in types.DraftRequest) (types.DraftResponse, error) {
	var out types.DraftResponse
	body, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/drafts", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	return out, c.do(req, &out)
}
