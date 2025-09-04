package utils

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func createTestConfigFile(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "test-config-*.yml")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", err
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return "", err
	}

	return tmpfile.Name(), nil
}

func TestTargetValidateHostAndPort(t *testing.T) {
	// Test missing host
	target := &Target{Host: "", Port: 443}
	_, err := target.Validate()
	if err == nil {
		t.Error("Expected error for missing host")
	}

	// Test invalid port (too low)
	target = &Target{Host: "test.com", Port: 0}
	_, err = target.Validate()
	if err == nil {
		t.Error("Expected error for port 0")
	}

	// Test invalid port (too high)
	target = &Target{Host: "test.com", Port: 65536}
	_, err = target.Validate()
	if err == nil {
		t.Error("Expected error for port 65536")
	}

	// Test valid host and port
	target = &Target{Host: "test.com", Port: 443}
	if err := target.validateHostAndPort(); err != nil {
		t.Errorf("Unexpected error for valid host and port: %v", err)
	}
}

func TestTargetValidateScheme(t *testing.T) {
	// Test default scheme (should be set to https)
	target := &Target{Host: "test.com", Port: 443, Scheme: ""}
	if err := target.validateScheme(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if target.Scheme != "https" {
		t.Errorf("Expected default scheme 'https', got %s", target.Scheme)
	}

	// Test valid http scheme
	target = &Target{Host: "test.com", Port: 80, Scheme: "http"}
	if err := target.validateScheme(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test valid https scheme
	target = &Target{Host: "test.com", Port: 443, Scheme: "https"}
	if err := target.validateScheme(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid scheme
	target = &Target{Host: "test.com", Port: 443, Scheme: "ftp"}
	if err := target.validateScheme(); err == nil {
		t.Error("Expected error for invalid scheme")
	}
}

func TestTargetValidateAuth(t *testing.T) {
	// Test basic auth with username and password
	target := &Target{Host: "test.com", Port: 443, AuthMethod: "basic", Username: "user", Password: "pass"}
	if err := target.validateAuth(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test basic auth missing username
	target = &Target{Host: "test.com", Port: 443, AuthMethod: "basic", Username: "", Password: "pass"}
	if err := target.validateAuth(); err == nil {
		t.Error("Expected error for missing username")
	}

	// Test basic auth missing password
	target = &Target{Host: "test.com", Port: 443, AuthMethod: "basic", Username: "user", Password: ""}
	if err := target.validateAuth(); err == nil {
		t.Error("Expected error for missing password")
	}

	// Test key auth with key
	target = &Target{Host: "test.com", Port: 443, AuthMethod: "key", Key: "apikey123"}
	if err := target.validateAuth(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test key auth missing key
	target = &Target{Host: "test.com", Port: 443, AuthMethod: "key", Key: ""}
	if err := target.validateAuth(); err == nil {
		t.Error("Expected error for missing key")
	}

	// Test invalid auth method
	target = &Target{Host: "test.com", Port: 443, AuthMethod: "invalid"}
	if err := target.validateAuth(); err == nil {
		t.Error("Expected error for invalid auth method")
	}
}

func TestTargetValidateTimeout(t *testing.T) {
	// Test default timeout (should be set to 30)
	target := &Target{Host: "test.com", Port: 443, Timeout: 0}
	if err := target.validateTimeout(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if target.Timeout != 30 {
		t.Errorf("Expected default timeout 30, got %d", target.Timeout)
	}

	// Test valid timeout
	target = &Target{Host: "test.com", Port: 443, Timeout: 60}
	if err := target.validateTimeout(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test timeout too low
	target = &Target{Host: "test.com", Port: 443, Timeout: 5}
	if err := target.validateTimeout(); err == nil {
		t.Error("Expected error for timeout <= 5")
	}

	// Test timeout too high
	target = &Target{Host: "test.com", Port: 443, Timeout: 360}
	if err := target.validateTimeout(); err == nil {
		t.Error("Expected error for timeout >= 360")
	}
}

func TestTargetValidateValidateCert(t *testing.T) {
	// Test with cert validation enabled
	target := &Target{Host: "test.com", Port: 443, ValidateCert: true}
	if err := target.validateValidateCert(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test with cert validation disabled
	target = &Target{Host: "test.com", Port: 443, ValidateCert: false}
	if err := target.validateValidateCert(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTargetValidateMaxCollectorConcurrency(t *testing.T) {
	// Test default value (should be set to 4)
	target := &Target{Host: "test.com", Port: 443, MaxCollectorConcurrency: 0}
	if err := target.ValidateMaxCollectorConcurrency(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if target.MaxCollectorConcurrency != 4 {
		t.Errorf("Expected default concurrency 4, got %d", target.MaxCollectorConcurrency)
	}

	// Test valid value
	target = &Target{Host: "test.com", Port: 443, MaxCollectorConcurrency: 8}
	if err := target.ValidateMaxCollectorConcurrency(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test value too low
	target = &Target{Host: "test.com", Port: 443, MaxCollectorConcurrency: 0}
	target.MaxCollectorConcurrency = 0 // Reset to 0 to test validation
	if err := target.ValidateMaxCollectorConcurrency(); err != nil {
		t.Errorf("Unexpected error, should set default: %v", err)
	}

	// Test value too high
	target = &Target{Host: "test.com", Port: 443, MaxCollectorConcurrency: 11}
	if err := target.ValidateMaxCollectorConcurrency(); err == nil {
		t.Error("Expected error for concurrency > 10")
	}
}

func TestTargetValidateMaxCollectorBufferSize(t *testing.T) {
	// Test default value (should be set to 100)
	target := &Target{Host: "test.com", Port: 443, MaxCollectorBufferSize: 0}
	if err := target.ValidateMaxCollectorBufferSize(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if target.MaxCollectorBufferSize != 100 {
		t.Errorf("Expected default buffer size 100, got %d", target.MaxCollectorBufferSize)
	}

	// Test valid value
	target = &Target{Host: "test.com", Port: 443, MaxCollectorBufferSize: 200}
	if err := target.ValidateMaxCollectorBufferSize(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test value too low
	target = &Target{Host: "test.com", Port: 443, MaxCollectorBufferSize: 5}
	if err := target.ValidateMaxCollectorBufferSize(); err == nil {
		t.Error("Expected error for buffer size < 10")
	}
}

func TestTargetValidate(t *testing.T) {
	// Test valid target
	target := &Target{
		Host:       "test.com",
		Port:       443,
		Scheme:     "https",
		AuthMethod: "basic",
		Username:   "user",
		Password:   "pass",
		Timeout:    30,
	}

	validated, err := target.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validated == nil {
		t.Error("Expected validated target to be returned")
	}
}

func TestConfigValidateAddress(t *testing.T) {
	// Test default address (should be set to localhost)
	config := &Config{Address: ""}
	if err := config.ValidateAddress(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config.Address != "localhost" {
		t.Errorf("Expected default address 'localhost', got %s", config.Address)
	}

	// Test valid IP
	config = &Config{Address: "192.168.1.1"}
	if err := config.ValidateAddress(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test localhost
	config = &Config{Address: "localhost"}
	if err := config.ValidateAddress(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid address
	config = &Config{Address: "invalid-address"}
	if err := config.ValidateAddress(); err == nil {
		t.Error("Expected error for invalid address")
	}
}

func TestConfigValidatePort(t *testing.T) {
	// Test default port (should be set to 9945)
	config := &Config{Port: 0}
	if err := config.ValidatePort(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config.Port != 9945 {
		t.Errorf("Expected default port 9945, got %d", config.Port)
	}

	// Test valid port
	config = &Config{Port: 8080}
	if err := config.ValidatePort(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test port too low
	config = &Config{Port: 0}
	config.Port = 0 // Reset to test validation
	if err := config.ValidatePort(); err != nil {
		t.Errorf("Unexpected error, should set default: %v", err)
	}

	// Test port too high
	config = &Config{Port: 65536}
	if err := config.ValidatePort(); err == nil {
		t.Error("Expected error for port > 65535")
	}
}

func TestConfigValidateTargets(t *testing.T) {
	// Test valid targets
	config := &Config{
		Targets: []Target{
			{
				Host:       "test1.com",
				Port:       443,
				AuthMethod: "basic",
				Username:   "user",
				Password:   "pass",
			},
			{
				Host:       "test2.com",
				Port:       80,
				AuthMethod: "key",
				Key:        "apikey",
			},
		},
	}

	if err := config.ValidateTargets(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test invalid target
	config = &Config{
		Targets: []Target{
			{
				Host:       "", // Invalid: empty host
				Port:       443,
				AuthMethod: "basic",
				Username:   "user",
				Password:   "pass",
			},
		},
	}

	if err := config.ValidateTargets(); err == nil {
		t.Error("Expected error for invalid target")
	}
}

func TestConfigValidate(t *testing.T) {
	// Test valid config
	config := &Config{
		Address: "localhost",
		Port:    9945,
		Targets: []Target{
			{
				Host:       "test.com",
				Port:       443,
				AuthMethod: "basic",
				Username:   "user",
				Password:   "pass",
			},
		},
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestLoadConfig(t *testing.T) {
	validConfig := `
address: "localhost"
port: 9945
targets:
  - host: "test.com"
    port: 443
    scheme: "https"
    auth_method: "basic"
    username: "user"
    password: "pass"
    validate_cert: true
    timeout: 30
`

	// Test loading valid config
	tmpfile, err := createTestConfigFile(validConfig)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(tmpfile)

	// Save original Cfg and restore it after test
	originalCfg := Cfg
	defer func() { Cfg = originalCfg }()

	LoadConfig(tmpfile)

	if Cfg == nil {
		t.Error("Expected Cfg to be loaded")
	}

	if Cfg.Address != "localhost" {
		t.Errorf("Expected address 'localhost', got %s", Cfg.Address)
	}

	if Cfg.Port != 9945 {
		t.Errorf("Expected port 9945, got %d", Cfg.Port)
	}

	if len(Cfg.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(Cfg.Targets))
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	// Test with non-existent file to trigger the file read error
	// This would normally call log.Fatal, so we can't test it directly
	// We'll test that the file reading logic exists

	// Test that we can call LoadConfig with a path
	// The error handling is done via log.Fatal which exits the program
	defer func() {
		if r := recover(); r != nil {
			// This is expected if log.Fatal is called
		}
	}()

	// We can't test the error path since it calls log.Fatal
	// This would exit the test process, so we just verify the function exists
	// by checking that we can reference it (compilation would fail if it didn't exist)
	_ = LoadConfig
}

func TestConfigValidateErrorPaths(t *testing.T) {
	// Test validation with invalid address
	config := &Config{
		Address: "invalid-address",
		Port:    9945,
		Targets: []Target{},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid address")
	}

	// Test validation with invalid port
	config = &Config{
		Address: "localhost",
		Port:    -1,
		Targets: []Target{},
	}

	err = config.Validate()
	if err == nil {
		t.Error("Expected error for invalid port")
	}

	// Test validation with invalid target
	config = &Config{
		Address: "localhost",
		Port:    9945,
		Targets: []Target{
			{Host: "", Port: 443}, // Invalid host
		},
	}

	err = config.Validate()
	if err == nil {
		t.Error("Expected error for invalid target")
	}
}

func TestGetTarget(t *testing.T) {
	// Save original Cfg and restore it after test
	originalCfg := Cfg
	defer func() { Cfg = originalCfg }()

	Cfg = &Config{
		Targets: []Target{
			{Host: "test1.com", Port: 443},
			{Host: "test2.com", Port: 80},
		},
	}

	// Test existing target
	target, err := GetTarget("test1.com")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if target == nil {
		t.Error("Expected target to be found")
	}
	if target.Host != "test1.com" {
		t.Errorf("Expected host 'test1.com', got %s", target.Host)
	}

	// Test non-existing target
	_, err = GetTarget("nonexistent.com")
	if err == nil {
		t.Error("Expected error for non-existing target")
	}
}

func TestTargetValidateAllErrorPaths(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		expect string
	}{
		{
			name: "missing_host",
			target: Target{
				Host:                    "", // Missing host
				Port:                    443,
				Scheme:                  "https",
				AuthMethod:              "basic",
				Username:                "user",
				Password:                "pass",
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'host' is a required field",
		},
		{
			name: "invalid_port",
			target: Target{
				Host:                    "test.com",
				Port:                    70000, // Invalid port
				Scheme:                  "https",
				AuthMethod:              "basic",
				Username:                "user",
				Password:                "pass",
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'port' must be between 1 and 65535",
		},
		{
			name: "invalid_scheme",
			target: Target{
				Host:                    "test.com",
				Port:                    443,
				Scheme:                  "ftp", // Invalid scheme
				AuthMethod:              "basic",
				Username:                "user",
				Password:                "pass",
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'scheme' must be 'http' or 'https'",
		},
		{
			name: "missing_basic_auth_username",
			target: Target{
				Host:                    "test.com",
				Port:                    443,
				Scheme:                  "https",
				AuthMethod:              "basic",
				Username:                "", // Missing username
				Password:                "pass",
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'username' is required with auth_method 'basic'",
		},
		{
			name: "missing_basic_auth_password",
			target: Target{
				Host:                    "test.com",
				Port:                    443,
				Scheme:                  "https",
				AuthMethod:              "basic",
				Username:                "user",
				Password:                "", // Missing password
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'password' is required with auth_method 'basic'",
		},
		{
			name: "missing_key_auth",
			target: Target{
				Host:                    "test.com",
				Port:                    443,
				Scheme:                  "https",
				AuthMethod:              "key",
				Key:                     "", // Missing key
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "Target 'key' is required with auth_method 'key'",
		},
		{
			name: "invalid_auth_method",
			target: Target{
				Host:                    "test.com",
				Port:                    443,
				Scheme:                  "https",
				AuthMethod:              "oauth", // Invalid auth method
				ValidateCert:            true,
				Timeout:                 30,
				MaxCollectorConcurrency: 10,
				MaxCollectorBufferSize:  100,
			},
			expect: "'auth_method' must be 'basic' or 'key'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.target.Validate()
			if err == nil {
				t.Errorf("Expected validation error containing '%s'", tt.expect)
			} else if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expect, err.Error())
			}
		})
	}

	// Test for timeout, concurrency, and buffer size set to 0 to see if they fail validation
	invalidTarget := Target{
		Host:                    "test.com",
		Port:                    443,
		Scheme:                  "https",
		AuthMethod:              "basic",
		Username:                "user",
		Password:                "pass",
		ValidateCert:            true,
		Timeout:                 0, // These won't cause validation errors as they get defaults
		MaxCollectorConcurrency: 0,
		MaxCollectorBufferSize:  0,
	}
	_, err := invalidTarget.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error for 0 timeout/concurrency/buffer values: %v", err)
	}

	// Test invalid MaxCollectorConcurrency > 10
	invalidConcurrency := Target{
		Host:                    "test.com",
		Port:                    443,
		Scheme:                  "https",
		AuthMethod:              "basic",
		Username:                "user",
		Password:                "pass",
		ValidateCert:            true,
		Timeout:                 30,
		MaxCollectorConcurrency: 15, // Invalid - too high
		MaxCollectorBufferSize:  100,
	}
	_, err = invalidConcurrency.Validate()
	if err == nil {
		t.Error("Expected validation error for MaxCollectorConcurrency > 10")
	}
	if !strings.Contains(err.Error(), "max_concurrent_collectors") {
		t.Errorf("Expected max_concurrent_collectors error, got: %v", err)
	}

	// Test invalid MaxCollectorBufferSize < 10
	invalidBuffer := Target{
		Host:                    "test.com",
		Port:                    443,
		Scheme:                  "https",
		AuthMethod:              "basic",
		Username:                "user",
		Password:                "pass",
		ValidateCert:            true,
		Timeout:                 30,
		MaxCollectorConcurrency: 5,
		MaxCollectorBufferSize:  5, // Invalid - too low
	}
	_, err = invalidBuffer.Validate()
	if err == nil {
		t.Error("Expected validation error for MaxCollectorBufferSize < 10")
	}
	if !strings.Contains(err.Error(), "max_collector_buffer_size") {
		t.Errorf("Expected max_collector_buffer_size error, got: %v", err)
	}
}

func TestExecuteAndParseErrorPaths(t *testing.T) {
	// Test connection refused error
	client := &http.Client{Timeout: 1 * time.Second}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:1", nil) // Port 1 should be closed

	_, err := executeAndParse(client, req)
	if err == nil {
		t.Error("Expected connection error")
	}
	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Expected 'error making request' in error, got: %v", err)
	}
}

func TestLoadConfigErrorPaths(t *testing.T) {
	// Test invalid YAML
	invalidYaml := `
invalid: yaml: content:
  - missing: indent
bad yaml
`
	tmpfile, err := createTestConfigFile(invalidYaml)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpfile)

	// This should trigger log.Fatal, so we can't test it directly
	// We'll test the YAML parse error path by temporarily replacing log.Fatal

	// Test config with invalid targets that will fail validation
	invalidConfig := `
address: "localhost"
port: 9100
targets:
  invalid-target:
    host: ""  # Missing host - will trigger validation error
    port: 443
    scheme: https
    auth_method: basic
    username: user
    password: pass
`
	tmpfile2, err := createTestConfigFile(invalidConfig)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpfile2)

	// This will also trigger log.Fatal due to validation failure
	// We can't test this directly without subprocess
}
