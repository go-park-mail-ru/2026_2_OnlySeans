package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	auth "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
)

var errConnLost = errors.New("connection lost")

// newPostgresUserRepo возвращает хранилище поверх поддельной базы и саму подделку,
// через которую тест задаёт ответы базы. В конце теста проверяется,
// что все ожидаемые запросы действительно были отправлены.
func newPostgresUserRepo(t *testing.T) (*auth.PostgresUserRepo, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	})

	return auth.NewPostgresUserRepo(db), mock
}

var userColumns = []string{"id", "email", "username", "password_hash"}

func TestPostgresUserRepo_Create(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`INSERT INTO account`).
		WithArgs("anna@example.com", "Анна", "hash").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	id, err := repo.Create(context.Background(), "anna@example.com", "Анна", "hash")
	require.NoError(t, err)
	assert.Equal(t, domain.UserID(7), id)
}

func TestPostgresUserRepo_CreateErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{
			name:    "занят email",
			dbErr:   &pgconn.PgError{Code: "23505", ConstraintName: "account_email_key"},
			wantErr: domain.ErrUserExists,
		},
		{
			name:    "занято имя",
			dbErr:   &pgconn.PgError{Code: "23505", ConstraintName: "account_username_key"},
			wantErr: domain.ErrUsernameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresUserRepo(t)
			mock.ExpectQuery(`INSERT INTO account`).WillReturnError(tt.dbErr)

			_, err := repo.Create(context.Background(), "anna@example.com", "anna", "hash")
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresUserRepo_CreateUnexpectedError(t *testing.T) {
	tests := []struct {
		name  string
		dbErr error
	}{
		{"обрыв соединения", errConnLost},
		{"нарушено другое ограничение", &pgconn.PgError{Code: "23514", ConstraintName: "account_email_check"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newPostgresUserRepo(t)
			mock.ExpectQuery(`INSERT INTO account`).WillReturnError(tt.dbErr)

			_, err := repo.Create(context.Background(), "anna@example.com", "anna", "hash")
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.dbErr, "исходная ошибка должна сохраниться внутри обёртки")
			assert.NotErrorIs(t, err, domain.ErrUserExists)
			assert.NotErrorIs(t, err, domain.ErrUsernameTaken)
		})
	}
}

func TestPostgresUserRepo_GetByEmail(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`FROM account\s+WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("anna@example.com").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(7, "anna@example.com", "Анна", "hash"))

	user, err := repo.GetByEmail(context.Background(), "anna@example.com")
	require.NoError(t, err)
	assert.Equal(t, &domain.User{ID: 7, Email: "anna@example.com", Username: "Анна", PasswordHash: "hash"}, user)
}

// вход с SQL в email не обходит проверку строка целиком ищется как email 00($1
func TestPostgresUserRepo_GetByEmailInjection(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)
	payload := `' OR '1'='1`

	mock.ExpectQuery(`WHERE email = \$1`).
		WithArgs(payload).
		WillReturnRows(sqlmock.NewRows(userColumns))

	_, err := repo.GetByEmail(context.Background(), payload)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestPostgresUserRepo_GetUser(t *testing.T) {
	repo, mock := newPostgresUserRepo(t)

	mock.ExpectQuery(`FROM account\s+WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(domain.UserID(7)).
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(7, "anna@example.com", "Анна", "hash"))

	user, err := repo.GetUser(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, domain.UserID(7), user.ID)
	assert.Equal(t, "Анна", user.Username)
}

func TestPostgresUserRepo_GetErrors(t *testing.T) {
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		dbErr   error
		wantErr error
	}{
		{"пользователя нет", sqlmock.NewRows(userColumns), nil, domain.ErrUserNotFound},
		{"база недоступна", nil, errConnLost, errConnLost},
	}

	getters := map[string]func(*auth.PostgresUserRepo) error{
		"GetByEmail": func(repo *auth.PostgresUserRepo) error {
			_, err := repo.GetByEmail(context.Background(), "anna@example.com")
			return err
		},
		"GetUser": func(repo *auth.PostgresUserRepo) error {
			_, err := repo.GetUser(context.Background(), 7)
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
