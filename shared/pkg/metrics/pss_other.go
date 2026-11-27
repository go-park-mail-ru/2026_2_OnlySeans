//go:build !linux

package metrics

import "github.com/prometheus/client_golang/prometheus"

func registerPSSCollector(_ *prometheus.Registry) {}
