package reviews_test

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviews "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
)

var reviewColumns = []string{"id", "account_id", "film_id", "title", "content",
	"contains_spoilers", "created_at", "updated_at"}

// newPostgresReviewRepo возвращает хранилище поверх поддельной базы и саму подделку.
// В конце теста проверяется, что все ожидаемые запросы были отправлены.
func newPostgresReviewRepo(t *testing.T) (*reviews.PostgresReviewRepo, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	})

	return reviews.NewPostgresReviewRepo(db), mock
}

func reviewRows(reviewsToReturn ...domain.Review) *sqlmock.Rows {
	rows := sqlmock.NewRows(reviewColumns)
	for _, r := range reviewsToReturn {
		rows.AddRow(r.ID, r.AuthorID, r.FilmID, r.Title, r.Content, r.ContainsSpoilers, r.CreatedAt, r.UpdatedAt)
	}
	return rows
}

func storedReview(id domain.ReviewID) domain.Review {
	created := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	return domain.Review{
		ID:        id,
		AuthorID:  7,
		FilmID:    existingFilm,
		Title:     validTitle,
		Content:   validContent,
		CreatedAt: created,
		UpdatedAt: created,
	}
}

func TestPostgresReviewRepo_Create(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	want := storedReview(1)
	want.ContainsSpoilers = true

	mock.ExpectQuery(`INSERT INTO review`).
		WithArgs(domain.UserID(7), existingFilm, validTitle, validContent, true).
		WillReturnRows(reviewRows(want))

	got, err := repo.Create(context.Background(), 7, existingFilm,
		domain.ReviewInput{Title: validTitle, Content: validContent, ContainsSpoilers: true})
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestPostgresReviewRepo_CreateErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{
			name:    "у автора уже есть отзыв на фильм",
			dbErr:   &pgconn.PgError{Code: "23505", ConstraintName: "review_active_unique_idx"},
			wantErr: domain.ErrReviewExists,
		},
		{
			name:    "фильма нет",
			dbErr:   &pgconn.PgError{Code: "23503", ConstraintName: "review_film_id_fkey"},
			wantErr: domain.ErrFilmNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresReviewRepo(t)
			mock.ExpectQuery(`INSERT INTO review`).WillReturnError(tt.dbErr)

			_, err := repo.Create(context.Background(), 7, existingFilm, validInput())
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresReviewRepo_CreateUnexpectedError(t *testing.T) {
	tests := []struct {
		name  string
		dbErr error
	}{
		{"обрыв соединения", errStorage},
		{"нет автора — это поломка, а не ошибка клиента", &pgconn.PgError{Code: "23503", ConstraintName: "review_account_id_fkey"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresReviewRepo(t)
			mock.ExpectQuery(`INSERT INTO review`).WillReturnError(tt.dbErr)

			_, err := repo.Create(context.Background(), 7, existingFilm, validInput())
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.dbErr)
			assert.NotErrorIs(t, err, domain.ErrFilmNotFound)
			assert.NotErrorIs(t, err, domain.ErrReviewExists)
		})
	}
}

func TestPostgresReviewRepo_GetByID(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	want := storedReview(5)

	mock.ExpectQuery(`FROM review\s+WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(domain.ReviewID(5)).
		WillReturnRows(reviewRows(want))

	got, err := repo.GetByID(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestPostgresReviewRepo_Update(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	want := storedReview(5)
	want.Title = "Передумала"
	want.ContainsSpoilers = true
	want.UpdatedAt = want.CreatedAt.Add(time.Hour)

	mock.ExpectQuery(`UPDATE review SET title = \$2`).
		WithArgs(domain.ReviewID(5), "Передумала", validContent, true).
		WillReturnRows(reviewRows(want))

	got, err := repo.Update(context.Background(), 5,
		domain.ReviewInput{Title: "Передумала", Content: validContent, ContainsSpoilers: true})
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

// SQL в заголовке уходит в базу отдельным значением $2 и сохраняется как обычный текст
func TestPostgresReviewRepo_UpdateInjection(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	payload := `'; DROP TABLE review; --`
	want := storedReview(5)
	want.Title = payload

	mock.ExpectQuery(`UPDATE review SET title = \$2`).
		WithArgs(domain.ReviewID(5), payload, validContent, false).
		WillReturnRows(reviewRows(want))

	got, err := repo.Update(context.Background(), 5, domain.ReviewInput{Title: payload, Content: validContent})
	require.NoError(t, err)
	assert.Equal(t, payload, got.Title)
}

func TestPostgresReviewRepo_GetAndUpdateErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{"отзыва нет или он удалён", nil, domain.ErrReviewNotFound},
		{"база недоступна", errStorage, errStorage},
	}

	calls := map[string]struct {
		query string
		call  func(*reviews.PostgresReviewRepo) error
	}{
		"GetByID": {`SELECT .+ FROM review`, func(repo *reviews.PostgresReviewRepo) error {
			_, err := repo.GetByID(context.Background(), 5)
			return err
		}},
		"Update": {`UPDATE review SET title`, func(repo *reviews.PostgresReviewRepo) error {
			_, err := repo.Update(context.Background(), 5, validInput())
			return err
		}},
	}

	for method, c := range calls {
		for _, tt := range tests {
			t.Run(method+": "+tt.name, func(t *testing.T) {
				repo, mock := newPostgresReviewRepo(t)
				expect := mock.ExpectQuery(c.query)
				if tt.dbErr != nil {
					expect.WillReturnError(tt.dbErr)
				} else {
					expect.WillReturnRows(reviewRows())
				}

				assert.ErrorIs(t, c.call(repo), tt.wantErr)
			})
		}
	}
}

