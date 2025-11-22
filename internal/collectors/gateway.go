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
	registry.Register(NewGatewayCollector())
}

// GatewayCollector collects metrics about gateway status.
type GatewayCollector struct {
	gatewayLossRatio     *prometheus.GaugeVec
	gatewayDelaySeconds  *prometheus.GaugeVec
	gatewayStdDevSeconds *prometheus.GaugeVec
	gatewayUp            *prometheus.GaugeVec
}

// GatewayStats represents the structure of the gateway status data returned by the API.
type GatewayStats struct {
	Name      string  `json:"name"`
	Loss      float64 `json:"loss"`
	Delay     float64 `json:"delay"`
	StdDev    float64 `json:"stddev"`
	Status    string  `json:"status"`
	Substatus string  `json:"substatus"`
	SourceIP  string  `json:"srcip"`
	MonitorIP string  `json:"monitorip"`
}

// NewGatewayCollector is the constructor
func NewGatewayCollector() *GatewayCollector {
	return &GatewayCollector{
		gatewayLossRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "gateway_loss_ratio",
				Help: "The loss ratio of the gateway as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host", "name", "srcip", "monitorip"},
		),
		gatewayDelaySeconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "gateway_delay_seconds",
				Help: "The delay of the gateway in seconds.",
			},
			[]string{"host", "name", "srcip", "monitorip"},
		),
		gatewayStdDevSeconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "gateway_stddev_seconds",
				Help: "The standard deviation of the gateway delay in seconds.",
			},
			[]string{"host", "name", "srcip", "monitorip"},
		),
		gatewayUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "gateway_up",
				Help: "The status of the gateway (0 = down, 1 = up).",
			},
			[]string{"host", "name", "srcip", "monitorip", "substatus"},
		),
	}
}

// Name returns the name of the collector.
func (c *GatewayCollector) Name() string {
	return "gateways"
}

// Describe sends the metric descriptions to the channel.
func (c *GatewayCollector) Describe(ch chan<- *prometheus.Desc) {
	c.gatewayLossRatio.Describe(ch)
	c.gatewayDelaySeconds.Describe(ch)
	c.gatewayStdDevSeconds.Describe(ch)
	c.gatewayUp.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *GatewayCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics from the target
	resp, err := utils.Request(target, "GET", "/api/v2/status/gateways")
	if err != nil {
		log.Error("gateways", "failed to fetch gateway statuses from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("gateways", "received nil response from host %s", target.Host)
		return
	}

	// Convert the data to an array of GatewayStats
	var gateways []GatewayStats
	if err := json.Unmarshal(resp.Data, &gateways); err != nil {
		log.Error("gateways", "failed to unmarshal gateways response from host %s: %s", target.Host, err.Error())
		return
	}

	// Reset metrics before collecting new data
	c.resetMetrics()

	// Extract metrics for each gateway identified
	for _, gw := range gateways {
		// Update the metrics
		c.gatewayLossRatio.WithLabelValues(target.Host, gw.Name, gw.SourceIP, gw.MonitorIP).Set(float64(gw.Loss / 100.0))        // Convert percentage to ratio
		c.gatewayDelaySeconds.WithLabelValues(target.Host, gw.Name, gw.SourceIP, gw.MonitorIP).Set(float64(gw.Delay / 1000.0))   // Convert milliseconds to seconds
		c.gatewayStdDevSeconds.WithLabelValues(target.Host, gw.Name, gw.SourceIP, gw.MonitorIP).Set(float64(gw.StdDev / 1000.0)) // Convert milliseconds to seconds
		c.gatewayUp.WithLabelValues(target.Host, gw.Name, gw.SourceIP, gw.MonitorIP, gw.Substatus).Set(float64(gatewayUpToFloat64(gw.Status)))
	}

	// Collect the metrics
	c.gatewayLossRatio.Collect(ch)
	c.gatewayDelaySeconds.Collect(ch)
	c.gatewayStdDevSeconds.Collect(ch)
	c.gatewayUp.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *GatewayCollector) resetMetrics() {
	c.gatewayLossRatio.Reset()
	c.gatewayDelaySeconds.Reset()
	c.gatewayStdDevSeconds.Reset()
	c.gatewayUp.Reset()
}

// gatewayUpToFloat64 converts the gateway status string to a float64 for Prometheus metrics.
func gatewayUpToFloat64(status string) float64 {
	if status == "online" {
		return 1.0
	}
	return 0.0
}
