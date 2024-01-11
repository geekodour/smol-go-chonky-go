package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func NewPrometheusRegistry() (*prometheus.Registry, error) {
	prometheusRegistry := prometheus.NewRegistry()
	err := prometheusRegistry.Register(collectors.NewGoCollector())
	if err != nil {
		return nil, err
	}

	err = prometheusRegistry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	if err != nil {
		return nil, err
	}

	return prometheusRegistry, nil
}
