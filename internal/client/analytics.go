package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/azure1489/mp-helper/internal/types"
)

// 数据分析与文章补录信息。

func (c *Client) getJSON(path string, q url.Values, out interface{}) error {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func rangeQuery(begin, end string) url.Values {
	q := url.Values{}
	q.Set("begin", begin)
	q.Set("end", end)
	return q
}

// AnalyticsArticles 汇总 [begin, end] 内发表的每篇文章；series 为 true 时附逐日明细。
func (c *Client) AnalyticsArticles(begin, end string, series bool) (types.ArticlesReport, error) {
	var out types.ArticlesReport
	q := rangeQuery(begin, end)
	if series {
		q.Set("series", "1")
	}
	return out, c.getJSON("/api/v1/analytics/articles", q, &out)
}

// AnalyticsDaily 汇总 [begin, end] 每天的粉丝增减、阅读与发文情况。
func (c *Client) AnalyticsDaily(begin, end string) (types.DailyReport, error) {
	var out types.DailyReport
	return out, c.getJSON("/api/v1/analytics/daily", rangeQuery(begin, end), &out)
}

// ListArticleMeta 列出全部文章补录信息。
func (c *Client) ListArticleMeta() ([]types.ArticleMeta, error) {
	var out []types.ArticleMeta
	return out, c.getJSON("/api/v1/articles/meta", nil, &out)
}

// SetArticleMeta 补录一篇文章的推送时刻、分类或备注（只写本服务的数据库，不触达微信）。
func (c *Client) SetArticleMeta(msgid string, in types.ArticleMetaRequest) (types.ArticleMeta, error) {
	var out types.ArticleMeta
	body, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest("PUT", c.baseURL+"/api/v1/articles/"+url.PathEscape(msgid)+"/meta", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	return out, c.do(req, &out)
}
