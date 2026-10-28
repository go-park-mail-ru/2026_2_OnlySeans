package fakes

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SessionStore struct {
	mu     sync.Mutex
	byID   map[string]*domain.Session
	nextID int
}

func NewSessionStore() *SessionStore {
	return &SessionStore{byID: make(map[string]*domain.Session), nextID: 1}
}

func (s *SessionStore) Create(_ context.Context, userID domain.UserID) (*domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("sid-%d", s.nextID)
	s.nextID++
	s.byID[id] = &domain.Session{ID: id, UserID: userID}
	return s.byID[id], nil
}

func (s *SessionStore) Get(_ context.Context, sessionID string) (*domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byID[sessionID]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return session, nil
}

func (s *SessionStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, sessionID)
	return nil
}
