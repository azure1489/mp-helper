package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuccessfulStatusTruncatedWriteReplyIsUncertain(t *testing.T) {
	writes := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writes++; io.WriteString(w, `{"media_id":`) }))
	defer ts.Close()
	c := New(ts.URL, "LOCAL-MOCK-KEY")
	r, _ := http.NewRequest("PUT", ts.URL+"/api/v1/drafts", strings.NewReader("{}"))
	var out map[string]interface{}
	err := c.doWrite(r, &out)
	if err == nil || !strings.Contains(err.Error(), "uncertain") || !strings.Contains(err.Error(), "no automatic retry") {
		t.Fatalf("wrong classification: %v", err)
	}
	if writes != 1 {
		t.Fatalf("writes=%d", writes)
	}
}
