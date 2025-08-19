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
	registry.Register(NewLoginProtectionCollector())
}

// LoginProtectionCollector collects metrics about the SSHGuard login protection.
type LoginProtectionCollector struct {
	loginProtectionBlockedIP      *prometheus.GaugeVec
	loginProtectionBlockedIPCount *prometheus.GaugeVec
}

// LoginProtectionStats represents the structure of the login protection status data.
type LoginProtectionStats struct {
	Id      string   `json:"id"`
	Entries []string `json:"entries"`
}

// NewLoginProtectionCollector is the constructor
func NewLoginProtectionCollector() *LoginProtectionCollector {
	return &LoginProtectionCollector{
		loginProtectionBlockedIP: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "login_protection_blocked_ip",
				Help: "Contains details about IPs blocked by Login Protection.",
			},
			[]string{"host", "ip"},
		),
		loginProtectionBlockedIPCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "login_protection_blocked_ip_count",
				Help: "Current number of IPs actively blocked by Login Protection.",
			},
			[]string{"host"},
		),
	}
}

// Name returns the name of the collector.
func (c *LoginProtectionCollector) Name() string {
	return "login_protection"
}

// Describe sends the metric descriptions to the channel.
func (c *LoginProtectionCollector) Describe(ch chan<- *prometheus.Desc) {
	c.loginProtectionBlockedIP.Describe(ch)
	c.loginProtectionBlockedIPCount.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *LoginProtectionCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics for each target
	resp, err := utils.Request(target, "GET", "/api/v2/diagnostics/table?id=sshguard")
	if err != nil {
		log.Error(target.Host, "failed to fetch Login Protection's sshguard table: "+err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("login_protection", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a LoginProtectionStats struct
	var stats LoginProtectionStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("login_protection", "failed to unmarshal Login Protection response from host %s: %s", target.Host, err.Error())
		return
	}

	// Update the metrics
	c.loginProtectionBlockedIP.Reset()
	for _, entry := range stats.Entries {
		c.loginProtectionBlockedIP.WithLabelValues(target.Host, entry).Set(1)
	}
	c.loginProtectionBlockedIPCount.WithLabelValues(target.Host).Set(float64(len(stats.Entries)))

	// Collect the metrics
	c.loginProtectionBlockedIP.Collect(ch)
	c.loginProtectionBlockedIPCount.Collect(ch)
}
