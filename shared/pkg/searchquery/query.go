package searchquery

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	DefaultLimit = 3
	MaxPageLimit = 100
)

var (
	ErrInvalidInput = errors.New("invalid search input")
	ErrQuery        = errors.New("q must be from 2 to 200 characters")
	ErrLimit        = errors.New("limit must be an integer from 1 to 5")
)

func Validate(query string, limit int) (string, error) {
	query, err := NormalizeQuery(query)
	if err != nil {
		return "", err
	}

	if limit < 1 || limit > 5 {
		return "", ErrLimit
	}

	return query, nil
}

func NormalizeQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if length := utf8.RuneCountInString(query); length < 2 || length > 200 {
		return "", ErrQuery
	}

	return query, nil
}

func ValidatePage(limit int, offset int64) error {
	if limit < 1 || limit > MaxPageLimit {
		return fmt.Errorf("limit must be an integer from 1 to %d", MaxPageLimit)
	}

	if offset < 0 {
		return errors.New("offset must be a non-negative integer")
	}

	return nil
}

func Patterns(query string) (prefix, contains string) {
	escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)
	return escaped + "%", "%" + escaped + "%"
}
