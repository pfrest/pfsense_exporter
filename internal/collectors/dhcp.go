package collectors

import (
	"encoding/binary"
	"encoding/json"
	"net"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registry.Register(NewDHCPCollector())
}

// DHCPCollector collects DHCP server and lease metrics.
type DHCPCollector struct {
	poolSize     *prometheus.GaugeVec
	leasesActive *prometheus.GaugeVec
	leasesOnline *prometheus.GaugeVec
	utilization  *prometheus.GaugeVec
	serverUp     *prometheus.GaugeVec
}

// DHCPServerConfig represents a DHCP server configuration per interface.
type DHCPServerConfig struct {
	ID        string                `json:"id"`
	Enable    bool                  `json:"enable"`
	RangeFrom string                `json:"range_from"`
	RangeTo   string                `json:"range_to"`
	Pool      []DHCPServerPoolEntry `json:"pool"`
}

// DHCPServerPoolEntry represents an additional address pool within a DHCP server.
type DHCPServerPoolEntry struct {
	RangeFrom string `json:"range_from"`
	RangeTo   string `json:"range_to"`
}

// DHCPLease represents a single DHCP lease from the leases endpoint.
type DHCPLease struct {
	IP           string `json:"ip"`
	MAC          string `json:"mac"`
	Hostname     string `json:"hostname"`
	Interface    string `json:"if"`
	Starts       string `json:"starts"`
	Ends         string `json:"ends"`
	ActiveStatus string `json:"active_status"`
	OnlineStatus string `json:"online_status"`
	Description  string `json:"descr"`
}

// NewDHCPCollector is the constructor.
func NewDHCPCollector() *DHCPCollector {
	return &DHCPCollector{
		poolSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_pool_size",
				Help: "Total number of IP addresses available in the DHCP pool.",
			},
			[]string{"host", "interface"},
		),
		leasesActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_leases_active",
				Help: "Number of active DHCP leases.",
			},
			[]string{"host", "interface"},
		),
		leasesOnline: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_leases_online",
				Help: "Number of online DHCP leases.",
			},
			[]string{"host", "interface"},
		),
		utilization: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_pool_utilization",
				Help: "DHCP pool utilization ratio (0.0 to 1.0).",
			},
			[]string{"host", "interface"},
		),
		serverUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_server_enabled",
				Help: "Whether the DHCP server is enabled (1) or disabled (0) for an interface.",
			},
			[]string{"host", "interface"},
		),
	}
}

// Name returns the name of the collector.
func (c *DHCPCollector) Name() string {
	return "dhcp"
}

// Describe sends the metric descriptions to the channel.
func (c *DHCPCollector) Describe(ch chan<- *prometheus.Desc) {
	c.poolSize.Describe(ch)
	c.leasesActive.Describe(ch)
	c.leasesOnline.Describe(ch)
	c.utilization.Describe(ch)
	c.serverUp.Describe(ch)
}

// CollectWithTarget fetches DHCP stats and sends them to the channel.
func (c *DHCPCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Fetch DHCP server configurations
	serverResp, err := utils.Request(target, "GET", "/api/v2/services/dhcp_servers")
	if err != nil {
		log.Error("dhcp", "failed to fetch DHCP server config from host %s: %s", target.Host, err.Error())
		return
	}
	if serverResp == nil || serverResp.Data == nil {
		log.Error("dhcp", "received nil DHCP server config response from host %s", target.Host)
		return
	}

	var servers []DHCPServerConfig
	if err := json.Unmarshal(serverResp.Data, &servers); err != nil {
		log.Error("dhcp", "failed to unmarshal DHCP server config from host %s: %s", target.Host, err.Error())
		return
	}

	// Fetch DHCP leases
	leaseResp, err := utils.Request(target, "GET", "/api/v2/status/dhcp_server/leases")
	if err != nil {
		log.Error("dhcp", "failed to fetch DHCP leases from host %s: %s", target.Host, err.Error())
		return
	}
	if leaseResp == nil || leaseResp.Data == nil {
		log.Error("dhcp", "received nil DHCP leases response from host %s", target.Host)
		return
	}

	var leases []DHCPLease
	if err := json.Unmarshal(leaseResp.Data, &leases); err != nil {
		log.Error("dhcp", "failed to unmarshal DHCP leases from host %s: %s", target.Host, err.Error())
		return
	}

	// Reset metrics before collecting new data
	c.resetMetrics()

	// Count active and online leases per interface
	activeByInterface := make(map[string]float64)
	onlineByInterface := make(map[string]float64)
	for _, lease := range leases {
		if lease.ActiveStatus == "active" {
			activeByInterface[lease.Interface]++
		}
		if lease.OnlineStatus == "online" {
			onlineByInterface[lease.Interface]++
		}
	}

	// Process each DHCP server (one per interface)
	for _, server := range servers {
		iface := server.ID

		// Server enabled status
		c.serverUp.WithLabelValues(target.Host, iface).Set(utils.BoolToFloat64(server.Enable))

		// Calculate total pool size (primary range + additional pools)
		totalPoolSize := ipRangeSize(server.RangeFrom, server.RangeTo)
		for _, pool := range server.Pool {
			totalPoolSize += ipRangeSize(pool.RangeFrom, pool.RangeTo)
		}

		c.poolSize.WithLabelValues(target.Host, iface).Set(float64(totalPoolSize))

		// Active and online lease counts
		active := activeByInterface[iface]
		online := onlineByInterface[iface]
		c.leasesActive.WithLabelValues(target.Host, iface).Set(active)
		c.leasesOnline.WithLabelValues(target.Host, iface).Set(online)

		// Utilization ratio
		if totalPoolSize > 0 {
			c.utilization.WithLabelValues(target.Host, iface).Set(active / float64(totalPoolSize))
		} else {
			c.utilization.WithLabelValues(target.Host, iface).Set(0)
		}
	}

	// Collect all metrics
	c.poolSize.Collect(ch)
	c.leasesActive.Collect(ch)
	c.leasesOnline.Collect(ch)
	c.utilization.Collect(ch)
	c.serverUp.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *DHCPCollector) resetMetrics() {
	c.poolSize.Reset()
	c.leasesActive.Reset()
	c.leasesOnline.Reset()
	c.utilization.Reset()
	c.serverUp.Reset()
}

// ipRangeSize calculates the number of usable IPs in a range (inclusive).
func ipRangeSize(from, to string) int64 {
	if from == "" || to == "" {
		return 0
	}

	fromIP := net.ParseIP(from).To4()
	toIP := net.ParseIP(to).To4()
	if fromIP == nil || toIP == nil {
		return 0
	}

	fromInt := int64(binary.BigEndian.Uint32(fromIP))
	toInt := int64(binary.BigEndian.Uint32(toIP))

	if toInt < fromInt {
		return 0
	}

	return toInt - fromInt + 1
}
