package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCARPCollector(t *testing.T) {
	collector := NewCARPCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.carpEnabled == nil {
		t.Error("Expected carpEnabled metric to be initialized")
	}

	if collector.carpMaintenanceModeEnabled == nil {
		t.Error("Expected carpMaintenanceModeEnabled metric to be initialized")
	}

	if collector.carpVirtualIPStatus == nil {
		t.Error("Expected carpVirtualIPStatus metric to be initialized")
	}
}

func TestCARPCollectorName(t *testing.T) {
	collector := NewCARPCollector()

	if collector.Name() != "carp" {
		t.Errorf("Expected name 'carp', got %s", collector.Name())
	}
}

func TestCARPCollectorDescribe(t *testing.T) {
	collector := NewCARPCollector()

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	// Count descriptions
	count := 0
	for range ch {
		count++
	}

	// Should have 2 descriptions (carpEnabled and carpMaintenanceModeEnabled)
	if count != 2 {
		t.Errorf("Expected 2 metric descriptions, got %d", count)
	}
}

func TestCARPCollectorCollectWithTarget(t *testing.T) {
	// Create test server that returns CARP data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/status/carp":
			response := utils.Response{
				Code:   200,
				Status: "success",
				Data:   json.RawMessage(`{"enable":true,"maintenance_mode":false}`),
			}
			json.NewEncoder(w).Encode(response)
		case "/api/v2/firewall/virtual_ips":
			if r.URL.Query().Get("mode") == "carp" {
				response := utils.Response{
					Code:   200,
					Status: "success",
					Data:   json.RawMessage(`[{"carp_status":"master","uniqid":"vip1","subnet":"192.168.1.1/24","vhid":1,"interface":"wan"}]`),
				}
				json.NewEncoder(w).Encode(response)
			}
		default:
			http.Error(w, "Not found", 404)
		}
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(serverURL.Port())

	target := &utils.Target{
		Host:         serverURL.Hostname(),
		Port:         port,
		Scheme:       serverURL.Scheme,
		Username:     "test",
		Password:     "test",
		AuthMethod:   "basic",
		ValidateCert: false,
		Timeout:      30,
	}

	collector := NewCARPCollector()

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	// Count metrics produced
	metricCount := 0
	for range ch {
		metricCount++
	}

	// Should produce at least 1 metric (carp enabled status)
	if metricCount == 0 {
		t.Error("Expected at least one metric from CARP collector")
	}
}

func TestCARPCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:         "nonexistent.host.invalid.test",
		Port:         443,
		Scheme:       "https",
		Username:     "test",
		Password:     "test",
		AuthMethod:   "basic",
		ValidateCert: false,
		Timeout:      1, // Short timeout
	}

	collector := NewCARPCollector()

	ch := make(chan prometheus.Metric, 10)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	// Drain the channel - might be empty due to error but shouldn't panic
	for range ch {
		// Just drain
	}

	// Test passes if no panic occurred
}

func TestCARPStatusToFloat64(t *testing.T) {
	tests := []struct {
		status   string
		expected float64
	}{
		{"master", 1},
		{"backup", 0},
		{"unknown", -1},
		{"init", -1},
		{"", -1},
		{"invalid", -1},
	}

	for _, tt := range tests {
		t.Run("status_"+tt.status, func(t *testing.T) {
			result := CARPStatusToFloat64(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %f for status '%s', got %f", tt.expected, tt.status, result)
			}
		})
	}
}

func TestCARPStatsStruct(t *testing.T) {
	stats := CARPStats{
		Enabled:         true,
		MaintenanceMode: false,
	}

	if !stats.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if stats.MaintenanceMode {
		t.Error("Expected MaintenanceMode to be false")
	}
}

func TestCARPVirtualIPStatusStruct(t *testing.T) {
	vip := CARPVirtualIPStatus{
		CARPStatus: "master",
		UniqID:     "test-vip",
		Subnet:     "192.168.1.1/24",
		VHID:       42,
		Interface:  "wan",
	}

	if vip.CARPStatus != "master" {
		t.Errorf("Expected CARPStatus 'master', got %s", vip.CARPStatus)
	}

	if vip.UniqID != "test-vip" {
		t.Errorf("Expected UniqID 'test-vip', got %s", vip.UniqID)
	}

	if vip.Subnet != "192.168.1.1/24" {
		t.Errorf("Expected Subnet '192.168.1.1/24', got %s", vip.Subnet)
	}

	if vip.VHID != 42 {
		t.Errorf("Expected VHID 42, got %d", vip.VHID)
	}

	if vip.Interface != "wan" {
		t.Errorf("Expected Interface 'wan', got %s", vip.Interface)
	}
}

func TestCARPCollectorCollectWithBadJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := utils.Response{
			Code:   200,
			Status: "success",
			Data:   json.RawMessage(`invalid json`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(serverURL.Port())

	target := &utils.Target{
		Host:         serverURL.Hostname(),
		Port:         port,
		Scheme:       serverURL.Scheme,
		Username:     "test",
		Password:     "test",
		AuthMethod:   "basic",
		ValidateCert: false,
		Timeout:      30,
	}

	collector := NewCARPCollector()

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	// Drain the channel - should handle JSON parse errors gracefully
	for range ch {
		// Just drain
	}

	// Test passes if no panic occurred
}
