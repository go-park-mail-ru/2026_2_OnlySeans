package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

// набор операций связывает HTTP-обработчик с локальными юзкейсами или с gRPC-клиентом
type Operations struct {
	List   func(context.Context, ListReviewsInput) (*ListReviewsOutput, error)
	Create func(context.Context, CreateReviewInput) (*CreateReviewOutput, error)
	Update func(context.Context, UpdateReviewInput) (*ReviewOutput, error)
	Delete func(context.Context, DeleteReviewInput) error
}

func NewOperations(reviews domain.ReviewRepository, films domain.FilmRepository, appLogger logger.Logger) (Operations, error) {
	list, err := NewListReviewsUseCase(reviews, films, appLogger)
	if err != nil {
		return Operations{}, err
	}
	create, err := NewCreateReviewUseCase(reviews, films, appLogger)
	if err != nil {
		return Operations{}, err
	}
	update, err := NewUpdateReviewUseCase(reviews, appLogger)
	if err != nil {
		return Operations{}, err
	}
	deleteReview, err := NewDeleteReviewUseCase(reviews, appLogger)
	if err != nil {
		return Operations{}, err
	}
	return Operations{List: list.Run, Create: create.Run, Update: update.Run, Delete: deleteReview.Run}, nil
}
