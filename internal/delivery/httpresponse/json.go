package httpresponse

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const InternalErrorMessage = "internal server error"

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, body any, appLogger logger.Logger) {
	data, err := json.Marshal(body)
	if err != nil {
		logError(appLogger, "encode response failed", err)
		status = http.StatusInternalServerError
		data, _ = json.Marshal(ErrorResponse{Error: InternalErrorMessage})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(append(data, '\n')); err != nil {
		logError(appLogger, "write response failed", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string, appLogger logger.Logger) {
	if status >= http.StatusInternalServerError {
		message = InternalErrorMessage
	}
	WriteJSON(w, status, ErrorResponse{Error: message}, appLogger)
}

func logError(appLogger logger.Logger, message string, err error) {
	if appLogger != nil {
		appLogger.Error(message, map[string]any{"error": err})
	}
}
