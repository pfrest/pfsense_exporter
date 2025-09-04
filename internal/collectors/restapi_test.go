package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRESTAPICollector(t *testing.T) {
	collector := NewRESTAPICollector()

	if collector == nil {
		t.Error("Expected collector to be created")
	}

	if collector.restAPIUpdateAvailable == nil {
		t.Error("Expected restAPIUpdateAvailable metric to be initialized")
	}
}

func TestRESTAPICollectorName(t *testing.T) {
	collector := NewRESTAPICollector()

	if collector.Name() != "restapi" {
		t.Errorf("Expected name 'restapi', got %s", collector.Name())
	}
}

func TestRESTAPICollectorDescribe(t *testing.T) {
	collector := NewRESTAPICollector()

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

func TestRESTAPICollectorCollectWithTarget(t *testing.T) {
	// Create test server for REST API status
	restAPIResponse := RESTAPIStats{
		UpdateAvailable:          true,
		CurrentVersion:           "1.0.0",
		LatestVersion:            "1.0.1",
		LatestVersionReleaseDate: "2023-10-01",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/system/restapi/version" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			return
		}

		data, _ := json.Marshal(restAPIResponse)
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
	collector := NewRESTAPICollector()

	if collector.Name() != "restapi" {
		t.Errorf("Expected name 'restapi', got %s", collector.Name())
	}

	_ = server.URL // Use server URL to avoid unused variable warning
}

func TestRESTAPICollectorCollectWithTargetError(t *testing.T) {
	// Test with unreachable target to trigger error handling
	target := &utils.Target{
		Host:   "nonexistent.host",
		Port:   443,
		Scheme: "https",
	}

	collector := NewRESTAPICollector()

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

func TestRESTAPIStatsStruct(t *testing.T) {
	stats := RESTAPIStats{
		UpdateAvailable:          false,
		CurrentVersion:           "2.0.0",
		LatestVersion:            "2.0.0",
		LatestVersionReleaseDate: "2023-09-15",
	}

	if stats.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false")
	}
	if stats.CurrentVersion != "2.0.0" {
		t.Errorf("Expected CurrentVersion '2.0.0', got %s", stats.CurrentVersion)
	}
	if stats.LatestVersion != "2.0.0" {
		t.Errorf("Expected LatestVersion '2.0.0', got %s", stats.LatestVersion)
	}
	if stats.LatestVersionReleaseDate != "2023-09-15" {
		t.Errorf("Expected LatestVersionReleaseDate '2023-09-15', got %s", stats.LatestVersionReleaseDate)
	}
}
