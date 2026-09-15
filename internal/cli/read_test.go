package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadCmds(t *testing.T) {
	var got []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Path+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"total_count":40,"item":[]}`))
	}))
	defer ts.Close()

	out, err := run(t, ts.URL, "published", "list", "--count", "5")
	if err != nil || !strings.Contains(out, `"total_count": 40`) {
		t.Fatalf("published list: out=%s err=%v", out, err)
	}
	if _, err := run(t, ts.URL, "stats", "getusersummary", "--begin", "2026-09-08", "--end", "2026-09-14"); err != nil {
		t.Fatalf("stats: %v", err)
	}
	if _, err := run(t, ts.URL, "draft", "get", "m1"); err != nil {
		t.Fatalf("draft get: %v", err)
	}
	want := []string{
		"/api/v1/published?count=5&no_content=1&offset=0",
		"/api/v1/datacube/getusersummary?begin=2026-09-08&end=2026-09-14",
		"/api/v1/drafts/m1?",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestStatsCmdRequiresDates(t *testing.T) {
	if _, err := run(t, "http://127.0.0.1:1", "stats", "getusersummary"); err == nil {
		t.Fatal("want error when --begin/--end missing")
	}
}
