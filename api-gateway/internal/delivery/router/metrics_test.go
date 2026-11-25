package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"io"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
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

type metricsTestPersons struct{ personsHandler }

func (metricsTestPersons) GetPerson(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestMetricsRouteTemplates(t *testing.T) {
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
	require.NoError(t, err)
	authMiddleware := middleware.NewAuthMiddleware(nil, nil, appLogger)
	metrics := &routeMetricsRecorder{}
	handler, err := NewRouter(Handlers{
		Auth:            auth.NewAuthHandler(nil, nil, appLogger),
		Persons:         metricsTestPersons{},
		MetricsObserver: metrics,
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}, authMiddleware, nil, appLogger)
	require.NoError(t, err)

	cases := []struct {
		method string
		path   string
		route  string
		status int
	}{
		{http.MethodGet, "/api/persons/1", "/api/persons/{id}", http.StatusOK},
		{http.MethodGet, "/api/persons/2", "/api/persons/{id}", http.StatusOK},
		{http.MethodGet, "/missing", "unmatched", http.StatusNotFound},
		{http.MethodPatch, "/api/persons/1", "unmatched", http.StatusMethodNotAllowed},
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
