package collectors

import (
	"encoding/json"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// init ensures the collector is automatically added to the registry.
func init() {
	registry.Register(NewServiceCollector())
}

// ServiceCollector collects metrics about service status.
type ServiceCollector struct {
	serviceUp      *prometheus.GaugeVec
	serviceEnabled *prometheus.GaugeVec
}

// ServiceStats represents the structure of the service status data returned by the API.
type ServiceStats struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  bool   `json:"status"`
}

// NewServiceCollector is the constructor
func NewServiceCollector() *ServiceCollector {
	return &ServiceCollector{
		serviceUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "service_up",
				Help: "Whether the service is up (1) or down (0).",
			},
			[]string{"host", "name"},
		),
		serviceEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "service_enabled",
				Help: "Whether the service is enabled (1) or disabled (0).",
			},
			[]string{"host", "name"},
		),
	}
}

// Name returns the name of the collector.
func (c *ServiceCollector) Name() string {
	return "service"
}

// Describe sends the metric descriptions to the channel.
func (c *ServiceCollector) Describe(ch chan<- *prometheus.Desc) {
	c.serviceUp.Describe(ch)
	c.serviceEnabled.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *ServiceCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics from the target
	resp, err := utils.Request(target, "GET", "/api/v2/status/services")
	if err != nil {
		log.Error("services", "failed to fetch service statuses from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("services", "received nil response from host %s", target.Host)
		return
	}

	// Convert the data to an array of ServiceStats
	var services []ServiceStats
	if err := json.Unmarshal(resp.Data, &services); err != nil {
		log.Error("services", "failed to unmarshal services response from host %s: %s", target.Host, err.Error())
		return
	}

	// Extract metrics for each service identified
	c.serviceUp.Reset()
	c.serviceEnabled.Reset()
	for _, svc := range services {
		// Update the metrics
		c.serviceUp.WithLabelValues(target.Host, svc.Name).Set(float64(utils.BoolToFloat64(svc.Status)))
		c.serviceEnabled.WithLabelValues(target.Host, svc.Name).Set(float64(utils.BoolToFloat64(svc.Enabled)))
	}

	// Collect the metrics
	c.serviceUp.Collect(ch)
	c.serviceEnabled.Collect(ch)
}
