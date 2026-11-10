package auth_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
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
		{"emoji в email", "user😊@example.com", utils.ErrInvalidEmail},
		{"кириллица в email", "пользователь@example.com", utils.ErrInvalidEmail},
		{"кавычка в email", "user'name@example.com", utils.ErrInvalidEmail},
		{"пустая строка", "", utils.ErrInvalidEmail},
		{"только пробелы", "   ", utils.ErrInvalidEmail},
		{"без @", "userexample.com", utils.ErrInvalidEmail},
		{"без домена", "user@", utils.ErrInvalidEmail},
		{"без точки в домене", "user@examplecom", utils.ErrInvalidEmail},
		{"254 символа — предел колонки в базе", strings.Repeat("a", 242) + "@example.com", nil},
		{"255 символов", strings.Repeat("a", 243) + "@example.com", utils.ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateEmail(tt.email)
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
		{"имя с пробелом, дефисом и подчёркиванием", "Света User_1", false},
		{"имя с дефисом", "user-name", false},
		{"слишком короткое", "a", true},
		{"пустая строка", "", true},
		{"только пробелы", "    ", true},
		{"33 символа", strings.Repeat("a", 33), true},
		{"emoji", "user😊", true},
		{"точка", "user.name", true},
		{"кавычка", "user'name", true},
		{"слэш", "user/name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateUsername(tt.username)
			if tt.wantErr {
				assert.ErrorIs(t, err, utils.ErrInvalidUsername)
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
		{"максимум 32 символа", "Password1!Password1!Password1!", false},
		{"разрешённая пунктуация", "Password1!@#$%^&*()", false},
		{"слишком короткий", "Ab1", true},
		{"33 символа", "Password1Password1Password1Pass12", true},
		{"без заглавной буквы", "password1", true},
		{"без цифры", "Password", true},
		{"пробел", "Pass word1", true},
		{"кавычка", "Pass\"word1", true},
		{"апостроф", "Pass'word1", true},
		{"обратный апостроф", "Pass`word1", true},
		{"обратная косая черта", "Pass\\word1", true},
		{"эмодзи", "Pass😊word1", true},
		{"кириллица", "Пароль12A", true},
		{"пустой", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, utils.ErrInvalidPassword)
				return
			}
			assert.NoError(t, err)
		})
	}
}
