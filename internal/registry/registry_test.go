package registry

import (
	"testing"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// MockCollector implements TargetedCollector for testing
type MockCollector struct {
	name   string
	called bool
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) Describe(ch chan<- *prometheus.Desc) {
	// Mock implementation
}

func (m *MockCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	m.called = true
	// Don't close the channel here - let the caller handle it
}

func TestRegister(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Reset collectors for test
	collectors = nil

	mock := &MockCollector{name: "test-collector"}
	Register(mock)

	if len(collectors) != 1 {
		t.Errorf("Expected 1 collector, got %d", len(collectors))
	}

	if collectors[0].Name() != "test-collector" {
		t.Errorf("Expected collector name 'test-collector', got %s", collectors[0].Name())
	}
}

func TestNewMasterCollector(t *testing.T) {
	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 4,
		MaxCollectorBufferSize:  100,
	}

	mc := NewMasterCollector(target)

	if mc == nil {
		t.Error("Expected MasterCollector to be created")
	}

	if mc.Target != target {
		t.Error("Expected target to be set correctly")
	}

	if mc.Target.Host != "test.com" {
		t.Errorf("Expected host 'test.com', got %s", mc.Target.Host)
	}
}

func TestMasterCollectorDescribe(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set up test collectors
	collectors = []TargetedCollector{
		&MockCollector{name: "collector1"},
		&MockCollector{name: "collector2"},
	}

	target := &utils.Target{Host: "test.com"}
	mc := NewMasterCollector(target)

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		mc.Describe(ch)
		close(ch)
	}()

	// Drain the channel
	for range ch {
		// Just consume the descriptions
	}

	// If we get here without hanging, the test passed
}

func TestMasterCollectorCollect(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set up test collectors
	mock1 := &MockCollector{name: "collector1"}
	mock2 := &MockCollector{name: "collector2"}
	collectors = []TargetedCollector{mock1, mock2}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 2,
		MaxCollectorBufferSize:  10,
		Collectors:              []string{"collector1", "collector2"}, // Include both collectors
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 20)
	go func() {
		mc.Collect(ch)
		close(ch)
	}()

	// Drain the channel
	for range ch {
		// Just consume the metrics
	}

	// Verify collectors were called
	if !mock1.called {
		t.Error("Expected collector1 to be called")
	}
	if !mock2.called {
		t.Error("Expected collector2 to be called")
	}
}

func TestMasterCollectorCollectWithFilteredCollectors(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set up test collectors
	mock1 := &MockCollector{name: "collector1"}
	mock2 := &MockCollector{name: "collector2"}
	collectors = []TargetedCollector{mock1, mock2}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 2,
		MaxCollectorBufferSize:  10,
		Collectors:              []string{"collector1"}, // Only include collector1
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 20)
	go func() {
		mc.Collect(ch)
		close(ch)
	}()

	// Drain the channel
	for range ch {
		// Just consume the metrics
	}

	// Verify only collector1 was called
	if !mock1.called {
		t.Error("Expected collector1 to be called")
	}
	if mock2.called {
		t.Error("Expected collector2 to NOT be called")
	}
}

func TestMasterCollectorCollectWithNilCollectorsList(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set up test collectors
	mock1 := &MockCollector{name: "collector1"}
	mock2 := &MockCollector{name: "collector2"}
	collectors = []TargetedCollector{mock1, mock2}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 2,
		MaxCollectorBufferSize:  10,
		Collectors:              nil, // No filter - should run all collectors
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 20)
	go func() {
		mc.Collect(ch)
		close(ch)
	}()

	// Drain the channel
	for range ch {
		// Just consume the metrics
	}

	// Verify both collectors were called
	if !mock1.called {
		t.Error("Expected collector1 to be called")
	}
	if !mock2.called {
		t.Error("Expected collector2 to be called")
	}
}

func TestMetricsPrefix(t *testing.T) {
	expected := "pfsense_"
	if MetricsPrefix != expected {
		t.Errorf("Expected MetricsPrefix '%s', got '%s'", expected, MetricsPrefix)
	}
}

func TestTargetedCollectorInterface(t *testing.T) {
	// Test that MockCollector implements TargetedCollector
	var _ TargetedCollector = &MockCollector{}

	mock := &MockCollector{name: "test"}

	if mock.Name() != "test" {
		t.Errorf("Expected name 'test', got %s", mock.Name())
	}
}

func TestMasterCollectorCollectConcurrencyLimiting(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set up many test collectors to test concurrency limiting
	mock1 := &MockCollector{name: "collector1"}
	mock2 := &MockCollector{name: "collector2"}
	mock3 := &MockCollector{name: "collector3"}
	mock4 := &MockCollector{name: "collector4"}
	collectors = []TargetedCollector{mock1, mock2, mock3, mock4}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 1, // Very limited concurrency
		MaxCollectorBufferSize:  5,
		Collectors:              nil, // All collectors allowed
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 100)
	done := make(chan bool)

	go func() {
		mc.Collect(ch)
		close(ch)
		done <- true
	}()

	// Wait for completion
	<-done

	// Verify all collectors were called despite concurrency limit
	if !mock1.called {
		t.Error("Expected collector1 to be called")
	}
	if !mock2.called {
		t.Error("Expected collector2 to be called")
	}
	if !mock3.called {
		t.Error("Expected collector3 to be called")
	}
	if !mock4.called {
		t.Error("Expected collector4 to be called")
	}
}

func TestMasterCollectorCollectEmptyCollectorsList(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Set empty collectors list
	collectors = []TargetedCollector{}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 2,
		MaxCollectorBufferSize:  10,
		Collectors:              nil,
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 10)
	done := make(chan bool)

	go func() {
		mc.Collect(ch)
		close(ch)
		done <- true
	}()

	// Wait for completion
	<-done

	// Should complete without issue
}

func TestMasterCollectorCollectWithSkippedCollectors(t *testing.T) {
	// Save original collectors and restore after test
	originalCollectors := collectors
	defer func() { collectors = originalCollectors }()

	// Enable verbose logging to hit debug log statement
	originalVerbose := log.Verbose
	log.Verbose = true
	defer func() { log.Verbose = originalVerbose }()

	// Set up test collectors
	mock1 := &MockCollector{name: "collector1"}
	mock2 := &MockCollector{name: "collector2"}
	mock3 := &MockCollector{name: "collector3"}
	collectors = []TargetedCollector{mock1, mock2, mock3}

	target := &utils.Target{
		Host:                    "test.com",
		MaxCollectorConcurrency: 2,
		MaxCollectorBufferSize:  10,
		Collectors:              []string{"collector1", "collector3"}, // Skip collector2
	}
	mc := NewMasterCollector(target)

	ch := make(chan prometheus.Metric, 20)
	done := make(chan bool)

	go func() {
		mc.Collect(ch)
		close(ch)
		done <- true
	}()

	// Wait for completion
	<-done

	// Verify only included collectors were called
	if !mock1.called {
		t.Error("Expected collector1 to be called")
	}
	if mock2.called {
		t.Error("Expected collector2 to be skipped")
	}
	if !mock3.called {
		t.Error("Expected collector3 to be called")
	}
}
