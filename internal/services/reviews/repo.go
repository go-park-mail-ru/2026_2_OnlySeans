package reviews

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

// для проверки, что автор ещё не писал отзыв на фильм
type reviewKey struct {
	authorID domain.UserID
	filmID   domain.FilmID
}

type InMemoryReviewRepo struct {
	mu           sync.Mutex
	byID         map[domain.ReviewID]domain.Review
	byAuthorFilm map[reviewKey]domain.ReviewID
	nextID       domain.ReviewID
}

func NewInMemoryReviewRepo() *InMemoryReviewRepo {
	return &InMemoryReviewRepo{
		byID:         make(map[domain.ReviewID]domain.Review),
		byAuthorFilm: make(map[reviewKey]domain.ReviewID),
		nextID:       1,
	}
}

func (r *InMemoryReviewRepo) Create(_ context.Context, authorID domain.UserID, filmID domain.FilmID, input domain.ReviewInput) (*domain.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := reviewKey{authorID: authorID, filmID: filmID}
	if _, exists := r.byAuthorFilm[key]; exists {
		return nil, domain.ErrReviewExists
	}

	now := time.Now()
	review := domain.Review{
		ID:               r.nextID,
		AuthorID:         authorID,
		FilmID:           filmID,
		Title:            input.Title,
		Content:          input.Content,
		ContainsSpoilers: input.ContainsSpoilers,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	r.byID[review.ID] = review
	r.byAuthorFilm[key] = review.ID
	r.nextID++
	return &review, nil
}

func (r *InMemoryReviewRepo) GetByID(_ context.Context, id domain.ReviewID) (*domain.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id]
	if !exists {
		return nil, domain.ErrReviewNotFound
	}
	return &review, nil
}

func (r *InMemoryReviewRepo) Update(_ context.Context, id domain.ReviewID, changes domain.ReviewUpdate) (*domain.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id]
	if !exists {
		return nil, domain.ErrReviewNotFound
	}

	if review.Version != changes.Version {
		return nil, domain.ErrVersionConflict
	}

	review.Title = changes.Input.Title
	review.Content = changes.Input.Content
	review.ContainsSpoilers = changes.Input.ContainsSpoilers
	review.Version++
	review.UpdatedAt = time.Now()

	r.byID[id] = review
	return &review, nil
}

// отдаёт страницу отзывов фильма, новые первыми
func (r *InMemoryReviewRepo) ListByFilm(_ context.Context, filmID domain.FilmID, limit, offset int) ([]domain.Review, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Словарь перебирается в случайном порядке, поэтому идём по номерам
	matched := make([]domain.Review, 0)
	for id := r.nextID - 1; id >= 1; id-- {
		review, exists := r.byID[id]
		if exists && review.FilmID == filmID {
			matched = append(matched, review)
		}
	}

	total := len(matched)
	start := min(offset, total)
	end := min(start+limit, total)
	return matched[start:end], total, nil
}

func (r *InMemoryReviewRepo) Delete(_ context.Context, id domain.ReviewID, _ domain.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id]
	if !exists {
		return domain.ErrReviewNotFound
	}

	delete(r.byID, id)
	delete(r.byAuthorFilm, reviewKey{authorID: review.AuthorID, filmID: review.FilmID})
	return nil
}
