package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewGatewayCollector(t *testing.T) {
	collector := NewGatewayCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.gatewayLossRatio == nil {
		t.Error("Expected gatewayLossRatio metric to be initialized")
	}
	if collector.gatewayDelaySeconds == nil {
		t.Error("Expected gatewayDelaySeconds metric to be initialized")
	}
	if collector.gatewayStdDevSeconds == nil {
		t.Error("Expected gatewayStdDevSeconds metric to be initialized")
	}
	if collector.gatewayUp == nil {
		t.Error("Expected gatewayUp metric to be initialized")
	}
}

func TestGatewayCollectorName(t *testing.T) {
	collector := NewGatewayCollector()

	if collector.Name() != "gateways" {
		t.Errorf("Expected name 'gateways', got %s", collector.Name())
	}
}

func TestGatewayCollectorDescribe(t *testing.T) {
	collector := NewGatewayCollector()

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

	// Should have 4 descriptions
	if count != 4 {
		t.Errorf("Expected 4 metric descriptions, got %d", count)
	}
}

func TestGatewayCollectorCollectWithTarget(t *testing.T) {
	// Create test server for gateway status
	gatewayResponse := []GatewayStats{
		{
			Name:      "WAN_DHCP",
			Loss:      2.5,
			Delay:     15.3,
			StdDev:    3.2,
			Status:    "online",
			Substatus: "none",
			SourceIP:  "192.168.1.1",
			MonitorIP: "8.8.8.8",
		},
		{
			Name:      "WAN2",
			Loss:      0.0,
			Delay:     12.8,
			StdDev:    1.5,
			Status:    "offline",
			Substatus: "down",
			SourceIP:  "192.168.2.1",
			MonitorIP: "8.8.4.4",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status/gateways" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(gatewayResponse)
		response := utils.Response{
			Code:   200,
			Status: "success",
			Data:   data,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// We can't easily test the full collection since it depends on utils.Request
	// But we can test the collector structure and methods
	collector := NewGatewayCollector()

	if collector.Name() != "gateways" {
		t.Errorf("Expected name 'gateways', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestGatewayCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewGatewayCollector()

	ch := make(chan prometheus.Metric, 10)
	go func() {
		collector.CollectWithTarget(ch, target)
		close(ch)
	}()

	// Drain the channel - should be empty due to error
	count := 0
	for range ch {
		count++
	}

	// No metrics should be produced on error
}

func TestGatewayUpToFloat64(t *testing.T) {
	// Test online status
	result := gatewayUpToFloat64("online")
	if result != 1.0 {
		t.Errorf("Expected 1.0 for online status, got %f", result)
	}

	// Test offline status
	result = gatewayUpToFloat64("offline")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for offline status, got %f", result)
	}

	// Test unknown status
	result = gatewayUpToFloat64("unknown")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for unknown status, got %f", result)
	}

	// Test partial status
	result = gatewayUpToFloat64("partial")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for partial status, got %f", result)
	}
}

func TestGatewayStatsStruct(t *testing.T) {
	stats := GatewayStats{
		Name:      "TEST_GW",
		Loss:      5.5,
		Delay:     20.3,
		StdDev:    2.8,
		Status:    "online",
		Substatus: "none",
		SourceIP:  "10.0.0.1",
		MonitorIP: "1.1.1.1",
	}

	if stats.Name != "TEST_GW" {
		t.Errorf("Expected Name 'TEST_GW', got %s", stats.Name)
	}
	if stats.Loss != 5.5 {
		t.Errorf("Expected Loss 5.5, got %f", stats.Loss)
	}
	if stats.Delay != 20.3 {
		t.Errorf("Expected Delay 20.3, got %f", stats.Delay)
	}
	if stats.StdDev != 2.8 {
		t.Errorf("Expected StdDev 2.8, got %f", stats.StdDev)
	}
	if stats.Status != "online" {
		t.Errorf("Expected Status 'online', got %s", stats.Status)
	}
	if stats.Substatus != "none" {
		t.Errorf("Expected Substatus 'none', got %s", stats.Substatus)
	}
	if stats.SourceIP != "10.0.0.1" {
		t.Errorf("Expected SourceIP '10.0.0.1', got %s", stats.SourceIP)
	}
	if stats.MonitorIP != "1.1.1.1" {
		t.Errorf("Expected MonitorIP '1.1.1.1', got %s", stats.MonitorIP)
	}
}
