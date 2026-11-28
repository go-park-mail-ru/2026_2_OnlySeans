package filecleanup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/filecleanup"
)

type failingStorage struct{ err error }

func (s failingStorage) DeleteFile(context.Context, string) error { return s.err }

func TestCleanupPreservesStorageAndRecoveryErrors(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	retryErr := errors.New("retry update unavailable")
	rollbackErr := errors.New("rollback unavailable")

	for _, tc := range []struct {
		name        string
		retryErr    error
		rollbackErr error
	}{
		{name: "retry scheduled"},
		{
			name:     "retry update fails",
			retryErr: retryErr,
		},
		{
			name:        "rollback fails",
			rollbackErr: rollbackErr,
		},
		{
			name:        "rollback and retry update fail",
			retryErr:    retryErr,
			rollbackErr: rollbackErr,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, db.Close())
				require.NoError(t, mock.ExpectationsWereMet())
			})
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT file_id FROM file_cleanup_queue").WillReturnRows(sqlmock.NewRows([]string{"file_id"}).AddRow(7))
			mock.ExpectQuery("SELECT storage_key FROM file").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"storage_key"}).AddRow("old.png"))
			mock.ExpectQuery("SELECT file_has_references").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"referenced"}).AddRow(false))

			rollback := mock.ExpectRollback()
			if tc.rollbackErr != nil {
				rollback.WillReturnError(tc.rollbackErr)
			}

			retry := mock.ExpectExec("UPDATE file_cleanup_queue").WithArgs(int64(7))
			if tc.retryErr != nil {
				retry.WillReturnError(tc.retryErr)
			} else {
				retry.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			mock.ExpectClose()

			processed, err := filecleanup.New(db, failingStorage{err: storageErr}).ProcessOne(t.Context())
			require.True(t, processed)
			require.ErrorIs(t, err, storageErr)

			if tc.retryErr != nil {
				require.ErrorIs(t, err, tc.retryErr)
				require.Contains(t, err.Error(), "schedule cleanup retry for file 7")
			}

			if tc.rollbackErr != nil {
				require.ErrorIs(t, err, tc.rollbackErr)
				require.Contains(t, err.Error(), "rollback file cleanup")
			}
		})
	}
}
