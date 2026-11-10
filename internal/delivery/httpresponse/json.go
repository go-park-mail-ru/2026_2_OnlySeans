package httpresponse

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const InternalErrorMessage = "internal server error"

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, body any, appLogger logger.Logger) {
	WriteJSONContext(context.Background(), w, status, body, appLogger)
}

func WriteJSONContext(ctx context.Context, w http.ResponseWriter, status int, body any, appLogger logger.Logger) {
	data, err := json.Marshal(body)
	if err != nil {
		logError(ctx, appLogger, "encode response failed", err)
		status = http.StatusInternalServerError
		data, _ = json.Marshal(ErrorResponse{Error: InternalErrorMessage})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(append(data, '\n')); err != nil {
		logError(ctx, appLogger, "write response failed", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string, appLogger logger.Logger) {
	WriteErrorContext(context.Background(), w, status, message, appLogger)
}

func WriteErrorContext(ctx context.Context, w http.ResponseWriter, status int, message string, appLogger logger.Logger) {
	if status >= http.StatusInternalServerError {
		message = InternalErrorMessage
	}
	WriteJSONContext(ctx, w, status, ErrorResponse{Error: message}, appLogger)
}

func logError(ctx context.Context, appLogger logger.Logger, message string, err error) {
	if appLogger != nil {
		appLogger.ErrorContext(ctx, message, map[string]any{"error": err})
	}
}
