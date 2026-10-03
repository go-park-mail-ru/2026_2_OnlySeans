package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/router"
)

func TestRequireAuth_SessionFlow(t *testing.T) {
	uc, err := auth.NewUseCase(auth.NewInMemoryUserRepo(), auth.NewInMemorySessionStore(time.Hour))
	require.NoError(t, err)

	h, err := auth.NewHandler(uc)
	require.NoError(t, err)

	body := `{"email":"flow@example.com","username":"flowuser","password":"Password1"}`
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	require.Equal(t, http.StatusCreated, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.Equal(t, auth.SessionCookieName, cookie.Name)
	require.NotEmpty(t, cookie.Value)
	require.True(t, cookie.HttpOnly, "кука должна быть HttpOnly")

	protected := router.RequireAuth(uc, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFromContext(r.Context()); !ok {
			w.WriteHeader(http.StatusInternalServerError) // мидлварь не положила ID
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	callMe := func(c *http.Cookie) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c != nil {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		protected(rec, req)
		return rec.Code
	}

	require.Equal(t, http.StatusUnauthorized, callMe(nil), "без куки")
	require.Equal(t, http.StatusOK, callMe(cookie), "с кукой")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Logout(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)

	require.Equal(t, http.StatusUnauthorized, callMe(cookie), "после logout")
}
