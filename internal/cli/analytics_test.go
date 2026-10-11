package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// runSplit 与 run 相同，但分开 stdout 与 stderr。
func runSplit(t *testing.T, baseURL string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("MP_HELPER_API_URL", baseURL)
	t.Setenv("MP_HELPER_API_KEY", "k1")
	cmd := NewRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestAnalyticsCmds(t *testing.T) {
	var got []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery+" "+string(body))
		switch {
		case strings.HasSuffix(r.URL.Path, "/articles"):
			_, _ = w.Write([]byte(`{"articles":[{"msgid":"1_1","title":"A,B","publish_date":"2026-10-01","notified":true,"delivered_est":4000,"push_open_rate":0.01,
				"latest":{"stat_date":"2026-10-07","read_user":500,"push_read_user":40,"read_by_source":{"推荐":400}}}],
				"warnings":[{"date":"2026-10-02","api":"getarticletotaldetail","message":"boom"}]}`))
		case strings.HasSuffix(r.URL.Path, "/daily"):
			_, _ = w.Write([]byte(`{"days":[{"date":"2026-10-01","day_type":"pushed","pushed_count":1,"new_user":5,"cancel_user":1,"cancel_rate":0.001,
				"read_by_source":{"公众号消息":30},"new_by_source":{"57":3,"0":2},"msgids":["1_1","1_2"]}]}`))
		default:
			_, _ = w.Write([]byte(`{"msgid":"1_1","category":"快讯"}`))
		}
	}))
	defer ts.Close()

	out, errOut, err := runSplit(t, ts.URL, "analytics", "articles", "--begin", "2026-10-01", "--end", "2026-10-07", "--format", "csv")
	if err != nil || !strings.Contains(errOut, "boom") {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "msgid,title,") || !strings.Contains(lines[1], `1_1,"A,B",2026-10-01`) ||
		!strings.Contains(lines[1], ",4000,0.01,") || !strings.Contains(lines[1], "2026-10-07,500,40,") {
		t.Fatalf("articles csv:\n%s", out)
	}
	if strings.Contains(out, "boom") {
		t.Fatal("warnings must not go to stdout in csv mode")
	}
	hdr := strings.Split(lines[0], ",")
	if len(hdr) != 11+3*(12+6)+1 {
		t.Fatalf("header columns = %d", len(hdr))
	}

	out, err = run(t, ts.URL, "analytics", "daily", "--begin", "2026-10-01", "--end", "2026-10-01", "--format", "csv")
	if err != nil || !strings.Contains(out, "2026-10-01,pushed,1,0,5,1,,,0.001,,30,") || !strings.Contains(out, "0:2;57:3,1_1;1_2") {
		t.Fatalf("daily csv: %s %v", out, err)
	}
	if _, err := run(t, ts.URL, "analytics", "daily", "--begin", "2026-10-01", "--end", "2026-10-01", "--format", "xml"); err == nil {
		t.Fatal("want error for bad --format")
	}
	if _, err := run(t, ts.URL, "article", "meta", "set", "1_1", "--category", "快讯"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, ts.URL, "article", "meta", "set", "1_1"); err == nil {
		t.Fatal("want error when no field is set")
	}
	want := []string{
		"GET /api/v1/analytics/articles?begin=2026-10-01&end=2026-10-07 ",
		"GET /api/v1/analytics/daily?begin=2026-10-01&end=2026-10-01 ",
		`PUT /api/v1/articles/1_1/meta? {"category":"快讯"}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
