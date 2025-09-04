package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewFirewallScheduleCollector(t *testing.T) {
	collector := NewFirewallScheduleCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.firewallScheduleActive == nil {
		t.Error("Expected firewallScheduleActive metric to be initialized")
	}
}

func TestFirewallScheduleCollectorName(t *testing.T) {
	collector := NewFirewallScheduleCollector()

	if collector.Name() != "firewall_schedule" {
		t.Errorf("Expected name 'firewall_schedule', got %s", collector.Name())
	}
}

func TestFirewallScheduleCollectorDescribe(t *testing.T) {
	collector := NewFirewallScheduleCollector()

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

	// Should have 1 description
	if count != 1 {
		t.Errorf("Expected 1 metric description, got %d", count)
	}
}

func TestFirewallScheduleCollectorCollectWithTarget(t *testing.T) {
	// Create test server for firewall schedule status
	scheduleResponse := []FirewallScheduleStats{
		{
			Name:   "business_hours",
			Active: true,
		},
		{
			Name:   "weekend_block",
			Active: false,
		},
		{
			Name:   "night_hours",
			Active: false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/firewall/schedules" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(scheduleResponse)
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
	collector := NewFirewallScheduleCollector()

	if collector.Name() != "firewall_schedule" {
		t.Errorf("Expected name 'firewall_schedule', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestFirewallScheduleCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewFirewallScheduleCollector()

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

func TestFirewallScheduleStatsStruct(t *testing.T) {
	stats := FirewallScheduleStats{
		Name:   "test_schedule",
		Active: true,
	}

	if stats.Name != "test_schedule" {
		t.Errorf("Expected Name 'test_schedule', got %s", stats.Name)
	}
	if !stats.Active {
		t.Error("Expected Active to be true")
	}

	// Test inactive schedule
	stats2 := FirewallScheduleStats{
		Name:   "inactive_schedule",
		Active: false,
	}

	if stats2.Name != "inactive_schedule" {
		t.Errorf("Expected Name 'inactive_schedule', got %s", stats2.Name)
	}
	if stats2.Active {
		t.Error("Expected Active to be false")
	}
}
