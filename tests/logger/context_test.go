package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	applogger "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestContextLoggerAddsRequestIDWithoutChangingFields(t *testing.T) {
	var output bytes.Buffer
	appLogger, err := applogger.NewLogger(&output, applogger.LogLevelDebug, "json")
	require.NoError(t, err)
	ctx := applogger.ContextWithRequestID(context.Background(), "request-42")
	fields := map[string]any{"user_id": 7, "request_id": "wrong-id"}
	for _, method := range []func(context.Context, string, map[string]any){appLogger.InfoContext, appLogger.DebugContext, appLogger.WarnContext, appLogger.ErrorContext} {
		output.Reset()
		method(ctx, "message", fields)
		var record map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &record))
		require.Equal(t, "request-42", record["request_id"])
		require.Equal(t, float64(7), record["user_id"])
		require.Equal(t, "wrong-id", fields["request_id"])
		require.Equal(t, 1, bytes.Count(output.Bytes(), []byte(`"request_id"`)))
	}
	output.Reset()
	appLogger.Info("application started", nil)
	var startup map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &startup))
	require.NotContains(t, startup, "request_id")
}
