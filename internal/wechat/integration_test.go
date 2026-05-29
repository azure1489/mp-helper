//go:build integration

package wechat

import (
	"os"
	"strings"
	"testing"

	"github.com/silenceper/wechat/v2/cache"
)

// 用真实公众号凭证跑：
//
//	MP_TEST_APPID=xxx MP_TEST_SECRET=yyy go test -tags integration ./internal/wechat/ -run TestIntegration
func TestIntegrationUploadMaterial(t *testing.T) {
	appid := os.Getenv("MP_TEST_APPID")
	secret := os.Getenv("MP_TEST_SECRET")
	img := os.Getenv("MP_TEST_IMAGE")
	if appid == "" || secret == "" || img == "" {
		t.Skip("set MP_TEST_APPID / MP_TEST_SECRET / MP_TEST_IMAGE to run")
	}
	f, err := os.Open(img)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := NewManager(cache.NewMemory())
	mediaID, url, err := m.UploadMaterial(appid, secret, "image", "test.png", f)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if mediaID == "" || !strings.HasPrefix(url, "http") {
		t.Fatalf("unexpected result mediaID=%q url=%q", mediaID, url)
	}
}
