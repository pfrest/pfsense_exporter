package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewDHCPCollector(t *testing.T) {
	collector := NewDHCPCollector()

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}
	if collector.poolSize == nil {
		t.Error("Expected poolSize desc to be initialized")
	}
	if collector.leasesActive == nil {
		t.Error("Expected leasesActive desc to be initialized")
	}
	if collector.leasesOnline == nil {
		t.Error("Expected leasesOnline desc to be initialized")
	}
	if collector.utilization == nil {
		t.Error("Expected utilization desc to be initialized")
	}
	if collector.serverEnabled == nil {
		t.Error("Expected serverEnabled desc to be initialized")
	}
	if collector.staticMappingsTotal == nil {
		t.Error("Expected staticMappingsTotal desc to be initialized")
	}
	if collector.staticMappingsOnline == nil {
		t.Error("Expected staticMappingsOnline desc to be initialized")
	}
}

func TestDHCPCollectorName(t *testing.T) {
	collector := NewDHCPCollector()

	if collector.Name() != "dhcp" {
		t.Errorf("Expected name 'dhcp', got %s", collector.Name())
	}
}

func TestDHCPCollectorDescribe(t *testing.T) {
	collector := NewDHCPCollector()

	ch := make(chan *prometheus.Desc, 20)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	if count != 7 {
		t.Errorf("Expected 7 metric descriptions, got %d", count)
	}
}

func TestDHCPCollectorCollectWithTarget(t *testing.T) {
	serverResponse := []DHCPServerConfig{
		{
			ID:        "lan",
			Enable:    true,
			RangeFrom: "192.168.1.100",
			RangeTo:   "192.168.1.200",
			Pool: []DHCPServerPoolEntry{
				{RangeFrom: "192.168.1.210", RangeTo: "192.168.1.220"},
			},
		},
	}

	leaseResponse := []DHCPLease{
		// In-pool static lease, online
		{IP: "192.168.1.100", Interface: "lan", ActiveStatus: "static", OnlineStatus: "active/online"},
		// In-pool dynamic lease (null interface), online
		{IP: "192.168.1.150", Interface: "", ActiveStatus: "active", OnlineStatus: "active/online"},
		// In-pool expired lease (should not count as active)
		{IP: "192.168.1.102", Interface: "", ActiveStatus: "expired", OnlineStatus: "idle/offline"},
		// Out-of-pool static mapping, online
		{IP: "192.168.1.50", Interface: "lan", ActiveStatus: "static", OnlineStatus: "active/online"},
		// Out-of-pool static mapping, offline
		{IP: "192.168.1.51", Interface: "lan", ActiveStatus: "static", OnlineStatus: "idle/offline"},
		// In secondary pool, online
		{IP: "192.168.1.215", Interface: "", ActiveStatus: "active", OnlineStatus: "active/online"},
	}

	server := newDHCPTestServer(t, serverResponse, leaseResponse)
	defer server.Close()

	target := targetFromTestServer(t, server)
	collector := NewDHCPCollector()
	metrics := collectMetrics(t, collector, target)

	// Expected: pool_size = 101 + 11 = 112
	assertMetric(t, metrics, "pfsense_dhcp_pool_size", 112, "lan")
	// Expected: 3 in-pool active/static leases (192.168.1.100, .150, .215)
	assertMetric(t, metrics, "pfsense_dhcp_leases_active", 3, "lan")
	// Expected: all 3 in-pool leases are online
	assertMetric(t, metrics, "pfsense_dhcp_leases_online", 3, "lan")
	// Expected: 2 out-of-pool static mappings (192.168.1.50, .51)
	assertMetric(t, metrics, "pfsense_dhcp_static_mappings_total", 2, "lan")
	// Expected: 1 out-of-pool static mapping online (192.168.1.50)
	assertMetric(t, metrics, "pfsense_dhcp_static_mappings_online", 1, "lan")
	// Expected: utilization = 3/112
	assertMetricApprox(t, metrics, "pfsense_dhcp_pool_utilization", 3.0/112.0, "lan")
	// Expected: server enabled
	assertMetric(t, metrics, "pfsense_dhcp_server_enabled", 1, "lan")
}

func TestDHCPCollectorMultipleInterfaces(t *testing.T) {
	serverResponse := []DHCPServerConfig{
		{ID: "lan", Enable: true, RangeFrom: "10.0.1.100", RangeTo: "10.0.1.200"},
		{ID: "opt1", Enable: true, RangeFrom: "10.0.2.100", RangeTo: "10.0.2.200"},
	}

	leaseResponse := []DHCPLease{
		// Dynamic lease in lan pool (null interface)
		{IP: "10.0.1.150", Interface: "", ActiveStatus: "active", OnlineStatus: "active/online"},
		// Dynamic lease in opt1 pool (null interface)
		{IP: "10.0.2.150", Interface: "", ActiveStatus: "active", OnlineStatus: "active/online"},
		{IP: "10.0.2.151", Interface: "", ActiveStatus: "active", OnlineStatus: "idle/offline"},
	}

	server := newDHCPTestServer(t, serverResponse, leaseResponse)
	defer server.Close()

	target := targetFromTestServer(t, server)
	collector := NewDHCPCollector()
	metrics := collectMetrics(t, collector, target)

	assertMetric(t, metrics, "pfsense_dhcp_leases_active", 1, "lan")
	assertMetric(t, metrics, "pfsense_dhcp_leases_active", 2, "opt1")
	assertMetric(t, metrics, "pfsense_dhcp_leases_online", 1, "lan")
	assertMetric(t, metrics, "pfsense_dhcp_leases_online", 1, "opt1")
}

