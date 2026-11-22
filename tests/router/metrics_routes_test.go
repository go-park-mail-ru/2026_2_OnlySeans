package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/stretchr/testify/require"
)

type httpMetricObservation struct {
	method string
	route  string
	status int
}

type routeMetricsRecorder struct {
	observations []httpMetricObservation
}

func (m *routeMetricsRecorder) ObserveHTTP(method, route string, status int, duration time.Duration) {
	m.observations = append(m.observations, httpMetricObservation{method, route, status})
}

func TestMetricsRouteTemplates(t *testing.T) {
	metrics := &routeMetricsRecorder{}
	handler, err := router.NewRouter(router.Handlers{
		MetricsObserver: metrics,
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}, []string{testOrigin}, newSpyLogger(t))
	require.NoError(t, err)

	cases := []struct {
		method string
		path   string
		route  string
		status int
	}{
		{http.MethodPut, "/api/films/1", "/api/films/{id}", http.StatusUnauthorized},
		{http.MethodPut, "/api/films/2", "/api/films/{id}", http.StatusUnauthorized},
		{http.MethodGet, "/missing", "unmatched", http.StatusNotFound},
		{http.MethodPatch, "/api/films/1", "unmatched", http.StatusMethodNotAllowed},
	}

	for _, tc := range cases {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			before := len(metrics.observations)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			require.Equal(t, tc.status, response.Code)
			require.Len(t, metrics.observations, before+1)
			require.Equal(t, httpMetricObservation{tc.method, tc.route, tc.status}, metrics.observations[before])
		})
	}

	before := len(metrics.observations)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Len(t, metrics.observations, before)
}
