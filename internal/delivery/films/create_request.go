package films

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type createFilmRequest struct {
	Title         string          `json:"title"`
	OriginalTitle *string         `json:"original_title"`
	FilmType      domain.FilmType `json:"film_type"`
	ReleaseYear   int16           `json:"release_year"`
	DurationMin   *int16          `json:"duration_min"`
	AgeLimit      *int16          `json:"age_limit"`
	Description   *string         `json:"description"`
}

func decodeCreateFilm(r *http.Request) (domain.CreateFilmInput, error) {
	var decoder = json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req createFilmRequest
	if err := decoder.Decode(&req); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			return domain.CreateFilmInput{}, err
		}
		return domain.CreateFilmInput{}, fmt.Errorf("invalid JSON request body: %w", domain.ErrInvalidFilmInput)
	}

	if req.AgeLimit == nil {
		return domain.CreateFilmInput{}, fmt.Errorf("age_limit is required: %w", domain.ErrInvalidFilmInput)
	}

	return domain.CreateFilmInput{
		Title:         req.Title,
		OriginalTitle: req.OriginalTitle,
		FilmType:      req.FilmType,
		ReleaseYear:   req.ReleaseYear,
		DurationMin:   req.DurationMin,
		AgeLimit:      *req.AgeLimit,
		Description:   req.Description,
	}, nil
}