func TestDHCPCollectorDisabledServer(t *testing.T) {
	serverResponse := []DHCPServerConfig{
		{ID: "lan", Enable: false, RangeFrom: "", RangeTo: ""},
	}

	server := newDHCPTestServer(t, serverResponse, []DHCPLease{})
	defer server.Close()

	target := targetFromTestServer(t, server)
	collector := NewDHCPCollector()
	metrics := collectMetrics(t, collector, target)

	assertMetric(t, metrics, "pfsense_dhcp_server_enabled", 0, "lan")
	assertMetric(t, metrics, "pfsense_dhcp_pool_size", 0, "lan")
	assertMetric(t, metrics, "pfsense_dhcp_pool_utilization", 0, "lan")
}

func TestDHCPCollectorCollectWithTargetError(t *testing.T) {
	target := &utils.Target{
		Host:    "nonexistent.host",
		Port:    443,
		Scheme:  "https",
		Timeout: 6,
	}

	collector := NewDHCPCollector()

	ch := make(chan prometheus.Metric, 20)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 metrics on error, got %d", count)
	}
}

func TestIPInUint32Ranges(t *testing.T) {
	r1, _ := parseUint32Range("192.168.1.100", "192.168.1.200")
	r2, _ := parseUint32Range("192.168.1.210", "192.168.1.220")

	tests := []struct {
		name     string
		ip       string
		ranges   []uint32Range
		expected bool
	}{
		{"in primary range", "192.168.1.150", []uint32Range{r1}, true},
		{"at range start", "192.168.1.100", []uint32Range{r1}, true},
		{"at range end", "192.168.1.200", []uint32Range{r1}, true},
		{"below range", "192.168.1.50", []uint32Range{r1}, false},
		{"above range", "192.168.1.250", []uint32Range{r1}, false},
		{"in second range", "192.168.1.215", []uint32Range{r1, r2}, true},
		{"between ranges", "192.168.1.205", []uint32Range{r1, r2}, false},
		{"empty ranges", "192.168.1.150", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := ipToUint32(tt.ip)
			result := ipInUint32Ranges(ip, tt.ranges)
			if result != tt.expected {
				t.Errorf("ipInUint32Ranges(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIPToUint32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{"valid ipv4", "192.168.1.1", 0xC0A80101},
		{"zeros", "0.0.0.0", 0},
		{"empty string", "", 0},
		{"invalid", "not-an-ip", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToUint32(tt.ip)
			if result != tt.expected {
				t.Errorf("ipToUint32(%q) = %d, want %d", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestParseUint32Range(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantOK  bool
	}{
		{"valid range", "192.168.1.100", "192.168.1.200", true},
		{"single IP", "192.168.1.1", "192.168.1.1", true},
		{"reversed", "192.168.1.200", "192.168.1.100", false},
		{"empty from", "", "192.168.1.200", false},
		{"empty to", "192.168.1.100", "", false},
		{"invalid from", "bad", "192.168.1.200", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseUint32Range(tt.from, tt.to)
			if ok != tt.wantOK {
				t.Errorf("parseUint32Range(%q, %q) ok = %v, want %v", tt.from, tt.to, ok, tt.wantOK)
			}
		})
	}
}

// --- Test helpers ---

func newDHCPTestServer(t *testing.T, servers []DHCPServerConfig, leases []DHCPLease) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		var err error
		switch r.URL.Path {
		case "/api/v2/services/dhcp_servers":
			data, err = json.Marshal(servers)
		case "/api/v2/status/dhcp_server/leases":
			data, err = json.Marshal(leases)
		default:
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}
		if err != nil {
			t.Fatalf("Failed to marshal test response: %v", err)
		}

		response := utils.Response{
			Code:   200,
			Status: "success",
			Data:   data,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func targetFromTestServer(t *testing.T, server *httptest.Server) *utils.Target {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("Failed to parse test server port: %v", err)
	}

	return &utils.Target{
		Host:    u.Hostname(),
		Port:    port,
		Scheme:  u.Scheme,
		Timeout: 10,
	}
}

func collectMetrics(t *testing.T, collector *DHCPCollector, target *utils.Target) []prometheus.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	return metrics
}

func assertMetric(t *testing.T, metrics []prometheus.Metric, name string, expected float64, iface string) {
	t.Helper()
	for _, m := range metrics {
		d := &dto.Metric{}
		m.Write(d)

		if !strings.Contains(m.Desc().String(), name) {
			continue
		}

		// Check interface label matches
		ifaceMatch := false
		for _, lp := range d.Label {
			if lp.GetName() == "interface" && lp.GetValue() == iface {
				ifaceMatch = true
			}
		}
		if !ifaceMatch {
			continue
		}

		got := d.Gauge.GetValue()
		if got != expected {
			t.Errorf("%s{interface=%q} = %v, want %v", name, iface, got, expected)
		}
		return
	}
	t.Errorf("metric %s{interface=%q} not found", name, iface)
}

func assertMetricApprox(t *testing.T, metrics []prometheus.Metric, name string, expected float64, iface string) {
	t.Helper()
	for _, m := range metrics {
		d := &dto.Metric{}
		m.Write(d)

		if !strings.Contains(m.Desc().String(), name) {
			continue
		}

		ifaceMatch := false
		for _, lp := range d.Label {
			if lp.GetName() == "interface" && lp.GetValue() == iface {
				ifaceMatch = true
			}
		}
		if !ifaceMatch {
			continue
		}

		got := d.Gauge.GetValue()
		diff := got - expected
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("%s{interface=%q} = %v, want ~%v", name, iface, got, expected)
		}
		return
	}
	t.Errorf("metric %s{interface=%q} not found", name, iface)
}
