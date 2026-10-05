package auth

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	minUsernameLen  = 2
	maxUsernameLen  = 32
	minPasswordLen  = 8
	maxPasswordLen  = 32
	passwordSymbols = "!@#$%^&*()-_=+[]{};:,.?/"
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

	for _, r := range username {
		if !isAllowedUsernameCharacter(r) {
			return fmt.Errorf("%w: contains an unsupported character", ErrInvalidUsername)
		}
	}

	return nil
}

func ValidatePassword(password string) error {
	if err := ValidatePasswordLength(password); err != nil {
		return err
	}

	if err := ValidatePasswordCharacters(password); err != nil {
		return err
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
		return fmt.Errorf("%w: at least one capital letter is required", ErrInvalidPassword)
	}

	if !hasDigit {
		return fmt.Errorf("%w: at least one number is needed", ErrInvalidPassword)
	}

	return nil
}

func ValidatePasswordLength(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minPasswordLen {
		return fmt.Errorf("%w: no shorter than %d characters", ErrInvalidPassword, minPasswordLen)
	}

	return ValidatePasswordMaxLength(password)
}

func ValidatePasswordMaxLength(password string) error {
	if utf8.RuneCountInString(password) > maxPasswordLen {
		return fmt.Errorf("%w: no longer than %d characters", ErrInvalidPassword, maxPasswordLen)
	}

	return nil
}

func ValidatePasswordCharacters(password string) error {
	for _, r := range password {
		if !isAllowedPasswordCharacter(r) {
			return fmt.Errorf("%w: contains an unsupported character", ErrInvalidPassword)
		}
	}

	return nil
}

func isAllowedPasswordCharacter(r rune) bool {
	return r >= 'A' && r <= 'Z' ||
		r >= 'a' && r <= 'z' ||
		r >= '0' && r <= '9' ||
		strings.ContainsRune(passwordSymbols, r)
}

func isAllowedUsernameCharacter(r rune) bool {
	return unicode.IsLetter(r) ||
		r >= '0' && r <= '9' ||
		unicode.IsSpace(r) ||
		r == '-' ||
		r == '_'
}
