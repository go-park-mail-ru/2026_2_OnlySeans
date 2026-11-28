package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

type refreshRPC struct {
	pb.AuthServiceClient
	response *pb.RefreshSessionResponse
	failure  error
}

func (c refreshRPC) RefreshSession(_ context.Context, req *pb.RefreshSessionRequest, _ ...grpc.CallOption) (*pb.RefreshSessionResponse, error) {
	if req.GetSessionId() != "session" {
		return nil, status.Error(codes.InvalidArgument, "invalid session")
	}

	return c.response, c.failure
}
func TestRefreshClientErrors(t *testing.T) {
	valid := &pb.Session{Id: "session", ExpiresAt: timestamppb.New(time.Now().Add(time.Hour))}
	for _, tc := range []struct {
		name     string
		response *pb.RefreshSessionResponse
		failure  error
		want     error
	}{
		{name: "success", response: &pb.RefreshSessionResponse{Session: valid}},
		{name: "missing", failure: status.Error(codes.NotFound, "missing"), want: authclient.ErrSessionNotFound},
		{name: "expired", failure: status.Error(codes.Unauthenticated, "expired"), want: authclient.ErrSessionNotFound},
		{name: "invalid", failure: status.Error(codes.InvalidArgument, "invalid"), want: authclient.ErrSessionNotFound},
		{name: "storage error", failure: status.Error(codes.Unavailable, "private"), want: authclient.ErrInternal},
		{name: "nil response", want: authclient.ErrInternal},
		{name: "empty response", response: &pb.RefreshSessionResponse{}, want: authclient.ErrInternal},
		{name: "missing expiry", response: &pb.RefreshSessionResponse{Session: &pb.Session{Id: "session"}}, want: authclient.ErrInternal},
		{name: "wrong session", response: &pb.RefreshSessionResponse{Session: &pb.Session{Id: "other", ExpiresAt: valid.ExpiresAt}}, want: authclient.ErrInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := authclient.NewAuthClient(refreshRPC{response: tc.response, failure: tc.failure})
			result, err := client.RefreshSession(t.Context(), "session")
			require.ErrorIs(t, err, tc.want)

			if tc.want == nil {
				require.Equal(t, valid, result)
			} else {
				require.Nil(t, result)
			}
		})
	}
}

func TestRefreshHTTPRouteRemoved(t *testing.T) {
	log := testAuthLogger(t)
	auth := authhttp.NewAuthHandler(nil, nil, log)
	routes, err := router.NewRouter(router.Handlers{Auth: auth}, middleware.NewAuthMiddleware(nil, nil, log), []string{"http://localhost:8000"}, log)
	require.NoError(t, err)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		result := httptest.NewRecorder()
		routes.ServeHTTP(result, httptest.NewRequest(method, "/api/refresh", nil))
		require.Equal(t, http.StatusNotFound, result.Code)
		require.Empty(t, result.Header().Get("Set-Cookie"))
	}
}

type checkSessionRPC struct {
	pb.AuthServiceClient
	response *pb.CheckSessionResponse
}

func (c checkSessionRPC) CheckSession(context.Context, *pb.CheckSessionRequest, ...grpc.CallOption) (*pb.CheckSessionResponse, error) {
	return c.response, nil
}

func TestCheckSessionDetectsOldAuthContract(t *testing.T) {
	for _, supported := range []bool{false, true} {
		response := &pb.CheckSessionResponse{UserId: 7, Role: "reader"}
		if supported {
			response.ExpiresAt = timestamppb.New(time.Now().Add(time.Hour))
			response.LastRefreshedAt = timestamppb.New(time.Time{})
		}

		session, err := authclient.NewAuthClient(checkSessionRPC{response: response}).CheckSession(t.Context(), "session")
		require.NoError(t, err)
		require.Equal(t, supported, session.RefreshSupported)
		require.EqualValues(t, 7, session.UserID)
	}
}
