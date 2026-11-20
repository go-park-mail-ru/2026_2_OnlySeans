package search

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func RegisterRoutes(r *mux.Router, handler *SearchHandler, appLogger logger.Logger) {
	requestError := func(w http.ResponseWriter, req *http.Request, err error) {
		field := "query"
		var missing *RequiredParamError
		var invalid *InvalidParamFormatError

		switch {
		case errors.As(err, &missing):
			field = missing.ParamName
		case errors.As(err, &invalid):
			field = invalid.ParamName
		}

		message := "invalid request parameters"

		switch field {
		case "q":
			message = domain.ErrInvalidSearchQuery.Error()
		case "limit":
			message = domain.ErrInvalidSearchLimit.Error()
		}

		appLogger.WarnContext(req.Context(), "search request parameters rejected", map[string]any{"field": field})
		httpresponse.WriteErrorContext(req.Context(), w, http.StatusBadRequest, message, appLogger)
	}
	server := NewStrictHandlerWithOptions(handler, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc: requestError,
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
			appLogger.ErrorContext(req.Context(), "search response failed", map[string]any{"error": err})
			// A failed network write must not append another response to JSON already sent.
			if state, ok := w.(*searchResponseWriter); ok && state.written {
				return
			}
			httpresponse.WriteErrorContext(req.Context(), w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
		},
	})
	HandlerWithOptions(server, GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: requestError,
		Middlewares: []MiddlewareFunc{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					next.ServeHTTP(&searchResponseWriter{ResponseWriter: w}, req)
				})
			},
		},
	})
}

type searchResponseWriter struct {
	http.ResponseWriter
	written bool
}

func (w *searchResponseWriter) WriteHeader(status int) {
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *searchResponseWriter) Write(data []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(data)
}

func (w *searchResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
