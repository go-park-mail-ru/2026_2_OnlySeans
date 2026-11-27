package utils_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "валидный email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "точка и плюс",
			email:   "user.name+tag@example.co.uk",
			wantErr: nil,
		},
		{
			name:    "пробелы по краям",
			email:   "  user@example.com  ",
			wantErr: nil,
		},
		{
			name:    "emoji в email",
			email:   "user😊@example.com",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "кириллица в email",
			email:   "пользователь@example.com",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "кавычка в email",
			email:   "user'name@example.com",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "пустая строка",
			email:   "",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "только пробелы",
			email:   "   ",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "без @",
			email:   "userexample.com",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "без домена",
			email:   "user@",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "без точки в домене",
			email:   "user@examplecom",
			wantErr: utils.ErrInvalidEmail,
		},
		{
			name:    "254 символа — предел колонки в базе",
			email:   strings.Repeat("a", 242) + "@example.com",
			wantErr: nil,
		},
		{
			name:    "255 символов",
			email:   strings.Repeat("a", 243) + "@example.com",
			wantErr: utils.ErrInvalidEmail,
		},
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
		{
			name:     "валидное имя",
			username: "svetlana",
			wantErr:  false,
		},
		{
			name:     "минимальная длина",
			username: "ab",
			wantErr:  false,
		},
		{
			name:     "максимальная длина",
			username: strings.Repeat("a", 32),
			wantErr:  false,
		},
		{
			name:     "кириллица считается по символам",
			username: "Светлана",
			wantErr:  false,
		},
		{
			name:     "имя с пробелом, дефисом и подчёркиванием",
			username: "Света User_1",
			wantErr:  false,
		},
		{
			name:     "имя с дефисом",
			username: "user-name",
			wantErr:  false,
		},
		{
			name:     "слишком короткое",
			username: "a",
			wantErr:  true,
		},
		{
			name:     "пустая строка",
			username: "",
			wantErr:  true,
		},
		{
			name:     "только пробелы",
			username: "    ",
			wantErr:  true,
		},
		{
			name:     "33 символа",
			username: strings.Repeat("a", 33),
			wantErr:  true,
		},
		{
			name:     "emoji",
			username: "user😊",
			wantErr:  true,
		},
		{
			name:     "точка",
			username: "user.name",
			wantErr:  true,
		},
		{
			name:     "кавычка",
			username: "user'name",
			wantErr:  true,
		},
		{
			name:     "слэш",
			username: "user/name",
			wantErr:  true,
		},
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
		{
			name:     "валидный пароль",
			password: "Password1",
			wantErr:  false,
		},
		{
			name:     "ровно 8 символов",
			password: "Passwor1",
			wantErr:  false,
		},
		{
			name:     "максимум 32 символа",
			password: "Password1!Password1!Password1!",
			wantErr:  false,
		},
		{
			name:     "разрешённая пунктуация",
			password: "Password1!@#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "слишком короткий",
			password: "Ab1",
			wantErr:  true,
		},
		{
			name:     "33 символа",
			password: "Password1Password1Password1Pass12",
			wantErr:  true,
		},
		{
			name:     "без заглавной буквы",
			password: "password1",
			wantErr:  true,
		},
		{
			name:     "без цифры",
			password: "Password",
			wantErr:  true,
		},
		{
			name:     "пробел",
			password: "Pass word1",
			wantErr:  true,
		},
		{
			name:     "кавычка",
			password: "Pass\"word1",
			wantErr:  true,
		},
		{
			name:     "апостроф",
			password: "Pass'word1",
			wantErr:  true,
		},
		{
			name:     "обратный апостроф",
			password: "Pass`word1",
			wantErr:  true,
		},
		{
			name:     "обратная косая черта",
			password: "Pass\\word1",
			wantErr:  true,
		},
		{
			name:     "эмодзи",
			password: "Pass😊word1",
			wantErr:  true,
		},
		{
			name:     "кириллица",
			password: "Пароль12A",
			wantErr:  true,
		},
		{
			name:     "пустой",
			password: "",
			wantErr:  true,
		},
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
