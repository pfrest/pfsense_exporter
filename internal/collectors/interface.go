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
	interfaceUp              *prometheus.Desc
	interfaceInErrsCount     *prometheus.Desc
	interfaceOutErrsCount    *prometheus.Desc
	interfaceCollisionsCount *prometheus.Desc
	interfaceInBytesCount    *prometheus.Desc
	interfaceInPassBytesCount *prometheus.Desc
	interfaceOutBytesCount    *prometheus.Desc
	interfaceOutPassBytesCount *prometheus.Desc
	interfaceInPktsCount      *prometheus.Desc
	interfaceInPassPktsCount  *prometheus.Desc
	interfaceOutPktsCount     *prometheus.Desc
	interfaceOutPassPktsCount *prometheus.Desc
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

// NewInterfaceCollector is the constructor.
func NewInterfaceCollector() *InterfaceCollector {
	gaugeLabels := []string{"host", "name", "descr", "hwif", "status"}
	counterLabels := []string{"host", "name", "descr", "hwif"}
	return &InterfaceCollector{
		interfaceUp: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_up",
			"Whether the interface is up (1) or down (0).",
			gaugeLabels, nil,
		),
		interfaceInErrsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_in_errs_count",
			"The number of input errors on the interface.",
			counterLabels, nil,
		),
		interfaceOutErrsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_out_errs_count",
			"The number of output errors on the interface.",
			counterLabels, nil,
		),
		interfaceCollisionsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_collisions_count",
			"The number of collisions on the interface.",
			counterLabels, nil,
		),
		interfaceInBytesCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_in_bytes",
			"The number of input bytes on the interface.",
			counterLabels, nil,
		),
		interfaceInPassBytesCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_in_pass_bytes",
			"The number of input bytes passed on the interface.",
			counterLabels, nil,
		),
		interfaceOutBytesCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_out_bytes",
			"The number of output bytes on the interface.",
			counterLabels, nil,
		),
		interfaceOutPassBytesCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_out_pass_bytes",
			"The number of output bytes passed on the interface.",
			counterLabels, nil,
		),
		interfaceInPktsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_in_pkts_count",
			"The number of input packets handled by the interface.",
			counterLabels, nil,
		),
		interfaceInPassPktsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_in_pass_pkts_count",
			"The number of input packets passed on the interface.",
			counterLabels, nil,
		),
		interfaceOutPktsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_out_pkts_count",
			"The number of output packets handled by the interface.",
			counterLabels, nil,
		),
		interfaceOutPassPktsCount: prometheus.NewDesc(
			registry.MetricsPrefix+"interface_out_pass_pkts_count",
			"The number of output packets passed on the interface.",
			counterLabels, nil,
		),
	}
}

// Name returns the name of the collector.
func (c *InterfaceCollector) Name() string {
	return "interface"
}

// Describe sends the metric descriptions to the channel.
func (c *InterfaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.interfaceUp
	ch <- c.interfaceInErrsCount
	ch <- c.interfaceOutErrsCount
	ch <- c.interfaceCollisionsCount
	ch <- c.interfaceInBytesCount
	ch <- c.interfaceInPassBytesCount
	ch <- c.interfaceOutBytesCount
	ch <- c.interfaceOutPassBytesCount
	ch <- c.interfaceInPktsCount
	ch <- c.interfaceInPassPktsCount
	ch <- c.interfaceOutPktsCount
	ch <- c.interfaceOutPassPktsCount
}

// CollectWithTarget fetches interface stats and sends them to the channel.
func (c *InterfaceCollector) CollectWithTarget(ch chan<- prometheus.Metric, target *utils.Target) {
	resp, err := utils.Request(target, "GET", "/api/v2/status/interfaces")
	if err != nil {
		log.Error("interfaces", "failed to fetch interface statuses from host %s: %s", target.Host, err.Error())
		return
	}
	if resp == nil || resp.Data == nil {
		log.Error("interfaces", "received nil response from host %s", target.Host)
		return
	}

	var interfaces []InterfaceStats
	if err := json.Unmarshal(resp.Data, &interfaces); err != nil {
		log.Error("interfaces", "failed to unmarshal interfaces response from host %s: %s", target.Host, err.Error())
		return
	}

	for _, iface := range interfaces {
		ch <- prometheus.MustNewConstMetric(c.interfaceUp, prometheus.GaugeValue, interfaceStatusToFloat64(iface.Status), target.Host, iface.Name, iface.Descr, iface.Hwif, iface.Status)
		ch <- prometheus.MustNewConstMetric(c.interfaceInErrsCount, prometheus.CounterValue, iface.InErrs, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceOutErrsCount, prometheus.CounterValue, iface.OutErrs, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceCollisionsCount, prometheus.CounterValue, iface.Collisions, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceInBytesCount, prometheus.CounterValue, iface.InBytes, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceInPassBytesCount, prometheus.CounterValue, iface.InBytesPass, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceOutBytesCount, prometheus.CounterValue, iface.OutBytes, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceOutPassBytesCount, prometheus.CounterValue, iface.OutBytesPass, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceInPktsCount, prometheus.CounterValue, iface.InPkts, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceInPassPktsCount, prometheus.CounterValue, iface.InPktsPass, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceOutPktsCount, prometheus.CounterValue, iface.OutPkts, target.Host, iface.Name, iface.Descr, iface.Hwif)
		ch <- prometheus.MustNewConstMetric(c.interfaceOutPassPktsCount, prometheus.CounterValue, iface.OutPktsPass, target.Host, iface.Name, iface.Descr, iface.Hwif)
	}
}

// interfaceStatusToFloat64 converts the interface status string to a float64 for Prometheus metrics.
func interfaceStatusToFloat64(status string) float64 {
	if status == "up" {
		return 1.0
	}
	return 0.0
}
