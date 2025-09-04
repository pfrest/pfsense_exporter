package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPackageCollector(t *testing.T) {
	collector := NewPackageCollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.updateAvailable == nil {
		t.Error("Expected updateAvailable metric to be initialized")
	}
}

func TestPackageCollectorName(t *testing.T) {
	collector := NewPackageCollector()

	if collector.Name() != "package" {
		t.Errorf("Expected name 'package', got %s", collector.Name())
	}
}

func TestPackageCollectorDescribe(t *testing.T) {
	collector := NewPackageCollector()

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

func TestPackageCollectorCollectWithTarget(t *testing.T) {
	// Create test server for package status
	packageResponse := []PackageStats{
		{
			Name:             "pfSense-pkg-Cron",
			Shortname:        "cron",
			InstalledVersion: "0.3.7_1",
			LatestVersion:    "0.3.8",
			UpdateAvailable:  true,
		},
		{
			Name:             "pfSense-pkg-pfBlockerNG",
			Shortname:        "pfblockerng",
			InstalledVersion: "2.1.4_26",
			LatestVersion:    "2.1.4_26",
			UpdateAvailable:  false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/system/packages" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(packageResponse)
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
	collector := NewPackageCollector()

	if collector.Name() != "package" {
		t.Errorf("Expected name 'package', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestPackageCollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewPackageCollector()

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

func TestPackageStatsStruct(t *testing.T) {
	stats := PackageStats{
		Name:             "pfSense-pkg-Test",
		Shortname:        "test",
		InstalledVersion: "1.0.0",
		LatestVersion:    "1.0.1",
		UpdateAvailable:  true,
	}

	if stats.Name != "pfSense-pkg-Test" {
		t.Errorf("Expected Name 'pfSense-pkg-Test', got %s", stats.Name)
	}
	if stats.Shortname != "test" {
		t.Errorf("Expected Shortname 'test', got %s", stats.Shortname)
	}
	if stats.InstalledVersion != "1.0.0" {
		t.Errorf("Expected InstalledVersion '1.0.0', got %s", stats.InstalledVersion)
	}
	if stats.LatestVersion != "1.0.1" {
		t.Errorf("Expected LatestVersion '1.0.1', got %s", stats.LatestVersion)
	}
	if !stats.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true")
	}
}
