package onlineconf

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type SessionReader interface {
	Get(context.Context, string) (*domain.Session, error)
}
type AccessUserReader interface {
	GetUser(context.Context, domain.UserID) (*domain.User, error)
}

type Access struct {
	sessions  SessionReader
	users     AccessUserReader
	appLogger logger.Logger
}

func NewAccess(sessions SessionReader, users AccessUserReader, appLogger logger.Logger) (*Access, error) {
	if sessions == nil {
		return nil, domain.ErrNilSessionStore
	}

	if users == nil {
		return nil, oc.ErrNilUserReader
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &Access{sessions: sessions, users: users, appLogger: appLogger}, nil
}

func (a *Access) AuthenticateAdmin(ctx context.Context, sessionID string) (oc.AdminIdentity, error) {
	if strings.TrimSpace(sessionID) == "" {
		return oc.AdminIdentity{}, domain.ErrUnauthorized
	}

	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrUnauthorized) {
			return oc.AdminIdentity{}, domain.ErrUnauthorized
		}

		return oc.AdminIdentity{}, fmt.Errorf("read onlineconf session: %w", err)
	}

	if session == nil || (!session.ExpiresAt.IsZero() && !session.ExpiresAt.After(time.Now())) {
		return oc.AdminIdentity{}, domain.ErrUnauthorized
	}

	if session.UserID < oc.MinUserID || int64(session.UserID) > oc.MaxSafeID {
		return oc.AdminIdentity{}, fmt.Errorf("invalid onlineconf session user: %w", oc.ErrInvalidUserID)
	}

	user, err := a.users.GetUser(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return oc.AdminIdentity{}, domain.ErrUnauthorized
		}

		return oc.AdminIdentity{}, fmt.Errorf("read onlineconf admin role: %w", err)
	}

	if user == nil {
		return oc.AdminIdentity{}, domain.ErrUnauthorized
	}

	if user.ID != session.UserID {
		return oc.AdminIdentity{}, fmt.Errorf("onlineconf session/user mismatch: %w", domain.ErrInternal)
	}

	if user.Role != domain.RoleAdmin {
		a.appLogger.WarnContext(ctx, "onlineconf admin access forbidden", map[string]any{"user_id": session.UserID})
		return oc.AdminIdentity{}, domain.ErrForbidden
	}

	return oc.AdminIdentity{ID: user.ID, Username: user.Username, Role: user.Role}, nil
}
