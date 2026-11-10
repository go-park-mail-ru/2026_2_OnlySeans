package ratings

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ratingKey struct {
	accountID domain.UserID
	filmID    domain.FilmID
}

type InMemoryRepo struct {
	mu      sync.RWMutex
	ratings map[ratingKey]domain.FilmRating
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{ratings: make(map[ratingKey]domain.FilmRating)}
}

func (r *InMemoryRepo) SetFilmRating(_ context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := ratingKey{accountID: accountID, filmID: filmID}
	now := time.Now()
	rating, exists := r.ratings[key]
	if !exists {
		rating.AccountID = accountID
		rating.FilmID = filmID
		rating.CreatedAt = now
	}
	rating.Score = score
	rating.UpdatedAt = now
	r.ratings[key] = rating
	return rating, nil
}

func (r *InMemoryRepo) GetFilmRatingSummary(_ context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	summary := domain.FilmRatingSummary{FilmID: filmID}
	var scoreTotal int64
	for key, rating := range r.ratings {
		if key.filmID == filmID {
			scoreTotal += int64(rating.Score)
			summary.Votes++
		}
	}
	if summary.Votes > 0 {
		summary.AverageScore = math.Round(float64(scoreTotal)*10/float64(summary.Votes)) / 10
	}
	return summary, nil
}

func (r *InMemoryRepo) GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
	summaries := make([]domain.FilmRatingSummary, 0, len(filmIDs))
	for _, filmID := range filmIDs {
		summary, err := r.GetFilmRatingSummary(ctx, filmID)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *InMemoryRepo) DeleteFilmRating(_ context.Context, accountID domain.UserID, filmID domain.FilmID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := ratingKey{accountID: accountID, filmID: filmID}
	if _, exists := r.ratings[key]; !exists {
		return domain.ErrRatingNotFound
	}
	delete(r.ratings, key)
	return nil
}

func (r *InMemoryRepo) GetMyScore(_ context.Context, accountID domain.UserID, filmID domain.FilmID) (*int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rating, ok := r.ratings[ratingKey{accountID: accountID, filmID: filmID}]
	if !ok {
		return nil, nil
	}
	score := rating.Score
	return &score, nil
}
