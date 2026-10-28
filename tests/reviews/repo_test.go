package reviews_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviews "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
)

func validInput() domain.ReviewInput {
	return domain.ReviewInput{Title: validTitle, Content: validContent}
}

func TestInMemoryReviewRepo_Create(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	input := domain.ReviewInput{Title: validTitle, Content: validContent, ContainsSpoilers: true}
	review, err := repo.Create(ctx, 7, 3, input)
	require.NoError(t, err)

	assert.Equal(t, domain.ReviewID(1), review.ID)
	assert.Equal(t, domain.UserID(7), review.AuthorID)
	assert.Equal(t, domain.FilmID(3), review.FilmID)
	assert.Equal(t, input.Title, review.Title)
	assert.Equal(t, input.Content, review.Content)
	assert.True(t, review.ContainsSpoilers)
	assert.False(t, review.CreatedAt.IsZero(), "дата создания должна быть выставлена")
	assert.Equal(t, review.CreatedAt, review.UpdatedAt)

	stored, err := repo.GetByID(ctx, review.ID)
	require.NoError(t, err)
	assert.Equal(t, review, stored)
}

func TestInMemoryReviewRepo_CreateIDsAreSequential(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	first, err := repo.Create(ctx, 1, 1, validInput())
	require.NoError(t, err)
	second, err := repo.Create(ctx, 2, 1, validInput())
	require.NoError(t, err)

	assert.Equal(t, domain.ReviewID(1), first.ID)
	assert.Equal(t, domain.ReviewID(2), second.ID)
}

func TestInMemoryReviewRepo_CreateDuplicate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		author  domain.UserID
		film    domain.FilmID
		wantErr error
	}{
		{"тот же автор на тот же фильм", 1, 1, domain.ErrReviewExists},
		{"тот же автор на другой фильм", 1, 2, nil},
		{"другой автор на тот же фильм", 2, 1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := reviews.NewInMemoryReviewRepo()
			_, err := repo.Create(ctx, 1, 1, validInput())
			require.NoError(t, err)

			_, err = repo.Create(ctx, tt.author, tt.film, validInput())
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestInMemoryReviewRepo_GetByIDNotFound(t *testing.T) {
	repo := reviews.NewInMemoryReviewRepo()

	_, err := repo.GetByID(context.Background(), 1)
	assert.ErrorIs(t, err, domain.ErrReviewNotFound)
}

func TestInMemoryReviewRepo_Update(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	created, err := repo.Create(ctx, 7, 3, validInput())
	require.NoError(t, err)

	changes := domain.ReviewInput{Title: "Передумала", Content: validContent + " Ещё раз.", ContainsSpoilers: true}
	updated, err := repo.Update(ctx, created.ID, changes)
	require.NoError(t, err)

	assert.Equal(t, changes.Title, updated.Title)
	assert.Equal(t, changes.Content, updated.Content)
	assert.True(t, updated.ContainsSpoilers)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, created.AuthorID, updated.AuthorID)
	assert.Equal(t, created.FilmID, updated.FilmID)
	assert.Equal(t, created.CreatedAt, updated.CreatedAt, "дата создания не должна меняться")
	assert.False(t, updated.UpdatedAt.Before(created.UpdatedAt))

	stored, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, updated, stored, "изменения должны сохраниться в хранилище")
}

func TestInMemoryReviewRepo_UpdateNotFound(t *testing.T) {
	repo := reviews.NewInMemoryReviewRepo()

	_, err := repo.Update(context.Background(), 1, validInput())
	assert.ErrorIs(t, err, domain.ErrReviewNotFound)
}

func TestInMemoryReviewRepo_Delete(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	created, err := repo.Create(ctx, 7, 3, validInput())
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, created.ID, 7))

	_, err = repo.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, domain.ErrReviewNotFound)

	_, total, err := repo.ListByFilm(ctx, 3, 10, 0)
	require.NoError(t, err)
	assert.Zero(t, total)

	_, err = repo.Create(ctx, 7, 3, validInput())
	assert.NoError(t, err, "после удаления автор может написать отзыв на тот же фильм заново")
}

func TestInMemoryReviewRepo_DeleteNotFound(t *testing.T) {
	repo := reviews.NewInMemoryReviewRepo()

	err := repo.Delete(context.Background(), 1, 1)
	assert.ErrorIs(t, err, domain.ErrReviewNotFound)
}

func TestInMemoryReviewRepo_ListByFilm(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	// На фильм 1 пишут авторы 1, 2, 3, на фильм 2 — автор 1.
	for author := domain.UserID(1); author <= 3; author++ {
		_, err := repo.Create(ctx, author, 1, validInput())
		require.NoError(t, err)
	}
	_, err := repo.Create(ctx, 1, 2, validInput())
	require.NoError(t, err)

	tests := []struct {
		name      string
		film      domain.FilmID
		limit     int
		offset    int
		wantIDs   []domain.ReviewID
		wantTotal int
	}{
		{"все отзывы фильма, новые первыми", 1, 10, 0, []domain.ReviewID{3, 2, 1}, 3},
		{"первая страница", 1, 2, 0, []domain.ReviewID{3, 2}, 3},
		{"вторая страница", 1, 2, 2, []domain.ReviewID{1}, 3},
		{"offset за концом списка", 1, 10, 10, []domain.ReviewID{}, 3},
		{"отзывы другого фильма не попадают", 2, 10, 0, []domain.ReviewID{4}, 1},
		{"фильм без отзывов", 5, 10, 0, []domain.ReviewID{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, total, err := repo.ListByFilm(ctx, tt.film, tt.limit, tt.offset)
			require.NoError(t, err)

			assert.NotNil(t, list, "пустой список должен быть [], а не nil")
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantIDs, reviewIDs(list))
		})
	}
}

func TestInMemoryReviewRepo_ListByFilmSkipsDeleted(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	for author := domain.UserID(1); author <= 3; author++ {
		_, err := repo.Create(ctx, author, 1, validInput())
		require.NoError(t, err)
	}
	require.NoError(t, repo.Delete(ctx, 2, 2))

	list, total, err := repo.ListByFilm(ctx, 1, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Equal(t, []domain.ReviewID{3, 1}, reviewIDs(list))
}

func TestInMemoryReviewRepo_ConcurrentCreateSameAuthor(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	const attempts = 50
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Create(ctx, 1, 1, validInput())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var created, duplicates int
	for err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, domain.ErrReviewExists):
			duplicates++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 1, created, "одновременно создаться должен ровно один отзыв")
	assert.Equal(t, attempts-1, duplicates)
}

func TestInMemoryReviewRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := reviews.NewInMemoryReviewRepo()

	const authors = 50
	var wg sync.WaitGroup
	for author := domain.UserID(1); author <= authors; author++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			review, err := repo.Create(ctx, author, 1, validInput())
			if !assert.NoError(t, err) {
				return
			}
			_, err = repo.Update(ctx, review.ID, validInput())
			assert.NoError(t, err)
			_, _, err = repo.ListByFilm(ctx, 1, 10, 0)
			assert.NoError(t, err)
			if author%2 == 0 {
				assert.NoError(t, repo.Delete(ctx, review.ID, author))
			}
		}()
	}
	wg.Wait()

	_, total, err := repo.ListByFilm(ctx, 1, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, authors/2, total)
}

func reviewIDs(list []domain.Review) []domain.ReviewID {
	ids := make([]domain.ReviewID, 0, len(list))
	for _, review := range list {
		ids = append(ids, review.ID)
	}
	return ids
}
