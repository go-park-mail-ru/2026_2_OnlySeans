package files_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type posterCapture struct {
	events             []string
	filmID             domain.FilmID
	userID             domain.UserID
	file               domain.PosterFile
	key, contentType   string
	data               []byte
	cleanupContextErr  error
	cleanupHasDeadline bool
}

func expectPosterUpload(t *testing.T, events []string, filmErr, uploadErr, saveErr, deleteErr error, cancel context.CancelFunc) (*mocks.MockPosterStorage, *mocks.MockPosterRepository, *mocks.MockPosterFilmReader, *posterCapture) {
	t.Helper()
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockPosterStorage(ctrl)
	repo := mocks.NewMockPosterRepository(ctrl)
	films := mocks.NewMockPosterFilmReader(ctrl)
	capture := &posterCapture{}
	var calls []any
	for _, event := range events {
		switch event {
		case "film":
			calls = append(calls, films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(9)).DoAndReturn(func(_ context.Context, id domain.FilmID) (domain.Film, error) {
				capture.events = append(capture.events, "film")
				capture.filmID = id
				return domain.Film{}, filmErr
			}))
		case "upload":
			calls = append(calls, storage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, key, mime string, body io.ReadSeeker) error {
				capture.events = append(capture.events, "upload")
				capture.key, capture.contentType = key, mime
				data, err := io.ReadAll(body)
				capture.data = data
				return errors.Join(err, uploadErr)
			}))
		case "save":
			calls = append(calls, repo.EXPECT().SavePoster(gomock.Any(), domain.FilmID(9), domain.UserID(7), gomock.Any()).DoAndReturn(func(_ context.Context, film domain.FilmID, user domain.UserID, file domain.PosterFile) error {
				capture.events = append(capture.events, "save")
				capture.filmID, capture.userID, capture.file = film, user, file
				require.Equal(t, capture.key, file.Filename)
				if cancel != nil {
					cancel()
				}
				return saveErr
			}))
		case "delete":
			calls = append(calls, storage.EXPECT().Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, key string) error {
				capture.events = append(capture.events, "delete")
				require.Equal(t, capture.key, key)
				capture.cleanupContextErr = ctx.Err()
				_, capture.cleanupHasDeadline = ctx.Deadline()
				return deleteErr
			}))
		}
	}
	gomock.InOrder(calls...)
	return storage, repo, films, capture
}

