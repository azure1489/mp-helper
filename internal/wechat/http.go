package wechat

import (
	"net/http"
	"time"
)

// NewWriteHTTPClient is installed once before serving. Redirects are forbidden:
// a 307/308 may replay a write and Referer may leak token-bearing query strings.
func NewWriteHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
