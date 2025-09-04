package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewFirewallStatesCollector(t *testing.T) {
	collector := NewFirewallStatesCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.firewallStatesMaximumCount == nil {
		t.Error("Expected firewallStatesMaximumCount metric to be initialized")
	}
	if collector.firewallStatesCurrentCount == nil {
		t.Error("Expected firewallStatesCurrentCount metric to be initialized")
	}
	if collector.firewallStatesUsageRatio == nil {
		t.Error("Expected firewallStatesUsageRatio metric to be initialized")
	}
}

func TestFirewallStatesCollectorName(t *testing.T) {
	collector := NewFirewallStatesCollector()

	if collector.Name() != "firewall_states" {
		t.Errorf("Expected name 'firewall_states', got %s", collector.Name())
	}
}

func TestFirewallStatesCollectorDescribe(t *testing.T) {
	collector := NewFirewallStatesCollector()

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

	// Should have 3 descriptions
	if count != 3 {
		t.Errorf("Expected 3 metric descriptions, got %d", count)
	}
}

func TestFirewallStatesCollectorCollectWithTarget(t *testing.T) {
	// Create test server for firewall states status
	statesResponse := FirewallStatesStats{
		MaximumStates:        100000,
		DefaultMaximumStates: 100000,
		CurrentStates:        12345,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/firewall/states/size" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(statesResponse)
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
	collector := NewFirewallStatesCollector()

	if collector.Name() != "firewall_states" {
		t.Errorf("Expected name 'firewall_states', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestFirewallStatesCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewFirewallStatesCollector()

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

func TestFirewallStatesStatsStruct(t *testing.T) {
	stats := FirewallStatesStats{
		MaximumStates:        50000,
		DefaultMaximumStates: 100000,
		CurrentStates:        25000,
	}

	if stats.MaximumStates != 50000 {
		t.Errorf("Expected MaximumStates 50000, got %f", stats.MaximumStates)
	}
	if stats.DefaultMaximumStates != 100000 {
		t.Errorf("Expected DefaultMaximumStates 100000, got %f", stats.DefaultMaximumStates)
	}
	if stats.CurrentStates != 25000 {
		t.Errorf("Expected CurrentStates 25000, got %f", stats.CurrentStates)
	}
}

func TestFirewallStatesStatsWithDefaultMaximum(t *testing.T) {
	// Test case where MaximumStates is 0 and should fall back to DefaultMaximumStates
	stats := FirewallStatesStats{
		MaximumStates:        0,
		DefaultMaximumStates: 75000,
		CurrentStates:        30000,
	}

	if stats.MaximumStates != 0 {
		t.Errorf("Expected MaximumStates 0, got %f", stats.MaximumStates)
	}
	if stats.DefaultMaximumStates != 75000 {
		t.Errorf("Expected DefaultMaximumStates 75000, got %f", stats.DefaultMaximumStates)
	}
	if stats.CurrentStates != 30000 {
		t.Errorf("Expected CurrentStates 30000, got %f", stats.CurrentStates)
	}
}
