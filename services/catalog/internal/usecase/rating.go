package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type RatingUseCase interface {
	SetFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error)
	GetFilmRating(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error)
	GetMyFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID) (*int, error)
	DeleteFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID) error
	GetCollectionRatings(ctx context.Context, id domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error)
}

type ratingUseCase struct {
	ratings           repository.RatingRepository
	collectionRatings repository.CollectionFilmRatingRepository
	films             repository.FilmReader
}

func NewRatingUseCase(ratings repository.RatingRepository, collectionRatings repository.CollectionFilmRatingRepository, films repository.FilmReader) RatingUseCase {
	return &ratingUseCase{
		ratings:           ratings,
		collectionRatings: collectionRatings,
		films:             films,
	}
}

func (u *ratingUseCase) checkFilm(ctx context.Context, filmID domain.FilmID) error {
	if filmID <= 0 {
		return ErrInvalidFilmID
	}

	if _, err := u.films.GetFilmByID(ctx, filmID); err != nil {
		return fmt.Errorf("check rating film: %w", err)
	}

	return nil
}

func (u *ratingUseCase) SetFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error) {
	if userID <= 0 {
		return domain.FilmRating{}, domain.ErrUnauthorized
	}

	if score < domain.MinFilmRating || score > domain.MaxFilmRating {
		return domain.FilmRating{}, domain.InvalidFilmRatingLimitsError()
	}

	if err := u.checkFilm(ctx, filmID); err != nil {
		return domain.FilmRating{}, err
	}

	rating, err := u.ratings.SetFilmRating(ctx, userID, filmID, score)
	if err != nil {
		return domain.FilmRating{}, fmt.Errorf("set film rating: %w", err)
	}

	return rating, nil
}

func (u *ratingUseCase) GetFilmRating(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	if err := u.checkFilm(ctx, filmID); err != nil {
		return domain.FilmRatingSummary{}, err
	}

	summary, err := u.ratings.GetFilmRatingSummary(ctx, filmID)
	if err != nil {
		return domain.FilmRatingSummary{}, fmt.Errorf("get film rating: %w", err)
	}

	return summary, nil
}

func (u *ratingUseCase) GetMyFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID) (*int, error) {
	if userID <= 0 {
		return nil, domain.ErrUnauthorized
	}

	if filmID <= 0 {
		return nil, domain.ErrFilmNotFound
	}

	if err := u.checkFilm(ctx, filmID); err != nil {
		return nil, err
	}

	score, err := u.ratings.GetMyScore(ctx, userID, filmID)
	if err != nil {
		return nil, fmt.Errorf("get personal film rating: %w", err)
	}

	return score, nil
}

func (u *ratingUseCase) DeleteFilmRating(ctx context.Context, userID domain.UserID, filmID domain.FilmID) error {
	if userID <= 0 {
		return domain.ErrUnauthorized
	}

	if err := u.checkFilm(ctx, filmID); err != nil {
		return err
	}

	if err := u.ratings.DeleteFilmRating(ctx, userID, filmID); err != nil {
		return fmt.Errorf("delete film rating: %w", err)
	}

	return nil
}

func (u *ratingUseCase) GetCollectionRatings(ctx context.Context, id domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error) {
	if id <= 0 {
		return nil, 0, ErrInvalidCollectionID
	}

	if err := validatePagination(limit, offset); err != nil {
		return nil, 0, err
	}

	ratings, total, err := u.collectionRatings.GetCollectionFilmRatingSummaries(ctx, id, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get collection ratings: %w", err)
	}

	return ratings, total, nil
}
