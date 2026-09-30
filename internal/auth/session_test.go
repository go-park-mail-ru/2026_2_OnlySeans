package auth

import (
	"testing"
	"time"
)

func TestInMemorySessionStore_CreateGetDelete(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)

	s, err := store.Create(42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.UserID != 42 {
		t.Errorf("UserID = %d, want 42", got.UserID)
	}

	if err := store.Delete(s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(s.ID); err != ErrSessionNotFound {
		t.Errorf("после Delete error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestInMemorySessionStore_Expired(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)

	s, err := store.Create(1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	store.now = func() time.Time { return time.Now().Add(2 * time.Hour) }

	if _, err := store.Get(s.ID); err != ErrSessionNotFound {
		t.Errorf("просроченная сессия: error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestInMemorySessionStore_Get_Unknown(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)
	if _, err := store.Get("nope"); err != ErrSessionNotFound {
		t.Errorf("error = %v, want %v", err, ErrSessionNotFound)
	}
}
