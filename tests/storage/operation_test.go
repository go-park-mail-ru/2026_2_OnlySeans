package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/metrics"
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

func TestExternalMetricsWithoutLogger(t *testing.T) {
	appMetrics := metrics.New()
	recorder := storage.NewOperationRecorder(nil, appMetrics)
	for _, tc := range []struct {
		system, operation, outcome string
		err                        error
	}{
		{"postgres", "library.get", "ok", nil},
		{"postgres", "library.set_saved", "error", fmt.Errorf("connection lost")},
		{"s3", "object.upload", "timeout", fmt.Errorf("upload: %w", context.DeadlineExceeded)},
		{"s3", "object.delete", "canceled", context.Canceled},
		{"redis", "session.get", "not_found", domain.ErrSessionNotFound},
		{"postgres", "account.create", "rejected", domain.ErrUserExists},
	} {
		recorder.Start(t.Context(), tc.system, tc.operation, map[string]any{"user_id": 7}).Finish(tc.err)

		response := httptest.NewRecorder()
		appMetrics.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		require.Equal(t, http.StatusOK, response.Code)
		require.Contains(t, response.Body.String(), fmt.Sprintf(
			`external_operations_total{operation=%q,outcome=%q,system=%q} 1`, tc.operation, tc.outcome, tc.system,
		))
		require.Contains(t, response.Body.String(), fmt.Sprintf(
			`external_operation_duration_seconds_count{operation=%q,system=%q} 1`, tc.operation, tc.system,
		))
		require.NotContains(t, response.Body.String(), "user_id=")
	}
}
