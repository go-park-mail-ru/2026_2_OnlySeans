package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionIDBytes = 32

type Session struct {
	ID        string
	UserID    UserID
	ExpiresAt time.Time
}

type SessionStore interface {
	Create(userID UserID) (*Session, error)
	Get(sessionID string) (*Session, error)
	Delete(sessionID string) error
}

type InMemorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
	ttl      time.Duration
	now      func() time.Time
}

func NewInMemorySessionStore(ttl time.Duration) *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		now:      time.Now,
	}
}

func generateSessionID() (string, error) {
	b := make([]byte, sessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (s *InMemorySessionStore) Create(userID UserID) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: s.now().Add(s.ttl),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[id] = session

	return session, nil
}

func (s *InMemorySessionStore) Get(sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	if !s.now().Before(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return nil, ErrSessionNotFound
	}

	return session, nil
}

func (s *InMemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)

	return nil
}
