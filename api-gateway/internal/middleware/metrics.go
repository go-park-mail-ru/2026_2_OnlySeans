package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type HTTPMetrics interface {
	ObserveHTTP(
		method string,
		route string,
		status int,
		duration time.Duration,
	)
}

type metricsRouteKey struct{}

type metricsRoute struct {
	template string
}

func WithMetrics(next http.Handler, metrics HTTPMetrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		started := time.Now()
		writer := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		routeInfo := &metricsRoute{template: "unmatched"}
		r = r.WithContext(context.WithValue(r.Context(), metricsRouteKey{}, routeInfo))

		next.ServeHTTP(writer, r)

		metrics.ObserveHTTP(
			metricMethod(r.Method),
			routeInfo.template,
			writer.status,
			time.Since(started),
		)
	})
}

func CaptureMetricsRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, ok := r.Context().Value(metricsRouteKey{}).(*metricsRoute)
		if ok {
			if route := mux.CurrentRoute(r); route != nil {
				if template, err := route.GetPathTemplate(); err == nil {
					info.template = template
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func metricMethod(method string) string {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}
