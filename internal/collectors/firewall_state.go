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
	registry.Register(NewFirewallStatesCollector())
}

// FirewallStatesCollector collects metrics about firewall states.
type FirewallStatesCollector struct {
	firewallStatesMaximumCount *prometheus.GaugeVec
	firewallStatesCurrentCount *prometheus.GaugeVec
	firewallStatesUsageRatio   *prometheus.GaugeVec
}

// FirewallStatesStats represents the structure of the firewall states status data returned by the API.
type FirewallStatesStats struct {
	MaximumStates        float64 `json:"maximumstates"`
	DefaultMaximumStates float64 `json:"defaultmaximumstates"`
	CurrentStates        float64 `json:"currentstates"`
}

// NewFirewallStatesCollector is the constructor
func NewFirewallStatesCollector() *FirewallStatesCollector {
	return &FirewallStatesCollector{
		firewallStatesMaximumCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "firewall_states_maximum_count",
				Help: "Maximum number of firewall states allowed by the host.",
			},
			[]string{"host"},
		),
		firewallStatesCurrentCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "firewall_states_current_count",
				Help: "Current number of firewall states registered on the host.",
			},
			[]string{"host"},
		),
		firewallStatesUsageRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "firewall_states_usage_ratio",
				Help: "Ratio of firewall states currently in use as a decimal percentage (0.0-1.0).",
			},
			[]string{"host"},
		),
	}
}

// Name returns the name of the collector.
func (c *FirewallStatesCollector) Name() string {
	return "firewall_states"
}

// Describe sends the metric descriptions to the channel.
func (c *FirewallStatesCollector) Describe(ch chan<- *prometheus.Desc) {
	c.firewallStatesMaximumCount.Describe(ch)
	c.firewallStatesCurrentCount.Describe(ch)
	c.firewallStatesUsageRatio.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *FirewallStatesCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics for each target
	resp, err := utils.Request(target, "GET", "/api/v2/firewall/states/size")
	if err != nil {
		log.Error(target.Host, "failed to fetch firewall states: "+err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("firewall_states", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a FirewallStatesStats struct
	var stats FirewallStatesStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("firewall_states", "failed to unmarshal firewall states response from host %s: %s", target.Host, err.Error())
		return
	}

	// Ensure a maximum state value is always present
	if stats.MaximumStates == 0 {
		stats.MaximumStates = stats.DefaultMaximumStates
	}

	// Update the metrics
	c.firewallStatesMaximumCount.WithLabelValues(target.Host).Set(float64(stats.MaximumStates))
	c.firewallStatesCurrentCount.WithLabelValues(target.Host).Set(float64(stats.CurrentStates))
	c.firewallStatesUsageRatio.WithLabelValues(target.Host).Set(float64(stats.CurrentStates) / float64(stats.MaximumStates))

	// Collect the metrics
	c.firewallStatesCurrentCount.Collect(ch)
	c.firewallStatesMaximumCount.Collect(ch)
	c.firewallStatesUsageRatio.Collect(ch)
}
