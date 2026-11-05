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
