package auth_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"валидный email", "user@example.com", nil},
		{"точка и плюс", "user.name+tag@example.co.uk", nil},
		{"пробелы по краям", "  user@example.com  ", nil},
		{"пустая строка", "", auth.ErrInvalidEmail},
		{"только пробелы", "   ", auth.ErrInvalidEmail},
		{"без @", "userexample.com", auth.ErrInvalidEmail},
		{"без домена", "user@", auth.ErrInvalidEmail},
		{"без точки в домене", "user@examplecom", auth.ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidateEmail(tt.email)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
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
		{"минимальная длина", "ab", false},
		{"максимальная длина", strings.Repeat("a", 32), false},
		{"кириллица считается по символам", "Светлана", false},
		{"слишком короткое", "a", true},
		{"пустая строка", "", true},
		{"только пробелы", "    ", true},
		{"33 символа", strings.Repeat("a", 33), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidateUsername(tt.username)
			if tt.wantErr {
				assert.ErrorIs(t, err, auth.ErrInvalidUsername)
				return
			}
			assert.NoError(t, err)
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
		{"ровно 8 символов", "Passwor1", false},
		{"слишком короткий", "Ab1", true},
		{"без заглавной буквы", "password1", true},
		{"без цифры", "Password", true},
		{"пустой", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, auth.ErrInvalidPassword)
				return
			}
			assert.NoError(t, err)
		})
	}
}
