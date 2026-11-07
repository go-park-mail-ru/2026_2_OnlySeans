package router_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/require"
)

type catalogPermissionStub struct {
	role domain.Role
	err  error
}

func (s catalogPermissionStub) CanEditPersons(context.Context, domain.UserID) (bool, error) {
	return s.role.CanEditPersons(), s.err
}

func TestRequireCatalogEditor(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		role          domain.Role
		err           error
		wantStatus    int
	}{
		{name: "anonymous", wantStatus: http.StatusUnauthorized},
		{name: "reader", authenticated: true, role: domain.RoleReader, wantStatus: http.StatusForbidden},
		{name: "editor", authenticated: true, role: domain.RoleEditor, wantStatus: http.StatusNoContent},
		{name: "admin", authenticated: true, role: domain.RoleAdmin, wantStatus: http.StatusNoContent},
		{name: "unknown role", authenticated: true, role: "other", wantStatus: http.StatusForbidden},
		{name: "deleted account", authenticated: true, err: domain.ErrUserNotFound, wantStatus: http.StatusUnauthorized},
		{name: "database failure", authenticated: true, err: errors.New("database down"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			appLogger := &spyLogger{}
      handler := router.RequireCatalogEditor(
        catalogPermissionStub{role: tt.role, err: tt.err},
        appLogger,
      )(func(w http.ResponseWriter, _ *http.Request) {
        called = true
        w.WriteHeader(http.StatusNoContent)
      })
			req := httptest.NewRequest(http.MethodPost, "/api/persons", nil)
			if tt.authenticated {
				req = req.WithContext(auth.ContextWithUserID(req.Context(), 1))
			}
			rec := httptest.NewRecorder()
			handler(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantStatus == http.StatusNoContent, called)
			require.NotContains(t, rec.Body.String(), "database down")
			if tt.wantStatus == http.StatusForbidden {
				require.Len(t, appLogger.entries, 1)
				entry := appLogger.entries[0]
				require.Equal(t, "warn", entry.level)
				require.Equal(t, "catalog modification forbidden", entry.msg)
				require.Equal(t, domain.UserID(1), entry.fields["user_id"])
				require.Equal(t, "/api/persons", entry.fields["path"])
			}

		})
	}
}

func TestPersonWriteRoutesRejectReader(t *testing.T) {
	handler := newTestRouter(t)
	register := httptest.NewRecorder()
	handler.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"email":"reader@example.com","username":"reader","password":"Password1"}`)))
	require.Equal(t, http.StatusCreated, register.Code)
	cookie := register.Result().Cookies()[0]
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "create", method: http.MethodPost, path: "/api/persons"},
		{name: "update", method: http.MethodPut, path: "/api/persons/1"},
		{name: "delete", method: http.MethodDelete, path: "/api/persons/1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}
