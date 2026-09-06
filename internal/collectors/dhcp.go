package collectors

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"strings"
	"sync"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registry.Register(NewDHCPCollector())
}

// DHCPCollector collects DHCP server and lease metrics.
// It uses prometheus.Desc + MustNewConstMetric instead of shared GaugeVec
// to avoid race conditions when multiple targets are scraped concurrently.
type DHCPCollector struct {
	poolSize             *prometheus.Desc
	leasesActive         *prometheus.Desc
	leasesOnline         *prometheus.Desc
	utilization          *prometheus.Desc
	serverEnabled        *prometheus.Desc
	staticMappingsTotal  *prometheus.Desc
	staticMappingsOnline *prometheus.Desc
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
	labels := []string{"host", "interface"}
	return &DHCPCollector{
		poolSize: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_pool_size",
			"Total number of IP addresses available in the DHCP pool.",
			labels, nil,
		),
		leasesActive: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_leases_active",
			"Number of active DHCP leases within the pool range.",
			labels, nil,
		),
		leasesOnline: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_leases_online",
			"Number of online DHCP leases within the pool range.",
			labels, nil,
		),
		utilization: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_pool_utilization",
			"DHCP pool utilization ratio (0.0 to 1.0).",
			labels, nil,
		),
		serverEnabled: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_server_enabled",
			"Whether the DHCP server is enabled (1) or disabled (0) for an interface.",
			labels, nil,
		),
		staticMappingsTotal: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_static_mappings_total",
			"Total number of DHCP static mappings per interface.",
			labels, nil,
		),
		staticMappingsOnline: prometheus.NewDesc(
			registry.MetricsPrefix+"dhcp_static_mappings_online",
			"Number of online DHCP static mappings per interface.",
			labels, nil,
		),
	}
}

// Name returns the name of the collector.
func (c *DHCPCollector) Name() string {
	return "dhcp"
}

// Describe sends the metric descriptions to the channel.
func (c *DHCPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.poolSize
	ch <- c.leasesActive
	ch <- c.leasesOnline
	ch <- c.utilization
	ch <- c.serverEnabled
	ch <- c.staticMappingsTotal
	ch <- c.staticMappingsOnline
}

