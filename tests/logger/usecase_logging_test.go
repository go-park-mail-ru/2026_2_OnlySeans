package logger

import (
	"context"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecaselogging "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	applogger "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestUseCaseFailureLevels(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		level string
	}{
		{
			name:  "missing rating",
			err:   fmt.Errorf("delete rating: %w", domain.ErrRatingNotFound),
			level: "DEBUG",
		},
		{
			name:  "invalid rating",
			err:   domain.InvalidFilmRatingLimitsError(),
			level: "WARN",
		},
		{
			name:  "review version conflict",
			err:   fmt.Errorf("update review: %w", domain.ErrVersionConflict),
			level: "WARN",
		},
		{
			name:  "invalid review version",
			err:   domain.ErrInvalidVersion,
			level: "WARN",
		},
		{
			name:  "missing film",
			err:   domain.ErrFilmNotFound,
			level: "DEBUG",
		},
		{
			name:  "wrapped missing person",
			err:   fmt.Errorf("get person: %w", domain.ErrPersonNotFound),
			level: "DEBUG",
		},
		{
			name:  "missing review",
			err:   domain.ErrReviewNotFound,
			level: "DEBUG",
		},
		{
			name:  "duplicate review",
			err:   domain.ErrReviewExists,
			level: "WARN",
		},
		{
			name:  "duplicate user",
			err:   domain.ErrUserExists,
			level: "WARN",
		},
		{
			name:  "forbidden operation",
			err:   domain.ErrForbidden,
			level: "WARN",
		},
		{
			name:  "technical failure",
			err:   errors.New("database unavailable"),
			level: "ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			appLogger, err := applogger.NewLogger(&output, applogger.LogLevelDebug, "json")
			require.NoError(t, err)
			fields := map[string]any{"user_id": 7}
			usecaselogging.Failure(context.Background(), appLogger, "operation failed", tt.err, fields)
			require.NotContains(t, fields, "error")
			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			require.Equal(t, tt.level, entry["level"])
			require.Equal(t, float64(7), entry["user_id"])
		})
	}
}

func TestValidationLoggingOmitsSubmittedValues(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		field string
	}{
		{
			name:  "password",
			err:   fmt.Errorf("%w: do-not-log-submitted-value", domain.ErrInvalidPassword),
			field: "password",
		},
		{
			name:  "review text",
			err:   fmt.Errorf("%w: do-not-log-submitted-value", domain.ErrInvalidContent),
			field: "content",
		},
		{
			name:  "birth date",
			err:   domain.ErrInvalidBirthDate,
			field: "birth_date",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			appLogger, err := applogger.NewLogger(&output, applogger.LogLevelDebug, "json")
			require.NoError(t, err)
			usecaselogging.Validation(context.Background(), appLogger, "invalid input", tt.err, nil)
			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			require.Equal(t, "WARN", entry["level"])
			require.Equal(t, tt.field, entry["field"])
			require.NotContains(t, output.String(), "do-not-log-submitted-value")
		})
	}
}
