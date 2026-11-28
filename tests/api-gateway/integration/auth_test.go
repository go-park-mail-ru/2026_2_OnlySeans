//go:build integration

package integration_test

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

// Authentication is isolated from persisted accounts: only this in-process fixture accepts this session.
type integrationSessions struct{ authclient.AuthClient }

func (integrationSessions) CheckSession(_ context.Context, session string) (authclient.SessionInfo, error) {
	if session != "integration-session" {
		return authclient.SessionInfo{}, authclient.ErrSessionNotFound
	}
	return authclient.SessionInfo{UserID: 1}, nil
}

type integrationUsers struct {
	userclient.UserClient
	role string
}

func (u *integrationUsers) GetByID(context.Context, int64) (*pb.User, error) {
	return &pb.User{Id: 1, Role: u.role}, nil
}
