package collectors

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"strings"

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
	poolSize             *prometheus.GaugeVec
	leasesActive         *prometheus.GaugeVec
	leasesOnline         *prometheus.GaugeVec
	utilization          *prometheus.GaugeVec
	serverUp             *prometheus.GaugeVec
	staticMappingsTotal  *prometheus.GaugeVec
	staticMappingsOnline *prometheus.GaugeVec
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
		staticMappingsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_static_mappings_total",
				Help: "Total number of DHCP static mappings per interface.",
			},
			[]string{"host", "interface"},
		),
		staticMappingsOnline: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "dhcp_static_mappings_online",
				Help: "Number of online DHCP static mappings per interface.",
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
	c.staticMappingsTotal.Describe(ch)
	c.staticMappingsOnline.Describe(ch)
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

	// Build a lookup of leases by interface
	leasesByInterface := make(map[string][]DHCPLease)
	for _, lease := range leases {
		leasesByInterface[lease.Interface] = append(leasesByInterface[lease.Interface], lease)
	}

	// Process each DHCP server (one per interface)
	for _, server := range servers {
		iface := server.ID

		// Server enabled status
		c.serverUp.WithLabelValues(target.Host, iface).Set(utils.BoolToFloat64(server.Enable))

		// Collect all pool ranges for this server (primary + additional)
		ranges := []ipRange{{server.RangeFrom, server.RangeTo}}
		for _, pool := range server.Pool {
			ranges = append(ranges, ipRange{pool.RangeFrom, pool.RangeTo})
		}

		// Calculate total pool size
		var totalPoolSize int64
		for _, r := range ranges {
			totalPoolSize += ipRangeSize(r.from, r.to)
		}
		c.poolSize.WithLabelValues(target.Host, iface).Set(float64(totalPoolSize))

		// Count leases that fall within the pool ranges
		// active_status can be "active", "static", "expired", etc.
		// online_status can be "active/online", "idle/offline", etc.
		var active, online float64
		for _, lease := range leasesByInterface[iface] {
			if !ipInRanges(lease.IP, ranges) {
				continue
			}
			status := strings.ToLower(lease.ActiveStatus)
			if status == "active" || status == "static" {
				active++
			}
			if strings.Contains(strings.ToLower(lease.OnlineStatus), "online") {
				online++
			}
		}
		c.leasesActive.WithLabelValues(target.Host, iface).Set(active)
		c.leasesOnline.WithLabelValues(target.Host, iface).Set(online)

		// Count static mappings (leases outside pool ranges with "static" status)
		var staticTotal, staticOnline float64
		for _, lease := range leasesByInterface[iface] {
			if strings.ToLower(lease.ActiveStatus) != "static" {
				continue
			}
			if ipInRanges(lease.IP, ranges) {
				continue
			}
			staticTotal++
			if strings.Contains(strings.ToLower(lease.OnlineStatus), "online") {
				staticOnline++
			}
		}
		c.staticMappingsTotal.WithLabelValues(target.Host, iface).Set(staticTotal)
		c.staticMappingsOnline.WithLabelValues(target.Host, iface).Set(staticOnline)

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
	c.staticMappingsTotal.Collect(ch)
	c.staticMappingsOnline.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *DHCPCollector) resetMetrics() {
	c.poolSize.Reset()
	c.leasesActive.Reset()
	c.leasesOnline.Reset()
	c.utilization.Reset()
	c.serverUp.Reset()
	c.staticMappingsTotal.Reset()
	c.staticMappingsOnline.Reset()
}

// ipRange represents a start and end IP address pair.
type ipRange struct {
	from string
	to   string
}

// ipInRanges checks if an IP address falls within any of the given ranges.
func ipInRanges(ip string, ranges []ipRange) bool {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return false
	}
	ipInt := int64(binary.BigEndian.Uint32(parsed))

	for _, r := range ranges {
		fromIP := net.ParseIP(r.from).To4()
		toIP := net.ParseIP(r.to).To4()
		if fromIP == nil || toIP == nil {
			continue
		}
		fromInt := int64(binary.BigEndian.Uint32(fromIP))
		toInt := int64(binary.BigEndian.Uint32(toIP))
		if ipInt >= fromInt && ipInt <= toInt {
			return true
		}
	}
	return false
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
