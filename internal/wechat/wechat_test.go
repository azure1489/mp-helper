package wechat

import (
	"sync"
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

// 并发命中同一 appid 的双检锁路径；配合 `go test -race` 可暴露数据竞争。
func TestManagerConcurrentGet(t *testing.T) {
	m := NewManager(cache.NewMemory())
	const n = 50
	got := make([]interface{}, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			got[i] = m.get("sameapp", "samesec")
		}(i)
	}
	wg.Wait()
	for i := 1; i < n; i++ {
		if got[i] != got[0] {
			t.Fatalf("concurrent get returned different instances for same appid")
		}
	}
}
