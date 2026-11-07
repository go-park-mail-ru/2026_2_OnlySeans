package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/stretchr/testify/require"
)

type routerAvatarStorage struct {
	calls int
	key   string
	data  []byte
}

func (s *routerAvatarStorage) Upload(_ context.Context, key, _ string, body io.ReadSeeker) error {
	s.calls++
	s.key = key
	var err error
	s.data, err = io.ReadAll(body)
	return err
}

func TestAvatarUploadRoute(t *testing.T) {
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	tests := []struct {
		name        string
		method      string
		session     string
		data        []byte
		wantStatus  int
		wantUploads int
	}{
		{
			name:       "no session",
			method:     http.MethodPost,
			data:       imageData.Bytes(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid session",
			method:     http.MethodPost,
			session:    "invalid",
			data:       imageData.Bytes(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "registered reader can upload",
			method:      http.MethodPost,
			session:     "valid",
			data:        imageData.Bytes(),
			wantStatus:  http.StatusCreated,
			wantUploads: 1,
		},
		{
			name:       "invalid image rejected",
			method:     http.MethodPost,
			session:    "valid",
			data:       []byte("not an image"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "GET is not upload",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &routerAvatarStorage{}
			handler := newTestRouterWithAvatarStorage(t, &spyLogger{}, storage)
			var cookie *http.Cookie
			switch tt.session {
			case "valid":
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"email":"avatar@example.com","username":"avataruser","password":"Password1"}`)))
				require.Equal(t, http.StatusCreated, rec.Code)
				cookies := rec.Result().Cookies()
				require.NotEmpty(t, cookies)
				cookie = cookies[0]
			case "invalid":
				cookie = &http.Cookie{Name: auth.SessionCookieName, Value: "invalid-session"}
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("avatar", "photo.png")
			require.NoError(t, err)
			_, err = part.Write(tt.data)
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			req := httptest.NewRequest(tt.method, "/api/avatars", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if cookie != nil {
				req.AddCookie(cookie)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			require.Equal(t, tt.wantUploads, storage.calls)
			if tt.wantStatus == http.StatusCreated {
				var response struct {
					Filename string `json:"filename"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
				require.Regexp(t, `^[0-9a-f]{32}\.png$`, response.Filename)
				require.Equal(t, response.Filename, storage.key)
				require.Equal(t, tt.data, storage.data)
				profileReq := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
				profileReq.AddCookie(cookie)
				profileRec := httptest.NewRecorder()
				handler.ServeHTTP(profileRec, profileReq)
				require.Equal(t, http.StatusOK, profileRec.Code)
				var profile struct {
					User struct {
						AvatarFilename string `json:"avatar_filename"`
					} `json:"user"`
				}
				require.NoError(t, json.Unmarshal(profileRec.Body.Bytes(), &profile))
				require.Equal(t, response.Filename, profile.User.AvatarFilename)
				getOwn := httptest.NewRequest(http.MethodGet, "/api/me/avatar", nil)
				getOwn.AddCookie(cookie)
				ownResponse := httptest.NewRecorder()
				handler.ServeHTTP(ownResponse, getOwn)
				require.Equal(t, http.StatusOK, ownResponse.Code)
				require.Contains(t, ownResponse.Body.String(), response.Filename)
				for range 2 {
					deleteReq := httptest.NewRequest(http.MethodDelete, "/api/me/avatar", nil)
					deleteReq.AddCookie(cookie)
					deleteRec := httptest.NewRecorder()
					handler.ServeHTTP(deleteRec, deleteReq)
					require.Equal(t, http.StatusNoContent, deleteRec.Code)
					require.Empty(t, deleteRec.Body.String())
				}
				missingResponse := httptest.NewRecorder()
				handler.ServeHTTP(missingResponse, getOwn)
				require.Equal(t, http.StatusNotFound, missingResponse.Code)
				profileAfter := httptest.NewRecorder()
				handler.ServeHTTP(profileAfter, profileReq)
				require.Equal(t, http.StatusOK, profileAfter.Code)
				require.NotContains(t, profileAfter.Body.String(), "avatar_filename")

			}
		})
	}
}

type routerAvatarURLStorage struct{}

func (*routerAvatarURLStorage) GetURL(_ context.Context, key string) (string, error) {
	return "https://storage.example/avatars/" + key, nil
}

func TestAvatarURLRoute(t *testing.T) {
	handler := newTestRouter(t)
	filename := strings.Repeat("a", 32) + ".png"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/avatars/"+filename, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"url":"https://storage.example/avatars/`+filename+`"}`, rec.Body.String())
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

func (s *routerAvatarStorage) Delete(context.Context, string) error { return nil }

func TestOwnAvatarRequiresSession(t *testing.T) {
	tests := []struct{ name, method, session string }{
		{
			name:   "GET without session",
			method: http.MethodGet,
		},
		{
			name:   "DELETE without session",
			method: http.MethodDelete,
		},
		{
			name:    "GET invalid session",
			method:  http.MethodGet,
			session: "invalid",
		},
		{
			name:    "DELETE invalid session",
			method:  http.MethodDelete,
			session: "invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestRouter(t)
			req := httptest.NewRequest(tt.method, "/api/me/avatar", nil)
			if tt.session != "" {
				req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: tt.session})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAvatarUpdateRoute(t *testing.T) {
	var validImage bytes.Buffer
	require.NoError(t, png.Encode(&validImage, image.NewRGBA(image.Rect(0, 0, 3, 3))))
	tests := []struct {
		name         string
		session      string
		existing     bool
		invalidImage bool
		wantStatus   int
	}{
		{
			name:       "replace existing avatar",
			session:    "valid",
			existing:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "set previously absent avatar",
			session:    "valid",
			wantStatus: http.StatusOK,
		},
		{
			name:       "no session",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid session",
			session:    "invalid",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:         "invalid replacement preserves existing avatar",
			session:      "valid",
			existing:     true,
			invalidImage: true,
			wantStatus:   http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &routerAvatarStorage{}
			handler := newTestRouterWithAvatarStorage(t, &spyLogger{}, storage)
			var cookie *http.Cookie
			if tt.session == "valid" {
				register := httptest.NewRecorder()
				handler.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"email":"update@example.com","username":"updateuser","password":"Password1"}`)))
				require.Equal(t, http.StatusCreated, register.Code)
				cookie = register.Result().Cookies()[0]
			} else if tt.session == "invalid" {
				cookie = &http.Cookie{Name: auth.SessionCookieName, Value: "invalid"}
			}
			sendImage := func(method, path string, data []byte) *httptest.ResponseRecorder {
				t.Helper()
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				part, err := writer.CreateFormFile("avatar", "photo.png")
				require.NoError(t, err)
				_, err = part.Write(data)
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				req := httptest.NewRequest(method, path, &body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				if cookie != nil {
					req.AddCookie(cookie)
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				return rec
			}
			var previousFilename string
			if tt.existing {
				initial := sendImage(http.MethodPost, "/api/avatars", validImage.Bytes())
				require.Equal(t, http.StatusCreated, initial.Code)
				var created struct {
					Filename string `json:"filename"`
				}
				require.NoError(t, json.Unmarshal(initial.Body.Bytes(), &created))
				previousFilename = created.Filename
			}
			imageData := validImage.Bytes()
			if tt.invalidImage {
				imageData = []byte("invalid image")
			}
			rec := sendImage(http.MethodPut, "/api/me/avatar", imageData)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			switch tt.wantStatus {
			case http.StatusOK:
				var updated struct {
					Filename string `json:"filename"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
				require.NotEmpty(t, updated.Filename)
				require.NotEqual(t, previousFilename, updated.Filename)
				require.Equal(t, updated.Filename, storage.key)
				profileReq := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
				profileReq.AddCookie(cookie)
				profile := httptest.NewRecorder()
				handler.ServeHTTP(profile, profileReq)
				require.Equal(t, http.StatusOK, profile.Code)
				require.Contains(t, profile.Body.String(), updated.Filename)
			case http.StatusBadRequest:
				require.Equal(t, 1, storage.calls)
				getReq := httptest.NewRequest(http.MethodGet, "/api/me/avatar", nil)
				getReq.AddCookie(cookie)
				current := httptest.NewRecorder()
				handler.ServeHTTP(current, getReq)
				require.Equal(t, http.StatusOK, current.Code)
				require.Contains(t, current.Body.String(), previousFilename)
			case http.StatusUnauthorized:
				require.Zero(t, storage.calls)
			}
		})
	}
}