func TestPostgresReviewRepo_Delete(t *testing.T) {
	tests := []struct {
		name     string
		affected int64
		dbErr    error
		wantErr  error
	}{
		{"отзыв помечен удалённым", 1, nil, nil},
		{"отзыва нет или он уже удалён", 0, nil, domain.ErrReviewNotFound},
		{"база недоступна", 0, errStorage, errStorage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresReviewRepo(t)
			expect := mock.ExpectExec(`UPDATE review SET deleted_at = now\(\), deleted_by = \$2`).
				WithArgs(domain.ReviewID(5), domain.UserID(7))
			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
				expect.WillReturnResult(sqlmock.NewResult(0, tt.affected))
			}

			err := repo.Delete(context.Background(), 5, 7)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresReviewRepo_DeleteRowsAffectedError(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	mock.ExpectExec(`UPDATE review SET deleted_at`).
		WillReturnResult(driver.ResultNoRows)

	err := repo.Delete(context.Background(), 5, 7)
	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrReviewNotFound)
}

func TestPostgresReviewRepo_ListByFilm(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	second, first := storedReview(2), storedReview(1)

	mock.ExpectQuery(`SELECT count\(\*\) FROM review`).
		WithArgs(existingFilm).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`ORDER BY id DESC\s+LIMIT \$2 OFFSET \$3`).
		WithArgs(existingFilm, 2, 3).
		WillReturnRows(reviewRows(second, first))

	list, total, err := repo.ListByFilm(context.Background(), existingFilm, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, total, "total — все отзывы фильма, а не только страница")
	assert.Equal(t, []domain.Review{second, first}, list)
}

func TestPostgresReviewRepo_ListByFilmEmpty(t *testing.T) {
	repo, mock := newPostgresReviewRepo(t)
	mock.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`ORDER BY id DESC`).WillReturnRows(reviewRows())

	list, total, err := repo.ListByFilm(context.Background(), existingFilm, 10, 0)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.NotNil(t, list, "пустой список должен уйти фронту как [], а не null")
	assert.Empty(t, list)
}

func TestPostgresReviewRepo_ListByFilmErrors(t *testing.T) {
	brokenRow := sqlmock.NewRows(reviewColumns).AddRow("не число", 7, 1, "t", "c", false, time.Now(), time.Now())
	interrupted := reviewRows(storedReview(2), storedReview(1)).RowError(1, errStorage)

	tests := []struct {
		name    string
		setup   func(sqlmock.Sqlmock)
		wantErr error
	}{
		{"не удалось посчитать", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`SELECT count`).WillReturnError(errStorage)
		}, errStorage},
		{"не удалось получить страницу", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			m.ExpectQuery(`ORDER BY id DESC`).WillReturnError(errStorage)
		}, errStorage},
		{"строка не раскладывается", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			m.ExpectQuery(`ORDER BY id DESC`).WillReturnRows(brokenRow)
		}, nil},
		{"чтение оборвалось на середине", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			m.ExpectQuery(`ORDER BY id DESC`).WillReturnRows(interrupted)
		}, errStorage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresReviewRepo(t)
			tt.setup(mock)

			list, total, err := repo.ListByFilm(context.Background(), existingFilm, 10, 0)
			require.Error(t, err)
			assert.Nil(t, list)
			assert.Zero(t, total)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
