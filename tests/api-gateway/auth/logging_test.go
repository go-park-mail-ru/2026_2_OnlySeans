package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pbauth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthFailuresLogCodeAndRequestIDWithoutCredentials(t *testing.T) {
	for _, operation := range []string{"register", "login", "logout", "authorised.get_user"} {
		t.Run(operation, func(t *testing.T) {
			var output bytes.Buffer
			l, err := logger.NewLogger(&output, logger.LogLevelDebug, "json")
			require.NoError(t, err)
			failure := status.Error(codes.Unavailable, "secret-password secret-cookie")
			ac := authClientStub{
				login:  func(context.Context, string, string) (*pbauth.Session, error) { return nil, failure },
				logout: func(context.Context, string) error { return failure },
			}
			uc := userClientStub{
				create: func(context.Context, string, string, string) (int64, error) { return 0, failure },
				get:    func(context.Context, int64) (*pbuser.User, error) { return nil, failure },
			}
			h := authhttp.NewAuthHandler(ac, uc, l)
			handlers := map[string]http.HandlerFunc{"register": h.Register, "login": h.Login, "logout": h.Logout, "authorised.get_user": h.Authorised}
			ctx := middleware.ContextWithUserID(logger.ContextWithRequestID(context.Background(), "http-42"), 7)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"user@example.com","username":"reader","password":"secret-password"}`)).WithContext(ctx)
			req.AddCookie(&http.Cookie{
				Name:  authhttp.SessionCookieName,
				Value: "secret-cookie",
			})
			w := httptest.NewRecorder()
			handlers[operation](w, req)
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.NotContains(t, w.Body.String(), "secret")
			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, "ERROR", record["level"])
			require.Equal(t, operation, record["operation"])
			require.Equal(t, "http-42", record["request_id"])
			require.Equal(t, "Unavailable", record["grpc_code"])
			require.NotContains(t, output.String(), "secret")
		})
	}
}
