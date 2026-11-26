package userclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pbUser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrAvatarNotFound     = errors.New("avatar not found")
	ErrInvalidAvatarName  = errors.New("invalid avatar filename")
	ErrAvatarTooLarge     = errors.New("avatar file is too large")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidArgument    = errors.New("invalid argument or empty update data")

	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUsernameTaken      = errors.New("user with this username already exists")
	ErrUserExists         = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInternal           = errors.New("internal server error")
)

type UserClient interface {
	CreateUser(ctx context.Context, email, username, password string) (int64, error)
	GetByID(ctx context.Context, userID int64) (*pbUser.User, error)
	GetProfile(ctx context.Context, userID int64) (*pbUser.GetProfileResponse, error)
	UpdateProfile(ctx context.Context, userID int64, email, username string) (*pbUser.User, error)
	GetAvatarURL(ctx context.Context, userID int64) (string, error)
	GetAvatarURLByFilename(ctx context.Context, filename string) (string, error)
	UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
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
		return 0, mapRegistrationError(err)
	}
	return resp.UserId, nil
}

func mapRegistrationError(err error) error {
	st := status.Convert(err)
	switch st.Code() {
	case codes.AlreadyExists:
		switch st.Message() {
		case "email taken":
			return ErrUserExists
		case "username taken":
			return ErrUsernameTaken
		}
	case codes.InvalidArgument:
		field, reason, _ := strings.Cut(st.Message(), ": ")
		var fieldErr error
		switch field {
		case "invalid email":
			fieldErr = ErrInvalidEmail
		case "invalid username":
			fieldErr = ErrInvalidUsername
		case "invalid password":
			fieldErr = ErrInvalidPassword
		default:
			return fmt.Errorf("%w: %w", ErrInternal, err)
		}
		if reason == "" {
			return fieldErr
		}
		return fmt.Errorf("%w: %s", fieldErr, reason)
	}

	return fmt.Errorf("%w: %w", ErrInternal, err)
}

func (c *userClient) GetByID(ctx context.Context, userID int64) (*pbUser.User, error) {
	resp, err := c.client.GetByID(ctx, &pbUser.GetByIDRequest{UserId: userID})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
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

func (c *userClient) GetAvatarURL(ctx context.Context, userID int64) (string, error) {
	resp, err := c.client.GetAvatarURL(ctx, &pbUser.GetAvatarURLRequest{
		UserId: userID,
	})
	if err != nil {
		return "", mapAvatarError(err)
	}

	return resp.GetUrl(), nil
}

func (c *userClient) GetAvatarURLByFilename(ctx context.Context, filename string) (string, error) {
	resp, err := c.client.GetAvatarURLByFilename(ctx, &pbUser.GetAvatarURLByFilenameRequest{
		Filename: filename,
	})
	if err != nil {
		return "", mapAvatarError(err)
	}

	return resp.GetUrl(), nil
}

func (c *userClient) UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error) {
	resp, err := c.client.UpdateAvatar(ctx, &pbUser.UpdateAvatarRequest{
		UserId:    userID,
		ImageData: imageData,
	})
	if err != nil {
		return "", mapAvatarError(err)
	}

	return resp.GetAvatarFilename(), nil
}

func (c *userClient) DeleteAvatar(ctx context.Context, userID int64) error {
	_, err := c.client.DeleteAvatar(ctx, &pbUser.DeleteAvatarRequest{
		UserId: userID,
	})
	if err != nil {
		return mapAvatarError(err)
	}

	return nil
}

func mapAvatarError(err error) error {
	st := status.Convert(err)

	switch st.Code() {
	case codes.NotFound:
		switch st.Message() {
		case "user not found":
			return ErrUserNotFound
		case "avatar not found":
			return ErrAvatarNotFound
		}
	case codes.InvalidArgument:
		if st.Message() == "invalid avatar filename" {
			return ErrInvalidAvatarName
		}
		return ErrInvalidArgument
	case codes.ResourceExhausted:
		return ErrAvatarTooLarge
	}

	return ErrInternal
}
