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
