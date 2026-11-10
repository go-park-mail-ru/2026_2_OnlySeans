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

func TestPostgresPosterRepoSave(t *testing.T) {
	dbErr := errors.New("database failure")
	tests := []struct {
		name        string
		failureAt   string
		wantErr     error
		wantMessage string
	}{
		{
			name: "metadata and main poster committed together",
		},
		{
			name:      "begin failure",
			failureAt: "begin",
			wantErr:   dbErr,
		},
		{
			name:      "missing film",
			failureAt: "missing film",
			wantErr:   domain.ErrFilmNotFound,
		},
		{
			name:      "film lock failure",
			failureAt: "lock",
			wantErr:   dbErr,
		},
		{
			name:        "invalid film ID from database",
			failureAt:   "lock scan",
			wantMessage: "lock poster film",
		},
		{
			name:      "metadata failure rolls back",
			failureAt: "insert",
			wantErr:   dbErr,
		},
		{
			name:        "invalid file ID rolls back",
			failureAt:   "insert scan",
			wantMessage: "insert poster metadata",
		},
		{
			name:      "poster binding failure rolls back metadata",
			failureAt: "binding",
			wantErr:   dbErr,
		},
		{
			name:      "commit outcome unknown",
			failureAt: "commit",
			wantErr:   domain.ErrCommitUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo, err := filesservice.NewPostgresPosterRepo(db, testFilesLogger(t))
			require.NoError(t, err)
			poster := domain.PosterFile{Filename: "poster.png", ContentType: "image/png", Size: 123}

			begin := mock.ExpectBegin()
			if tt.failureAt == "begin" {
				begin.WillReturnError(dbErr)
			} else {
				lock := mock.ExpectQuery(`SELECT id FROM film WHERE id = \$1 FOR UPDATE`).WithArgs(domain.FilmID(9))
				switch tt.failureAt {
				case "missing film":
					lock.WillReturnError(sql.ErrNoRows)
				case "lock":
					lock.WillReturnError(dbErr)
				case "lock scan":
					lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("invalid"))
				default:
					lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
					insert := mock.ExpectQuery("INSERT INTO file").WithArgs(poster.Filename, poster.ContentType, poster.Size, domain.UserID(7))
					switch tt.failureAt {
					case "insert":
						insert.WillReturnError(dbErr)
					case "insert scan":
						insert.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("invalid"))
					default:
						insert.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
						binding := mock.ExpectExec(`INSERT INTO film_file .* VALUES \(\$1, \$2, 'poster', 1\) ON CONFLICT \(film_id, file_role, sort_order\) DO UPDATE SET file_id = EXCLUDED.file_id, created_at = now\(\)`).WithArgs(domain.FilmID(9), int64(42))
						if tt.failureAt == "binding" {
							binding.WillReturnError(dbErr)
						} else {
							binding.WillReturnResult(sqlmock.NewResult(0, 1))
						}
					}
				}
				switch tt.failureAt {
				case "":
					mock.ExpectCommit()
				case "commit":
					mock.ExpectCommit().WillReturnError(dbErr)
				default:
					mock.ExpectRollback()
				}
			}

			err = repo.SavePoster(t.Context(), 9, 7, poster)
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantMessage != "":
				require.ErrorContains(t, err, tt.wantMessage)
			default:
				require.NoError(t, err)
			}
			if tt.failureAt == "commit" {
				require.ErrorIs(t, err, dbErr)
			} else {
				require.NotErrorIs(t, err, domain.ErrCommitUnknown)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
