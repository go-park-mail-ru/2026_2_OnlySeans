package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type sliceConverter struct{}

func (sliceConverter) ConvertValue(value any) (driver.Value, error) {
	if ids, ok := value.([]int64); ok {
		return ids, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(value)
}

func testDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual), sqlmock.ValueConverterOption(sliceConverter{}))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})

	return db, mock
}

func testLogger(t *testing.T) logger.Logger {
	t.Helper()

	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
	require.NoError(t, err)

	return appLogger
}

func filmRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "title", "original_title", "film_type", "production_year", "duration_min", "age_limit", "description", "poster_url", "trailer_url", "poster_storage_key",
	})
}

func TestFilmRepoList(t *testing.T) {
	failure := errors.New("database failure")

	cases := []struct {
		name      string
		empty     bool
		queryErr  error
		commitErr error
		want      error
	}{
		{
			name: "films and genres",
		},
		{
			name:  "empty page preserves total",
			empty: true,
		},
		{
			name:     "query failure rolls back",
			queryErr: failure,
			want:     failure,
		},
		{
			name:      "commit failure",
			commitErr: failure,
			want:      failure,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			mock.ExpectBegin()
			mock.ExpectQuery(countFilmsQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(37))

			query := mock.ExpectQuery(listFilmsQuery).WithArgs(20, 40)
			if tc.queryErr != nil {
				query.WillReturnError(tc.queryErr)
				mock.ExpectRollback()
			} else {
				rows := filmRows()
				if !tc.empty {
					rows.AddRow(7, "Film", nil, "movie", 2020, nil, 12, nil, nil, nil, "poster.png")
				}

				query.WillReturnRows(rows).RowsWillBeClosed()

				if !tc.empty {
					mock.ExpectQuery(filmGenresQuery).WithArgs(sqlmock.AnyArg()).WillReturnRows(
						sqlmock.NewRows([]string{"film_id", "id", "name", "slug"}).AddRow(7, 3, "Drama", "drama"),
					).RowsWillBeClosed()
				}

				commit := mock.ExpectCommit()
				if tc.commitErr != nil {
					commit.WillReturnError(tc.commitErr)
				}
			}

			films, total, err := NewFilmRepo(db, nil).ListFilms(t.Context(), 20, 40)
			require.ErrorIs(t, err, tc.want)

			if tc.want != nil {
				return
			}

			require.Equal(t, 37, total)

			if tc.empty {
				require.Empty(t, films)
				require.NotNil(t, films)
			} else {
				require.Len(t, films, 1)
				require.Equal(t, domain.FilmID(7), films[0].ID)
				require.Equal(t, []domain.Genre{{
					ID:   3,
					Name: "Drama",
					Slug: "drama",
				}}, films[0].Genres)
				require.NotNil(t, films[0].PosterStorageKey)
				require.Equal(t, "poster.png", *films[0].PosterStorageKey)
			}
		})
	}
}

func TestFilmRepoWriteNotFound(t *testing.T) {
	cases := []struct {
		name     string
		update   bool
		affected int64
		want     error
	}{
		{
			name:     "update existing",
			update:   true,
			affected: 1,
		},
		{
			name:   "update missing",
			update: true,
			want:   domain.ErrFilmNotFound,
		},
		{
			name:     "delete existing",
			affected: 1,
		},
		{
			name: "delete missing",
			want: domain.ErrFilmNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			repo := NewFilmRepo(db, nil)
			var err error

			if tc.update {
				mock.ExpectExec(updateFilmQuery).WithArgs(domain.FilmID(7), "Film", nil, domain.FilmTypeMovie, int16(2020), nil, int16(12), nil).WillReturnResult(sqlmock.NewResult(0, tc.affected))
				err = repo.UpdateFilm(t.Context(), 7, domain.CreateFilmInput{
					Title:       "Film",
					FilmType:    domain.FilmTypeMovie,
					ReleaseYear: 2020,
					AgeLimit:    12,
				})
			} else {
				mock.ExpectExec(deleteFilmQuery).WithArgs(domain.FilmID(7)).WillReturnResult(sqlmock.NewResult(0, tc.affected))

				err = repo.DeleteFilm(t.Context(), 7)
			}

			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCollectionRepoPage(t *testing.T) {
	cases := []struct {
		name    string
		missing bool
		want    error
	}{
		{
			name: "empty page preserves total",
		},
		{
			name:    "missing collection",
			missing: true,
			want:    domain.ErrCollectionNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)

			lookup := mock.ExpectQuery(getCollectionQuery).WithArgs(domain.CollectionID(5))
			if tc.missing {
				lookup.WillReturnError(sql.ErrNoRows)
			} else {
				lookup.WillReturnRows(sqlmock.NewRows([]string{"id", "title", "slug", "description", "cover_url"}).AddRow(5, "Collection", "collection", nil, nil))
				mock.ExpectBegin()
				mock.ExpectQuery(countCollectionFilmsQuery).WithArgs(domain.CollectionID(5)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery(listCollectionFilmsQuery).WithArgs(domain.CollectionID(5), 20, 100).WillReturnRows(filmRows()).RowsWillBeClosed()
				mock.ExpectCommit()
			}

			films, total, err := NewFilmRepo(db, nil).ListCollectionFilms(t.Context(), 5, 20, 100)
			require.ErrorIs(t, err, tc.want)

			if tc.want == nil {
				require.Empty(t, films)
				require.NotNil(t, films)
				require.Equal(t, 3, total)
			}
		})
	}
}

func TestRatingRepoErrors(t *testing.T) {
	failure := errors.New("database failure")

	cases := []struct {
		name  string
		dbErr error
		want  error
	}{
		{
			name: "missing film",
			dbErr: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: "film_rating_film_id_fkey",
			},
			want: domain.ErrFilmNotFound,
		},
		{
			name: "invalid score",
			dbErr: &pgconn.PgError{
				Code:           checkViolation,
				ConstraintName: "film_rating_score_check",
			},
			want: domain.ErrInvalidFilmRating,
		},
		{
			name: "unrelated constraint",
			dbErr: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: "film_rating_account_id_fkey",
			},
		},
		{
			name:  "database failure",
			dbErr: failure,
			want:  failure,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			mock.ExpectQuery(setFilmRatingQuery).WithArgs(domain.UserID(7), domain.FilmID(42), 8).WillReturnError(tc.dbErr)

			_, err := NewRatingRepo(db, nil).SetFilmRating(t.Context(), 7, 42, 8)

			want := tc.want
			if want == nil {
				want = tc.dbErr
			}

			require.ErrorIs(t, err, want)
		})
	}
}

