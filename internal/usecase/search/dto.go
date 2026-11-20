package usecase

import (
	"strings"
	"unicode/utf8"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SearchSuggestionUsecaseInput struct {
	Query string
	Limit int
}

func NewSearchSuggestionUsecaseInput(query string, limit int) *SearchSuggestionUsecaseInput {
	return &SearchSuggestionUsecaseInput{
		Query: query,
		Limit: limit,
	}
}

func prepareSearchInput(input SearchSuggestionUsecaseInput) (*SearchSuggestionUsecaseInput, error) {
	input.Query = strings.TrimSpace(input.Query)

	length := utf8.RuneCountInString(input.Query)
	if length < 2 || length > 200 {
		return nil, domain.ErrInvalidSearchQuery
	}

	if input.Limit < 1 || input.Limit > 5 {
		return nil, domain.ErrInvalidSearchLimit
	}

	return &input, nil
}

type SearchSuggestionUsecaseOutput struct {
	FilmSuggestions   []domain.SearchFilm
	PersonSuggestions []domain.SearchPerson
}

func NewSearchSuggestionUsecaseOutput(filmSuggestions []domain.SearchFilm, personSuggestions []domain.SearchPerson) *SearchSuggestionUsecaseOutput {
	return &SearchSuggestionUsecaseOutput{
		FilmSuggestions:   filmSuggestions,
		PersonSuggestions: personSuggestions,
	}
}
