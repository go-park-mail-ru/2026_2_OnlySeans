package films

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type createFilmRequest struct {
	Title         string  `json:"title"`
	OriginalTitle *string `json:"original_title"`
	FilmType      string  `json:"film_type"`
	ReleaseYear   int16   `json:"release_year"`
	DurationMin   *int16  `json:"duration_min"`
	AgeLimit      *int16  `json:"age_limit"`
	Description   *string `json:"description"`
}

func decodeCreateFilm(r *http.Request) (*pb.FilmInput, error) {
	var decoder = json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req createFilmRequest
	if err := decoder.Decode(&req); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			return nil, err
		}

		return nil, fmt.Errorf("invalid JSON request body: %w", catalogclient.ErrInvalidFilmInput)
	}

	if req.AgeLimit == nil {
		return nil, fmt.Errorf("age_limit is required: %w", catalogclient.ErrInvalidFilmInput)
	}

	age := int32(*req.AgeLimit)
	var duration *int32

	if req.DurationMin != nil {
		value := int32(*req.DurationMin)
		duration = &value
	}

	return &pb.FilmInput{
		Title:         req.Title,
		OriginalTitle: req.OriginalTitle,
		FilmType:      req.FilmType,
		ReleaseYear:   int32(req.ReleaseYear),
		DurationMin:   duration,
		AgeLimit:      &age,
		Description:   req.Description,
	}, nil
}
