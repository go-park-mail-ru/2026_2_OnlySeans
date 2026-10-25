package reviews

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type reviewResponseDTO struct {
	ID               domain.ReviewID `json:"id"`
	AuthorID         domain.UserID   `json:"author_id"`
	FilmID           domain.FilmID   `json:"film_id"`
	Title            string          `json:"title"`
	Content          string          `json:"content"`
	ContainsSpoilers bool            `json:"contains_spoilers"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type reviewResponse struct {
	Review reviewResponseDTO `json:"review"`
}

type reviewsPage struct {
	Reviews []reviewResponseDTO `json:"reviews"`
	Total   int                 `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toReviewResponseDTO(review domain.Review) reviewResponseDTO {
	return reviewResponseDTO{
		ID:               review.ID,
		AuthorID:         review.AuthorID,
		FilmID:           review.FilmID,
		Title:            review.Title,
		Content:          review.Content,
		ContainsSpoilers: review.ContainsSpoilers,
		CreatedAt:        review.CreatedAt,
		UpdatedAt:        review.UpdatedAt,
	}
}

// toReviewResponseDTOs всегда возвращает срез, чтобы пустой список ушёл как [], а не null.
func toReviewResponseDTOs(reviews []domain.Review) []reviewResponseDTO {
	result := make([]reviewResponseDTO, 0, len(reviews))
	for _, review := range reviews {
		result = append(result, toReviewResponseDTO(review))
	}
	return result
}
