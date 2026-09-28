package auth

import "testing"

func TestInMemoryUserRepo_Create(t *testing.T) {
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
				if _, err := repo.Create(tt.existingUser, "existing", "hash"); err != nil {
					t.Fatalf("не удалось подготовить существующего пользователя: %v", err)
				}
			}

			_, err := repo.Create(tt.email, "username", "hash")
			if err != tt.wantErr {
				t.Errorf("Create(%q) error = %v, want %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryUserRepo_FindByEmail(t *testing.T) {
	repo := NewInMemoryUserRepo()
	created, err := repo.Create("find@example.com", "finduser", "hash123")
	if err != nil {
		t.Fatalf("не удалось создать пользователя для теста: %v", err)
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
			user, err := repo.FindByEmail(tt.email)
			if err != tt.wantErr {
				t.Fatalf("FindByEmail(%q) error = %v, want %v", tt.email, err, tt.wantErr)
			}
			if err == nil && user.ID != created.ID {
				t.Errorf("FindByEmail вернул пользователя с ID %d, ожидали %d", user.ID, created.ID)
			}
		})
	}
}
