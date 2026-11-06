package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func filmTestPointer[T any](value T) *T { return &value }

func TestCreateFilmInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*CreateFilmInput)
		invalid bool
	}{
		{"minimal movie", func(in *CreateFilmInput) {}, false},
		{"empty title", func(in *CreateFilmInput) { in.Title = "" }, true},
		{"blank title", func(in *CreateFilmInput) { in.Title = " \t\n" }, true},
		{"one character title", func(in *CreateFilmInput) { in.Title = "Я" }, false},
		{"255 unicode characters", func(in *CreateFilmInput) { in.Title = strings.Repeat("Я", 255) }, false},
		{"256 unicode characters", func(in *CreateFilmInput) { in.Title = strings.Repeat("Я", 256) }, true},
		{"year minimum", func(in *CreateFilmInput) { in.ReleaseYear = 1888 }, false},
		{"year below minimum", func(in *CreateFilmInput) { in.ReleaseYear = 1887 }, true},
		{"year maximum", func(in *CreateFilmInput) { in.ReleaseYear = 2200 }, false},
		{"year above maximum", func(in *CreateFilmInput) { in.ReleaseYear = 2201 }, true},
		{"unknown type", func(in *CreateFilmInput) { in.FilmType = "documentary" }, true},
		{"empty type", func(in *CreateFilmInput) { in.FilmType = "" }, true},
		{"age zero", func(in *CreateFilmInput) { in.AgeLimit = 0 }, false},
		{"age six", func(in *CreateFilmInput) { in.AgeLimit = 6 }, false},
		{"age twelve", func(in *CreateFilmInput) { in.AgeLimit = 12 }, false},
		{"age eighteen", func(in *CreateFilmInput) { in.AgeLimit = 18 }, false},
		{"invalid intermediate age", func(in *CreateFilmInput) { in.AgeLimit = 7 }, true},
		{"negative age", func(in *CreateFilmInput) { in.AgeLimit = -1 }, true},
		{"duration minimum", func(in *CreateFilmInput) { in.DurationMin = filmTestPointer[int16](1) }, false},
		{"duration maximum", func(in *CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6000) }, false},
		{"duration zero", func(in *CreateFilmInput) { in.DurationMin = filmTestPointer[int16](0) }, true},
		{"duration negative", func(in *CreateFilmInput) { in.DurationMin = filmTestPointer[int16](-1) }, true},
		{"duration above maximum", func(in *CreateFilmInput) { in.DurationMin = filmTestPointer[int16](6001) }, true},
		{"series without duration", func(in *CreateFilmInput) { in.FilmType = FilmTypeSeries }, false},
		{"series with duration", func(in *CreateFilmInput) { in.FilmType = FilmTypeSeries; in.DurationMin = filmTestPointer[int16](90) }, true},
		{"empty original title", func(in *CreateFilmInput) { in.OriginalTitle = filmTestPointer("") }, true},
		{"original title maximum", func(in *CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 255)) }, false},
		{"original title above maximum", func(in *CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("Я", 256)) }, true},
		{"original title trailing space counts", func(in *CreateFilmInput) { in.OriginalTitle = filmTestPointer(strings.Repeat("A", 255) + " ") }, true},
		{"empty description", func(in *CreateFilmInput) { in.Description = filmTestPointer("") }, false},
		{"description maximum", func(in *CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10000)) }, false},
		{"description above maximum", func(in *CreateFilmInput) { in.Description = filmTestPointer(strings.Repeat("Я", 10001)) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateFilmInput{Title: "Alien", FilmType: FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16}
			tt.change(&input)
			err := input.Validate()
			if tt.invalid {
				assert.ErrorIs(t, err, ErrInvalidFilmInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
