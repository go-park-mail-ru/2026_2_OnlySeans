package reviews_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/reviews"
)

const (
	validTitle   = "Отлично"
	validContent = "Посмотрела фильм два раза, и второй раз понравился даже больше первого."
)

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		wantErr error
	}{
		{"обычный отзыв", validTitle, validContent, nil},
		{"пустой заголовок", "", validContent, reviews.ErrInvalidTitle},
		{"заголовок 2 символа", "ab", validContent, reviews.ErrInvalidTitle},
		{"заголовок 3 символа", "abc", validContent, nil},
		{"заголовок 255 символов", strings.Repeat("a", 255), validContent, nil},
		{"заголовок 256 символов", strings.Repeat("a", 256), validContent, reviews.ErrInvalidTitle},
		{"кириллица считается символами, а не байтами: 2 буквы", "Да", validContent, reviews.ErrInvalidTitle},
		{"кириллица считается символами, а не байтами: 3 буквы", "Ого", validContent, nil},
		{"заголовок 255 кириллических символов", strings.Repeat("я", 255), validContent, nil},
		{"текст 49 символов", validTitle, strings.Repeat("я", 49), reviews.ErrInvalidContent},
		{"текст 50 символов", validTitle, strings.Repeat("я", 50), nil},
		{"текст 20000 символов", validTitle, strings.Repeat("я", 20000), nil},
		{"текст 20001 символ", validTitle, strings.Repeat("я", 20001), reviews.ErrInvalidContent},
		{"заголовок проверяется раньше текста", "", "", reviews.ErrInvalidTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reviews.ValidateInput(reviews.ReviewInput{Title: tt.title, Content: tt.content})

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
