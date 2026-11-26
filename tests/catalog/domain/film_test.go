package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

func filmTestPointer[T any](value T) *T { return &value }

func TestCreateFilmInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*domain.CreateFilmInput)
		invalid bool
	}{
		{
			name:    "minimal movie",
			change:  func(in *domain.CreateFilmInput) {},
			invalid: false,
		},
		{
			name:    "empty title",
			change:  func(in *domain.CreateFilmInput) { in.Title = "" },
			invalid: true,
		},
		{
			name:    "blank title",
			change:  func(in *domain.CreateFilmInput) { in.Title = " \t\n" },
			invalid: true,
		},
		{
			name:    "one character title",
			change:  func(in *domain.CreateFilmInput) { in.Title = "Я" },
			invalid: false,
		},
		{
			name:    "255 unicode characters",
			change:  func(in *domain.CreateFilmInput) { in.Title = strings.Repeat("Я", 255) },
			invalid: false,
		},
		{
			name:    "256 unicode characters",
			change:  func(in *domain.CreateFilmInput) { in.Title = strings.Repeat("Я", 256) },
			invalid: true,
		},
		{
			name:    "year minimum",
			change:  func(in *domain.CreateFilmInput) { in.ReleaseYear = 1888 },
			invalid: false,
		},
		{
			name:    "year below minimum",
			change:  func(in *domain.CreateFilmInput) { in.ReleaseYear = 1887 },
			invalid: true,
		},
		{
			name:    "year maximum",
			change:  func(in *domain.CreateFilmInput) { in.ReleaseYear = 2200 },
			invalid: false,
		},
		{
			name:    "year above maximum",
			change:  func(in *domain.CreateFilmInput) { in.ReleaseYear = 2201 },
			invalid: true,
		},
		{
			name:    "unknown type",
			change:  func(in *domain.CreateFilmInput) { in.FilmType = "documentary" },
			invalid: true,
		},
		{
			name:    "empty type",
			change:  func(in *domain.CreateFilmInput) { in.FilmType = "" },
			invalid: true,
		},
		{
			name:    "age zero",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = 0 },
			invalid: false,
		},
		{
			name:    "age six",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = 6 },
			invalid: false,
		},
		{
			name:    "age twelve",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = 12 },
			invalid: false,
		},
		{
			name:    "age eighteen",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = 18 },
			invalid: false,
		},
		{
			name:    "invalid intermediate age",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = 7 },
			invalid: true,
		},
		{
			name:    "negative age",
			change:  func(in *domain.CreateFilmInput) { in.AgeLimit = -1 },
			invalid: true,
		},
		{
			name:    "duration minimum",
			change:  func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](1) },
			invalid: false,
		},
		{
			name:    "duration maximum",
			change:  func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6000) },
			invalid: false,
		},
		{
			name:    "duration zero",
			change:  func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](0) },
			invalid: true,
		},
		{
			name:    "duration negative",
			change:  func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](-1) },
			invalid: true,
		},
		{
			name:    "duration above maximum",
			change:  func(in *domain.CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6001) },
			invalid: true,
		},
		{
			name:    "series without duration",
			change:  func(in *domain.CreateFilmInput) { in.FilmType = domain.FilmTypeSeries },
			invalid: false,
		},
		{
			name: "series with duration",
			change: func(in *domain.CreateFilmInput) {
				in.FilmType = domain.FilmTypeSeries
				in.DurationMin = filmTestPointer[int16](90)
			},
			invalid: true,
		},
		{
			name:    "empty original title",
			change:  func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer("") },
			invalid: true,
		},
		{
			name:    "original title maximum",
			change:  func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 255)) },
			invalid: false,
		},
		{
			name:    "original title above maximum",
			change:  func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 256)) },
			invalid: true,
		},
		{
			name:    "original title trailing space counts",
			change:  func(in *domain.CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("A", 255) + " ") },
			invalid: true,
		},
		{
			name:    "empty description",
			change:  func(in *domain.CreateFilmInput) { in.Description = filmTestPointer("") },
			invalid: false,
		},
		{
			name:    "description maximum",
			change:  func(in *domain.CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10000)) },
			invalid: false,
		},
		{
			name:    "description above maximum",
			change:  func(in *domain.CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10001)) },
			invalid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := domain.CreateFilmInput{
				Title:       "Alien",
				FilmType:    domain.FilmTypeMovie,
				ReleaseYear: 1979,
				AgeLimit:    16,
			}
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
