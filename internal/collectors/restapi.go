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
	registry.Register(NewRESTAPICollector())
}

// RESTAPICollector collects metrics about the REST API package.
type RESTAPICollector struct {
	restAPIUpdateAvailable *prometheus.GaugeVec
}

// RESTAPIStats represents the structure of the REST API status data returned by the API.
type RESTAPIStats struct {
	UpdateAvailable          bool   `json:"update_available"`
	CurrentVersion           string `json:"current_version"`
	LatestVersion            string `json:"latest_version"`
	LatestVersionReleaseDate string `json:"latest_version_release_date"`
}

// NewRESTAPICollector is the constructor
func NewRESTAPICollector() *RESTAPICollector {
	return &RESTAPICollector{
		restAPIUpdateAvailable: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "restapi_update_available",
				Help: "Whether a REST API update is available (1 = available, 0 = not available).",
			},
			[]string{"host", "current_version", "latest_version", "latest_version_release_date"},
		),
	}
}

// Name returns the name of the collector.
func (c *RESTAPICollector) Name() string {
	return "restapi"
}

// Describe sends the metric descriptions to the channel.
func (c *RESTAPICollector) Describe(ch chan<- *prometheus.Desc) {
	c.restAPIUpdateAvailable.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *RESTAPICollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics for each target
	resp, err := utils.Request(target, "GET", "/api/v2/system/restapi/version")
	if err != nil {
		log.Error(target.Host, "failed to fetch REST API version: "+err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("restapi", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a RESTAPIStats struct
	var stats RESTAPIStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("restapi", "failed to unmarshal REST API response from host %s: %s", target.Host, err.Error())
		return
	}

	// Reset metrics before collecting new data
	c.resetMetrics()

	// Update the metrics
	c.restAPIUpdateAvailable.WithLabelValues(target.Host, stats.CurrentVersion, stats.LatestVersion, stats.LatestVersionReleaseDate).Set(utils.BoolToFloat64(stats.UpdateAvailable))

	// Collect the metrics
	c.restAPIUpdateAvailable.Collect(ch)
}

// resetMetrics resets all metrics in the collector.
func (c *RESTAPICollector) resetMetrics() {
	c.restAPIUpdateAvailable.Reset()
}
