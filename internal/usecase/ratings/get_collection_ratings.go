package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const collectionRatingsPageSize = 100

type GetCollectionRatingsUseCase struct {
	ratings     domain.FilmRatingRepository
	collections domain.CollectionRepository
}

func NewGetCollectionRatingsUseCase(ratings domain.FilmRatingRepository, collections domain.CollectionRepository) (*GetCollectionRatingsUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if collections == nil {
		return nil, domain.ErrNilCollectionsRepo
	}
	return &GetCollectionRatingsUseCase{ratings: ratings, collections: collections}, nil
}

func (uc *GetCollectionRatingsUseCase) Run(ctx context.Context, input GetCollectionRatingsInput) (*GetCollectionRatingsOutput, error) {
	filmIDs := make([]domain.FilmID, 0)
	for offset := 0; ; {
		films, _, err := uc.collections.ListCollectionFilms(ctx, input.CollectionID, collectionRatingsPageSize, offset)
		if err != nil {
			if errors.Is(err, domain.ErrCollectionNotFound) {
				return nil, domain.ErrCollectionNotFound
			}
			return nil, err
		}

		for _, film := range films {
			filmIDs = append(filmIDs, film.ID)
		}
		if len(films) < collectionRatingsPageSize {
			break
		}
		offset += len(films)
	}

	ratings, err := uc.ratings.GetSummaries(ctx, filmIDs)
	if err != nil {
		return nil, err
	}
	return NewGetCollectionRatingsOutput(ratings), nil
}
