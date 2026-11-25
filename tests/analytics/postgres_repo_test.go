package analytics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	analytics "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/analytics"
)

func newPostgresRepo(t *testing.T) (*analytics.PostgresRepo, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	})

	return analytics.NewPostgresRepo(db, nil), mock
}

func TestPostgresRepo_TopGenresByFilmOpens(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	limit := 10

	mock.ExpectQuery(`FROM analytic_event AS ae.*ORDER BY open_count DESC, g.id ASC`).
		WithArgs(from, to, limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "open_count"}).
			AddRow(int64(1), "drama", "drama", int64(12)).
			AddRow(int64(3), "comedy", "comedy", int64(7)))

	stats, err := repo.TopGenresByFilmOpens(context.Background(), from, to, limit)
	require.NoError(t, err)
	assert.Equal(t, []domain.GenreOpenStat{
		{GenreID: 1, Name: "drama", Slug: "drama", OpenCount: 12},
		{GenreID: 3, Name: "comedy", Slug: "comedy", OpenCount: 7},
	}, stats)
}

func TestPostgresRepo_TopGenresByFilmOpensEmpty(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM analytic_event AS ae`).
		WithArgs(from, to, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "open_count"}))

	stats, err := repo.TopGenresByFilmOpens(context.Background(), from, to, 5)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Empty(t, stats)
}

func TestPostgresRepo_TopGenresByFilmOpensQueryError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	wantErr := errors.New("database unavailable")

	mock.ExpectQuery(`FROM analytic_event AS ae`).
		WithArgs(from, to, 5).
		WillReturnError(wantErr)

	stats, err := repo.TopGenresByFilmOpens(context.Background(), from, to, 5)
	assert.ErrorIs(t, err, wantErr)
	assert.NotNil(t, stats)
	assert.Empty(t, stats)
}

func TestPostgresRepo_TopGenresByFilmOpensScanError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM analytic_event AS ae`).
		WithArgs(from, to, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug"}).
			AddRow(int64(1), "drama", "drama"))

	stats, err := repo.TopGenresByFilmOpens(context.Background(), from, to, 5)
	require.Error(t, err)
	assert.Empty(t, stats)
}

func TestPostgresRepo_TopGenresByFilmOpensRowsError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	wantErr := errors.New("rows interrupted")
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "open_count"}).
		AddRow(int64(1), "drama", "drama", int64(12)).
		RowError(0, wantErr)

	mock.ExpectQuery(`FROM analytic_event AS ae`).
		WithArgs(from, to, 5).
		WillReturnRows(rows)

	stats, err := repo.TopGenresByFilmOpens(context.Background(), from, to, 5)
	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, stats)
}

func TestPostgresRepo_GenreOpensByAge(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WITH filtered_events AS.*ORDER BY CASE age_group`).
		WithArgs(from, to, "drama").
		WillReturnRows(sqlmock.NewRows([]string{"age_group", "genre_open_count", "total_open_count", "share"}).
			AddRow("18_24", int64(3), int64(12), 0.25).
			AddRow("unknown", int64(2), int64(4), 0.5))

	stats, err := repo.GenreOpensByAge(context.Background(), "drama", from, to)
	require.NoError(t, err)
	assert.Equal(t, []domain.AgeGroupOpenStat{
		{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 3, TotalOpenCount: 12, Share: 0.25},
		{AgeGroup: domain.AgeGroupUnknown, GenreOpenCount: 2, TotalOpenCount: 4, Share: 0.5},
	}, stats)
}

func TestPostgresRepo_GenreOpensByAgeEmpty(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WITH filtered_events AS`).
		WithArgs(from, to, "drama").
		WillReturnRows(sqlmock.NewRows([]string{"age_group", "genre_open_count", "total_open_count", "share"}))

	stats, err := repo.GenreOpensByAge(context.Background(), "drama", from, to)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Empty(t, stats)
}

func TestPostgresRepo_GenreOpensByAgeQueryError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	wantErr := errors.New("database unavailable")

	mock.ExpectQuery(`WITH filtered_events AS`).
		WithArgs(from, to, "drama").
		WillReturnError(wantErr)

	stats, err := repo.GenreOpensByAge(context.Background(), "drama", from, to)
	assert.ErrorIs(t, err, wantErr)
	assert.NotNil(t, stats)
	assert.Empty(t, stats)
}

func TestPostgresRepo_GenreOpensByAgeScanError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WITH filtered_events AS`).
		WithArgs(from, to, "drama").
		WillReturnRows(sqlmock.NewRows([]string{"age_group", "genre_open_count", "total_open_count"}).
			AddRow("18_24", int64(3), int64(12)))

	stats, err := repo.GenreOpensByAge(context.Background(), "drama", from, to)
	require.Error(t, err)
	assert.Empty(t, stats)
}

func TestPostgresRepo_GenreOpensByAgeRowsError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	wantErr := errors.New("rows interrupted")
	rows := sqlmock.NewRows([]string{"age_group", "genre_open_count", "total_open_count", "share"}).
		AddRow("18_24", int64(3), int64(12), 0.25).
		RowError(0, wantErr)

	mock.ExpectQuery(`WITH filtered_events AS`).
		WithArgs(from, to, "drama").
		WillReturnRows(rows)

	stats, err := repo.GenreOpensByAge(context.Background(), "drama", from, to)
	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, stats)
}

func TestPostgresRepo_RecordFilmOpen(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	accountID := domain.UserID(42)
	event := domain.FilmOpenEvent{
		VisitorPublicID: "visitor_public_1234",
		AccountID:       &accountID,
		FilmID:          7,
		OccurredAt:      time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC),
	}

	mock.ExpectQuery(`WITH upserted_visitor AS.*INSERT INTO analytic_event`).
		WithArgs(event.VisitorPublicID, accountID, event.FilmID, event.OccurredAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))

	err := repo.RecordFilmOpen(context.Background(), event)
	require.NoError(t, err)
}

func TestPostgresRepo_RecordFilmOpenGuest(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	event := domain.FilmOpenEvent{
		VisitorPublicID: "visitor_public_1234",
		FilmID:          7,
		OccurredAt:      time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC),
	}

	mock.ExpectQuery(`WITH upserted_visitor AS.*INSERT INTO analytic_event`).
		WithArgs(event.VisitorPublicID, nil, event.FilmID, event.OccurredAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(100)))

	err := repo.RecordFilmOpen(context.Background(), event)
	require.NoError(t, err)
}

func TestPostgresRepo_RecordFilmOpenError(t *testing.T) {
	repo, mock := newPostgresRepo(t)
	wantErr := errors.New("database unavailable")
	event := domain.FilmOpenEvent{
		VisitorPublicID: "visitor_public_1234",
		FilmID:          7,
		OccurredAt:      time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC),
	}

	mock.ExpectQuery(`WITH upserted_visitor AS.*INSERT INTO analytic_event`).
		WithArgs(event.VisitorPublicID, nil, event.FilmID, event.OccurredAt).
		WillReturnError(wantErr)

	err := repo.RecordFilmOpen(context.Background(), event)
	assert.ErrorIs(t, err, wantErr)
}
