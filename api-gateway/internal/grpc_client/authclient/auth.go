package authclient

import (
	"context"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrInternal           = errors.New("internal server error")
)

type AuthClient interface {
	IssueSession(ctx context.Context, userID int64, role string) (*pbAuth.Session, error)
	Login(ctx context.Context, email, password string) (*pbAuth.Session, error)
	Logout(ctx context.Context, sessionID string) error
	CheckSession(ctx context.Context, sessionID string) (SessionInfo, error)
}

type SessionInfo struct {
	UserID int64
	Role   string
}

type authClient struct {
	client pbAuth.AuthServiceClient
}

func NewAuthClient(cl pbAuth.pbAuthServiceClient) AuthClient {
	return &authClient{
		client: cl,
	}
}

func (c *authClient) IssueSession(ctx context.Context, userID int64, role string) (*pbAuth.Session, error) {
	resp, err := c.client.IssueSession(ctx, &pbAuth.IssueSessionRequest{
		UserID: userID,
		Role:   role,
	})
	if err != nil {
		return nil, ErrInternal
	}

	return resp.Session, nil
}

func (c *authClient) Login(ctx context.Context, email, password string) (*pbAuth.Session, error) {
	resp, err := c.client.Login(ctx, &pbAuth.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return nil, ErrInvalidCredentials
		}

		return nil, ErrInternal
	}

	return resp.Session, nil
}

func (c *authClient) Logout(ctx context.Context, sessionID string) error {
	_, err := c.client.Logout(ctx, &pbAuth.LogoutRequest{
		SessionID: sessionID,
	})
	return err
}

func (c *authClient) CheckSession(ctx context.Context, sessionId string) (SessionInfo, error) {
	resp, err := c.client.CheckSession(ctx, &pbAuth.CheckSessionRequest{
		SessionId: sessionId,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.NotFound || st.Code() == codes.Unaurhenticated) {
			return SessionInfo{}, ErrSessionNotFound
		}

		return SessionInfo{}, ErrInternal
	}
	return SessionInfo{
		UserID: resp.UserID,
		Role:   resp.Role,
	}, nil
}
