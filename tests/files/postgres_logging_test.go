package files_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	files "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestPosterTransactionLogsActualOutcome(t *testing.T) {
	for _, stage := range []string{"begin", "query", "commit", "success"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			var buffer bytes.Buffer
			log, err := logger.NewLogger(&buffer, logger.LogLevelInfo, "json")
			require.NoError(t, err)
			repo, err := files.NewPostgresPosterRepo(db, log)
			require.NoError(t, err)
			failure := errors.New("database unavailable")
			begin := mock.ExpectBegin()
			if stage == "begin" {
				begin.WillReturnError(failure)
			} else {
				lock := mock.ExpectQuery("SELECT id").WithArgs(domain.FilmID(5))
				if stage == "query" {
					lock.WillReturnError(failure)
					mock.ExpectRollback()
				} else {
					lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
					mock.ExpectQuery("INSERT INTO file").WithArgs("poster.png", "image/png", int64(100), domain.UserID(7)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
					mock.ExpectExec("INSERT INTO film_file").WithArgs(domain.FilmID(5), int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
					commit := mock.ExpectCommit()
					if stage == "commit" {
						commit.WillReturnError(failure)
					}
				}
			}
			err = repo.SavePoster(context.Background(), 5, 7, domain.PosterFile{Filename: "poster.png", ContentType: "image/png", Size: 100})
			if stage == "success" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, failure)
			}
			require.NoError(t, mock.ExpectationsWereMet())
			var record map[string]any
			require.NoError(t, json.Unmarshal(buffer.Bytes(), &record))
			if stage == "success" {
				require.Equal(t, "ok", record["outcome"])
				require.Equal(t, "commit", record["stage"])
			} else {
				require.Equal(t, "error", record["outcome"])
				require.Equal(t, stage, record["stage"])
				require.Equal(t, "ERROR", record["level"])
			}
		})
	}
}
