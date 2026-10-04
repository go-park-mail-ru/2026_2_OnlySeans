package auth

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
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
	length := utf8.RuneCountInString(username)

	if length < minUsernameLen {
		return fmt.Errorf("%w: no shorter than %d characters", ErrInvalidUsername, minUsernameLen)
	}

	if length > maxUsernameLen {
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
