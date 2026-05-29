package store

import (
	"errors"
	"testing"
)

func TestAccountCRUD(t *testing.T) {
	s := newTestStore(t)

	a, err := s.CreateAccount("gz", "appid1", "secret1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := s.GetAccount(a.ID)
	if err != nil || got.AppID != "appid1" {
		t.Fatalf("get mismatch: %+v err=%v", got, err)
	}

	name := "gz2"
	if err := s.UpdateAccount(a.ID, &name, nil, nil); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetAccount(a.ID)
	if got.Name != "gz2" || got.AppID != "appid1" {
		t.Fatalf("update mismatch: %+v", got)
	}

	list, err := s.ListAccounts()
	if err != nil || len(list) != 1 {
		t.Fatalf("list mismatch: %v len=%d", err, len(list))
	}

	if err := s.DeleteAccount(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAccount(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateAccountDuplicateName(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateAccount("dup", "a", "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount("dup", "c", "d"); err == nil {
		t.Fatal("expected unique constraint error")
	}
}
