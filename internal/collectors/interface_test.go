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

func TestNewInterfaceCollector(t *testing.T) {
	collector := NewInterfaceCollector()

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}
	if collector.interfaceUp == nil {
		t.Error("Expected interfaceUp desc to be initialized")
	}
	if collector.interfaceInErrsCount == nil {
		t.Error("Expected interfaceInErrsCount desc to be initialized")
	}
	if collector.interfaceOutErrsCount == nil {
		t.Error("Expected interfaceOutErrsCount desc to be initialized")
	}
	if collector.interfaceCollisionsCount == nil {
		t.Error("Expected interfaceCollisionsCount desc to be initialized")
	}
	if collector.interfaceInBytesCount == nil {
		t.Error("Expected interfaceInBytesCount desc to be initialized")
	}
	if collector.interfaceInPassBytesCount == nil {
		t.Error("Expected interfaceInPassBytesCount desc to be initialized")
	}
	if collector.interfaceOutBytesCount == nil {
		t.Error("Expected interfaceOutBytesCount desc to be initialized")
	}
	if collector.interfaceOutPassBytesCount == nil {
		t.Error("Expected interfaceOutPassBytesCount desc to be initialized")
	}
	if collector.interfaceInPktsCount == nil {
		t.Error("Expected interfaceInPktsCount desc to be initialized")
	}
	if collector.interfaceInPassPktsCount == nil {
		t.Error("Expected interfaceInPassPktsCount desc to be initialized")
	}
	if collector.interfaceOutPktsCount == nil {
		t.Error("Expected interfaceOutPktsCount desc to be initialized")
	}
	if collector.interfaceOutPassPktsCount == nil {
		t.Error("Expected interfaceOutPassPktsCount desc to be initialized")
	}
}

func TestInterfaceCollectorName(t *testing.T) {
	collector := NewInterfaceCollector()

	if collector.Name() != "interface" {
		t.Errorf("Expected name 'interface', got %s", collector.Name())
	}
}

func TestInterfaceCollectorDescribe(t *testing.T) {
	collector := NewInterfaceCollector()

	ch := make(chan *prometheus.Desc, 20)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	if count != 12 {
		t.Errorf("Expected 12 metric descriptions, got %d", count)
	}
}

func TestInterfaceCollectorCollectWithTarget(t *testing.T) {
	interfaceResponse := []InterfaceStats{
		{
			Name:         "wan",
			Descr:        "WAN",
			Hwif:         "em0",
			Status:       "up",
			InErrs:       10,
			OutErrs:      5,
			Collisions:   2,
			InBytes:      1024000,
			InBytesPass:  1020000,
			OutBytes:     512000,
			OutBytesPass: 510000,
			InPkts:       5000,
			InPktsPass:   4950,
			OutPkts:      2500,
			OutPktsPass:  2480,
		},
		{
			Name:         "lan",
			Descr:        "LAN",
			Hwif:         "em1",
			Status:       "down",
			InErrs:       0,
			OutErrs:      0,
			Collisions:   0,
			InBytes:      0,
			InBytesPass:  0,
			OutBytes:     0,
			OutBytesPass: 0,
			InPkts:       0,
			InPktsPass:   0,
			OutPkts:      0,
			OutPktsPass:  0,
		},
	}

	server := newInterfaceTestServer(t, interfaceResponse)
	defer server.Close()

	target := interfaceTargetFromTestServer(t, server)
	collector := NewInterfaceCollector()
	metrics := collectInterfaceMetrics(t, collector, target)

	// WAN interface
	assertInterfaceMetric(t, metrics, "pfsense_interface_up", 1, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_errs_count", 10, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_out_errs_count", 5, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_collisions_count", 2, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_bytes", 1024000, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_pass_bytes", 1020000, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_out_bytes", 512000, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_out_pass_bytes", 510000, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_pkts_count", 5000, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_pass_pkts_count", 4950, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_out_pkts_count", 2500, "wan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_out_pass_pkts_count", 2480, "wan")

	// LAN interface
	assertInterfaceMetric(t, metrics, "pfsense_interface_up", 0, "lan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_errs_count", 0, "lan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_collisions_count", 0, "lan")
	assertInterfaceMetric(t, metrics, "pfsense_interface_in_bytes", 0, "lan")
}

func TestInterfaceCollectorCollectWithTargetError(t *testing.T) {
	target := &utils.Target{
		Host:    "nonexistent.host",
		Port:    443,
		Scheme:  "https",
		Timeout: 6,
	}

	collector := NewInterfaceCollector()

	ch := make(chan prometheus.Metric, 50)
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

func TestInterfaceStatusToFloat64(t *testing.T) {
	tests := []struct {
		status   string
		expected float64
	}{
		{"up", 1.0},
		{"down", 0.0},
		{"unknown", 0.0},
		{"no carrier", 0.0},
	}

	for _, tt := range tests {
		result := interfaceStatusToFloat64(tt.status)
		if result != tt.expected {
			t.Errorf("interfaceStatusToFloat64(%q) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestInterfaceMetricTypes(t *testing.T) {
	interfaceResponse := []InterfaceStats{
		{
			Name: "wan", Descr: "WAN", Hwif: "em0", Status: "up",
			InErrs: 1, OutErrs: 1, Collisions: 1,
			InBytes: 1, InBytesPass: 1, OutBytes: 1, OutBytesPass: 1,
			InPkts: 1, InPktsPass: 1, OutPkts: 1, OutPktsPass: 1,
		},
	}

	server := newInterfaceTestServer(t, interfaceResponse)
	defer server.Close()

	target := interfaceTargetFromTestServer(t, server)
	collector := NewInterfaceCollector()
	metrics := collectInterfaceMetrics(t, collector, target)

	for _, m := range metrics {
		d := &dto.Metric{}
		m.Write(d)

		name := m.Desc().String()
		if strings.Contains(name, "interface_up") {
			if d.Gauge == nil {
				t.Errorf("%s should be a gauge", name)
			}
		} else {
			if d.Counter == nil {
				t.Errorf("%s should be a counter", name)
			}
		}
	}
}

// --- Test helpers ---

func newInterfaceTestServer(t *testing.T, interfaces []InterfaceStats) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status/interfaces" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}
		data, err := json.Marshal(interfaces)
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

func interfaceTargetFromTestServer(t *testing.T, server *httptest.Server) *utils.Target {
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

func collectInterfaceMetrics(t *testing.T, collector *InterfaceCollector, target *utils.Target) []prometheus.Metric {
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

func assertInterfaceMetric(t *testing.T, metrics []prometheus.Metric, name string, expected float64, ifaceName string) {
	t.Helper()
	for _, m := range metrics {
		d := &dto.Metric{}
		m.Write(d)

		if !strings.Contains(m.Desc().String(), name) {
			continue
		}

		nameMatch := false
		for _, lp := range d.Label {
			if lp.GetName() == "name" && lp.GetValue() == ifaceName {
				nameMatch = true
			}
		}
		if !nameMatch {
			continue
		}

		var got float64
		if d.Gauge != nil {
			got = d.Gauge.GetValue()
		} else if d.Counter != nil {
			got = d.Counter.GetValue()
		}

		if got != expected {
			t.Errorf("%s{name=%q} = %v, want %v", name, ifaceName, got, expected)
		}
		return
	}
	t.Errorf("metric %s{name=%q} not found", name, ifaceName)
}
