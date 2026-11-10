package auth_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authrepo "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

var profileFilmColumns = []string{"id", "title", "production_year", "film_type", "age_limit", "genres", "poster_url", "storage_key", "average"}

func TestProfileRepoGet(t *testing.T) {
	created := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	failure := errors.New("database unavailable")
	for _, phase := range []string{"success", "empty", "begin", "missing", "user", "favorites", "favorite scan", "favorite rows", "ratings", "rating scan", "commit"} {
		t.Run(phase, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo := authrepo.NewProfileRepo(db, nil)
			begin := mock.ExpectBegin()
			if phase == "begin" {
				begin.WillReturnError(failure)
			} else {
				user := mock.ExpectQuery("FROM account AS a WHERE a.id = \\$1 AND a.deleted_at IS NULL").WithArgs(domain.UserID(7))
				if phase == "missing" || phase == "user" {
					if phase == "missing" {
						user.WillReturnError(sql.ErrNoRows)
					} else {
						user.WillReturnError(failure)
					}
					mock.ExpectRollback()
				} else {
					user.WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "created_at", "avatar", "reviews"}).AddRow(7, "anna@example.com", "anna", created, "avatar.png", 3))
					favorites := mock.ExpectQuery("FROM folder AS folder").WithArgs(domain.UserID(7))
					if phase == "favorites" {
						favorites.WillReturnError(failure)
						mock.ExpectRollback()
					} else {
						rows := sqlmock.NewRows(profileFilmColumns)
						if phase == "favorite scan" {
							rows.AddRow("invalid-id", "Film", 2020, "movie", 16, "Drama", nil, nil, 8.5)
						} else if phase != "empty" {
							rows.AddRow(5, "Film", 2020, "movie", 16, "Drama, Sci-Fi", nil, "poster.png", 8.5)
						}
						if phase == "favorite rows" {
							rows.RowError(0, failure)
						}
						favorites.WillReturnRows(rows).RowsWillBeClosed()
						if phase == "favorite scan" || phase == "favorite rows" {
							mock.ExpectRollback()
						} else {
							ratings := mock.ExpectQuery("FROM film_rating AS rating").WithArgs(domain.UserID(7))
							if phase == "ratings" {
								ratings.WillReturnError(failure)
								mock.ExpectRollback()
							} else {
								rows := sqlmock.NewRows(append(append([]string{}, profileFilmColumns...), "score"))
								if phase == "rating scan" {
									rows.AddRow(5, "Film", 2020, "movie", 16, "Drama", nil, nil, 8.5, "invalid-score")
								} else if phase != "empty" {
									rows.AddRow(5, "Film", 2020, "movie", 16, "Drama", "https://example.com/poster.png", nil, 8.5, 9)
								}
								ratings.WillReturnRows(rows).RowsWillBeClosed()
								if phase == "rating scan" {
									mock.ExpectRollback()
								} else {
									commit := mock.ExpectCommit()
									if phase == "commit" {
										commit.WillReturnError(failure)
									}
								}
							}
						}
					}
				}
			}
			result, err := repo.GetProfile(t.Context(), 7)
			switch phase {
			case "success", "empty":
				require.NoError(t, err)
				require.Equal(t, domain.UserID(7), result.User.ID)
				require.Equal(t, created, result.User.CreatedAt)
				require.Equal(t, 3, result.Reviews)
				require.NotNil(t, result.Favorites)
				require.NotNil(t, result.Ratings)
				if phase == "success" {
					require.Len(t, result.Favorites, 1)
					require.Equal(t, "Drama, Sci-Fi", result.Favorites[0].Genres)
					require.Equal(t, "poster.png", *result.Favorites[0].PosterStorageKey)
					require.Equal(t, 8.5, result.Favorites[0].Average)
					require.Nil(t, result.Favorites[0].Score)
					require.Equal(t, 9, *result.Ratings[0].Score)
				} else {
					require.Empty(t, result.Favorites)
					require.Empty(t, result.Ratings)
				}
			case "missing":
				require.ErrorIs(t, err, domain.ErrUserNotFound)
				require.Nil(t, result)
			default:
				require.Error(t, err)
				require.Nil(t, result)
				if phase != "favorite scan" && phase != "rating scan" {
					require.ErrorIs(t, err, failure)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProfileRepoUpdate(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tc := range []struct {
		name           string
		dbErr, wantErr error
	}{
		{"success", nil, nil},
		{"missing", sql.ErrNoRows, domain.ErrUserNotFound},
		{"email", &pgconn.PgError{Code: "23505", ConstraintName: "account_email_key"}, domain.ErrUserExists},
		{"username", &pgconn.PgError{Code: "23505", ConstraintName: "account_username_key"}, domain.ErrUsernameTaken},
		{"failure", failure, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			query := mock.ExpectQuery("UPDATE account SET email=\\$2,username=\\$3").WithArgs(domain.UserID(7), "anna@example.com", "anna")
			if tc.dbErr != nil {
				query.WillReturnError(tc.dbErr)
			} else {
				query.WillReturnRows(sqlmock.NewRows(userColumns).AddRow(7, "anna@example.com", "anna", "hash", "reader", nil))
			}
			result, err := authrepo.NewPostgresUserRepo(db, nil).UpdateProfile(t.Context(), 7, "anna@example.com", "anna")
			require.ErrorIs(t, err, tc.wantErr)
			if err == nil {
				require.Equal(t, domain.UserID(7), result.ID)
			} else {
				require.Nil(t, result)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProfileRepoUnknownConflictIsNotEmailConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	failure := &pgconn.PgError{Code: "23505", ConstraintName: "unknown_constraint"}
	mock.ExpectQuery("UPDATE account").WillReturnError(failure)
	_, err = authrepo.NewPostgresUserRepo(db, nil).UpdateProfile(t.Context(), 7, "anna@example.com", "anna")
	require.ErrorIs(t, err, failure)
	require.NotErrorIs(t, err, domain.ErrUserExists)
	require.NotErrorIs(t, err, domain.ErrUsernameTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}
