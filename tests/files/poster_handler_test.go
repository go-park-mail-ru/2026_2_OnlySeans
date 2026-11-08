package files_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestPosterUploadHTTP(t *testing.T) {
	var picture bytes.Buffer
	require.NoError(t, png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	tests := []struct {
		name                        string
		id                          string
		anonymous, raw              bool
		copies                      int
		field                       string
		data                        []byte
		filmErr, uploadErr, saveErr error
		wantStatus                  int
	}{
		{
			name: "PNG saved", id: "9", copies: 1, field: "poster", data: picture.Bytes(), wantStatus: http.StatusCreated,
		},
		{
			name: "anonymous", id: "9", anonymous: true, wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid film ID", id: "abc", wantStatus: http.StatusBadRequest,
		},
		{
			name: "negative film ID", id: "-9", wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing file", id: "9", wantStatus: http.StatusBadRequest,
		},
		{
			name: "wrong field", id: "9", copies: 1, field: "avatar", data: picture.Bytes(), wantStatus: http.StatusBadRequest,
		},
		{
			name: "multiple posters", id: "9", copies: 2, field: "poster", data: picture.Bytes(), wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid multipart", id: "9", raw: true, wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid image", id: "9", copies: 1, field: "poster", data: []byte("not a PNG"), wantStatus: http.StatusBadRequest,
		},
		{
			name: "file too large", id: "9", copies: 1, field: "poster", data: make([]byte, 1025), wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "whole request too large", id: "9", copies: 1, field: "poster", data: make([]byte, (1<<20)+1025), wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "film not found", id: "9", copies: 1, field: "poster", data: picture.Bytes(), filmErr: domain.ErrFilmNotFound, wantStatus: http.StatusNotFound,
		},
		{
			name: "storage error hidden", id: "9", copies: 1, field: "poster", data: picture.Bytes(), uploadErr: errors.New("private storage failure"), wantStatus: http.StatusInternalServerError,
		},
		{
			name: "database error hidden", id: "9", copies: 1, field: "poster", data: picture.Bytes(), saveErr: errors.New("private database failure"), wantStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &posterUploadDependencies{filmErr: tt.filmErr, uploadErr: tt.uploadErr, saveErr: tt.saveErr}
			uc, err := filesusecase.NewUploadPosterUseCase(deps, deps, deps, filesusecase.UploadPosterOptions{
				MaxSize: 1024, MaxDimension: 64, CleanupTimeout: time.Second,
			}, testFilesLogger(t))
			require.NoError(t, err)
			handler, err := filesdelivery.NewPosterHandler(uc, testFilesLogger(t))
			require.NoError(t, err)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for i := 0; i < tt.copies; i++ {
				part, err := writer.CreateFormFile(tt.field, "original.png")
				require.NoError(t, err)
				_, err = part.Write(tt.data)
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			contentType := writer.FormDataContentType()
			if tt.raw {
				body.Reset()
				body.WriteString(`{}`)
				contentType = "application/json"
			}
			req := httptest.NewRequest(http.MethodPost, "/api/films/"+tt.id+"/poster", &body)
			req.Header.Set("Content-Type", contentType)
			req = mux.SetURLVars(req, map[string]string{"id": tt.id})
			if !tt.anonymous {
				req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), 7))
			}
			rec := httptest.NewRecorder()
			handler.UploadPoster(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			require.NotContains(t, rec.Body.String(), "private")
			if tt.wantStatus == http.StatusCreated {
				var result map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
				require.Equal(t, map[string]string{"filename": deps.file.Filename}, result)
			} else if tt.saveErr == nil {
				for _, event := range deps.events {
					require.NotEqual(t, "save", event)
				}
			}
		})
	}
}
