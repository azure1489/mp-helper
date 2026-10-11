package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azure1489/mp-helper/internal/types"
)

func TestAnalyticsEndpoints(t *testing.T) {
	var got []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.EscapedPath()+"?"+r.URL.RawQuery+" "+string(body))
		switch {
		case strings.HasSuffix(r.URL.Path, "/articles"):
			_, _ = w.Write([]byte(`{"begin":"2026-10-01","articles":[{"msgid":"1_1"}]}`))
		case strings.HasSuffix(r.URL.Path, "/daily"):
			_, _ = w.Write([]byte(`{"days":[{"date":"2026-10-01","day_type":"quiet"}]}`))
		case r.Method == "PUT":
			_, _ = w.Write([]byte(`{"msgid":"1_1","published_at":"2026-10-01 21:00"}`))
		default:
			_, _ = w.Write([]byte(`[{"msgid":"1_1"}]`))
		}
	}))
	defer ts.Close()

	c := New(ts.URL, "k1")
	ar, err := c.AnalyticsArticles("2026-10-01", "2026-10-02", true)
	if err != nil || len(ar.Articles) != 1 {
		t.Fatalf("articles: %+v %v", ar, err)
	}
	dr, err := c.AnalyticsDaily("2026-10-01", "2026-10-02")
	if err != nil || dr.Days[0].DayType != "quiet" {
		t.Fatalf("daily: %+v %v", dr, err)
	}
	pa := "2026-10-01 21:00"
	m, err := c.SetArticleMeta("1_1", types.ArticleMetaRequest{PublishedAt: &pa})
	if err != nil || m.PublishedAt != pa {
		t.Fatalf("set meta: %+v %v", m, err)
	}
	if ms, err := c.ListArticleMeta(); err != nil || len(ms) != 1 {
		t.Fatalf("list meta: %+v %v", ms, err)
	}
	want := []string{
		"GET /api/v1/analytics/articles?begin=2026-10-01&end=2026-10-02&series=1 ",
		"GET /api/v1/analytics/daily?begin=2026-10-01&end=2026-10-02 ",
		`PUT /api/v1/articles/1_1/meta? {"published_at":"2026-10-01 21:00"}`,
		"GET /api/v1/articles/meta? ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
