package usecase

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/repository"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type SessionUseCase interface {
	Create(ctx context.Context, userID domain.UserID, role string) (domain.Session, error)
	Check(ctx context.Context, id uuid.UUID) (domain.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type sessionUseCase struct {
	sessionRepo repository.SessionRepository
	sessionTTL  time.Duration
}

func NewSessionUseCase(sr repository.SessionRepository, ttl time.Duration) SessionUseCase {
	return &sessionUseCase{
		sessionRepo: sr,
		sessionTTL:  ttl,
	}
}

func (s *sessionUseCase) Create(ctx context.Context, userID domain.UserID, role string) (domain.Session, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("user.id", int64(userID)),
		attribute.String("user.role", role),
	)

	sessionID := uuid.New()
	expiresAt := time.Now().Add(s.sessionTTL)

	span.SetAttributes(attribute.String("session.id", sessionID.String()))

	session := domain.Session{
		ID:        sessionID,
		UserID:    userID,
		Role:      role,
		ExpiresAt: expiresAt,
	}

	err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return domain.Session{}, err
	}

	return session, nil
}

func (s *sessionUseCase) Check(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("session.id", id.String()),
	)

	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if session == nil {
		return domain.Session{}, domain.ErrSessionNotFound
	}

	span.SetAttributes(
		attribute.Int64("user.id", int64(session.UserID)),
		attribute.String("user.role", session.Role),
	)

	if !session.ExpiresAt.After(time.Now()) {
		span.AddEvent("session_expired")
		return domain.Session{}, domain.ErrSessionExpired
	}

	return *session, nil
}

func (s *sessionUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("session.id", id.String()),
	)

	return s.sessionRepo.Delete(ctx, id)
}
