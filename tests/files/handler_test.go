package files_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	fileusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type avatarStorage struct {
	err         error
	calls       int
	key         string
	contentType string
	data        []byte
}

func (s *avatarStorage) Upload(_ context.Context, key, contentType string, body io.ReadSeeker) error {
	s.calls++
	s.key, s.contentType = key, contentType
	var err error
	s.data, err = io.ReadAll(body)
	if err != nil {
		return err
	}
	return s.err
}

func TestUploadAvatar(t *testing.T) {
	var pngData, jpegData, wideData bytes.Buffer
	require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	require.NoError(t, jpeg.Encode(&jpegData, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil))
	require.NoError(t, png.Encode(&wideData, image.NewRGBA(image.Rect(0, 0, testMaxAvatarDimension+1, 1))))
	storageErr := errors.New("private storage failure")
	tests := []struct {
		name          string
		data          []byte
		field         string
		copies        int
		rawBody       bool
		storageErr    error
		wantStatus    int
		wantCalls     int
		wantType      string
		wantExtension string
	}{
		{
			name: "PNG upload",
			data: pngData.Bytes(), field: "avatar", copies: 1,
			wantStatus: http.StatusCreated, wantCalls: 1,
			wantType: "image/png", wantExtension: "png",
		},
		{
			name: "JPEG upload",
			data: jpegData.Bytes(), field: "avatar", copies: 1,
			wantStatus: http.StatusCreated, wantCalls: 1,
			wantType: "image/jpeg", wantExtension: "jpg",
		},
		{
			name:       "missing file",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "wrong field name",
			data: pngData.Bytes(), field: "photo", copies: 1,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate avatar files",
			data: pngData.Bytes(), field: "avatar", copies: 2,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "empty file",
			field: "avatar", copies: 1,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid image despite PNG name",
			data: []byte("not an image"), field: "avatar", copies: 1,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "truncated PNG",
			data: pngData.Bytes()[:33], field: "avatar", copies: 1,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "excessive dimensions",
			data: wideData.Bytes(), field: "avatar", copies: 1,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "file exceeds limit",
			data: bytes.Repeat([]byte("a"), testMaxAvatarSize+1), field: "avatar", copies: 1,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "request exceeds limit",
			data: bytes.Repeat([]byte("a"), testMaxAvatarSize+(1<<20)), field: "avatar", copies: 1,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "JSON instead of multipart",
			rawBody:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "storage error is hidden",
			data: pngData.Bytes(), field: "avatar", copies: 1, storageErr: storageErr,
			wantStatus: http.StatusInternalServerError, wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &avatarStorage{err: tt.storageErr}
			uc, err := fileusecase.NewUploadAvatarUseCase(storage, &avatarMetadataRepo{}, testAvatarOptions(), testFilesLogger(t))
			require.NoError(t, err)
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
			require.NoError(t, err)
			urlUC, err := fileusecase.NewGetAvatarURLUseCase(&avatarURLStorage{}, testFilesLogger(t))
			require.NoError(t, err)
			handler, err := filesdelivery.NewHandler(uc, urlUC, testAccountAvatar(t, &avatarURLStorage{}), appLogger)
			require.NoError(t, err)

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for i := 0; i < tt.copies; i++ {
				part, err := writer.CreateFormFile(tt.field, "untrusted.png")
				require.NoError(t, err)
				_, err = part.Write(tt.data)
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			req := httptest.NewRequest(http.MethodPost, "/api/avatars", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if tt.rawBody {
				req = httptest.NewRequest(http.MethodPost, "/api/avatars", bytes.NewBufferString(`{}`))
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), 1))
			handler.UploadAvatar(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			require.Equal(t, tt.wantCalls, storage.calls)
			var response map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			if tt.wantStatus == http.StatusCreated {
				require.Len(t, response, 1)
				filename, ok := response["filename"].(string)
				require.True(t, ok)
				require.Regexp(t, `^[0-9a-f]{32}\.`+tt.wantExtension+`$`, filename)
				require.Equal(t, filename, storage.key)
				require.Equal(t, tt.wantType, storage.contentType)
				require.Equal(t, tt.data, storage.data)
			} else {
				require.NotContains(t, response, "filename")
			}
			if tt.wantStatus == http.StatusInternalServerError {
				require.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
			}
			require.NotContains(t, rec.Body.String(), "private storage failure")
		})
	}
}

type avatarURLStorage struct {
	url   string
	err   error
	calls int
}

