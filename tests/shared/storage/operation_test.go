package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

func TestDatabaseOperationLogs(t *testing.T) {
	for _, tc := range []struct {
		name, backend, outcome, level string
		err                           error
	}{
		{
			name:    "success",
			backend: "postgres",
			outcome: "ok",
			level:   "INFO",
			err:     nil,
		},
		{
			name:    "not found",
			backend: "postgres",
			outcome: "not_found",
			level:   "INFO",
			err:     errors.New("user not found"),
		},
		{
			name:    "conflict",
			backend: "postgres",
			outcome: "rejected",
			level:   "WARN",
			err:     errors.New("username taken"),
		},
		{
			name:    "canceled",
			backend: "postgres",
			outcome: "error",
			level:   "ERROR",
			err:     context.Canceled,
		},
		{
			name:    "deadline",
			backend: "redis",
			outcome: "error",
			level:   "ERROR",
			err:     context.DeadlineExceeded,
		},
		{
			name:    "expired session",
			backend: "redis",
			outcome: "not_found",
			level:   "INFO",
			err:     errors.New("session not found"),
		},
		{
			name:    "postgres diagnostic",
			backend: "postgres",
			outcome: "error",
			level:   "ERROR",
			err: fmt.Errorf("query: %w", &pgconn.PgError{
				Code:    "23505",
				Message: "secret-password",
				Detail:  "secret-token",
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			appLogger, err := logger.NewLogger(&buffer, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			fields := map[string]any{
				"user_id": 7,
			}
			op := storage.StartOperation(logger.ContextWithRequestID(context.Background(), "request-123"), appLogger, tc.backend, "account.get_by_id", fields)
			op.Stage = "query"
			op.NotFound = tc.outcome == "not_found"
			op.Rejected = tc.outcome == "rejected"
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
			require.Equal(t, map[string]any{
				"user_id": 7,
			}, fields)
			require.NotContains(t, buffer.String(), "secret-password")
			require.NotContains(t, buffer.String(), "secret-token")

			if tc.name == "postgres diagnostic" {
				require.Equal(t, "23505", record["sqlstate"])
			}
		})
	}
}
