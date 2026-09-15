package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadEndpoints(t *testing.T) {
	var got []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer k1" {
			t.Errorf("method=%s auth=%q", r.Method, r.Header.Get("Authorization"))
		}
		got = append(got, r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"total_count":2}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "k1")
	calls := []func() ([]byte, error){
		func() ([]byte, error) { return c.ListPublished(0, 20, false) },
		func() ([]byte, error) { return c.GetPublished("a/b") },
		func() ([]byte, error) { return c.ListDrafts(10, 5, true) },
		func() ([]byte, error) { return c.GetDraft("m1") },
		func() ([]byte, error) { return c.DataCube("getusersummary", "2026-09-08", "2026-09-14") },
	}
	for i, call := range calls {
		raw, err := call()
		if err != nil || !strings.Contains(string(raw), "total_count") {
			t.Fatalf("call %d: raw=%s err=%v", i, raw, err)
		}
	}
	want := []string{
		"/api/v1/published?count=20&no_content=1&offset=0",
		"/api/v1/published/a%2Fb?",
		"/api/v1/drafts?count=5&no_content=0&offset=10",
		"/api/v1/drafts/m1?",
		"/api/v1/datacube/getusersummary?begin=2026-09-08&end=2026-09-14",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
