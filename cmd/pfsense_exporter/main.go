package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"

	_ "github.com/pfrest/pfsense_exporter/internal/collectors"
	"github.com/pfrest/pfsense_exporter/internal/log"
	"github.com/pfrest/pfsense_exporter/internal/registry"
	"github.com/pfrest/pfsense_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// version holds the current version of the application. This can be overridden during the build process.
var version = "0.0.0"

// Args holds the command-line arguments for the application.
type Args struct {
	Config  string
	Version bool
	Verbose bool
}

// ParseArgs parses command-line flags and returns an Args struct.
func ParseArgs() *Args {
	// Define command-line flags
	configFlag := flag.String(
		"config",
		"config.yml",
		"Path to the configuration file.",
	)
	versionFlag := flag.Bool(
		"version",
		false,
		"Print the version information.",
	)
	flag.BoolVar(
		&log.Verbose,
		"verbose",
		false,
		"Enable verbose logging.",
	)

	// Parse and return the command-line arguments
	flag.Parse()
	return &Args{
		Config:  *configFlag,
		Version: *versionFlag,
		Verbose: log.Verbose,
	}
}

// main is the entry point of the application.
func main() {
	// Parse command-line arguments and load the configuration file
	args := ParseArgs()

	// Just print the version information if requested
	if args.Version {
		fmt.Print(version + "\n")
		return
	}

	// Load the config and registry
	utils.LoadConfig(args.Config)
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Get the target from the request parameters.
		targetParam := r.URL.Query().Get("target")
		target, err := utils.GetTarget(targetParam)
		if err != nil {
			http.Error(w, "Bad target", http.StatusBadRequest)
			return
		}

		// Create a new Prometheus registry for the target and setup collectors
		reg := prometheus.NewRegistry()
		reg.MustRegister(registry.NewMasterCollector(target))
		h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})

	// Start the exporter
	log.Info("main", "Starting pfsense_exporter on %s:%d", utils.Cfg.Address, utils.Cfg.Port)
	if err := http.ListenAndServe(utils.Cfg.Address+":"+strconv.Itoa(utils.Cfg.Port), nil); err != nil {
		log.Fatal("main", "Error starting HTTP server: %v", err)
	}
}
