package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_RequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		withUserID bool
		role       domain.Role
		getUserErr error
		wantStatus int
		wantError  string
		wantNext   bool
	}{
		{"without user ID", false, "", nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"user not found", true, "", nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"role lookup failure", true, "", errors.New("database unavailable"), http.StatusInternalServerError, domain.ErrInternal.Error(), false},
		{"reader is forbidden", true, domain.RoleReader, nil, http.StatusForbidden, domain.ErrForbidden.Error(), false},
		{"admin is allowed", true, domain.RoleAdmin, nil, http.StatusOK, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler *auth.Handler
			ctx := t.Context()

			switch {
			case tt.getUserErr != nil:
				ctrl := gomock.NewController(t)
				users := mocks.NewMockUserRepository(ctrl)
				users.EXPECT().GetUser(gomock.Any(), domain.UserID(42)).Return(nil, tt.getUserErr)
				handler = newMockAuthHandler(t, users, mocks.NewMockSessionStore(ctrl), mocks.NewMockPasswordHasher(ctrl))
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