func TestRatingRepoPersonalScore(t *testing.T) {
	failure := errors.New("database failure")

	cases := []struct {
		name    string
		present bool
		dbErr   error
	}{
		{
			name:    "saved",
			present: true,
		},
		{
			name: "not yet rated",
		},
		{
			name:  "database failure",
			dbErr: failure,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)

			query := mock.ExpectQuery("SELECT score FROM film_rating WHERE account_id = $1 AND film_id = $2").WithArgs(domain.UserID(7), domain.FilmID(42))
			if tc.dbErr != nil {
				query.WillReturnError(tc.dbErr)
			} else {
				rows := sqlmock.NewRows([]string{"score"})
				if tc.present {
					rows.AddRow(8)
				}

				query.WillReturnRows(rows)
			}

			score, err := NewRatingRepo(db, nil).GetMyScore(t.Context(), 7, 42)
			require.ErrorIs(t, err, tc.dbErr)

			if tc.present {
				require.NotNil(t, score)
				require.Equal(t, 8, *score)
			} else {
				require.Nil(t, score)
			}
		})
	}
}

func TestLibraryRepoSetSaved(t *testing.T) {
	failure := errors.New("remove watchlater failed")

	cases := []struct {
		name       string
		list       domain.List
		saved      bool
		cleanupErr error
		want       error
	}{
		{
			name:  "add favorite",
			list:  domain.Favorite,
			saved: true,
		},
		{
			name:  "watched removes watchlater atomically",
			list:  domain.Watched,
			saved: true,
		},
		{
			name:       "watchlater removal rolls back",
			list:       domain.Watched,
			saved:      true,
			cleanupErr: failure,
			want:       failure,
		},
		{
			name: "remove saved film",
			list: domain.WatchLater,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			target := folders[tc.list]

			mock.ExpectBegin()
			mock.ExpectExec(lockAccountQuery).WithArgs(domain.UserID(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(folderQuery).WithArgs(domain.UserID(7), target.title, target.folderType).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

			query := removeFilmQuery
			if tc.saved {
				query = addFilmQuery
			}

			mock.ExpectExec(query).WithArgs(int64(3), domain.FilmID(42)).WillReturnResult(sqlmock.NewResult(0, 1))

			if tc.saved && tc.list == domain.Watched {
				cleanup := mock.ExpectExec(removeFromWatchLaterQuery).WithArgs(domain.UserID(7), domain.FilmID(42))
				if tc.cleanupErr != nil {
					cleanup.WillReturnError(tc.cleanupErr)
				} else {
					cleanup.WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}

			if tc.want != nil {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}

			err := NewLibraryRepo(db, nil).SetSaved(t.Context(), 7, tc.list, 42, tc.saved)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPreferencesRepoRead(t *testing.T) {
	disabled := false
	grid := domain.CatalogViewGrid

	cases := []struct {
		name    string
		raw     string
		missing bool
		want    domain.Preferences
		wantErr error
	}{
		{
			name: "explicit false and grid",
			raw:  `{"frame:sidebar-collapsed":false,"frame:catalog-view":"grid"}`,
			want: domain.Preferences{
				SidebarCollapsed: &disabled,
				CatalogView:      &grid,
			},
		},
		{
			name: "unset fields",
			raw:  `{}`,
		},
		{
			name: "unrelated settings",
			raw:  `{"other-setting":123}`,
		},
		{
			name:    "missing account",
			missing: true,
			wantErr: sql.ErrNoRows,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)

			query := mock.ExpectQuery(preferencesQuery).WithArgs(domain.UserID(7))
			if tc.missing {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"preferences"}).AddRow([]byte(tc.raw)))
			}

			result, err := NewPreferencesRepo(db, nil).Preferences(t.Context(), 7)
			require.ErrorIs(t, err, tc.wantErr)
			require.True(t, reflect.DeepEqual(tc.want, result), "result = %+v", result)
		})
	}
}

func TestPreferencesRepoSave(t *testing.T) {
	disabled := false
	enabled := true
	grid := domain.CatalogViewGrid
	list := domain.CatalogViewList

	cases := []struct {
		name      string
		update    domain.PreferenceUpdate
		key       string
		jsonValue string
		want      error
	}{
		{
			name: "explicit false",
			update: domain.PreferenceUpdate{
				SidebarCollapsed: &disabled,
			},
			key:       "frame:sidebar-collapsed",
			jsonValue: "false",
		},
		{
			name: "explicit true",
			update: domain.PreferenceUpdate{
				SidebarCollapsed: &enabled,
			},
			key:       "frame:sidebar-collapsed",
			jsonValue: "true",
		},
		{
			name: "grid",
			update: domain.PreferenceUpdate{
				CatalogView: &grid,
			},
			key:       "frame:catalog-view",
			jsonValue: `"grid"`,
		},
		{
			name: "list",
			update: domain.PreferenceUpdate{
				CatalogView: &list,
			},
			key:       "frame:catalog-view",
			jsonValue: `"list"`,
		},
		{
			name: "empty update",
			want: domain.ErrInvalidPreferences,
		},
		{
			name: "two settings",
			update: domain.PreferenceUpdate{
				SidebarCollapsed: &disabled,
				CatalogView:      &grid,
			},
			want: domain.ErrInvalidPreferences,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			if tc.want == nil {
				mock.ExpectExec(savePreferenceQuery).WithArgs(domain.UserID(7), tc.key, tc.jsonValue).WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := NewPreferencesRepo(db, nil).SavePreference(t.Context(), 7, tc.update)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestRatingRepoCollectionPage(t *testing.T) {
	cases := []struct {
		name    string
		missing bool
		empty   bool
		want    error
	}{
		{
			name: "film with rating",
		},
		{
			name:  "empty page preserves total",
			empty: true,
		},
		{
			name:    "missing collection",
			missing: true,
			want:    domain.ErrCollectionNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)
			rows := sqlmock.NewRows([]string{"collection_id", "total", "film_id", "average", "votes"})

			if !tc.missing {
				if tc.empty {
					rows.AddRow(5, 3, nil, 0, 0)
				} else {
					rows.AddRow(5, 3, 42, 8.5, 2)
				}
			}

			mock.ExpectQuery(getCollectionFilmRatingSummariesQuery).WithArgs(domain.CollectionID(5), 20, 100).WillReturnRows(rows).RowsWillBeClosed()

			result, total, err := NewRatingRepo(db, nil).GetCollectionFilmRatingSummaries(t.Context(), 5, 20, 100)
			require.ErrorIs(t, err, tc.want)

			if tc.want != nil {
				return
			}

			require.Equal(t, 3, total)

			if tc.empty {
				require.Empty(t, result)
				require.NotNil(t, result)
			} else {
				require.Equal(t, []domain.FilmRatingSummary{{
					FilmID:       42,
					AverageScore: 8.5,
					Votes:        2,
				}}, result)
			}
		})
	}
}

func TestPosterRepoRead(t *testing.T) {
	cases := []struct {
		name     string
		missing  bool
		noPoster bool
		want     error
	}{
		{
			name: "poster present",
		},
		{
			name:    "missing film",
			missing: true,
			want:    domain.ErrFilmNotFound,
		},
		{
			name:     "film without poster",
			noPoster: true,
			want:     domain.ErrFileNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := testDB(t)

			query := mock.ExpectQuery(getFilmPosterQuery).WithArgs(domain.FilmID(42))
			if tc.missing {
				query.WillReturnError(sql.ErrNoRows)
			} else if tc.noPoster {
				query.WillReturnRows(sqlmock.NewRows([]string{"key", "mime", "size", "external_url"}).AddRow("", nil, 0, nil))
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"key", "mime", "size", "external_url"}).AddRow("poster.png", "image/png", 123, nil))
			}

			repo, err := NewPosterRepo(db, testLogger(t))
			require.NoError(t, err)
			poster, err := repo.GetPoster(t.Context(), 42)
			require.ErrorIs(t, err, tc.want)

			if tc.want == nil {
				require.Equal(t, &domain.PosterFile{
					Filename:    "poster.png",
					ContentType: "image/png",
					Size:        123,
				}, poster)
			}
		})
	}
}
