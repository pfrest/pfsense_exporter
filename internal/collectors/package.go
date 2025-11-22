package collectors

import (
	"encoding/json"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// init ensures the collector is automatically added to the registry.
func init() {
	registry.Register(NewPackageCollector())
}

// PackageCollector collects metrics about package status.
type PackageCollector struct {
	updateAvailable *prometheus.GaugeVec
}

// PackageStats represents the structure of the package status data returned by the API.
type PackageStats struct {
	Name             string `json:"name"`
	Shortname        string `json:"shortname"`
	InstalledVersion string `json:"installed_version"`
	LatestVersion    string `json:"latest_version"`
	UpdateAvailable  bool   `json:"update_available"`
}

// NewPackageCollector is the constructor
func NewPackageCollector() *PackageCollector {
	return &PackageCollector{
		updateAvailable: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "package_update_available",
				Help: "Whether an update is available for the package (1) or not (0).",
			},
			[]string{"host", "name", "shortname", "installed_version", "latest_version"},
		),
	}
}

// Name returns the name of the collector.
func (c *PackageCollector) Name() string {
	return "package"
}

// Describe sends the metric descriptions to the channel.
func (c *PackageCollector) Describe(ch chan<- *prometheus.Desc) {
	c.updateAvailable.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *PackageCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics from the target
	resp, err := utils.Request(target, "GET", "/api/v2/system/packages")
	if err != nil {
		log.Error("packages", "failed to fetch package statuses from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("packages", "received nil response from host %s", target.Host)
		return
	}

	// Convert the data to an array of PackageStats
	var packages []PackageStats
	if err := json.Unmarshal(resp.Data, &packages); err != nil {
		log.Error("packages", "failed to unmarshal packages response from host %s: %s", target.Host, err.Error())
		return
	}

	// Reset metrics before collecting new data
	c.resetMetrics()

	// Extract metrics for each package identified
	for _, pkg := range packages {
		// Update the metrics
		c.updateAvailable.WithLabelValues(target.Host, pkg.Name, pkg.Shortname, pkg.InstalledVersion, pkg.LatestVersion).Set(float64(utils.BoolToFloat64(pkg.UpdateAvailable)))
	}

	// Collect the metrics
	c.updateAvailable.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *PackageCollector) resetMetrics() {
	c.updateAvailable.Reset()
}
