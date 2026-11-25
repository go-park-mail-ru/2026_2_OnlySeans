package ratings

import (
	"time"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type filmRatingResponseDTO struct {
	FilmID    int64     `json:"film_id"`
	Score     int32     `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type filmRatingSummaryResponseDTO struct {
	FilmID       int64   `json:"film_id"`
	AverageScore float64 `json:"average_score"`
	Votes        int64   `json:"votes"`
}

type ratingResponse struct {
	Rating filmRatingResponseDTO `json:"rating"`
}

type ratingSummaryResponse struct {
	Rating filmRatingSummaryResponseDTO `json:"rating"`
}

type collectionRatingsResponse struct {
	Ratings []filmRatingSummaryResponseDTO `json:"ratings"`
	Total   int64                          `json:"total"`
	Limit   int                            `json:"limit"`
	Offset  int                            `json:"offset"`
}

func toFilmRatingResponseDTO(rating *pb.FilmRating) filmRatingResponseDTO {
	return filmRatingResponseDTO{
		FilmID:    rating.FilmId,
		Score:     rating.Score,
		CreatedAt: rating.GetCreatedAt().AsTime(),
		UpdatedAt: rating.GetUpdatedAt().AsTime(),
	}
}

func toFilmRatingSummaryResponseDTO(summary *pb.FilmRatingSummary) filmRatingSummaryResponseDTO {
	return filmRatingSummaryResponseDTO{
		FilmID:       summary.FilmId,
		AverageScore: summary.AverageScore,
		Votes:        summary.Votes,
	}
}

func toFilmRatingSummaryResponseDTOs(summaries []*pb.FilmRatingSummary) []filmRatingSummaryResponseDTO {
	result := make([]filmRatingSummaryResponseDTO, len(summaries))
	for i, summary := range summaries {
		result[i] = toFilmRatingSummaryResponseDTO(summary)
	}

	return result
}
