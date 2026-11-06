package files_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/stretchr/testify/require"
)

func TestPostgresAvatarRepoSave(t *testing.T) {
	dbErr := errors.New("database failure")
	tests := []struct {
		name    string
		failure string
		wantErr error
	}{
		{name: "file and account committed together"},
		{name: "missing or deleted account", failure: "account", wantErr: domain.ErrUserNotFound},
		{name: "insert failure rolls back", failure: "insert", wantErr: dbErr},
		{name: "account update failure rolls back", failure: "update", wantErr: dbErr},
		{name: "commit outcome unknown", failure: "commit", wantErr: domain.ErrAvatarCommitUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo, err := filesservice.NewPostgresAvatarRepo(db)
			require.NoError(t, err)
			mock.ExpectBegin()
			lock := mock.ExpectQuery("SELECT id FROM account.*FOR UPDATE").WithArgs(domain.UserID(7))
			if tt.failure == "account" {
				lock.WillReturnError(sql.ErrNoRows)
			} else {
				lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
				insert := mock.ExpectQuery("INSERT INTO file").WithArgs("avatar.png", "image/png", int64(123), domain.UserID(7))
				if tt.failure == "insert" {
					insert.WillReturnError(dbErr)
				} else {
					insert.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
					update := mock.ExpectExec("UPDATE account SET avatar_file_id").WithArgs(int64(42), domain.UserID(7))
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
			err = repo.SaveAvatar(t.Context(), 7, domain.AvatarFile{Filename: "avatar.png", ContentType: "image/png", Size: 123})
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
		{name: "current metadata"},
		{name: "missing avatar", missingAvatar: true, wantErr: domain.ErrFileNotFound},
		{name: "missing user", missingUser: true, wantErr: domain.ErrUserNotFound},
		{name: "database failure", queryErr: failure, wantErr: failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo, err := filesservice.NewPostgresAvatarRepo(db)
			require.NoError(t, err)
			query := mock.ExpectQuery("SELECT f.storage_key, f.mime_type, f.size_bytes").WithArgs(domain.UserID(7))
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
	tests := []struct{ name, failureAt string }{
		{name: "detached"},
		{name: "missing user", failureAt: "user"},
		{name: "update rolled back", failureAt: "update"},
		{name: "ordinary commit error", failureAt: "commit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo, err := filesservice.NewPostgresAvatarRepo(db)
			require.NoError(t, err)
			mock.ExpectBegin()
			lock := mock.ExpectQuery("SELECT id FROM account.*FOR UPDATE").WithArgs(domain.UserID(7))
			if tt.failureAt == "user" {
				lock.WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			} else {
				lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
				update := mock.ExpectExec("UPDATE account SET avatar_file_id = NULL").WithArgs(domain.UserID(7))
				if tt.failureAt == "update" {
					update.WillReturnError(failure)
					mock.ExpectRollback()
				} else {
					update.WillReturnResult(sqlmock.NewResult(0, 1))
					if tt.failureAt == "commit" {
						mock.ExpectCommit().WillReturnError(failure)
					} else {
						mock.ExpectCommit()
					}
				}
			}
			err = repo.DeleteAvatar(t.Context(), 7)
			if tt.failureAt == "user" {
				require.ErrorIs(t, err, domain.ErrUserNotFound)
			} else if tt.failureAt != "" {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
			require.NotErrorIs(t, err, domain.ErrAvatarCommitUnknown)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
