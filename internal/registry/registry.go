package registry

import (
	"slices"
	"sync"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsPrefix is the prefix for all metrics exposed by the exporter.
const MetricsPrefix = "pfsense_"

// collectors is an un-exported global variable that holds all registered collectors.
var collectors []TargetedCollector

// MasterCollector is the entry point for Prometheus scrapes.
type MasterCollector struct {
	Target *utils.Target
}

type TargetedCollector interface {
	Name() string
	Describe(ch chan<- *prometheus.Desc)
	CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target)
}

// Register adds a new collector to the registry.
func Register(c TargetedCollector) {
	collectors = append(collectors, c)
}

// NewMasterCollector creates a new MasterCollector.
func NewMasterCollector(target *utils.Target) *MasterCollector {
	return &MasterCollector{Target: target}
}

// Describe iterates over the registry and calls Describe on each collector.
func (mc *MasterCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range collectors {
		c.Describe(ch)
	}
}

// Collect iterates over the registry and calls Collect on each collector in parallel.
func (mc *MasterCollector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrent collectors
	semaphore := make(chan struct{}, mc.Target.MaxCollectorConcurrency)

	for _, c := range collectors {
		wg.Add(1)
		go func(collector TargetedCollector) {
			defer wg.Done()

			// Acquire an available semaphore slot and release it when done
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Skip this collector if it's not in the target's collector list
			if mc.Target.Collectors != nil && !slices.Contains(mc.Target.Collectors, collector.Name()) {
				log.Debug("config", "skipping collector %s for target %s", collector.Name(), mc.Target.Host)
				return
			}

			// Create a buffered channel for this collector's metrics
			collectorCh := make(chan prometheus.Metric, mc.Target.MaxCollectorBufferSize)

			// Collect metrics from this collector
			collector.CollectWithTarget(collectorCh, mc.Target)
			close(collectorCh)

			// Forward all metrics to the main channel
			for metric := range collectorCh {
				ch <- metric
			}
		}(c)
	}

	wg.Wait()
}
