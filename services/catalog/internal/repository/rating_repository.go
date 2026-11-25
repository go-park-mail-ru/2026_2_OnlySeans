package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type RatingRepository interface {
	GetMyScore(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) (*int, error)
	SetFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error)
	GetFilmRatingSummary(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error)
	GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error)
	DeleteFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) error
}

type CollectionFilmRatingRepository interface {
	GetCollectionFilmRatingSummaries(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error)
}
