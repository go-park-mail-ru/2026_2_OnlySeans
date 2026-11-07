package ratings

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type InMemoryCollectionRatingRepository struct {
	ratings     domain.FilmRatingRepository
	collections domain.CollectionRepository
}

func NewInMemoryCollectionRatingRepository(
	ratings domain.FilmRatingRepository,
	collections domain.CollectionRepository,
) (*InMemoryCollectionRatingRepository, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if collections == nil {
		return nil, domain.ErrNilCollectionsRepo
	}
	return &InMemoryCollectionRatingRepository{
		ratings:     ratings,
		collections: collections,
	}, nil
}

func (r *InMemoryCollectionRatingRepository) GetCollectionFilmRatingSummaries(
	ctx context.Context,
	collectionID domain.CollectionID,
	limit, offset int,
) ([]domain.FilmRatingSummary, int, error) {
	films, total, err := r.collections.ListCollectionFilms(ctx, collectionID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	filmIDs := make([]domain.FilmID, len(films))
	for i, film := range films {
		filmIDs[i] = film.ID
	}

	summaries, err := r.ratings.GetFilmRatingsSummaries(ctx, filmIDs)
	if err != nil {
		return nil, 0, err
	}
	return summaries, total, nil
}
