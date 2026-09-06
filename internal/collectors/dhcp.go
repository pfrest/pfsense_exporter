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

	// Build per-server pool ranges and calculate pool sizes
	type serverInfo struct {
		config DHCPServerConfig
		ranges []ipRange
		poolSize int64
	}
	serversByID := make(map[string]*serverInfo)
	var serverOrder []string
	for _, server := range servers {
		ranges := []ipRange{{server.RangeFrom, server.RangeTo}}
		for _, pool := range server.Pool {
			ranges = append(ranges, ipRange{pool.RangeFrom, pool.RangeTo})
		}
		var poolSize int64
		for _, r := range ranges {
			poolSize += ipRangeSize(r.from, r.to)
		}
		serversByID[server.ID] = &serverInfo{config: server, ranges: ranges, poolSize: poolSize}
		serverOrder = append(serverOrder, server.ID)
	}

	// Classify each lease: match to an interface by the "if" field if set,
	// otherwise by checking which server's pool ranges contain the lease IP.
	// active_status can be "active", "static", "expired", etc.
	// online_status can be "active/online", "idle/offline", etc.
	activeByInterface := make(map[string]float64)
	onlineByInterface := make(map[string]float64)
	staticTotalByInterface := make(map[string]float64)
	staticOnlineByInterface := make(map[string]float64)

	for _, lease := range leases {
		// Determine which interface this lease belongs to
		iface := lease.Interface
		if iface == "" {
			// Dynamic leases may have a null interface; match by IP to pool ranges
			for id, info := range serversByID {
				if ipInRanges(lease.IP, info.ranges) {
					iface = id
					break
				}
			}
		}
		if iface == "" {
			continue
		}

		info, ok := serversByID[iface]
		if !ok {
			continue
		}

		inPool := ipInRanges(lease.IP, info.ranges)
		status := strings.ToLower(lease.ActiveStatus)
		isOnline := strings.Contains(strings.ToLower(lease.OnlineStatus), "online")

		if inPool && (status == "active" || status == "static") {
			activeByInterface[iface]++
			if isOnline {
				onlineByInterface[iface]++
			}
		} else if !inPool && status == "static" {
			staticTotalByInterface[iface]++
			if isOnline {
				staticOnlineByInterface[iface]++
			}
		}
	}

	// Emit metrics for each server
	for _, iface := range serverOrder {
		info := serversByID[iface]

		c.serverUp.WithLabelValues(target.Host, iface).Set(utils.BoolToFloat64(info.config.Enable))
		c.poolSize.WithLabelValues(target.Host, iface).Set(float64(info.poolSize))
		c.leasesActive.WithLabelValues(target.Host, iface).Set(activeByInterface[iface])
		c.leasesOnline.WithLabelValues(target.Host, iface).Set(onlineByInterface[iface])
		c.staticMappingsTotal.WithLabelValues(target.Host, iface).Set(staticTotalByInterface[iface])
		c.staticMappingsOnline.WithLabelValues(target.Host, iface).Set(staticOnlineByInterface[iface])

		if info.poolSize > 0 {
			c.utilization.WithLabelValues(target.Host, iface).Set(activeByInterface[iface] / float64(info.poolSize))
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
