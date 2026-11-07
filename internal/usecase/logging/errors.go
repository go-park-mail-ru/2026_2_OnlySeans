package logging

import (
	"errors"
	"maps"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func Failure(appLogger logger.Logger, message string, err error, fields map[string]any) {
	fields = maps.Clone(fields)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["error"] = err
	switch {
	case matches(err, domain.ErrFilmNotFound, domain.ErrCollectionNotFound, domain.ErrPersonNotFound, domain.ErrReviewNotFound, domain.ErrUserNotFound, domain.ErrSessionNotFound, domain.ErrFileNotFound):
		appLogger.Debug(message, fields)
	case matches(err, domain.ErrUserExists, domain.ErrUsernameTaken, domain.ErrReviewExists, domain.ErrFilmExists, domain.ErrCollectionExists, domain.ErrDuplicateCollectionFilm, domain.ErrForbidden, domain.ErrUnauthorized, domain.ErrInvalidCredentials):
		appLogger.Warn(message, fields)
	default:
		appLogger.Error(message, fields)
	}
}

func Validation(appLogger logger.Logger, message string, err error, fields map[string]any) {
	fields = maps.Clone(fields)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["field"] = invalidField(err)
	appLogger.Warn(message, fields)
}

func invalidField(err error) string {
	for _, field := range []struct {
		err  error
		name string
	}{
		{domain.ErrInvalidEmail, "email"},
		{domain.ErrInvalidUsername, "username"},
		{domain.ErrInvalidPassword, "password"},
		{domain.ErrInvalidFirstName, "first_name"},
		{domain.ErrInvalidLastName, "last_name"},
		{domain.ErrInvalidBirthDate, "birth_date"},
		{domain.ErrInvalidTitle, "title"},
		{domain.ErrInvalidContent, "content"},
	} {
		if errors.Is(err, field.err) {
			return field.name
		}
	}
	return "input"
}

func matches(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
