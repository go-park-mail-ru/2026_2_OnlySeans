package grpcclient

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	users pb.UserServiceClient
}

func New(users pb.UserServiceClient) *Client {
	return &Client{
		users: users,
	}
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	resp, err := c.users.GetByEmail(ctx, &pb.GetByEmailRequest{Email: email})

	if status.Code(err) == codes.NotFound {
		return domain.User{}, domain.ErrUserNotFound
	}

	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:           domain.UserID(resp.GetUserId()),
		Role:         domain.Role(resp.GetRole()),
		PasswordHash: resp.GetPasswordHash(),
	}, nil
}
