package store

import (
	"errors"
	"testing"
	"time"
)

func strp(s string) *string { return &s }

func TestArticleMetaUpsertPartial(t *testing.T) {
	s := newTestStore(t)
	acc, err := s.CreateAccount("a", "app", "sec")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertArticleMeta(acc.ID, "100_1", strp("2026-10-05 21:00"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertArticleMeta(acc.ID, "100_1", nil, strp("攻略"), nil); err != nil {
		t.Fatal(err)
	}
	ms, err := s.ListArticleMeta(acc.ID)
	if err != nil || len(ms) != 1 {
		t.Fatalf("list: %v %v", ms, err)
	}
	if ms[0].PublishedAt != "2026-10-05 21:00" || ms[0].Category != "攻略" || ms[0].Note != "" {
		t.Fatalf("partial update lost fields: %+v", ms[0])
	}
	other, _ := s.CreateAccount("b", "app2", "sec")
	if ms, _ := s.ListArticleMeta(other.ID); len(ms) != 0 {
		t.Fatalf("meta leaked across accounts: %v", ms)
	}
}

func TestDataCubeCache(t *testing.T) {
	s := newTestStore(t)
	acc, _ := s.CreateAccount("a", "app", "sec")
	if _, _, err := s.GetCachedDataCube(acc.ID, "x", "2026-10-01"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	now := time.Unix(1760000000, 0)
	_ = s.PutCachedDataCube(acc.ID, "x", "2026-10-01", []byte(`{"a":1}`), now)
	_ = s.PutCachedDataCube(acc.ID, "x", "2026-10-01", []byte(`{"a":2}`), now.Add(time.Hour))
	body, ts, err := s.GetCachedDataCube(acc.ID, "x", "2026-10-01")
	if err != nil || string(body) != `{"a":2}` || !ts.Equal(now.Add(time.Hour)) {
		t.Fatalf("got %s %v %v", body, ts, err)
	}
}
