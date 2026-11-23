package userclient

import (
	"context"
	"errors"

	pbUser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidArgument    = errors.New("invalid argument or empty update data")

	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrUserExists         = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInternal           = errors.New("internal server error")
)

type UserClient interface {
	CreateUser(ctx context.Context, email, username, password string) (int64, error)
	GetByID(ctx context.Context, userID int64) (*pbUser.User, error)
	GetProfile(ctx context.Context, userID int64) (*pbUser.GetProfileResponse, error)
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
		Username: username,
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
	resp, err := c.client.GetByID(ctx, &pbUser.GetByIDRequest{UserId: userID})
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

func (c *userClient) GetProfile(ctx context.Context, userID int64) (*pbUser.GetProfileResponse, error) {
	resp, err := c.client.GetProfile(ctx, &pbUser.GetProfileRequest{UserId: userID})
	if status.Code(err) == codes.NotFound {
		return nil, ErrUserNotFound
	}
	if err != nil || resp.GetUser() == nil || resp.GetCreatedAt() == nil {
		return nil, ErrInternal
	}
	return resp, nil
}

func (c *userClient) UpdateProfile(ctx context.Context, userID int64, email, username string) (*pbUser.User, error) {
	req := &pbUser.UpdateProfileRequest{
		UserId:   userID,
		Username: &username,
		Email:    &email,
	}

	resp, err := c.client.UpdateProfile(ctx, req)
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.NotFound:
			return nil, ErrUserNotFound
		case codes.AlreadyExists:
			switch st.Message() {
			case "username taken":
				return nil, ErrUsernameTaken
			case "email taken":
				return nil, ErrEmailAlreadyExists
			}
		case codes.InvalidArgument:
			switch st.Message() {
			case "invalid email":
				return nil, ErrInvalidEmail
			case "invalid username":
				return nil, ErrInvalidUsername
			}
			return nil, ErrInvalidArgument
		}
		return nil, ErrInternal
	}
	if resp.GetUser() == nil {
		return nil, ErrInternal
	}
	return resp.User, nil
}
