package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pfrest/pfsense_exporter/internal/log"
)

// Response describes the API JSON response structure.
type Response struct {
	Code       int             `json:"code"`
	Status     string          `json:"status"`
	ResponseID string          `json:"response_id"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

// Request performs an HTTP request by coordinating client creation,
// request execution, and response parsing.
func Request(target *Target, method string, endpoint string) (*Response, error) {
	// 1. Configure and create the HTTP client.
	client := newHTTPClient(target)

	// 2. Format the full URL.
	fullURL := formatURL(target, endpoint)

	// 3. Create the basic HTTP request object.
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// 4. Set the necessary headers.
	setHeaders(req, target)

	// 5. Execute the request and parse the response.
	return executeAndParse(client, req)
}

// newHTTPClient creates and configures an HTTP client based on the target's settings.
func newHTTPClient(target *Target) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !target.ValidateCert},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(target.Timeout) * time.Second,
	}
}

// formatURL builds the complete request URL from the target and endpoint.
func formatURL(target *Target, endpoint string) string {
	return fmt.Sprintf("%s://%s:%d%s", target.Scheme, target.Host, target.Port, endpoint)
}

// executeAndParse sends the request, validates the status, and parses the JSON response.
func executeAndParse(client *http.Client, req *http.Request) (*Response, error) {
	// Log the request details
	log.Debug("http", "sending %s request to %s", req.Method, req.URL)

	// Send the request to the target
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Log the response details
	log.Debug("http", "received %d response from %s: %s", resp.StatusCode, req.URL, body)

	// Unmarshal the body into our Response struct
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	// Ensure we received a successful response
	if response.Code != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code %d: %s", response.Code, response.Message)
	}

	return &response, nil
}

// setHeaders sets the necessary headers for the HTTP request.
func setHeaders(req *http.Request, target *Target) {
	// Only accept responses from the API in JSON format
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Configure authentication based on the target's AuthMethod
	switch target.AuthMethod {
	case "basic":
		req.SetBasicAuth(target.Username, target.Password)
	case "key":
		req.Header.Set("X-API-Key", target.Key)
	}
}
