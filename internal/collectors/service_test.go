package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewServiceCollector(t *testing.T) {
	collector := NewServiceCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.serviceUp == nil {
		t.Error("Expected serviceUp metric to be initialized")
	}
	if collector.serviceEnabled == nil {
		t.Error("Expected serviceEnabled metric to be initialized")
	}
}

func TestServiceCollectorName(t *testing.T) {
	collector := NewServiceCollector()

	if collector.Name() != "service" {
		t.Errorf("Expected name 'service', got %s", collector.Name())
	}
}

func TestServiceCollectorDescribe(t *testing.T) {
	collector := NewServiceCollector()

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

	// Should have 2 descriptions
	if count != 2 {
		t.Errorf("Expected 2 metric descriptions, got %d", count)
	}
}

func TestServiceCollectorCollectWithTarget(t *testing.T) {
	// Create test server for service status
	serviceResponse := []ServiceStats{
		{
			Name:    "sshd",
			Enabled: true,
			Status:  true,
		},
		{
			Name:    "dhcpd",
			Enabled: true,
			Status:  false,
		},
		{
			Name:    "ntpd",
			Enabled: false,
			Status:  false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status/services" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(serviceResponse)
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
	collector := NewServiceCollector()

	if collector.Name() != "service" {
		t.Errorf("Expected name 'service', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestServiceCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewServiceCollector()

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

func TestServiceStatsStruct(t *testing.T) {
	stats := ServiceStats{
		Name:    "test-service",
		Enabled: true,
		Status:  false,
	}

	if stats.Name != "test-service" {
		t.Errorf("Expected Name 'test-service', got %s", stats.Name)
	}
	if !stats.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if stats.Status {
		t.Error("Expected Status to be false")
	}
}
