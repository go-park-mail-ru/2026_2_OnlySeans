package auth_test

import (
	"context"
	"errors"
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

type refreshHTTPClient struct {
	authclient.AuthClient
	session *pb.Session
	failure error
	calls   int
}

func (c *refreshHTTPClient) RefreshSession(_ context.Context, id string) (*pb.Session, error) {
	c.calls++

	if id != "session" {
		return nil, authclient.ErrSessionNotFound
	}

	return c.session, c.failure
}
func TestRefreshHTTP(t *testing.T) {
	expiry := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)

	for _, tc := range []struct {
		name    string
		cookie  *http.Cookie
		failure error
		want    int
	}{
		{name: "success", cookie: &http.Cookie{Name: "session_id", Value: "session"}, want: 204},
		{name: "no cookie", want: 401},
		{name: "empty cookie", cookie: &http.Cookie{Name: "session_id"}, want: 401},
		{name: "expired", cookie: &http.Cookie{Name: "session_id", Value: "session"}, failure: authclient.ErrSessionNotFound, want: 401},
		{name: "invalid", cookie: &http.Cookie{Name: "session_id", Value: "invalid"}, want: 401},
		{name: "storage failure", cookie: &http.Cookie{Name: "session_id", Value: "session"}, failure: errors.New("private detail"), want: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &refreshHTTPClient{session: &pb.Session{Id: "session", ExpiresAt: timestamppb.New(expiry)}, failure: tc.failure}
			handler := authhttp.NewAuthHandler(client, nil, testAuthLogger(t))
			handler.CookieSecure = true

			req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}

			result := httptest.NewRecorder()
			handler.Refresh(result, req)
			require.Equal(t, tc.want, result.Code)
			require.Equal(t, "no-store", result.Header().Get("Cache-Control"))

			if tc.want == 204 {
				require.Empty(t, result.Body.String())
				require.Len(t, result.Result().Cookies(), 1)
				cookie := result.Result().Cookies()[0]
				require.Equal(t, "session", cookie.Value)
				require.Equal(t, expiry, cookie.Expires)
				require.True(t, cookie.HttpOnly)
				require.True(t, cookie.Secure)
				require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
				require.Equal(t, "/", cookie.Path)
			}

			if tc.cookie == nil || tc.cookie.Value == "" {
				require.Zero(t, client.calls)
			}

			if tc.want == 500 {
				require.NotContains(t, result.Body.String(), "private detail")
				require.Empty(t, result.Result().Cookies())
			}

			if tc.want == 401 && tc.cookie != nil && tc.cookie.Value != "" {
				require.Len(t, result.Result().Cookies(), 1)
				require.Negative(t, result.Result().Cookies()[0].MaxAge)
			}
		})
	}
}
func TestRefreshRouteAndCSRF(t *testing.T) {
	client := &refreshHTTPClient{session: &pb.Session{Id: "session", ExpiresAt: timestamppb.New(time.Now().Add(time.Hour))}}
	log := testAuthLogger(t)
	auth := authhttp.NewAuthHandler(client, nil, log)
	routes, err := router.NewRouter(router.Handlers{Auth: auth}, middleware.NewAuthMiddleware(client, nil, log), []string{"http://localhost:8000"}, log)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

	result := httptest.NewRecorder()
	routes.ServeHTTP(result, req)
	require.Equal(t, 204, result.Code)
	require.Equal(t, 1, client.calls)

	result = httptest.NewRecorder()
	routes.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/api/refresh", nil))
	require.Equal(t, 405, result.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req.Header.Set("Origin", "https://untrusted.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

	result = httptest.NewRecorder()
	routes.ServeHTTP(result, req)
	require.Equal(t, 403, result.Code)
	require.Equal(t, 1, client.calls)
}
