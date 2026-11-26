package ratings

import "time"

type setFilmRatingRequestDTO struct {
	Score *int `json:"score"`
}

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