// CollectWithTarget fetches DHCP stats and sends them to the channel.
func (c *DHCPCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Fetch both API responses in parallel
	var (
		servers []DHCPServerConfig
		leases  []DHCPLease
		srvErr  error
		lseErr  error
		wg      sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		servers, srvErr = fetchDHCPServers(target)
	}()
	go func() {
		defer wg.Done()
		leases, lseErr = fetchDHCPLeases(target)
	}()
	wg.Wait()

	if srvErr != nil {
		log.Error("dhcp", "failed to fetch DHCP server config from host %s: %s", target.Host, srvErr.Error())
		return
	}
	if lseErr != nil {
		log.Error("dhcp", "failed to fetch DHCP leases from host %s: %s", target.Host, lseErr.Error())
		return
	}

	// Build per-server pool ranges (pre-parsed to uint32) and calculate pool sizes
	type serverInfo struct {
		config   DHCPServerConfig
		ranges   []uint32Range
		poolSize int64
	}
	serversByID := make(map[string]*serverInfo, len(servers))
	serverOrder := make([]string, 0, len(servers))
	for _, server := range servers {
		var ranges []uint32Range
		if r, ok := parseUint32Range(server.RangeFrom, server.RangeTo); ok {
			ranges = append(ranges, r)
		}
		for _, pool := range server.Pool {
			if r, ok := parseUint32Range(pool.RangeFrom, pool.RangeTo); ok {
				ranges = append(ranges, r)
			}
		}
		var poolSize int64
		for _, r := range ranges {
			poolSize += r.size()
		}
		serversByID[server.ID] = &serverInfo{config: server, ranges: ranges, poolSize: poolSize}
		serverOrder = append(serverOrder, server.ID)
	}

	// Classify each lease: match to an interface by the "if" field if set,
	// otherwise by checking which server's pool ranges contain the lease IP.
	// Ranges across servers should not overlap; if they do, the first match wins.
	activeByInterface := make(map[string]float64, len(servers))
	onlineByInterface := make(map[string]float64, len(servers))
	staticTotalByInterface := make(map[string]float64, len(servers))
	staticOnlineByInterface := make(map[string]float64, len(servers))

	for _, lease := range leases {
		leaseIP := ipToUint32(lease.IP)
		if leaseIP == 0 {
			continue
		}

		// Determine which interface this lease belongs to
		iface := lease.Interface
		var inPool bool
		if iface == "" {
			// Dynamic leases may have a null interface; match by IP to pool ranges
			for _, id := range serverOrder {
				if ipInUint32Ranges(leaseIP, serversByID[id].ranges) {
					iface = id
					inPool = true
					break
				}
			}
		} else {
			info, ok := serversByID[iface]
			if ok {
				inPool = ipInUint32Ranges(leaseIP, info.ranges)
			}
		}
		if iface == "" {
			continue
		}
		if _, ok := serversByID[iface]; !ok {
			continue
		}

		isOnline := strings.Contains(strings.ToLower(lease.OnlineStatus), "online")

		if inPool && (strings.EqualFold(lease.ActiveStatus, "active") || strings.EqualFold(lease.ActiveStatus, "static")) {
			activeByInterface[iface]++
			if isOnline {
				onlineByInterface[iface]++
			}
		} else if !inPool && strings.EqualFold(lease.ActiveStatus, "static") {
			staticTotalByInterface[iface]++
			if isOnline {
				staticOnlineByInterface[iface]++
			}
		}
	}

	// Emit metrics for each server
	for _, iface := range serverOrder {
		info := serversByID[iface]

		ch <- prometheus.MustNewConstMetric(c.serverEnabled, prometheus.GaugeValue, utils.BoolToFloat64(info.config.Enable), target.Host, iface)
		ch <- prometheus.MustNewConstMetric(c.poolSize, prometheus.GaugeValue, float64(info.poolSize), target.Host, iface)
		ch <- prometheus.MustNewConstMetric(c.leasesActive, prometheus.GaugeValue, activeByInterface[iface], target.Host, iface)
		ch <- prometheus.MustNewConstMetric(c.leasesOnline, prometheus.GaugeValue, onlineByInterface[iface], target.Host, iface)
		ch <- prometheus.MustNewConstMetric(c.staticMappingsTotal, prometheus.GaugeValue, staticTotalByInterface[iface], target.Host, iface)
		ch <- prometheus.MustNewConstMetric(c.staticMappingsOnline, prometheus.GaugeValue, staticOnlineByInterface[iface], target.Host, iface)

		if info.poolSize > 0 {
			ch <- prometheus.MustNewConstMetric(c.utilization, prometheus.GaugeValue, activeByInterface[iface]/float64(info.poolSize), target.Host, iface)
		} else {
			ch <- prometheus.MustNewConstMetric(c.utilization, prometheus.GaugeValue, 0, target.Host, iface)
		}
	}
}

// fetchDHCPServers retrieves DHCP server configurations from the target.
func fetchDHCPServers(target *utils.Target) ([]DHCPServerConfig, error) {
	resp, err := utils.Request(target, "GET", "/api/v2/services/dhcp_servers")
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, nil
	}
	var servers []DHCPServerConfig
	if err := json.Unmarshal(resp.Data, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

// fetchDHCPLeases retrieves DHCP lease data from the target.
func fetchDHCPLeases(target *utils.Target) ([]DHCPLease, error) {
	resp, err := utils.Request(target, "GET", "/api/v2/status/dhcp_server/leases")
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, nil
	}
	var leases []DHCPLease
	if err := json.Unmarshal(resp.Data, &leases); err != nil {
		return nil, err
	}
	return leases, nil
}

// uint32Range represents a pre-parsed IPv4 address range.
type uint32Range struct {
	from uint32
	to   uint32
}

// size returns the number of IPs in the range (inclusive).
func (r uint32Range) size() int64 {
	return int64(r.to-r.from) + 1
}

// parseUint32Range parses two IPv4 address strings into a uint32Range.
func parseUint32Range(from, to string) (uint32Range, bool) {
	f := ipToUint32(from)
	t := ipToUint32(to)
	if f == 0 || t == 0 || t < f {
		return uint32Range{}, false
	}
	return uint32Range{from: f, to: t}, true
}

// ipToUint32 parses an IPv4 address string to a uint32. Returns 0 on failure.
func ipToUint32(s string) uint32 {
	if s == "" {
		return 0
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return 0
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

// ipInUint32Ranges checks if a uint32 IP falls within any of the given ranges.
func ipInUint32Ranges(ip uint32, ranges []uint32Range) bool {
	for _, r := range ranges {
		if ip >= r.from && ip <= r.to {
			return true
		}
	}
	return false
}

