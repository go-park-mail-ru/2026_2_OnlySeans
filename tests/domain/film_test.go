package domain_test

import (
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func filmTestPointer[T any](value T) *T { return &value }

func TestCreateFilmInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*domain.CreateFilmInput)
		invalid bool
	}{
		{"minimal movie", func(in *domain.CreateFilmInput) {}, false},
		{"empty title", func(in *domain.CreateFilmInput) { in.Title = "" }, true},
		{"blank title", func(in *domain.CreateFilmInput) { in.Title = " \t\n" }, true},
		{"one character title", func(in *domain.CreateFilmInput) { in.Title = "Я" }, false},
		{"255 unicode characters", func(in *domain.CreateFilmInput) { in.Title = strings.Repeat("Я", 255) }, false},
		{"256 unicode characters", func(in *domain.CreateFilmInput) { in.Title = strings.Repeat("Я", 256) }, true},
		{"year minimum", func(in *domain.CreateFilmInput) { in.ReleaseYear = 1888 }, false},
		{"year below minimum", func(in *domain.CreateFilmInput) { in.ReleaseYear = 1887 }, true},
		{"year maximum", func(in *domain.CreateFilmInput) { in.ReleaseYear = 2200 }, false},
		{"year above maximum", func(in *domain.CreateFilmInput) { in.ReleaseYear = 2201 }, true},
		{"unknown type", func(in *domain.CreateFilmInput) { in.FilmType = "documentary" }, true},
		{"empty type", func(in *domain.CreateFilmInput) { in.FilmType = "" }, true},
		{"age zero", func(in *domain.CreateFilmInput) { in.AgeLimit = 0 }, false},
		{"age six", func(in *domain.CreateFilmInput) { in.AgeLimit = 6 }, false},
		{"age twelve", func(in *domain.CreateFilmInput) { in.AgeLimit = 12 }, false},
		{"age eighteen", func(in *domain.CreateFilmInput) { in.AgeLimit = 18 }, false},
		{"invalid intermediate age", func(in *domain.CreateFilmInput) { in.AgeLimit = 7 }, true},
		{"negative age", func(in *domain.CreateFilmInput) { in.AgeLimit = -1 }, true},
		{"duration minimum", func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](1) }, false},
		{"duration maximum", func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6000) }, false},
		{"duration zero", func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](0) }, true},
		{"duration negative", func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](-1) }, true},
		{"duration above maximum", func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6001) }, true},
		{"series without duration", func(in *domain.CreateFilmInput) { in.FilmType = domain.FilmTypeSeries }, false},
		{"series with duration", func(in *domain.CreateFilmInput) {
			in.FilmType = domain.FilmTypeSeries
			in.DurationMin = filmTestPointer[int16](90)
		}, true},
		{"empty original title", func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer("") }, true},
		{"original title maximum", func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 255)) }, false},
		{"original title above maximum", func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 256)) }, true},
		{"original title trailing space counts", func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("A", 255) + " ") }, true},
		{"empty description", func(in *domain.CreateFilmInput) { in.Description = filmTestPointer("") }, false},
		{"description maximum", func(in *domain.CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10000)) }, false},
		{"description above maximum", func(in *domain.CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10001)) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := domain.CreateFilmInput{Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16}
			tt.change(&input)
			err := input.Validate()
			if tt.invalid {
				assert.ErrorIs(t, err, domain.ErrInvalidFilmInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
