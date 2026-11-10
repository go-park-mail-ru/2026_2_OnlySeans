package files_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	repo "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	uc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestFilmPosterURL(t *testing.T) {
	external := "https://images.example.org/poster.png"
	failure := errors.New("object unavailable")
	for _, tc := range []struct {
		name                         string
		id                           domain.FilmID
		poster                       *domain.PosterFile
		repoErr, storageErr, wantErr error
		wantURL                      string
	}{
		{name: "private poster", id: 9, poster: &domain.PosterFile{Filename: "poster.png"}, wantURL: "signed-url"},
		{name: "external poster", id: 9, poster: &domain.PosterFile{ExternalURL: &external}, wantURL: external},
		{name: "invalid id", id: 0, wantErr: uc.ErrInvalidPosterFilmID},
		{name: "missing film", id: 9, repoErr: domain.ErrFilmNotFound, wantErr: domain.ErrFilmNotFound},
		{name: "missing poster", id: 9, repoErr: domain.ErrFileNotFound, wantErr: domain.ErrFileNotFound},
		{name: "empty repository result", id: 9, wantErr: domain.ErrFileNotFound},
		{name: "storage failure", id: 9, poster: &domain.PosterFile{Filename: "poster.png"}, storageErr: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			posters := mocks.NewMockPosterRepository(ctrl)
			objects := mocks.NewMockAvatarURLStorage(ctrl)
			if tc.id > 0 {
				read := posters.EXPECT().GetPoster(t.Context(), tc.id).Return(tc.poster, tc.repoErr)
				if tc.repoErr == nil && tc.poster != nil && tc.poster.ExternalURL == nil {
					url := tc.wantURL
					objects.EXPECT().GetURL(t.Context(), tc.poster.Filename).Return(url, tc.storageErr).After(read)
				}
			}
			usecase, err := uc.NewFilmPosterUseCase(posters, objects, testFilesLogger(t))
			require.NoError(t, err)
			url, err := usecase.GetURL(t.Context(), tc.id)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.wantURL, url)
		})
	}
}

func TestFilmPosterDelete(t *testing.T) {
	for _, wantErr := range []error{nil, domain.ErrFilmNotFound, errors.New("database failure")} {
		ctrl := gomock.NewController(t)
		posters := mocks.NewMockPosterRepository(ctrl)
		objects := mocks.NewMockAvatarURLStorage(ctrl)
		posters.EXPECT().DeletePoster(t.Context(), domain.FilmID(9)).Return(wantErr)
		usecase, err := uc.NewFilmPosterUseCase(posters, objects, testFilesLogger(t))
		require.NoError(t, err)
		require.ErrorIs(t, usecase.Delete(t.Context(), 9), wantErr)
		require.ErrorIs(t, usecase.Delete(t.Context(), 0), uc.ErrInvalidPosterFilmID)
	}
}

func TestPostgresGetFilmPoster(t *testing.T) {
	for _, tc := range []struct {
		name             string
		mime             any
		repoErr, wantErr error
	}{
		{name: "poster", mime: "image/png"},
		{name: "no poster", wantErr: domain.ErrFileNotFound},
		{name: "no film", repoErr: sql.ErrNoRows, wantErr: domain.ErrFilmNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			posters, err := repo.NewPostgresPosterRepo(db, nil)
			require.NoError(t, err)
			q := mock.ExpectQuery(`SELECT COALESCE\(fl.storage_key`).WithArgs(domain.FilmID(9))
			if tc.repoErr != nil {
				q.WillReturnError(tc.repoErr)
			} else {
				q.WillReturnRows(sqlmock.NewRows([]string{"storage_key", "mime_type", "size_bytes", "external_url"}).AddRow("poster.png", tc.mime, 123, nil))
			}
			result, err := posters.GetPoster(t.Context(), 9)
			require.ErrorIs(t, err, tc.wantErr)
			if err == nil {
				require.Equal(t, "poster.png", result.Filename)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresDeleteFilmPoster(t *testing.T) {
	dbErr := errors.New("database failure")
	for _, stage := range []string{"ok", "missing film", "delete", "commit"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			posters, err := repo.NewPostgresPosterRepo(db, nil)
			require.NoError(t, err)
			mock.ExpectBegin()
			lock := mock.ExpectQuery(`SELECT id FROM film WHERE id = \$1 FOR UPDATE`).WithArgs(domain.FilmID(9))
			var wantErr error
			if stage == "missing film" {
				lock.WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
				wantErr = domain.ErrFilmNotFound
			} else {
				lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
				del := mock.ExpectExec(`DELETE FROM film_file WHERE film_id = \$1 AND file_role = 'poster'`).WithArgs(domain.FilmID(9))
				if stage == "delete" {
					del.WillReturnError(dbErr)
					mock.ExpectRollback()
					wantErr = dbErr
				} else {
					del.WillReturnResult(sqlmock.NewResult(0, 1))
					commit := mock.ExpectCommit()
					if stage == "commit" {
						commit.WillReturnError(dbErr)
						wantErr = domain.ErrCommitUnknown
					}
				}
			}
			err = posters.DeletePoster(t.Context(), 9)
			require.ErrorIs(t, err, wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
