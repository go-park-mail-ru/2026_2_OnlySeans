package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	auth "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/postgres"
)

var errConnLost = errors.New("connection lost")

func newPostgresUserRepo(t *testing.T) (repository.UserRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		mock.ExpectClose()
		assert.NoError(t, db.Close())
	})

	return auth.NewPostgresUserRepo(db, nil), mock
}

var userColumns = []string{"id", "email", "username", "password_hash", "role_code", "avatar_filename"}

func checkPostgresUserRepo_Create(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`INSERT INTO account`).
		WithArgs("anna@example.com", "Анна", "hash").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	id, err := repo.CreateUser(context.Background(), domain.User{
		Email:        "anna@example.com",
		Username:     "Анна",
		PasswordHash: "hash",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(7), id)
}

func checkPostgresUserRepo_CreateErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{
			name: "занят email",
			dbErr: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "account_email_key",
			},
			wantErr: domain.ErrUserExists,
		},
		{
			name: "занято имя",
			dbErr: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "account_username_key",
			},
			wantErr: domain.ErrUsernameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresUserRepo(t)
			mock.ExpectQuery(`INSERT INTO account`).WillReturnError(tt.dbErr)

			_, err := repo.CreateUser(context.Background(), domain.User{
				Email:        "anna@example.com",
				Username:     "anna",
				PasswordHash: "hash",
			})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func checkPostgresUserRepo_CreateUnexpectedError(t *testing.T) {
	tests := []struct {
		name  string
		dbErr error
	}{
		{
			name:  "обрыв соединения",
			dbErr: errConnLost,
		},
		{
			name: "нарушено другое ограничение",
			dbErr: &pgconn.PgError{
				Code:           "23514",
				ConstraintName: "account_email_check",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresUserRepo(t)
			mock.ExpectQuery(`INSERT INTO account`).WillReturnError(tt.dbErr)

			_, err := repo.CreateUser(context.Background(), domain.User{
				Email:        "anna@example.com",
				Username:     "anna",
				PasswordHash: "hash",
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.dbErr, "исходная ошибка должна сохраниться внутри обёртки")
			assert.NotErrorIs(t, err, domain.ErrUserExists)
			assert.NotErrorIs(t, err, domain.ErrUsernameTaken)
		})
	}
}

func checkPostgresUserRepo_GetByEmail(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`FROM account\s+WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("anna@example.com").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(7, "anna@example.com", "Анна", "hash", "reader", nil))

	user, err := repo.GetUserByEmail(context.Background(), "anna@example.com")
	require.NoError(t, err)
	assert.Equal(t, domain.User{
		Role:         "reader",
		ID:           7,
		Email:        "anna@example.com",
		Username:     "Анна",
		PasswordHash: "hash",
	}, user)
}

func checkPostgresUserRepo_GetUser(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`FROM account\s+WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(7, "anna@example.com", "Анна", "hash", "reader", nil))

	user, err := repo.GetUserByID(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(7), user.ID)
	assert.Equal(t, "Анна", user.Username)
}

func checkPostgresUserRepo_GetErrors(t *testing.T) {
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		dbErr   error
		wantErr error
	}{
		{
			name:    "пользователя нет",
			rows:    sqlmock.NewRows(userColumns),
			dbErr:   nil,
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:    "база недоступна",
			rows:    nil,
			dbErr:   errConnLost,
			wantErr: errConnLost,
		},
	}

	getters := map[string]func(repository.UserRepository) error{
		"GetByEmail": func(repo repository.UserRepository) error {
			_, err := repo.GetUserByEmail(context.Background(), "anna@example.com")
			return err
		},
		"GetUser": func(repo repository.UserRepository) error {
			_, err := repo.GetUserByID(context.Background(), 7)
			return err
		},
	}

	for method, call := range getters {
		for _, tt := range tests {
			t.Run(method+": "+tt.name, func(t *testing.T) {
				repo, mock := newPostgresUserRepo(t)
				query := mock.ExpectQuery(`FROM account`)

				if tt.dbErr != nil {
					query.WillReturnError(tt.dbErr)
				} else {
					query.WillReturnRows(tt.rows)
				}

				assert.ErrorIs(t, call(repo), tt.wantErr)
			})
		}
	}
}
