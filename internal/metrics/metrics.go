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
