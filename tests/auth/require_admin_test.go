package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
)

func TestHandler_RequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		withUserID bool
		role       domain.UserRole
		getUserErr error
		wantStatus int
		wantError  string
		wantNext   bool
	}{
		{"without user ID", false, "", nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"user not found", true, "", nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"role lookup failure", true, "", errors.New("database unavailable"), http.StatusInternalServerError, domain.ErrInternal.Error(), false},
		{"reader is forbidden", true, domain.UserRoleReader, nil, http.StatusForbidden, domain.ErrForbidden.Error(), false},
		{"admin is allowed", true, domain.UserRoleAdmin, nil, http.StatusOK, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *auth.Handler
			ctx := t.Context()

			switch {
			case tt.getUserErr != nil:
				handler, _ = newFlakyHandler(t, flakyRepo{getUserErr: tt.getUserErr}, flakyStore{})
				ctx = auth.ContextWithUserID(ctx, 42)
			case tt.withUserID && tt.role == "":
				handler = newTestHandler(t)
				ctx = auth.ContextWithUserID(ctx, 42)
			case tt.withUserID:
				var usersID domain.UserID
				var users *authservice.InMemoryUserRepo
				handler, users, _ = newTestAuth(t)
				usersID, err := users.Create(ctx, "admin@example.com", "admin", "hash")
				require.NoError(t, err)
				user, err := users.GetUser(ctx, usersID)
				require.NoError(t, err)
				user.Role = tt.role
				ctx = auth.ContextWithUserID(ctx, usersID)
			default:
				handler = newTestHandler(t)
			}

			nextCalled := false
			protected := handler.RequireAdmin(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/api/films", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			protected(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled, "next handler call")
			if tt.wantError == "" {
				return
			}

			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantError, body["error"])
		})
	}
}
