package userclient

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidArgument    = errors.New("invalid argument or empty update data")
	ErrInternal           = errors.New("internal server error")

	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrUserExists         = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

type UserClient interface {
	CreateUser(ctx context.Context, email, username, password string) (int64, error)
	GetByID(ctx context.Context, userID int64) (*pbUser.User, error)
	GetUserProfile(ctx context.Context, userID int64) (*pbUser.ClientProfile, error)
	UpdateProfile(ctx context.Context, userID int64, email, username string) (*pbUser.User, error)
}

type userClient struct {
	client pbUser.UserServiceClient
}

func NewUserClient(cl pbUser.UserServiceClient) UserClient {
	return &userClient{client: cl}
}

func (c *userClient) CreateUser(ctx context.Context, email, username, password string) (int64, error) {
	resp, err := c.client.CreateUser(ctx, &pbUser.CreateUserRequest{
		UserName: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			return 0, ErrEmailAlreadyExists
		}
		return 0, ErrInternal
	}
	return resp.UserId, nil
}

func (c *userClient) GetByID(ctx context.Context, userID int64) (*pbUser.User, error) {
	resp, err := c.client.GetByID(ctx, &pbUser.GetUserByIDRequest{UserId: userID})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil, ErrUserNotFound
		}
		return nil, ErrInternal
	}

	if resp.GetUser() == nil {
		return nil, ErrInternal
	}

	return resp.User, nil
}

func (c *userClient) GetUserProfile(ctx context.Context, userID int64) (*pbUser.ClientProfile, error) {
	resp, err := c.client.GetUserProfile(ctx, &pbUser.GetUserProfileRequest{UserId: userID})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil, ErrUserNotFound
		}
		return nil, ErrInternal
	}
	return resp.Profile, nil
}

func (c *userClient) UpdateProfile(ctx context.Context, userID int64, email, username string) (*pbUser.User, error) {
	req := &pbUser.UpdateProfileRequest{
		UserId:   userID,
		UserName: username,
		Email:    email,
	}

	resp, err := c.client.UpdateProfile(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				return nil, ErrEmailAlreadyExists
			case codes.InvalidArgument:
				return nil, ErrInvalidArgument
			}
		}
		return nil, ErrInternal
	}
	return resp.User, nil
}
