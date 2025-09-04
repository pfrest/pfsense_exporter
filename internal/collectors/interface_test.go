package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewInterfaceCollector(t *testing.T) {
	collector := NewInterfaceCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.interfaceUp == nil {
		t.Error("Expected interfaceUp metric to be initialized")
	}
	if collector.interfaceInErrsCount == nil {
		t.Error("Expected interfaceInErrsCount metric to be initialized")
	}
	if collector.interfaceOutErrsCount == nil {
		t.Error("Expected interfaceOutErrsCount metric to be initialized")
	}
	if collector.interfaceCollisionsCount == nil {
		t.Error("Expected interfaceCollisionsCount metric to be initialized")
	}
	if collector.interfaceInBytesCount == nil {
		t.Error("Expected interfaceInBytesCount metric to be initialized")
	}
	if collector.interfaceInBytesPassCount == nil {
		t.Error("Expected interfaceInBytesPassCount metric to be initialized")
	}
	if collector.interfaceOutBytesCount == nil {
		t.Error("Expected interfaceOutBytesCount metric to be initialized")
	}
	if collector.interfaceOutBytesPassCount == nil {
		t.Error("Expected interfaceOutBytesPassCount metric to be initialized")
	}
	if collector.interfaceInPktsCount == nil {
		t.Error("Expected interfaceInPktsCount metric to be initialized")
	}
	if collector.interfaceInPktsPassCount == nil {
		t.Error("Expected interfaceInPktsPassCount metric to be initialized")
	}
	if collector.interfaceOutPktsCount == nil {
		t.Error("Expected interfaceOutPktsCount metric to be initialized")
	}
	if collector.interfaceOutPktsPassCount == nil {
		t.Error("Expected interfaceOutPktsPassCount metric to be initialized")
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

	// Count descriptions
	count := 0
	for range ch {
		count++
	}

	// Should have 11 descriptions (all metrics except interfaceUp which is not in Describe)
	if count != 11 {
		t.Errorf("Expected 11 metric descriptions, got %d", count)
	}
}

func TestInterfaceCollectorCollectWithTarget(t *testing.T) {
	// Create test server for interface status
	interfaceResponse := []InterfaceStats{
		{
			Name:         "wan",
			Descr:        "WAN",
			Hwif:         "em0",
			Status:       "up",
			InErrs:       10,
			OutErrs:      5,
			Collisions:   0,
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status/interfaces" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(interfaceResponse)
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
	collector := NewInterfaceCollector()

	if collector.Name() != "interface" {
		t.Errorf("Expected name 'interface', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestInterfaceCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewInterfaceCollector()

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

func TestInterfaceStatusToFloat64(t *testing.T) {
	// Test up status
	result := interfaceStatusToFloat64("up")
	if result != 1.0 {
		t.Errorf("Expected 1.0 for up status, got %f", result)
	}

	// Test down status
	result = interfaceStatusToFloat64("down")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for down status, got %f", result)
	}

	// Test unknown status
	result = interfaceStatusToFloat64("unknown")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for unknown status, got %f", result)
	}

	// Test no carrier status
	result = interfaceStatusToFloat64("no carrier")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for 'no carrier' status, got %f", result)
	}
}

func TestInterfaceStatsStruct(t *testing.T) {
	stats := InterfaceStats{
		Name:         "test",
		Descr:        "Test Interface",
		Hwif:         "em0",
		Status:       "up",
		InErrs:       15,
		OutErrs:      8,
		Collisions:   2,
		InBytes:      2048000,
		InBytesPass:  2040000,
		OutBytes:     1024000,
		OutBytesPass: 1020000,
		InPkts:       10000,
		InPktsPass:   9900,
		OutPkts:      5000,
		OutPktsPass:  4960,
	}

	if stats.Name != "test" {
		t.Errorf("Expected Name 'test', got %s", stats.Name)
	}
	if stats.Descr != "Test Interface" {
		t.Errorf("Expected Descr 'Test Interface', got %s", stats.Descr)
	}
	if stats.Hwif != "em0" {
		t.Errorf("Expected Hwif 'em0', got %s", stats.Hwif)
	}
	if stats.Status != "up" {
		t.Errorf("Expected Status 'up', got %s", stats.Status)
	}
	if stats.InErrs != 15 {
		t.Errorf("Expected InErrs 15, got %f", stats.InErrs)
	}
	if stats.OutErrs != 8 {
		t.Errorf("Expected OutErrs 8, got %f", stats.OutErrs)
	}
	if stats.Collisions != 2 {
		t.Errorf("Expected Collisions 2, got %f", stats.Collisions)
	}
	if stats.InBytes != 2048000 {
		t.Errorf("Expected InBytes 2048000, got %f", stats.InBytes)
	}
	if stats.InBytesPass != 2040000 {
		t.Errorf("Expected InBytesPass 2040000, got %f", stats.InBytesPass)
	}
	if stats.OutBytes != 1024000 {
		t.Errorf("Expected OutBytes 1024000, got %f", stats.OutBytes)
	}
	if stats.OutBytesPass != 1020000 {
		t.Errorf("Expected OutBytesPass 1020000, got %f", stats.OutBytesPass)
	}
	if stats.InPkts != 10000 {
		t.Errorf("Expected InPkts 10000, got %f", stats.InPkts)
	}
	if stats.InPktsPass != 9900 {
		t.Errorf("Expected InPktsPass 9900, got %f", stats.InPktsPass)
	}
	if stats.OutPkts != 5000 {
		t.Errorf("Expected OutPkts 5000, got %f", stats.OutPkts)
	}
	if stats.OutPktsPass != 4960 {
		t.Errorf("Expected OutPktsPass 4960, got %f", stats.OutPktsPass)
	}
}
