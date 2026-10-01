package auth

import (
	"context"
	"testing"
)

func TestInMemoryUserRepo_Create(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		existingUser string
		email        string
		wantErr      error
	}{
		{"новый пользователь создаётся без ошибок", "", "new@example.com", nil},
		{"повторный email возвращает ErrUserExists", "dup@example.com", "dup@example.com", ErrUserExists},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryUserRepo()

			if tt.existingUser != "" {
				if _, err := repo.Create(ctx, tt.existingUser, "existing", "hash"); err != nil {
					t.Fatalf("failed to set up existing user: %v", err)
				}
			}

			_, err := repo.Create(ctx, tt.email, "username", "hash")
			if err != tt.wantErr {
				t.Errorf("Create(%q) error = %v, want %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryUserRepo_GetByEmail(t *testing.T) {
	repo := NewInMemoryUserRepo()
	ctx := context.Background()

	createdID, err := repo.Create(ctx, "find@example.com", "finduser", "hash123")
	if err != nil {
		t.Fatalf("failed to create user for test: %v", err)
	}

	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"существующий пользователь находится", "find@example.com", nil},
		{"несуществующий email возвращает ErrUserNotFound", "ghost@example.com", ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByEmail(ctx, tt.email)
			if err != tt.wantErr {
				t.Fatalf("GetEmail(%q) error = %v, want %v", tt.email, err, tt.wantErr)
			}
			if err == nil && user.ID != createdID {
				t.Errorf("GetByEmail returned user with ID %d, want %d", user.ID, createdID)
			}
		})
	}
}
