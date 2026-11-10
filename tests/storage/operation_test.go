package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestDatabaseOperationLogs(t *testing.T) {
	for _, tc := range []struct {
		name, backend, outcome, level string
		err                           error
	}{
		{"success", "postgres", "ok", "INFO", nil},
		{"not found", "postgres", "not_found", "INFO", domain.ErrUserNotFound},
		{"conflict", "postgres", "rejected", "WARN", domain.ErrUsernameTaken},
		{"canceled", "postgres", "error", "ERROR", context.Canceled},
		{"deadline", "redis", "error", "ERROR", context.DeadlineExceeded},
		{"expired session", "redis", "not_found", "INFO", domain.ErrSessionNotFound},
		{"postgres diagnostic", "postgres", "error", "ERROR", fmt.Errorf("query: %w", &pgconn.PgError{Code: "23505", Message: "secret-password", Detail: "secret-token"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			appLogger, err := logger.NewLogger(&buffer, logger.LogLevelInfo, "json")
			require.NoError(t, err)
			fields := map[string]any{"user_id": 7}
			op := storage.StartOperation(logger.ContextWithRequestID(context.Background(), "request-123"), appLogger, tc.backend, "account.get_by_id", fields)
			op.Stage = "query"
			op.Finish(tc.err)
			var record map[string]any
			require.NoError(t, json.Unmarshal(buffer.Bytes(), &record))
			require.Equal(t, tc.backend, record["backend"])
			require.Equal(t, tc.outcome, record["outcome"])
			require.Equal(t, tc.level, record["level"])
			require.Equal(t, "request-123", record["request_id"])
			require.Equal(t, "account.get_by_id", record["operation"])
			require.Equal(t, "query", record["stage"])
			require.Contains(t, record, "duration_ms")
			require.Equal(t, map[string]any{"user_id": 7}, fields)
			require.NotContains(t, buffer.String(), "secret-password")
			require.NotContains(t, buffer.String(), "secret-token")
			if tc.name == "postgres diagnostic" {
				require.Equal(t, "23505", record["sqlstate"])
			}
		})
	}
}
