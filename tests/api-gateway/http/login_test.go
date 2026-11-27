package http_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

func TestGatewayLoginErrors(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		code        codes.Code
		message     string
		httpStatus  int
		httpMessage string
	}{
		{
			name:        "wrong password",
			email:       "reader@example.com",
			password:    "wrong-password",
			code:        codes.Unauthenticated,
			message:     "invalid email or password",
			httpStatus:  http.StatusUnauthorized,
			httpMessage: "invalid email or password",
		},
		{
			name:        "unknown email",
			email:       "unknown@example.com",
			password:    "Password1",
			code:        codes.Unauthenticated,
			message:     "invalid email or password",
			httpStatus:  http.StatusUnauthorized,
			httpMessage: "invalid email or password",
		},
		{
			name:        "empty password",
			email:       "reader@example.com",
			code:        codes.InvalidArgument,
			message:     "invalid password: no shorter than 8 characters",
			httpStatus:  http.StatusBadRequest,
			httpMessage: "invalid password: no shorter than 8 characters",
		},
		{
			name:        "invalid email",
			email:       "invalid-email",
			password:    "Password1",
			code:        codes.InvalidArgument,
			message:     "invalid email: invalid format",
			httpStatus:  http.StatusBadRequest,
			httpMessage: "invalid email: invalid format",
		},
		{
			name:        "internal error",
			email:       "reader@example.com",
			password:    "Password1",
			code:        codes.Internal,
			message:     "database unavailable",
			httpStatus:  http.StatusInternalServerError,
			httpMessage: "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := grpc.NewServer(grpc.UnaryInterceptor(func(_ context.Context, request any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
				calls.Add(1)

				req, ok := request.(*pb.LoginRequest)
				if !ok || req.GetEmail() != tt.email || req.GetPassword() != tt.password {
					return nil, status.Error(codes.Internal, "unexpected login request")
				}

				return nil, status.Error(tt.code, tt.message)
			}))
			pb.RegisterAuthServiceServer(server, &pb.UnimplementedAuthServiceServer{})

			listener := bufconn.Listen(1 << 20)
			served := make(chan error, 1)

			go func() { served <- server.Serve(listener) }()

			connection, err := grpc.NewClient("passthrough:///auth",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
			)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, connection.Close())
				server.Stop()
				require.NoError(t, listener.Close())
				require.NoError(t, <-served)
			})

			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			auth := authclient.NewAuthClient(pb.NewAuthServiceClient(connection))
			users := userStub{}
			handler, err := router.NewRouter(router.Handlers{
				Auth: authhttp.NewAuthHandler(auth, users, appLogger),
			}, middleware.NewAuthMiddleware(auth, users, appLogger), []string{"http://localhost:8000"}, appLogger)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/login", strings.NewReader(`{"email":"`+tt.email+`","password":"`+tt.password+`"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", "http://localhost:8000")

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tt.httpStatus, response.Code)
			require.JSONEq(t, `{"error":"`+tt.httpMessage+`"}`, response.Body.String())
			require.Empty(t, response.Result().Cookies())
			require.EqualValues(t, 1, calls.Load())
		})
	}
}
