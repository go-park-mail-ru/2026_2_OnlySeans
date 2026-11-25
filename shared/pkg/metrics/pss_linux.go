package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

type pssCollector struct {
	process procfs.Proc
	desc    *prometheus.Desc
}

func registerPSSCollector(registry *prometheus.Registry) {
	process, err := procfs.Self()
	if err != nil {
		return
	}
	registry.MustRegister(&pssCollector{
		process: process,
		desc: prometheus.NewDesc(
			"process_proportional_resident_memory_bytes",
			"Proportional set size (PSS) of the process in bytes.",
			nil,
			nil,
		),
	})
}

func (c *pssCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *pssCollector) Collect(ch chan<- prometheus.Metric) {
	smaps, err := c.process.ProcSMapsRollup()
	if err != nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(smaps.Pss))
}
