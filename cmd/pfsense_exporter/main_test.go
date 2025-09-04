package main

import (
	"net/http"
	"os"
	"testing"
)

// TestMain is a basic test to ensure we can start the exporter with a basic config.
func TestMain(t *testing.T) {
	// Write a temporary config file to ./config.yml
	config := `
address: "localhost"
port: 28080
targets:
  - host: "127.0.0.1"
    port: 443
    username: "admin"
    password: "pfsense"
    auth_method: "basic"
`
	// Create the config file
	if err := os.WriteFile("./config.yml", []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Clean up the config file after the test
	defer os.Remove("./config.yml")

	// Start the main function in a goroutine
	go func() {
		main()
	}()

	// Make a request to the /metrics endpoint
	resp, err := http.Get("http://localhost:28080/metrics")
	if err != nil {
		t.Fatalf("Failed to make GET request to /metrics: %v", err)
	}
	defer resp.Body.Close()
}
