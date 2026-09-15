package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// 只读查询：返回 mp-server 透传的微信原始响应。

func (c *Client) getRaw(path string, q url.Values) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	err = c.do(req, &out)
	return out, err
}

func pageQuery(offset, count int, withContent bool) url.Values {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("count", strconv.Itoa(count))
	if withContent {
		q.Set("no_content", "0")
	} else {
		q.Set("no_content", "1")
	}
	return q
}

// ListPublished 分页列出已发布图文。
func (c *Client) ListPublished(offset, count int, withContent bool) (json.RawMessage, error) {
	return c.getRaw("/api/v1/published", pageQuery(offset, count, withContent))
}

// GetPublished 读取一篇已发布图文。
func (c *Client) GetPublished(articleID string) (json.RawMessage, error) {
	return c.getRaw("/api/v1/published/"+url.PathEscape(articleID), nil)
}

// ListDrafts 分页列出草稿。
func (c *Client) ListDrafts(offset, count int, withContent bool) (json.RawMessage, error) {
	return c.getRaw("/api/v1/drafts", pageQuery(offset, count, withContent))
}

// GetDraft 读取一篇草稿。
func (c *Client) GetDraft(mediaID string) (json.RawMessage, error) {
	return c.getRaw("/api/v1/drafts/"+url.PathEscape(mediaID), nil)
}

// DataCube 调用数据统计接口，日期为 YYYY-MM-DD。
func (c *Client) DataCube(api, begin, end string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("begin", begin)
	q.Set("end", end)
	return c.getRaw("/api/v1/datacube/"+url.PathEscape(api), q)
}
