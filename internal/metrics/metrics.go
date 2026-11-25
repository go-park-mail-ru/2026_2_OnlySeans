package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry         *prometheus.Registry
	requestsTotal    *prometheus.CounterVec
	requestsDuration *prometheus.HistogramVec
	externalDuration *prometheus.HistogramVec
	externalTotal    *prometheus.CounterVec
	businessTotal    *prometheus.CounterVec
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "HTTP_requests_total",
				Help: "Total number of completed HTTP reqs",
			},
			[]string{"method", "route", "status"},
		),
		requestsDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "HTTP_requests_duration",
				Help: "HTTP requests duration in seconds",
				Buckets: []float64{
					0.005, 0.01, 0.025, 0.05, 0.1,
					0.25, 0.5, 1, 2.5, 5, 10,
				},
			},
			[]string{"method", "route"},
		),
	}

	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requestsTotal,
		m.requestsDuration,
	)

	m.externalDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "external_operation_duration_seconds",
			Help: "Duration of external operations as observed by the backend.",
		},
		[]string{"system", "operation"},
	)

	m.externalTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_operations_total",
			Help: "Completed external operations by outcome.",
		},
		[]string{"system", "operation", "outcome"},
	)

	m.businessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_operations_total",
			Help: "Successfully completed business operations.",
		},
		[]string{"operation"},
	)

	m.registry.MustRegister(m.externalDuration, m.externalTotal, m.businessTotal)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{},
	)
}

func (m *Metrics) ObserveHTTP(
	method string,
	route string,
	status int,
	duration time.Duration,
) {
	m.requestsTotal.WithLabelValues(
		method,
		route,
		strconv.Itoa(status),
	).Inc()

	m.requestsDuration.WithLabelValues(
		method,
		route,
	).Observe(duration.Seconds())
}

func (m *Metrics) ObserveExternal(
	system, operation, outcome string,
	duration time.Duration,
) {
	m.externalDuration.WithLabelValues(system, operation).Observe(duration.Seconds())
	m.externalTotal.WithLabelValues(system, operation, outcome).Inc()
}

func (m *Metrics) RecordBusinessSuccess(operation string) {
	m.businessTotal.WithLabelValues(operation).Inc()
}
