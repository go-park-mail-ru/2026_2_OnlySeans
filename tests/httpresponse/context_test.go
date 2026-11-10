package httpresponse_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

type failingResponseWriter struct{ *httptest.ResponseRecorder }

func (w failingResponseWriter) Write([]byte) (int, error) { return 0, errors.New("connection closed") }

func TestResponseFailureLogsKeepRequestID(t *testing.T) {
	for _, encodeFailure := range []bool{true, false} {
		var buffer bytes.Buffer
		log, err := logger.NewLogger(&buffer, logger.LogLevelInfo, "json")
		require.NoError(t, err)
		ctx := logger.ContextWithRequestID(context.Background(), "request-123")
		if encodeFailure {
			httpresponse.WriteJSONContext(ctx, httptest.NewRecorder(), http.StatusOK, make(chan int), log)
		} else {
			httpresponse.WriteErrorContext(ctx, failingResponseWriter{httptest.NewRecorder()}, http.StatusBadRequest, "invalid body", log)
		}
		var record map[string]any
		require.NoError(t, json.Unmarshal(buffer.Bytes(), &record))
		require.Equal(t, "request-123", record["request_id"])
		require.Equal(t, "ERROR", record["level"])
		if encodeFailure {
			require.Equal(t, "encode response failed", record["msg"])
		} else {
			require.Equal(t, "write response failed", record["msg"])
		}
	}
}
