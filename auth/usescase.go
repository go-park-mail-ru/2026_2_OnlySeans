package auth

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("неверный email или пароль")
	ErrInternal           = errors.New("внутренняя ошибка сервера")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email обязателен")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("некорректный формат email")
	}
	return nil
}

func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return errors.New("имя пользователя должно быть не короче 3 символов")
	}
	if len(username) > 32 {
		return errors.New("имя пользователя слишком длинное (максимум 32 символа)")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("пароль должен быть не короче 8 символов")
	}
	var hasUpper, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper {
		return errors.New("пароль должен содержать хотя бы одну заглавную букву")
	}
	if !hasDigit {
		return errors.New("пароль должен содержать хотя бы одну цифру")
	}
	return nil
}

type SessionIssuer interface {
	IssueSession(userID int) (string, error)
}

type AuthResult struct {
	User  *User
	Token string // пусто, пока Session == nil
}

type UseCase struct {
	Repo    UserRepository
	Session SessionIssuer
}

func NewUseCase(repo UserRepository, session SessionIssuer) *UseCase {
	return &UseCase{Repo: repo, Session: session}
}

func (uc *UseCase) Register(email, username, password string) (*AuthResult, error) {
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

	user, err := uc.Repo.Create(email, username, string(hash))
	if err != nil {
		return nil, err // тут будет ErrUserExists из repo.go
	}

	return uc.buildResult(user)
}

func (uc *UseCase) Login(email, password string) (*AuthResult, error) {
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, errors.New("пароль обязателен")
	}

	user, err := uc.Repo.FindByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return uc.buildResult(user)
}

func (uc *UseCase) buildResult(user *User) (*AuthResult, error) {
	result := &AuthResult{User: user}
	if uc.Session != nil {
		if token, err := uc.Session.IssueSession(user.ID); err == nil {
			result.Token = token
		}
	}
	return result, nil
}
