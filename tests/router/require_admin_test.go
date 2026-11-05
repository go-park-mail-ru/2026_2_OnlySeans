package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoleGetter struct {
	role      domain.UserRole
	err       error
	gotUserID domain.UserID
}

func (s *fakeRoleGetter) GetUserRole(_ context.Context, userID domain.UserID) (domain.UserRole, error) {
	s.gotUserID = userID
	return s.role, s.err
}

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		withUserID bool
		role       domain.UserRole
		roleErr    error
		wantStatus int
		wantError  string
		wantNext   bool
	}{
		{"without user ID", false, "", nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"user not found", true, "", domain.ErrUserNotFound, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"role lookup failure", true, "", errors.New("database unavailable"), http.StatusInternalServerError, domain.ErrInternal.Error(), false},
		{"reader is forbidden", true, domain.UserRoleReader, nil, http.StatusForbidden, domain.ErrForbidden.Error(), false},
		{"admin is allowed", true, domain.UserRoleAdmin, nil, http.StatusOK, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := &fakeRoleGetter{role: tt.role, err: tt.roleErr}
			nextCalled := false
			handler := router.RequireAdmin(roles, func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			ctx := context.Background()
			if tt.withUserID {
				ctx = auth.ContextWithUserID(ctx, 42)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/films", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled, "next handler call")
			if tt.withUserID {
				assert.Equal(t, domain.UserID(42), roles.gotUserID)
			}

			if tt.wantError == "" {
				return
			}

			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantError, body["error"])
		})
	}
}
