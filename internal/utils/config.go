package utils

import (
	"fmt"
	"net"
	"os"

	"github.com/pfrest/pfsense_exporter/internal/log"
	"gopkg.in/yaml.v3"
)

var Cfg *Config

// Config is the top-level structure for the YAML config file.
type Config struct {
	Address string   `yaml:"address"` // Address is the address the exporter will bind to.
	Port    int      `yaml:"port"`    // Port is the port the exporter will listen on.
	Targets []Target `yaml:"targets"` // Targets contains the configuration for the targets to scrape.
}

// Target represents a single target object in the YAML.
type Target struct {
	Host                    string   `yaml:"host"`                      // Host is the hostname or IP address of the target.
	Port                    int      `yaml:"port"`                      // Port is the port number of the target.
	Scheme                  string   `yaml:"scheme"`                    // Scheme is the URL scheme (http or https) to use for the target.
	AuthMethod              string   `yaml:"auth_method"`               // AuthMethod is the authentication method to use for the target.
	Username                string   `yaml:"username,omitempty"`        // Username is the username for basic authentication.
	Password                string   `yaml:"password,omitempty"`        // Password is the password for basic authentication.
	Key                     string   `yaml:"key,omitempty"`             // Key is the API key to use for key-based authentication.
	ValidateCert            bool     `yaml:"validate_cert"`             // ValidateCert determines whether to validate the TLS certificate.
	Timeout                 int      `yaml:"timeout"`                   // Timeout is the timeout for requests to the target.
	Collectors              []string `yaml:"collectors"`                // Collectors is the list of collectors to use for the target.
	MaxCollectorConcurrency int      `yaml:"max_collector_concurrency"` // MaxCollectorConcurrency is the maximum number of collectors allowed to run concurrently.
	MaxCollectorBufferSize  int      `yaml:"max_collector_buffer_size"` // MaxCollectorBufferSize is the maximum size of the collector's metric buffer.
}

// Validate validates the fields of a given Target.
func (t *Target) Validate() (*Target, error) {
	if err := t.validateHostAndPort(); err != nil {
		return nil, err
	}
	if err := t.validateScheme(); err != nil {
		return nil, err
	}
	if err := t.validateAuth(); err != nil {
		return nil, err
	}
	if err := t.validateTimeout(); err != nil {
		return nil, err
	}
	if err := t.validateValidateCert(); err != nil {
		return nil, err
	}
	if err := t.ValidateMaxCollectorConcurrency(); err != nil {
		return nil, err
	}
	if err := t.ValidateMaxCollectorBufferSize(); err != nil {
		return nil, err
	}

	return t, nil
}

// validateHostAndPort checks that the host and port are present.
func (t *Target) validateHostAndPort() error {
	if t.Host == "" {
		return fmt.Errorf("Target 'host' is a required field")
	}
	if t.Port < 1 || t.Port > 65535 {
		return fmt.Errorf("Target 'port' must be between 1 and 65535 for host '%s'", t.Host)
	}
	return nil
}

// validateScheme ensures the scheme is either 'http' or 'https'.
func (t *Target) validateScheme() error {
	// Default to https
	if t.Scheme == "" {
		t.Scheme = "https"
	}

	// Ensure value is either 'http' or 'https'
	if t.Scheme != "http" && t.Scheme != "https" {
		return fmt.Errorf("Target 'scheme' must be 'http' or 'https' for host '%s'", t.Host)
	}

	return nil
}

// validateAuth checks the auth_method and its dependent fields.
func (t *Target) validateAuth() error {
	// Ensure a valid auth method is set
	switch t.AuthMethod {
	case "basic":
		if t.Username == "" {
			return fmt.Errorf("Target 'username' is required with auth_method 'basic' on host '%s'", t.Host)
		}
		if t.Password == "" {
			return fmt.Errorf("Target 'password' is required with auth_method 'basic' on host '%s'", t.Host)
		}
		log.Debug("config", "using basic auth for %s", t.Host)
	case "key":
		if t.Key == "" {
			return fmt.Errorf("Target 'key' is required with auth_method 'key' on host '%s'", t.Host)
		}
		log.Debug("config", "using API key auth for %s", t.Host)
	default:
		return fmt.Errorf("'auth_method' must be 'basic' or 'key' for host '%s'", t.Host)
	}
	return nil
}

