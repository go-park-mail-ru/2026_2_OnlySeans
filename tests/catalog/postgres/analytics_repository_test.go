package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/postgres"
)

func newAnalyticsRepo(t *testing.T) (*postgres.AnalyticsRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})
	return postgres.NewAnalyticsRepo(db, nil), mock
}

func TestAnalyticsRepoReports(t *testing.T) {
	repo, mock := newAnalyticsRepo(t)
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	mock.ExpectQuery(`FROM analytic_event AS ae.*ORDER BY open_count DESC`).
		WithArgs(from, to, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "open_count"}).AddRow(1, "drama", "drama", 12))
	top, err := repo.TopGenresByFilmOpens(t.Context(), from, to, 10)
	require.NoError(t, err)
	require.Equal(t, []domain.GenreOpenStat{{GenreID: 1, Name: "drama", Slug: "drama", OpenCount: 12}}, top)

	mock.ExpectQuery(`WITH filtered_events AS.*ORDER BY CASE age_group`).
		WithArgs(from, to, "drama").
		WillReturnRows(sqlmock.NewRows([]string{"age_group", "genre_open_count", "total_open_count", "share"}).AddRow("18_24", 3, 12, 0.25))
	age, err := repo.GenreOpensByAge(t.Context(), "drama", from, to)
	require.NoError(t, err)
	require.Equal(t, []domain.AgeGroupOpenStat{{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 3, TotalOpenCount: 12, Share: 0.25}}, age)
}

func TestAnalyticsRepoRecordsFilmOpen(t *testing.T) {
	repo, mock := newAnalyticsRepo(t)
	event := domain.FilmOpenEvent{VisitorPublicID: "visitor_public_1234", FilmID: 7, OccurredAt: time.Now().UTC()}
	mock.ExpectQuery(`WITH upserted_visitor AS.*INSERT INTO analytic_event`).
		WithArgs(event.VisitorPublicID, nil, event.FilmID, event.OccurredAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))
	require.NoError(t, repo.RecordFilmOpen(t.Context(), event))
}

func TestAnalyticsRepoReturnsStorageError(t *testing.T) {
	repo, mock := newAnalyticsRepo(t)
	wantErr := errors.New("storage unavailable")
	from := time.Now().UTC()
	to := from.Add(time.Hour)
	mock.ExpectQuery(`FROM analytic_event AS ae`).WithArgs(from, to, 5).WillReturnError(wantErr)
	_, err := repo.TopGenresByFilmOpens(t.Context(), from, to, 5)
	require.ErrorIs(t, err, wantErr)
}