func (s *avatarURLStorage) GetURL(context.Context, string) (string, error) {
	s.calls++
	return s.url, s.err
}

func TestGetAvatarURL(t *testing.T) {
	const validName = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"
	tests := []struct {
		name       string
		filename   string
		err        error
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "existing PNG",
			filename:   validName,
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "existing JPEG",
			filename:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg",
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "path traversal",
			filename:   "../private.png",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong extension",
			filename:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.svg",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing object",
			filename:   validName,
			err:        domain.ErrFileNotFound,
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
		},
		{
			name:       "storage error",
			filename:   validName,
			err:        errors.New("private storage failure"),
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upload, err := fileusecase.NewUploadAvatarUseCase(&avatarStorage{}, &avatarMetadataRepo{}, testAvatarOptions(), testFilesLogger(t))
			require.NoError(t, err)
			storage := &avatarURLStorage{url: "https://media.example/avatar.png?signature=test", err: tt.err}
			getURL, err := fileusecase.NewGetAvatarURLUseCase(storage, testFilesLogger(t))
			require.NoError(t, err)
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
			require.NoError(t, err)
			handler, err := filesdelivery.NewHandler(upload, getURL, testAccountAvatar(t, &avatarURLStorage{}), appLogger)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodGet, "/api/avatars/test", nil)
			req = mux.SetURLVars(req, map[string]string{"filename": tt.filename})
			rec := httptest.NewRecorder()
			handler.GetAvatarURL(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCalls, storage.calls)
			var response map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			if tt.wantStatus == http.StatusOK {
				require.Equal(t, storage.url, response["url"])
				require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			} else {
				require.NotContains(t, response, "url")
			}
			require.NotContains(t, rec.Body.String(), "private storage failure")
		})
	}
}

func (s *avatarStorage) Delete(context.Context, string) error { return nil }

type avatarMetadataRepo struct{}

func (*avatarMetadataRepo) SaveAvatar(context.Context, domain.UserID, domain.AvatarFile) error {
	return nil
}

func TestAccountAvatarHandlers(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		userID     domain.UserID
		repoErr    error
		wantStatus int
	}{
		{name: "get own avatar", method: http.MethodGet, userID: 7, wantStatus: http.StatusOK},
		{name: "get without session", method: http.MethodGet, wantStatus: http.StatusUnauthorized},
		{name: "get missing avatar", method: http.MethodGet, userID: 7, repoErr: domain.ErrFileNotFound, wantStatus: http.StatusNotFound},
		{name: "delete own avatar", method: http.MethodDelete, userID: 7, wantStatus: http.StatusNoContent},
		{name: "delete without session", method: http.MethodDelete, wantStatus: http.StatusUnauthorized},
		{name: "delete unknown user", method: http.MethodDelete, userID: 7, repoErr: domain.ErrUserNotFound, wantStatus: http.StatusUnauthorized},
		{name: "private error hidden", method: http.MethodDelete, userID: 7, repoErr: errors.New("private database error"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &accountAvatarStub{file: &domain.AvatarFile{Filename: "avatar.png"}, err: tt.repoErr}
			storage := &avatarURLStorage{url: "https://example.com/avatar"}
			account, err := fileusecase.NewAccountAvatarUseCase(repo, storage, testFilesLogger(t))
			require.NoError(t, err)
			upload, err := fileusecase.NewUploadAvatarUseCase(&avatarStorage{}, &avatarMetadataRepo{}, testAvatarOptions(), testFilesLogger(t))
			require.NoError(t, err)
			getURL, err := fileusecase.NewGetAvatarURLUseCase(storage, testFilesLogger(t))
			require.NoError(t, err)
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
			require.NoError(t, err)
			handler, err := filesdelivery.NewHandler(upload, getURL, account, appLogger)
			require.NoError(t, err)
			req := httptest.NewRequest(tt.method, "/api/me/avatar", nil)
			if tt.userID > 0 {
				req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), tt.userID))
			}
			rec := httptest.NewRecorder()
			if tt.method == http.MethodGet {
				handler.GetMyAvatarURL(rec, req)
			} else {
				handler.DeleteMyAvatar(rec, req)
			}
			require.Equal(t, tt.wantStatus, rec.Code)
			require.NotContains(t, rec.Body.String(), "private database error")
			if tt.wantStatus == http.StatusNoContent {
				require.Empty(t, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			}
			if tt.userID > 0 {
				require.Equal(t, tt.userID, repo.userID)
			}
		})
	}
}
