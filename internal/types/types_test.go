package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArticleJSONTags(t *testing.T) {
	b, err := json.Marshal(Article{Title: "t", Content: "c", ThumbMediaID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"title"`, `"content"`, `"thumb_media_id"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing json tag %s in %s", want, s)
		}
	}
}

func TestDraftRequestRoundTrip(t *testing.T) {
	in := DraftRequest{Articles: []Article{{Title: "hi", Content: "<p>x</p>", ThumbMediaID: "mid"}}}
	b, _ := json.Marshal(in)
	var out DraftRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Articles) != 1 || out.Articles[0].Title != "hi" {
		t.Fatalf("round trip mismatch: %+v", out)
	}
}
