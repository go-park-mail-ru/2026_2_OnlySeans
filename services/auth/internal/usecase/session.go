package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/repository"
)

type SessionUseCase interface {
	Create(ctx context.Context, userID domain.UserID, role string) (domain.Session, error)
	Check(ctx context.Context, id uuid.UUID) (domain.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Refresh(ctx context.Context, id uuid.UUID) (domain.Session, error)
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
	sessionID := uuid.New()
	expiresAt := time.Now().Add(s.sessionTTL)

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
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}

	if session == nil {
		return domain.Session{}, domain.ErrSessionNotFound
	}

	if !session.ExpiresAt.After(time.Now()) {
		return domain.Session{}, domain.ErrSessionExpired
	}

	return *session, nil
}

func (s *sessionUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return s.sessionRepo.Delete(ctx, id)
}

func (s *sessionUseCase) Refresh(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	session, err := s.sessionRepo.Refresh(ctx, id, time.Now().Add(s.sessionTTL))
	if err != nil {
		return domain.Session{}, err
	}

	if session == nil {
		return domain.Session{}, domain.ErrSessionNotFound
	}

	if !session.ExpiresAt.After(time.Now()) {
		return domain.Session{}, domain.ErrSessionExpired
	}

	return *session, nil
}
