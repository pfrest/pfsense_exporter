package collectors

import (
	"encoding/json"
	"fmt"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// init ensures the collector is automatically added to the registry.
func init() {
	registry.Register(NewCARPCollector())
}

// CARPCollector collects metrics about CARP status.
type CARPCollector struct {
	carpEnabled                *prometheus.GaugeVec
	carpMaintenanceModeEnabled *prometheus.GaugeVec
	carpVirtualIPStatus        *prometheus.GaugeVec
}

// CARPStats represents the structure of the system's CARP status data returned by the API.
type CARPStats struct {
	Enabled         bool `json:"enable"`
	MaintenanceMode bool `json:"maintenance_mode"`
}

// CARPVirtualIPStatus represents the structure of a virtual IP's CARP status data returned by the API.
type CARPVirtualIPStatus struct {
	CARPStatus string `json:"carp_status"`
	UniqID     string `json:"uniqid"`
	Subnet     string `json:"subnet"`
	VHID       int64  `json:"vhid"`
	Interface  string `json:"interface"`
}

// NewCARPCollector is the constructor
func NewCARPCollector() *CARPCollector {
	return &CARPCollector{
		carpEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "carp_enabled",
				Help: "Whether CARP is enabled (1 = enabled, 0 = disabled).",
			},
			[]string{"host"},
		),
		carpMaintenanceModeEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "carp_maintenance_mode_enabled",
				Help: "Whether CARP maintenance mode is enabled (1 = enabled, 0 = disabled).",
			},
			[]string{"host"},
		),
		carpVirtualIPStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "carp_virtual_ip_status",
				Help: "CARP virtual IP status (1 = MASTER, 0 = BACKUP, -1 = OTHER).",
			},
			[]string{"host", "carp_status", "uniqid", "subnet", "vhid", "interface"},
		),
	}
}

// Name returns the name of the collector.
func (c *CARPCollector) Name() string {
	return "carp"
}

// Describe sends the metric descriptions to the channel.
func (c *CARPCollector) Describe(ch chan<- *prometheus.Desc) {
	c.carpEnabled.Describe(ch)
	c.carpMaintenanceModeEnabled.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *CARPCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect system CARP metrics from the target
	resp, err := utils.Request(target, "GET", "/api/v2/status/carp")
	if err != nil {
		log.Error("carp", "failed to fetch carp status from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("carp", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a CARPStats struct
	var stats CARPStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("carp", "failed to unmarshal carp response from host %s: %s", target.Host, err.Error())
		return
	}

	// Reset old metrics before collecting new data
	c.resetMetrics()

	// Update the system CARP metrics
	c.carpEnabled.WithLabelValues(target.Host).Set(float64(utils.BoolToFloat64(stats.Enabled)))
	c.carpMaintenanceModeEnabled.WithLabelValues(target.Host).Set(float64(utils.BoolToFloat64(stats.MaintenanceMode)))

	// Collect virtual IP CARP metrics from the target
	resp, err = utils.Request(target, "GET", "/api/v2/firewall/virtual_ips?mode=carp")
	if err != nil {
		log.Error("carp", "failed to fetch virtual IP status from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("carp", "received nil response for virtual IPs from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a slice of CARPVirtualIPStatus structs
	var virtualIPs []CARPVirtualIPStatus
	if err := json.Unmarshal(resp.Data, &virtualIPs); err != nil {
		log.Error("carp", "failed to unmarshal virtual IP response from host %s: %s", target.Host, err.Error())
		return
	}

	// Extract metrics for each virtual IP identified
	for _, ip := range virtualIPs {
		c.carpVirtualIPStatus.WithLabelValues(
			target.Host,
			ip.CARPStatus,
			ip.UniqID,
			ip.Subnet,
			fmt.Sprintf("%d", ip.VHID), // Convert VHID from int64 to string
			ip.Interface,
		).Set(CARPStatusToFloat64(ip.CARPStatus))
	}

	// Collect the metrics
	c.carpEnabled.Collect(ch)
	c.carpMaintenanceModeEnabled.Collect(ch)
	c.carpVirtualIPStatus.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *CARPCollector) resetMetrics() {
	c.carpEnabled.Reset()
	c.carpMaintenanceModeEnabled.Reset()
	c.carpVirtualIPStatus.Reset()
}

// CARPStatusToFloat64 converts a CARP status string to a float64 value.
func CARPStatusToFloat64(status string) float64 {
	switch status {
	case "master":
		return 1
	case "backup":
		return 0
	default:
		return -1
	}
}