func TestUploadPoster(t *testing.T) {
	var pngData, jpegData bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	require.NoError(t, png.Encode(&pngData, picture))
	require.NoError(t, jpeg.Encode(&jpegData, picture, nil))
	databaseErr := errors.New("database unavailable")
	storageErr := errors.New("storage unavailable")
	cleanupErr := errors.New("cleanup unavailable")
	tests := []struct {
		name                                            string
		data                                            []byte
		mime, extension                                 string
		userID                                          domain.UserID
		filmID                                          domain.FilmID
		declaredSize                                    int64
		maxSize                                         int64
		maxDimension                                    int
		missing, cancelBefore, cancelOnSave             bool
		filmErr, uploadErr, saveErr, deleteErr, wantErr error
		events                                          []string
	}{
		{
			name: "PNG saved and attached", data: pngData.Bytes(), mime: "image/png", extension: ".png",
			events: []string{"film", "upload", "save"},
		},
		{
			name: "JPEG saved and attached", data: jpegData.Bytes(), mime: "image/jpeg", extension: ".jpg",
			events: []string{"film", "upload", "save"},
		},
		{
			name: "exact size and dimension limits", data: pngData.Bytes(), mime: "image/png", extension: ".png",
			maxSize: int64(pngData.Len()), maxDimension: 2, events: []string{"film", "upload", "save"},
		},
		{
			name: "invalid user", userID: -1, wantErr: domain.ErrUnauthorized,
		},
		{
			name: "invalid film", filmID: -1, wantErr: filesusecase.ErrInvalidPosterFilmID,
		},
		{
			name: "canceled before lookup", cancelBefore: true, wantErr: context.Canceled,
		},
		{
			name: "missing film", filmErr: domain.ErrFilmNotFound, wantErr: domain.ErrFilmNotFound, events: []string{"film"},
		},
		{
			name: "film lookup failed", filmErr: databaseErr, wantErr: databaseErr, events: []string{"film"},
		},
		{
			name: "missing body", missing: true, wantErr: filesusecase.ErrPosterMissing, events: []string{"film"},
		},
		{
			name: "negative declared size", declaredSize: -1, wantErr: filesusecase.ErrInvalidPosterSize, events: []string{"film"},
		},
		{
			name: "declared size exceeds limit", declaredSize: 2048, maxSize: 1024, wantErr: filesusecase.ErrPosterTooLarge, events: []string{"film"},
		},
		{
			name: "actual size exceeds limit", data: pngData.Bytes(), declaredSize: 1, maxSize: 10,
			wantErr: filesusecase.ErrPosterTooLarge, events: []string{"film"},
		},
		{
			name: "empty actual body", data: []byte{}, declaredSize: 1, wantErr: filesusecase.ErrInvalidPosterSize, events: []string{"film"},
		},
		{
			name: "text disguised as image", data: []byte("not an image"), wantErr: filesusecase.ErrInvalidPosterImage, events: []string{"film"},
		},
		{
			name: "truncated PNG", data: pngData.Bytes()[:pngData.Len()-15], wantErr: filesusecase.ErrInvalidPosterImage, events: []string{"film"},
		},
		{
			name: "dimensions exceed limit", maxDimension: 1, wantErr: filesusecase.ErrPosterDimensionsTooLarge, events: []string{"film"},
		},
		{
			name: "upload failed", uploadErr: storageErr, wantErr: storageErr, events: []string{"film", "upload"},
		},
		{
			name: "metadata failure removes object", saveErr: databaseErr, wantErr: databaseErr,
			events: []string{"film", "upload", "save", "delete"},
		},
		{
			name: "film deleted during upload", saveErr: domain.ErrFilmNotFound, wantErr: domain.ErrFilmNotFound,
			events: []string{"film", "upload", "save", "delete"},
		},
		{
			name: "ambiguous commit retains object", saveErr: errors.Join(domain.ErrCommitUnknown, databaseErr),
			wantErr: domain.ErrCommitUnknown, events: []string{"film", "upload", "save"},
		},
		{
			name: "cleanup preserves both failures", saveErr: databaseErr, deleteErr: cleanupErr, wantErr: databaseErr,
			events: []string{"film", "upload", "save", "delete"},
		},
		{
			name: "cleanup survives request cancellation", saveErr: context.Canceled, cancelOnSave: true, wantErr: context.Canceled,
			events: []string{"film", "upload", "save", "delete"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var cancelOnSave context.CancelFunc
			if tt.cancelBefore {
				cancel()
			}
			if tt.cancelOnSave {
				cancelOnSave = cancel
			}
			options := filesusecase.UploadPosterOptions{MaxSize: 1 << 20, MaxDimension: 4096, CleanupTimeout: time.Second}
			if tt.maxSize != 0 {
				options.MaxSize = tt.maxSize
			}
			if tt.maxDimension != 0 {
				options.MaxDimension = tt.maxDimension
			}
			storage, repo, films, deps := expectPosterUpload(t, tt.events, tt.filmErr, tt.uploadErr, tt.saveErr, tt.deleteErr, cancelOnSave)
			uc, err := filesusecase.NewUploadPosterUseCase(storage, repo, films, options, testFilesLogger(t))
			require.NoError(t, err)
			data := tt.data
			if data == nil {
				data = pngData.Bytes()
			}
			input := filesusecase.UploadPosterInput{UserID: 7, FilmID: 9, Body: bytes.NewReader(data), Size: int64(len(data))}
			if tt.userID != 0 {
				input.UserID = tt.userID
			}
			if tt.filmID != 0 {
				input.FilmID = tt.filmID
			}
			if tt.declaredSize != 0 {
				input.Size = tt.declaredSize
			}
			if tt.missing {
				input.Body = nil
			}
			result, err := uc.Run(ctx, input)
			require.Equal(t, tt.events, deps.events)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, result)
				if tt.deleteErr != nil {
					require.ErrorIs(t, err, tt.deleteErr)
				}
			} else {
				require.NoError(t, err)
				require.Regexp(t, `^[a-f0-9]{32}\`+tt.extension+`$`, result.Filename)
				require.Equal(t, result.Filename, deps.file.Filename)
				require.Equal(t, tt.mime, deps.file.ContentType)
				require.Equal(t, tt.mime, deps.contentType)
				require.Equal(t, int64(len(data)), deps.file.Size)
				require.Equal(t, data, deps.data)
				require.Equal(t, domain.UserID(7), deps.userID)
				require.Equal(t, domain.FilmID(9), deps.filmID)
			}
			if len(deps.events) > 0 && deps.events[len(deps.events)-1] == "delete" {
				require.NoError(t, deps.cleanupContextErr)
				require.True(t, deps.cleanupHasDeadline)
			}
		})
	}
}
