package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CreateReviewUseCase struct {
	reviews domain.ReviewRepository
	films   domain.FilmRepository
}

func NewCreateReviewUseCase(reviews domain.ReviewRepository, films domain.FilmRepository) (*CreateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}

	return &CreateReviewUseCase{reviews: reviews, films: films}, nil
}

// Повторный отзыв того же автора на тот же фильм отклоняет хранилище, без отдельной проверки заранее.
func (uc *CreateReviewUseCase) Run(ctx context.Context, input CreateReviewInput) (*ReviewOutput, error) {
	review, err := prepareReview(input.Review)
	if err != nil {
		return nil, err
	}

	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	created, err := uc.reviews.Create(ctx, input.AuthorID, input.FilmID, *review)
	if err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}
	return NewReviewOutput(created), nil
}
