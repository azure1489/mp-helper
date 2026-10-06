package wechat

import (
	"fmt"
	"github.com/azure1489/mp-helper/internal/types"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/util"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSDKRedirectsNeverReplayWrites(t *testing.T) {
	for _, status := range []int{301, 302, 307, 308} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			targets, writes := 0, 0
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targets++
				io.WriteString(w, `{"url":"mock","errcode":0}`)
			}))
			defer target.Close()
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/cgi-bin/token" {
					io.WriteString(w, `{"access_token":"LOCAL-TEST-TOKEN","expires_in":7200}`)
					return
				}
				writes++
				http.Redirect(w, r, target.URL, status)
			}))
			defer origin.Close()
			old := util.DefaultHTTPClient
			util.DefaultHTTPClient = NewWriteHTTPClient(time.Second)
			defer func() { util.DefaultHTTPClient = old }()
			util.SetURIModifier(func(s string) string { u, _ := url.Parse(s); return origin.URL + u.RequestURI() })
			defer util.SetURIModifier(nil)
			m := NewManager(cache.NewMemory())
			if _, err := m.UploadBodyImage("local-app", "local-secret", "sample.png", strings.NewReader("local-test")); err == nil {
				t.Fatal("redirect unexpectedly succeeded")
			}
			if err := m.UpdateDraft("local-app", "local-secret", "same-ID", 0, types.Article{}); err == nil {
				t.Fatal("redirect update unexpectedly succeeded")
			}
			if targets != 0 || writes != 2 {
				t.Fatalf("redirect replay: targets=%d writes=%d", targets, writes)
			}
		})
	}
	if NewWriteHTTPClient(0) == http.DefaultClient {
		t.Fatal("must not mutate shared http.DefaultClient")
	}
}
