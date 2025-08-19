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
	registry.Register(NewSystemCollector())
}

// SystemCollector collects metrics about system status.
type SystemCollector struct {
	systemTemperatureCelsius *prometheus.GaugeVec
	systemCPUCount           *prometheus.GaugeVec
	systemCPUUsage           *prometheus.GaugeVec
	systemDiskUsage          *prometheus.GaugeVec
	systemMemoryUsage        *prometheus.GaugeVec
	systemSwapUsage          *prometheus.GaugeVec
	systemMbufUsage          *prometheus.GaugeVec
}

// SystemStats represents the structure of the system status data returned by the API.
type SystemStats struct {
	TempC     float64 `json:"temp_c"`
	CPUCount  float64 `json:"cpu_count"`
	CPUUsage  float64 `json:"cpu_usage"`
	DiskUsage float64 `json:"disk_usage"`
	MemUsage  float64 `json:"mem_usage"`
	SwapUsage float64 `json:"swap_usage"`
	MbufUsage float64 `json:"mbuf_usage"`
}

// NewSystemCollector is the constructor
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{
		systemTemperatureCelsius: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_temperature_celsius",
				Help: "Current system temperature in Celsius.",
			},
			[]string{"host"},
		),
		systemCPUCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_cpu_count",
				Help: "Number of CPU cores available on the system.",
			},
			[]string{"host"},
		),
		systemCPUUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_cpu_usage_ratio",
				Help: "Current CPU usage as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host"},
		),
		systemDiskUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_disk_usage_ratio",
				Help: "Current disk usage as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host"},
		),
		systemMemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_memory_usage_ratio",
				Help: "Current memory usage as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host"},
		),
		systemSwapUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_swap_usage_ratio",
				Help: "Current swap usage as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host"},
		),
		systemMbufUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "system_mbuf_usage_ratio",
				Help: "Current mbuf usage as a decimal percentage (0.0 - 1.0).",
			},
			[]string{"host"},
		),
	}
}

// Name returns the name of the collector.
func (c *SystemCollector) Name() string {
	return "system"
}

// Describe sends the metric descriptions to the channel.
func (c *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	c.systemTemperatureCelsius.Describe(ch)
	c.systemCPUCount.Describe(ch)
	c.systemCPUUsage.Describe(ch)
	c.systemDiskUsage.Describe(ch)
	c.systemMemoryUsage.Describe(ch)
	c.systemSwapUsage.Describe(ch)
	c.systemMbufUsage.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *SystemCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics for the target
	resp, err := utils.Request(target, "GET", "/api/v2/status/system")
	if err != nil {
		log.Error(target.Host, "failed to fetch system status: "+err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("system", "received nil response from host %s", target.Host)
		return
	}

	// Unmarshal the response data into a SystemStats struct
	var stats SystemStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		log.Error("system", "failed to unmarshal system response from host %s: %s", target.Host, err.Error())
		return
	}

	// Update the metrics
	c.systemTemperatureCelsius.WithLabelValues(target.Host).Set(float64(stats.TempC))
	c.systemCPUCount.WithLabelValues(target.Host).Set(float64(stats.CPUCount))
	c.systemCPUUsage.WithLabelValues(target.Host).Set(float64(stats.CPUUsage) / 100)
	c.systemDiskUsage.WithLabelValues(target.Host).Set(float64(stats.DiskUsage) / 100)
	c.systemMemoryUsage.WithLabelValues(target.Host).Set(float64(stats.MemUsage) / 100)
	c.systemSwapUsage.WithLabelValues(target.Host).Set(float64(stats.SwapUsage) / 100)
	c.systemMbufUsage.WithLabelValues(target.Host).Set(float64(stats.MbufUsage) / 100)

	// Collect the metrics
	c.systemTemperatureCelsius.Collect(ch)
	c.systemCPUCount.Collect(ch)
	c.systemCPUUsage.Collect(ch)
	c.systemDiskUsage.Collect(ch)
	c.systemMemoryUsage.Collect(ch)
	c.systemSwapUsage.Collect(ch)
	c.systemMbufUsage.Collect(ch)
}
