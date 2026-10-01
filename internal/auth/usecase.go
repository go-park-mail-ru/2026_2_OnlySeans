package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minUsernameLen = 2
	maxUsernameLen = 32
	minPasswordLen = 8
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidEmail)
	}

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("%w: invalid format", ErrInvalidEmail)
	}

	return nil
}

func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	lenUserName := utf8.RuneCountInString(username)

	if lenUserName < minUsernameLen {
		return fmt.Errorf("%w: no shorter than %d characters", ErrInvalidUsername, minUsernameLen)
	}

	if lenUserName > maxUsernameLen {
		return fmt.Errorf("%w: name is too long, maximum %d characters", ErrInvalidUsername, maxUsernameLen)
	}

	return nil
}

func ValidatePassword(password string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("%w: no shorter than %d characters", ErrInvalidPassword, minPasswordLen)
	}

	var hasUpper, hasDigit bool

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("%w: at least one capital letter is required", ErrInvalidPassword)
	}

	if !hasDigit {
		return fmt.Errorf("%w: at least one number is needed", ErrInvalidPassword)
	}

	return nil
}

type AuthResult struct {
	User    *User
	Session *Session
}

type UseCase struct {
	Repo     UserRepository
	Sessions SessionStore
}

func NewUseCase(repo UserRepository, sessions SessionStore) (*UseCase, error) {
	if repo == nil {
		return nil, ErrNilRepo
	}

	if sessions == nil {
		return nil, ErrNilSessionStore
	}

	return &UseCase{Repo: repo, Sessions: sessions}, nil
}

func (uc *UseCase) Register(ctx context.Context, email, username, password string) (*AuthResult, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	if err := ValidateUsername(username); err != nil {
		return nil, err
	}

	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrInternal
	}

	userID, err := uc.Repo.Create(ctx, email, username, string(hash))
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			return nil, ErrUserExists
		}

		return nil, ErrInternal
	}

	user, err := uc.Repo.GetUser(ctx, userID)
	if err != nil {
		return nil, ErrInternal
	}

	return uc.buildResult(user)
}

func (uc *UseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	if password == "" {
		return nil, fmt.Errorf("%w: password required", ErrInvalidPassword)
	}

	user, err := uc.Repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return uc.buildResult(user)
}

func (uc *UseCase) buildResult(user *User) (*AuthResult, error) {
	session, err := uc.Sessions.Create(user.ID)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthResult{User: user, Session: session}, nil
}

func (uc *UseCase) Logout(sessionID string) error {
	if err := uc.Sessions.Delete(sessionID); err != nil {
		return ErrInternal
	}

	return nil
}

// Authenticate проверяет сессию и возвращает ID её владельца.
func (uc *UseCase) Authenticate(ctx context.Context, sessionID string) (UserID, error) {
	session, err := uc.Sessions.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return 0, ErrUnauthorized
		}

		return 0, ErrInternal
	}

	return session.UserID, nil
}
