package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ListReviewsUseCase struct {
	reviews domain.ReviewRepository
	films   domain.FilmRepository
}

func NewListReviewsUseCase(reviews domain.ReviewRepository, films domain.FilmRepository) (*ListReviewsUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}

	return &ListReviewsUseCase{reviews: reviews, films: films}, nil
}

func (uc *ListReviewsUseCase) Run(ctx context.Context, input ListReviewsInput) (*ListReviewsOutput, error) {
	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	reviews, total, err := uc.reviews.ListByFilm(ctx, input.FilmID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("list reviews of film %d: %w", input.FilmID, err)
	}
	return NewListReviewsOutput(reviews, total), nil
}
