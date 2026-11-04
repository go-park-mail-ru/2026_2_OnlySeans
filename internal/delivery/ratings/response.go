package ratings

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type filmRatingResponseDTO struct {
	FilmID    domain.FilmID `json:"film_id"`
	Score     int16         `json:"score"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type filmRatingSummaryResponseDTO struct {
	FilmID       domain.FilmID `json:"film_id"`
	AverageScore float64       `json:"average_score"`
	Votes        int64         `json:"votes"`
}

type ratingResponse struct {
	Rating filmRatingResponseDTO `json:"rating"`
}

type ratingSummaryResponse struct {
	Rating filmRatingSummaryResponseDTO `json:"rating"`
}

type collectionRatingsResponse struct {
	Ratings []filmRatingSummaryResponseDTO `json:"ratings"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toFilmRatingResponseDTO(rating domain.FilmRating) filmRatingResponseDTO {
	return filmRatingResponseDTO{
		FilmID:    rating.FilmID,
		Score:     rating.Score,
		CreatedAt: rating.CreatedAt,
		UpdatedAt: rating.UpdatedAt,
	}
}

func toFilmRatingSummaryResponseDTO(summary domain.FilmRatingSummary) filmRatingSummaryResponseDTO {
	return filmRatingSummaryResponseDTO{
		FilmID:       summary.FilmID,
		AverageScore: summary.AverageScore,
		Votes:        summary.Votes,
	}
}

func toFilmRatingSummaryResponseDTOs(summaries []domain.FilmRatingSummary) []filmRatingSummaryResponseDTO {
	result := make([]filmRatingSummaryResponseDTO, len(summaries))
	for i, summary := range summaries {
		result[i] = toFilmRatingSummaryResponseDTO(summary)
	}
	return result
}
