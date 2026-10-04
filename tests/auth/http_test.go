package auth_test

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

const validRegisterBody = `{"email":"anna@example.com","username":"anna","password":"Password1"}`

func rawRequest(handlerFunc http.HandlerFunc, method, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handlerFunc(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, auth.SessionCookieName, cookies[0].Name)
	return cookies[0]
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	handlers := map[string]http.HandlerFunc{
		"register": h.Register,
		"login":    h.Login,
		"logout":   h.Logout,
	}

	for name, handlerFunc := range handlers {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
			t.Run(name+" "+method, func(t *testing.T) {
				rec := rawRequest(handlerFunc, method, validRegisterBody)
				assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
				assert.Empty(t, rec.Result().Cookies())
			})
		}
	}
}

func TestHandler_InvalidBody(t *testing.T) {
	h := newTestHandler(t)
	bodies := map[string]string{
		"пустое тело":           "",
		"оборванный json":       `{"email":`,
		"массив вместо объекта": `[]`,
		"число вместо строки":   `{"email":1,"password":2}`,
		"не json":               "email=anna@example.com",
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			for _, handlerFunc := range []http.HandlerFunc{h.Register, h.Login} {
				rec := rawRequest(handlerFunc, http.MethodPost, body)
				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Empty(t, rec.Result().Cookies())
			}
		})
	}
}

func TestHandler_ResponseHidesPassword(t *testing.T) {
	h := newTestHandler(t)

	for _, step := range []struct {
		name        string
		handlerFunc http.HandlerFunc
		wantStatus  int
	}{
		{"register", h.Register, http.StatusCreated},
		{"login", h.Login, http.StatusOK},
	} {
		t.Run(step.name, func(t *testing.T) {
			rec := rawRequest(step.handlerFunc, http.MethodPost, validRegisterBody)
			require.Equal(t, step.wantStatus, rec.Code)

			body := rec.Body.String()
			assert.NotContains(t, strings.ToLower(body), "password")
			assert.NotContains(t, body, "$2a$")
			assert.JSONEq(t, `{"user":{"id":1,"email":"anna@example.com","username":"anna"}}`, body)
		})
	}

	user, err := h.UseCase.Repo.GetByEmail(context.Background(), "anna@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, "Password1", user.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("Password1")))
}

func TestHandler_SessionCookie(t *testing.T) {
	for _, secure := range []bool{false, true} {
		h := newTestHandler(t)
		h.CookieSecure = secure

		rec := rawRequest(h.Register, http.MethodPost, validRegisterBody)
		require.Equal(t, http.StatusCreated, rec.Code)

		cookie := sessionCookie(t, rec)
		raw, err := hex.DecodeString(cookie.Value)
		require.NoError(t, err)
		assert.Len(t, raw, 32)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, secure, cookie.Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, "/", cookie.Path)
		assert.True(t, cookie.Expires.After(time.Now()))
	}
}

func TestHandler_Logout(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

	cookie := sessionCookie(t, rawRequest(h.Register, http.MethodPost, validRegisterBody))
	_, err := h.UseCase.Authenticate(ctx, cookie.Value)
	require.NoError(t, err)

	rec := rawRequest(h.Logout, http.MethodPost, "", cookie)
	require.Equal(t, http.StatusNoContent, rec.Code)

	cleared := sessionCookie(t, rec)
	assert.Empty(t, cleared.Value)
	assert.Negative(t, cleared.MaxAge)

	_, err = h.UseCase.Authenticate(ctx, cookie.Value)
	assert.ErrorIs(t, err, auth.ErrUnauthorized)

	assert.Equal(t, http.StatusNoContent, rawRequest(h.Logout, http.MethodPost, "", cookie).Code)
	assert.Equal(t, http.StatusNoContent, rawRequest(h.Logout, http.MethodPost, "").Code)
}

func TestUseCase_SessionsAreIndependent(t *testing.T) {
	uc := newTestUseCase(t)
	ctx := context.Background()

	registered, err := uc.Register(ctx, "anna@example.com", "anna", "Password1")
	require.NoError(t, err)
	phone, err := uc.Login(ctx, "anna@example.com", "Password1")
	require.NoError(t, err)
	laptop, err := uc.Login(ctx, "anna@example.com", "Password1")
	require.NoError(t, err)

	assert.NotEqual(t, registered.Session.ID, phone.Session.ID)
	assert.NotEqual(t, phone.Session.ID, laptop.Session.ID)
	assert.NotEqual(t, registered.Session.ID, laptop.Session.ID)

	require.NoError(t, uc.Logout(phone.Session.ID))

	_, err = uc.Authenticate(ctx, phone.Session.ID)
	assert.ErrorIs(t, err, auth.ErrUnauthorized)

	for _, id := range []string{registered.Session.ID, laptop.Session.ID} {
		userID, err := uc.Authenticate(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, registered.User.ID, userID)
	}
}
