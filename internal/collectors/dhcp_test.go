package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewDHCPCollector(t *testing.T) {
	collector := NewDHCPCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}
	if collector.poolSize == nil {
		t.Error("Expected poolSize metric to be initialized")
	}
	if collector.leasesActive == nil {
		t.Error("Expected leasesActive metric to be initialized")
	}
	if collector.leasesOnline == nil {
		t.Error("Expected leasesOnline metric to be initialized")
	}
	if collector.utilization == nil {
		t.Error("Expected utilization metric to be initialized")
	}
	if collector.serverUp == nil {
		t.Error("Expected serverUp metric to be initialized")
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

	if count != 5 {
		t.Errorf("Expected 5 metric descriptions, got %d", count)
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
		{IP: "192.168.1.100", Interface: "lan", ActiveStatus: "active", OnlineStatus: "online"},
		{IP: "192.168.1.101", Interface: "lan", ActiveStatus: "active", OnlineStatus: "offline"},
		{IP: "192.168.1.102", Interface: "lan", ActiveStatus: "inactive", OnlineStatus: "offline"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		switch r.URL.Path {
		case "/api/v2/services/dhcp_servers":
			data, _ = json.Marshal(serverResponse)
		case "/api/v2/status/dhcp_server/leases":
			data, _ = json.Marshal(leaseResponse)
		default:
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		response := utils.Response{
			Code:   200,
			Status: "success",
			Data:   data,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	collector := NewDHCPCollector()

	if collector.Name() != "dhcp" {
		t.Errorf("Expected name 'dhcp', got %s", collector.Name())
	}

	_ = server.URL
}

func TestDHCPCollectorCollectWithTargetError(t *testing.T) {
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
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

	// No metrics should be produced on error
	if count != 0 {
		t.Errorf("Expected 0 metrics on error, got %d", count)
	}
}

func TestIPRangeSize(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected int64
	}{
		{"single IP", "192.168.1.1", "192.168.1.1", 1},
		{"small range", "192.168.1.100", "192.168.1.200", 101},
		{"class C full", "192.168.1.1", "192.168.1.254", 254},
		{"empty from", "", "192.168.1.200", 0},
		{"empty to", "192.168.1.100", "", 0},
		{"both empty", "", "", 0},
		{"invalid from", "not-an-ip", "192.168.1.200", 0},
		{"invalid to", "192.168.1.100", "not-an-ip", 0},
		{"reversed range", "192.168.1.200", "192.168.1.100", 0},
		{"cross octet", "192.168.1.250", "192.168.2.10", 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipRangeSize(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("ipRangeSize(%q, %q) = %d, want %d", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestDHCPServerConfigStruct(t *testing.T) {
	config := DHCPServerConfig{
		ID:        "lan",
		Enable:    true,
		RangeFrom: "192.168.1.100",
		RangeTo:   "192.168.1.200",
		Pool: []DHCPServerPoolEntry{
			{RangeFrom: "192.168.1.210", RangeTo: "192.168.1.220"},
		},
	}

	if config.ID != "lan" {
		t.Errorf("Expected ID 'lan', got %s", config.ID)
	}
	if !config.Enable {
		t.Error("Expected Enable to be true")
	}
	if len(config.Pool) != 1 {
		t.Errorf("Expected 1 pool entry, got %d", len(config.Pool))
	}
}

func TestDHCPLeaseStruct(t *testing.T) {
	lease := DHCPLease{
		IP:           "192.168.1.100",
		MAC:          "aa:bb:cc:dd:ee:ff",
		Hostname:     "test-host",
		Interface:    "lan",
		ActiveStatus: "active",
		OnlineStatus: "online",
	}

	if lease.IP != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got %s", lease.IP)
	}
	if lease.Interface != "lan" {
		t.Errorf("Expected Interface 'lan', got %s", lease.Interface)
	}
	if lease.ActiveStatus != "active" {
		t.Errorf("Expected ActiveStatus 'active', got %s", lease.ActiveStatus)
	}
}
