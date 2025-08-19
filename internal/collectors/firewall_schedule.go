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
	registry.Register(NewFirewallScheduleCollector())
}

// FirewallScheduleCollector collects metrics about firewall schedule status.
type FirewallScheduleCollector struct {
	firewallScheduleActive *prometheus.GaugeVec
}

// FirewallScheduleStats represents the structure of the firewall schedule status data returned by the API.
type FirewallScheduleStats struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// NewFirewallScheduleCollector is the constructor
func NewFirewallScheduleCollector() *FirewallScheduleCollector {
	return &FirewallScheduleCollector{
		firewallScheduleActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "firewall_schedule_active",
				Help: "Whether the firewall schedule is active (1) or inactive (0).",
			},
			[]string{"host", "name"},
		),
	}
}

// Name returns the name of the collector.
func (c *FirewallScheduleCollector) Name() string {
	return "firewall_schedule"
}

// Describe sends the metric descriptions to the channel.
func (c *FirewallScheduleCollector) Describe(ch chan<- *prometheus.Desc) {
	c.firewallScheduleActive.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *FirewallScheduleCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics for each target
	resp, err := utils.Request(target, "GET", "/api/v2/firewall/schedules")
	if err != nil {
		log.Error("firewall_schedules", "failed to fetch firewall schedule status from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("firewall_schedules", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a FirewallScheduleStats struct
	var stats []FirewallScheduleStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("firewall_schedules", "failed to unmarshal firewall schedule response from host %s: %s", target.Host, err.Error())
		return
	}

	// Update the metrics
	c.firewallScheduleActive.Reset()
	for _, schedule := range stats {
		c.firewallScheduleActive.WithLabelValues(target.Host, schedule.Name).Set(float64(utils.BoolToFloat64(schedule.Active)))
	}

	// Collect the metrics
	c.firewallScheduleActive.Collect(ch)
}
