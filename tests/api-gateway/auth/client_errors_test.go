package auth_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/grpcutil"
	pbauth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingAuthRPC struct {
	pbauth.AuthServiceClient
	err error
}

func (c failingAuthRPC) Login(context.Context, *pbauth.LoginRequest, ...grpc.CallOption) (*pbauth.LoginResponse, error) {
	return nil, c.err
}
func (c failingAuthRPC) IssueSession(context.Context, *pbauth.IssueSessionRequest, ...grpc.CallOption) (*pbauth.IssueSessionResponse, error) {
	return nil, c.err
}

type failingUserRPC struct {
	pbuser.UserServiceClient
	err error
}

func (c failingUserRPC) GetByID(context.Context, *pbuser.GetByIDRequest, ...grpc.CallOption) (*pbuser.GetByIDResponse, error) {
	return nil, c.err
}
func (c failingUserRPC) CreateUser(context.Context, *pbuser.CreateUserRequest, ...grpc.CallOption) (*pbuser.CreateUserResponse, error) {
	return nil, c.err
}

func TestClientsPreserveTechnicalFailureStatus(t *testing.T) {
	failure := status.Error(codes.Unavailable, "secret-server-detail")
	auth := authclient.NewAuthClient(failingAuthRPC{err: failure})
	users := userclient.NewUserClient(failingUserRPC{err: failure})
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		sentinel error
		call     func() error
	}{
		{"login", authclient.ErrInternal, func() error { _, err := auth.Login(ctx, "user@example.com", "password"); return err }},
		{"issue", authclient.ErrInternal, func() error { _, err := auth.IssueSession(ctx, 7, "reader"); return err }},
		{"get user", userclient.ErrInternal, func() error { _, err := users.GetByID(ctx, 7); return err }},
		{"register", userclient.ErrInternal, func() error { _, err := users.CreateUser(ctx, "user@example.com", "reader", "password"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.ErrorIs(t, err, tc.sentinel)
			require.Equal(t, codes.Unavailable, status.Code(err))
			fields := grpcutil.ErrorFields(err)
			require.Equal(t, "RPC failed: Unavailable", fields["error"])
		})
	}
}
