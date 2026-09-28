package auth

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"валидный email", "user@example.com", false},
		{"валидный email с точкой и плюсом", "user.name+tag@example.co.uk", false},
		{"пустая строка", "", true},
		{"без @", "userexample.com", true},
		{"без домена", "user@", true},
		{"без точки в домене", "user@examplecom", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateEmail(tt.email); (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"валидное имя", "svetlana", false},
		{"минимальная длина", "abc", false},
		{"слишком короткое", "ab", true},
		{"пустая строка", "", true},
		{"слишком длинное (33 символа)", "abcdefghijklmnopqrstuvwxyz1234567", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateUsername(tt.username); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"валидный пароль", "Password1", false},
		{"слишком короткий", "Ab1", true},
		{"без заглавной буквы", "password1", true},
		{"без цифры", "Password", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePassword(tt.password); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestUseCase_Register(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		username string
		password string
		wantErr  bool
	}{
		{"успешная регистрация", "new@example.com", "newuser", "Password1", false},
		{"некорректный email", "not-an-email", "someuser", "Password1", true},
		{"слабый пароль", "user2@example.com", "someuser", "weak", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewUseCase(NewInMemoryUserRepo(), nil)
			_, err := uc.Register(tt.email, tt.username, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register(%q, %q, %q) error = %v, wantErr %v", tt.email, tt.username, tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestUseCase_Register_DuplicateEmail(t *testing.T) {
	uc := NewUseCase(NewInMemoryUserRepo(), nil)

	if _, err := uc.Register("dup@example.com", "first", "Password1"); err != nil {
		t.Fatalf("первая регистрация не должна была упасть: %v", err)
	}

	_, err := uc.Register("dup@example.com", "second", "Password1")
	if err != ErrUserExists {
		t.Errorf("error = %v, want %v", err, ErrUserExists)
	}
}

func TestUseCase_Login(t *testing.T) {
	uc := NewUseCase(NewInMemoryUserRepo(), nil)
	if _, err := uc.Register("login@example.com", "loginuser", "Password1"); err != nil {
		t.Fatalf("не удалось подготовить пользователя: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"верные данные", "login@example.com", "Password1", nil},
		{"неверный пароль", "login@example.com", "WrongPass1", ErrInvalidCredentials},
		{"несуществующий email", "ghost@example.com", "Password1", ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Login(tt.email, tt.password)
			if err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
