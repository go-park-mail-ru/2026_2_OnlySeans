package postgres_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/postgres"
)

func TestPostgresAvatarRepoSave(t *testing.T) {
	dbErr := errors.New("database failure")

	tests := []struct {
		name    string
		failure string
		wantErr error
	}{
		{
			name: "file and account committed together",
		},
		{
			name:    "missing or deleted account",
			failure: "account",
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:    "insert failure rolls back",
			failure: "insert",
			wantErr: dbErr,
		},
		{
			name:    "account update failure rolls back",
			failure: "update",
			wantErr: dbErr,
		},
		{
			name:    "commit outcome unknown",
			failure: "commit",
			wantErr: domain.ErrCommitUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)

			t.Cleanup(func() { mock.ExpectClose(); require.NoError(t, db.Close()) })

			repo := filesservice.NewPostgresAvatarRepo(db)

			mock.ExpectBegin()

			lock := mock.ExpectQuery("SELECT id FROM account.*FOR UPDATE").WithArgs(int64(7))
			if tt.failure == "account" {
				lock.WillReturnError(sql.ErrNoRows)
			} else {
				lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

				insert := mock.ExpectQuery("INSERT INTO file").WithArgs("avatar.png", "image/png", int64(123), int64(7))
				if tt.failure == "insert" {
					insert.WillReturnError(dbErr)
				} else {
					insert.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

					update := mock.ExpectExec("UPDATE account SET avatar_file_id").WithArgs(int64(42), int64(7))
					if tt.failure == "update" {
						update.WillReturnError(dbErr)
					} else {
						update.WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			}

			switch tt.failure {
			case "account", "insert", "update":
				mock.ExpectRollback()
			case "commit":
				mock.ExpectCommit().WillReturnError(dbErr)
			default:
				mock.ExpectCommit()
			}

			err = repo.SaveAvatar(t.Context(), 7, domain.AvatarFile{
				Filename:    "avatar.png",
				ContentType: "image/png",
				Size:        123,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresAvatarRepoGet(t *testing.T) {
	failure := errors.New("database unavailable")

	tests := []struct {
		name          string
		missingUser   bool
		missingAvatar bool
		queryErr      error
		wantErr       error
	}{
		{
			name: "current metadata",
		},
		{
			name:          "missing avatar",
			missingAvatar: true,
			wantErr:       domain.ErrAvatarNotFound,
		},
		{
			name:        "missing user",
			missingUser: true,
			wantErr:     domain.ErrUserNotFound,
		},
		{
			name:     "database failure",
			queryErr: failure,
			wantErr:  failure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)

			t.Cleanup(func() { mock.ExpectClose(); require.NoError(t, db.Close()) })

			repo := filesservice.NewPostgresAvatarRepo(db)

			query := mock.ExpectQuery("SELECT f.storage_key, f.mime_type, f.size_bytes").WithArgs(int64(7))

			switch {
			case tt.missingUser:
				query.WillReturnError(sql.ErrNoRows)
			case tt.queryErr != nil:
				query.WillReturnError(tt.queryErr)
			case tt.missingAvatar:
				query.WillReturnRows(sqlmock.NewRows([]string{"key", "type", "size"}).AddRow(nil, nil, nil))
			default:
				query.WillReturnRows(sqlmock.NewRows([]string{"key", "type", "size"}).AddRow("avatar.png", "image/png", 123))
			}

			file, err := repo.GetAvatar(t.Context(), 7)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, file)
			} else {
				require.NoError(t, err)
				require.Equal(t, "avatar.png", file.Filename)
				require.Equal(t, int64(123), file.Size)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresAvatarRepoDelete(t *testing.T) {
	failure := errors.New("database unavailable")

	tests := []struct {
		name     string
		queryErr error
		wantErr  error
	}{
		{
			name: "detached",
		},
		{
			name:     "missing user",
			queryErr: sql.ErrNoRows,
			wantErr:  domain.ErrUserNotFound,
		},
		{
			name:     "database failure",
			queryErr: failure,
			wantErr:  failure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)

			t.Cleanup(func() { mock.ExpectClose(); require.NoError(t, db.Close()) })

			repo := filesservice.NewPostgresAvatarRepo(db)

			query := mock.ExpectQuery("UPDATE account SET avatar_file_id = NULL").WithArgs(int64(7))
			if tt.queryErr != nil {
				query.WillReturnError(tt.queryErr)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
			}

			err = repo.DeleteAvatar(t.Context(), 7)
			require.ErrorIs(t, err, tt.wantErr)
			require.NotErrorIs(t, err, domain.ErrCommitUnknown)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
