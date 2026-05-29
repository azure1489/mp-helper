package wechat

import (
	"testing"

	"github.com/silenceper/wechat/v2/cache"
)

func TestManagerCachesByAppID(t *testing.T) {
	m := NewManager(cache.NewMemory())

	a := m.get("app1", "sec1")
	b := m.get("app1", "sec1")
	if a != b {
		t.Fatal("same appid should return cached instance")
	}

	c := m.get("app2", "sec2")
	if a == c {
		t.Fatal("different appid should return different instance")
	}

	m.Invalidate("app1")
	d := m.get("app1", "sec1")
	if a == d {
		t.Fatal("after Invalidate, expected a freshly built instance")
	}
}
