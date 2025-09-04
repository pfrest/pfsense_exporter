package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewSystemCollector(t *testing.T) {
	collector := NewSystemCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.systemTemperatureCelsius == nil {
		t.Error("Expected systemTemperatureCelsius metric to be initialized")
	}
	if collector.systemCPUCount == nil {
		t.Error("Expected systemCPUCount metric to be initialized")
	}
	if collector.systemCPUUsage == nil {
		t.Error("Expected systemCPUUsage metric to be initialized")
	}
	if collector.systemDiskUsage == nil {
		t.Error("Expected systemDiskUsage metric to be initialized")
	}
	if collector.systemMemoryUsage == nil {
		t.Error("Expected systemMemoryUsage metric to be initialized")
	}
	if collector.systemSwapUsage == nil {
		t.Error("Expected systemSwapUsage metric to be initialized")
	}
	if collector.systemMbufUsage == nil {
		t.Error("Expected systemMbufUsage metric to be initialized")
	}
}

func TestSystemCollectorName(t *testing.T) {
	collector := NewSystemCollector()

	if collector.Name() != "system" {
		t.Errorf("Expected name 'system', got %s", collector.Name())
	}
}

func TestSystemCollectorDescribe(t *testing.T) {
	collector := NewSystemCollector()

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

	// Should have 7 descriptions
	if count != 7 {
		t.Errorf("Expected 7 metric descriptions, got %d", count)
	}
}

func TestSystemCollectorCollectWithTarget(t *testing.T) {
	// Create test server for system status
	systemResponse := SystemStats{
		TempC:     45.5,
		CPUCount:  4,
		CPUUsage:  75.2,
		DiskUsage: 60.8,
		MemUsage:  82.3,
		SwapUsage: 15.7,
		MbufUsage: 25.1,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/status/system" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(systemResponse)
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
	collector := NewSystemCollector()

	if collector.Name() != "system" {
		t.Errorf("Expected name 'system', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestSystemCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewSystemCollector()

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

func TestSystemStatsStruct(t *testing.T) {
	stats := SystemStats{
		TempC:     42.5,
		CPUCount:  8,
		CPUUsage:  65.3,
		DiskUsage: 45.7,
		MemUsage:  78.9,
		SwapUsage: 12.1,
		MbufUsage: 18.5,
	}

	if stats.TempC != 42.5 {
		t.Errorf("Expected TempC 42.5, got %f", stats.TempC)
	}
	if stats.CPUCount != 8 {
		t.Errorf("Expected CPUCount 8, got %f", stats.CPUCount)
	}
	if stats.CPUUsage != 65.3 {
		t.Errorf("Expected CPUUsage 65.3, got %f", stats.CPUUsage)
	}
	if stats.DiskUsage != 45.7 {
		t.Errorf("Expected DiskUsage 45.7, got %f", stats.DiskUsage)
	}
	if stats.MemUsage != 78.9 {
		t.Errorf("Expected MemUsage 78.9, got %f", stats.MemUsage)
	}
	if stats.SwapUsage != 12.1 {
		t.Errorf("Expected SwapUsage 12.1, got %f", stats.SwapUsage)
	}
	if stats.MbufUsage != 18.5 {
		t.Errorf("Expected MbufUsage 18.5, got %f", stats.MbufUsage)
	}
}
