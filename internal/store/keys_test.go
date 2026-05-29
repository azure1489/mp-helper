package store

import (
	"errors"
	"strings"
	"testing"
)

func TestKeyLifecycle(t *testing.T) {
	s := newTestStore(t)
	acc, _ := s.CreateAccount("gz", "appidX", "secretX")

	id, plaintext, prefix, err := s.CreateKey(acc.ID, "ci")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 || !strings.HasPrefix(plaintext, "mpk_") || prefix != plaintext[:12] {
		t.Fatalf("bad key: id=%d plaintext=%q prefix=%q", id, plaintext, prefix)
	}

	resolved, err := s.ResolveAccountByKey(plaintext)
	if err != nil || resolved.AppID != "appidX" {
		t.Fatalf("resolve mismatch: %+v err=%v", resolved, err)
	}

	keys, _ := s.ListKeys()
	if len(keys) != 1 || keys[0].Prefix != prefix {
		t.Fatalf("list mismatch: %+v", keys)
	}

	if err := s.RevokeKey(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveAccountByKey(plaintext); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked key should not resolve, got %v", err)
	}
}

func TestResolveUnknownKey(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.ResolveAccountByKey("mpk_deadbeef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteAccountCascadesKeys(t *testing.T) {
	s := newTestStore(t)
	acc, _ := s.CreateAccount("gz", "a", "b")
	_, plaintext, _, _ := s.CreateKey(acc.ID, "")
	if err := s.DeleteAccount(acc.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveAccountByKey(plaintext); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected cascade delete, got %v", err)
	}
}