// validateTimeout checks that the timeout is a positive integer greater than 5 but less than 360
func (t *Target) validateTimeout() error {
	// Default to 30 seconds if not set
	if t.Timeout == 0 {
		t.Timeout = 30
	}

	// Check if the timeout is within the valid range
	if t.Timeout <= 5 || t.Timeout >= 360 {
		return fmt.Errorf("Target 'timeout' must be between 5 and 360 seconds for host '%s'", t.Host)
	}
	return nil
}

// validateValidateCert checks that the validate_cert field is set correctly.
func (t *Target) validateValidateCert() error {
	if t.ValidateCert {
		log.Debug("config", "certificate validation is enabled for target %s", t.Host)
	} else {
		log.Warn("config", "certificate validation is disabled for target %s, your credentials may be at risk!", t.Host)
	}
	return nil
}

// ValidateTargets checks each individual Target in a Config for correctness.
func (c *Config) ValidateTargets() error {
	for idx, target := range c.Targets {
		validated_target, err := target.Validate()
		if err != nil {
			return fmt.Errorf("validation error for target %d: %w", idx, err)
		}
		c.Targets[idx] = *validated_target
	}
	return nil
}

// ValidateAddress checks that the global 'address' field is set and is a valid IP
func (c *Config) ValidateAddress() error {
	// Default to localhost if not set
	if c.Address == "" {
		c.Address = "localhost"
	}

	// Ensure value is an IP address or localhost
	if net.ParseIP(c.Address) == nil && c.Address != "localhost" {
		return fmt.Errorf("global 'address' must be a valid IP address")
	}

	return nil
}

// ValidatePort checks that the global 'port' field is set and is a valid port number.
func (c *Config) ValidatePort() error {
	// Default to 9945 if not set
	if c.Port == 0 {
		c.Port = 9945
	}

	// Check if the port is within the valid range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("global 'port' must be between 1 and 65535")
	}
	return nil
}

// ValidateMaxCollectorConcurrency checks that the global 'max_concurrent_collectors' field is set and is a valid number.
func (t *Target) ValidateMaxCollectorConcurrency() error {
	// Default to 4 if not set
	if t.MaxCollectorConcurrency == 0 {
		t.MaxCollectorConcurrency = 4
	}

	// Check if the max_concurrent_collectors is within the valid range
	if t.MaxCollectorConcurrency < 1 || t.MaxCollectorConcurrency > 10 {
		return fmt.Errorf("global 'max_concurrent_collectors' must be between 1 and 10")
	}

	// Log the number of concurrent collectors
	log.Debug("config", "target %s using %d concurrent collectors", t.Host, t.MaxCollectorConcurrency)
	return nil
}

// ValidateMaxCollectorBufferSize checks that the global 'max_collector_buffer_size' field is set and is a valid number.
func (t *Target) ValidateMaxCollectorBufferSize() error {
	// Default to 100 if not set
	if t.MaxCollectorBufferSize == 0 {
		t.MaxCollectorBufferSize = 100
	}

	// Check if the max_collector_buffer_size is within the valid range
	if t.MaxCollectorBufferSize < 10 {
		return fmt.Errorf("global 'max_collector_buffer_size' must be at least 10")
	}

	// Log the buffer size
	log.Debug("config", "target %s using collector buffer size of %d", t.Host, t.MaxCollectorBufferSize)
	return nil
}

// Validate checks the entire Config for correctness.
func (c *Config) Validate() error {
	if err := c.ValidateAddress(); err != nil {
		return fmt.Errorf("address validation failed: %w", err)
	}
	if err := c.ValidatePort(); err != nil {
		return fmt.Errorf("port validation failed: %w", err)
	}
	if err := c.ValidateTargets(); err != nil {
		return fmt.Errorf("target validation failed: %w", err)
	}
	return nil
}

// LoadConfig reads a configuration file from a given path, parses it,
// validates it, and loads the global Cfg variable.
func LoadConfig(path string) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("config", "error reading YAML file at %s: %s", path, err)
	}

	if err := yaml.Unmarshal(yamlFile, &Cfg); err != nil {
		log.Fatal("config", "error unmarshaling YAML: %s", err)
	}

	if err := Cfg.Validate(); err != nil {
		log.Fatal("config", "error validating configuration: %s", err)
	}
}

// GetTarget obtains the Target configuration for a specific target host.
func GetTarget(host string) (*Target, error) {
	for _, target := range Cfg.Targets {
		if target.Host == host {
			return &target, nil
		}
	}
	return nil, fmt.Errorf("target not configured: %s", host)
}
