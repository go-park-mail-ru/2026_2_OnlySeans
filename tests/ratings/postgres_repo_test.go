package ratings_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	ratingsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func newPostgresRatingRepo(t *testing.T) (domain.FilmRatingCollectionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "text")
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		assert.NoError(t, db.Close())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	return ratingsservice.NewPostgresRepo(db, appLogger), mock
}

func TestPostgresRatingRepo_Set(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`INSERT INTO film_rating`).
		WithArgs(domain.UserID(7), domain.FilmID(3), 9).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "film_id", "score", "created_at", "updated_at"}).
			AddRow(7, 3, 9, now, now))

	got, err := repo.SetFilmRating(context.Background(), 7, 3, 9)
	require.NoError(t, err)
	assert.Equal(t, domain.FilmRating{
		AccountID: 7,
		FilmID:    3,
		Score:     9,
		CreatedAt: now,
		UpdatedAt: now,
	}, got)
}

func TestPostgresRatingRepo_SetMapsDatabaseErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{
			name:    "film not found",
			dbErr:   &pgconn.PgError{Code: storage.ForeignKeyViolation, ConstraintName: "film_rating_film_id_fkey"},
			wantErr: domain.ErrFilmNotFound,
		},
		{
			name:    "invalid score",
			dbErr:   &pgconn.PgError{Code: storage.CheckViolation, ConstraintName: "film_rating_score_check"},
			wantErr: domain.ErrInvalidFilmRating,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresRatingRepo(t)
			mock.ExpectQuery(`INSERT INTO film_rating`).WillReturnError(tt.dbErr)
			_, err := repo.SetFilmRating(context.Background(), 7, 3, 9)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresRatingRepo_GetFilmRatingSummary(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	mock.ExpectQuery(`SELECT COALESCE\(ROUND\(AVG\(score\), 1\)`).
		WithArgs(domain.FilmID(3)).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce", "count"}).AddRow(8.5, 2))

	got, err := repo.GetFilmRatingSummary(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, domain.FilmRatingSummary{FilmID: 3, AverageScore: 8.5, Votes: 2}, got)
}

func TestPostgresRatingRepo_GetSummaries(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	mock.ExpectQuery(`WITH requested_films AS`).
		WithArgs(int64(3), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"film_id", "average_score", "votes"}).
			AddRow(3, 8.5, 2).
			AddRow(1, 0, 0))

	got, err := repo.GetFilmRatingsSummaries(context.Background(), []domain.FilmID{3, 1})
	require.NoError(t, err)
	assert.Equal(t, []domain.FilmRatingSummary{
		{FilmID: 3, AverageScore: 8.5, Votes: 2},
		{FilmID: 1, AverageScore: 0, Votes: 0},
	}, got)
}

func TestPostgresRatingRepo_GetSummariesEmpty(t *testing.T) {
	repo, _ := newPostgresRatingRepo(t)
	got, err := repo.GetFilmRatingsSummaries(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPostgresRatingRepo_GetCollectionFilmRatingSummaries(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	mock.ExpectQuery(`WITH target_collection AS`).
		WithArgs(domain.CollectionID(2), 1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"collection_id", "total", "film_id", "average_score", "votes"}).
			AddRow(2, 6, 3, 8.5, 2))

	got, total, err := repo.GetCollectionFilmRatingSummaries(context.Background(), 2, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Equal(t, []domain.FilmRatingSummary{
		{FilmID: 3, AverageScore: 8.5, Votes: 2},
	}, got)
}

func TestPostgresRatingRepo_GetCollectionFilmRatingSummariesEmptyPage(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	mock.ExpectQuery(`WITH target_collection AS`).
		WithArgs(domain.CollectionID(2), 20, 100).
		WillReturnRows(sqlmock.NewRows([]string{"collection_id", "total", "film_id", "average_score", "votes"}).
			AddRow(2, 6, nil, 0, 0))

	got, total, err := repo.GetCollectionFilmRatingSummaries(context.Background(), 2, 20, 100)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, 6, total)
}

func TestPostgresRatingRepo_GetCollectionFilmRatingSummariesMissingCollection(t *testing.T) {
	repo, mock := newPostgresRatingRepo(t)
	mock.ExpectQuery(`WITH target_collection AS`).
		WithArgs(domain.CollectionID(999), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"collection_id", "total", "film_id", "average_score", "votes"}))

	_, _, err := repo.GetCollectionFilmRatingSummaries(context.Background(), 999, 20, 0)
	assert.ErrorIs(t, err, domain.ErrCollectionNotFound)
}

func TestPostgresRatingRepo_Delete(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		result  driver.Result
		execErr error
		wantErr error
	}{
		{
			name:   "deleted",
			result: sqlmock.NewResult(0, 1),
		},
		{
			name:    "rating does not exist",
			result:  sqlmock.NewResult(0, 0),
			wantErr: domain.ErrRatingNotFound,
		},
		{
			name:    "database error",
			execErr: databaseErr,
			wantErr: databaseErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresRatingRepo(t)
			expectation := mock.ExpectExec(`DELETE FROM film_rating`).
				WithArgs(domain.UserID(7), domain.FilmID(3))
			if tt.execErr != nil {
				expectation.WillReturnError(tt.execErr)
			} else {
				expectation.WillReturnResult(tt.result)
			}

			err := repo.DeleteFilmRating(context.Background(), 7, 3)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
