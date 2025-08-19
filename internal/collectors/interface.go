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
	registry.Register(NewInterfaceCollector())
}

// InterfaceCollector collects metrics about interface status.
type InterfaceCollector struct {
	interfaceUp                *prometheus.GaugeVec
	interfaceInErrsCount       *prometheus.GaugeVec
	interfaceOutErrsCount      *prometheus.GaugeVec
	interfaceCollisionsCount   *prometheus.GaugeVec
	interfaceInBytesCount      *prometheus.GaugeVec
	interfaceInBytesPassCount  *prometheus.GaugeVec
	interfaceOutBytesCount     *prometheus.GaugeVec
	interfaceOutBytesPassCount *prometheus.GaugeVec
	interfaceInPktsCount       *prometheus.GaugeVec
	interfaceInPktsPassCount   *prometheus.GaugeVec
	interfaceOutPktsCount      *prometheus.GaugeVec
	interfaceOutPktsPassCount  *prometheus.GaugeVec
}

// InterfaceStats represents the structure of the interface status data returned by the API.
type InterfaceStats struct {
	Name         string  `json:"name"`
	Descr        string  `json:"descr"`
	Hwif         string  `json:"hwif"`
	Status       string  `json:"status"`
	InErrs       float64 `json:"inerrs"`
	OutErrs      float64 `json:"outerrs"`
	Collisions   float64 `json:"collisions"`
	InBytes      float64 `json:"inbytes"`
	InBytesPass  float64 `json:"inbytespass"`
	OutBytes     float64 `json:"outbytes"`
	OutBytesPass float64 `json:"outbytespass"`
	InPkts       float64 `json:"inpkts"`
	InPktsPass   float64 `json:"inpktspass"`
	OutPkts      float64 `json:"outpkts"`
	OutPktsPass  float64 `json:"outpktspass"`
}

// NewInterfaceCollector is the constructor
func NewInterfaceCollector() *InterfaceCollector {
	return &InterfaceCollector{
		interfaceUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_up",
				Help: "Whether the interface is up (1) or down (0).",
			},
			[]string{"host", "name", "descr", "hwif", "status"},
		),
		interfaceInErrsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_in_errs_count",
				Help: "The number of input errors on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceOutErrsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_out_errs_count",
				Help: "The number of output errors on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceCollisionsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_collisions_count",
				Help: "The number of collisions on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceInBytesCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_in_bytes",
				Help: "The number of input bytes on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceInBytesPassCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_in_pass_bytes",
				Help: "The number of input bytes passed on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceOutBytesCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_out_bytes",
				Help: "The number of output bytes on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceOutBytesPassCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_out_pass_bytes",
				Help: "The number of output bytes passed on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceInPktsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_in_pkts_count",
				Help: "The number of input packets handled by the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceInPktsPassCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_in_pass_pkts_count",
				Help: "The number of input packets passed on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceOutPktsCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_out_pkts_count",
				Help: "The number of output packets handled by the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
		interfaceOutPktsPassCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: registry.MetricsPrefix + "interface_out_pass_pkts_count",
				Help: "The number of output packets passed on the interface.",
			},
			[]string{"host", "name", "descr", "hwif"},
		),
	}
}

// Name returns the name of the collector.
func (c *InterfaceCollector) Name() string {
	return "interface"
}

// Describe sends the metric descriptions to the channel.
func (c *InterfaceCollector) Describe(ch chan<- *prometheus.Desc) {
	c.interfaceInErrsCount.Describe(ch)
	c.interfaceOutErrsCount.Describe(ch)
	c.interfaceCollisionsCount.Describe(ch)
	c.interfaceInBytesCount.Describe(ch)
	c.interfaceInBytesPassCount.Describe(ch)
	c.interfaceOutBytesCount.Describe(ch)
	c.interfaceOutBytesPassCount.Describe(ch)
	c.interfaceInPktsCount.Describe(ch)
	c.interfaceInPktsPassCount.Describe(ch)
	c.interfaceOutPktsCount.Describe(ch)
	c.interfaceOutPktsPassCount.Describe(ch)
}

// Collect fetches the stats and sends them to the channel.
func (c *InterfaceCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	// Collect metrics from the target
	resp, err := utils.Request(target, "GET", "/api/v2/status/interfaces")
	if err != nil {
		log.Error("interfaces", "failed to fetch interface statuses from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("interfaces", "received nil response from host %s", target.Host)
		return
	}

	// Convert the data to an array of InterfaceStats
	var interfaces []InterfaceStats
	if err := json.Unmarshal(resp.Data, &interfaces); err != nil {
		log.Error("interfaces", "failed to unmarshal interfaces response from host %s: %s", target.Host, err.Error())
		return
	}

	// Extract metrics for each interface identified
	c.interfaceUp.Reset()
	c.interfaceInErrsCount.Reset()
	c.interfaceOutErrsCount.Reset()
	c.interfaceCollisionsCount.Reset()
	c.interfaceInBytesCount.Reset()
	c.interfaceInBytesPassCount.Reset()
	c.interfaceOutBytesCount.Reset()
	c.interfaceOutBytesPassCount.Reset()
	c.interfaceInPktsCount.Reset()
	c.interfaceInPktsPassCount.Reset()
	c.interfaceOutPktsCount.Reset()
	c.interfaceOutPktsPassCount.Reset()
	for _, iface := range interfaces {
		// Update the metrics
		c.interfaceUp.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif, iface.Status).Set(interfaceStatusToFloat64(iface.Status))
		c.interfaceInErrsCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.InErrs))
		c.interfaceOutErrsCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.OutErrs))
		c.interfaceCollisionsCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.Collisions))
		c.interfaceInBytesCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.InBytes))
		c.interfaceInBytesPassCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.InBytesPass))
		c.interfaceOutBytesCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.OutBytes))
		c.interfaceOutBytesPassCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.OutBytesPass))
		c.interfaceInPktsCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.InPkts))
		c.interfaceInPktsPassCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.InPktsPass))
		c.interfaceOutPktsCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.OutPkts))
		c.interfaceOutPktsPassCount.WithLabelValues(target.Host, iface.Name, iface.Descr, iface.Hwif).Set(float64(iface.OutPktsPass))
	}

	// Collect the metrics
	c.interfaceUp.Collect(ch)
	c.interfaceInErrsCount.Collect(ch)
	c.interfaceOutErrsCount.Collect(ch)
	c.interfaceCollisionsCount.Collect(ch)
	c.interfaceInBytesCount.Collect(ch)
	c.interfaceInBytesPassCount.Collect(ch)
	c.interfaceOutBytesCount.Collect(ch)
	c.interfaceOutBytesPassCount.Collect(ch)
	c.interfaceInPktsCount.Collect(ch)
	c.interfaceInPktsPassCount.Collect(ch)
	c.interfaceOutPktsCount.Collect(ch)
	c.interfaceOutPktsPassCount.Collect(ch)
}

// statusToFloat64 converts the interface status string to a float64 for Prometheus metrics.
func interfaceStatusToFloat64(status string) float64 {
	if status == "up" {
		return 1.0
	}
	return 0.0
}
