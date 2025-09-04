package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	// Test successful request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and endpoint
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/test/endpoint" {
			t.Errorf("Expected /test/endpoint, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Expected Accept header to be application/json")
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Error("Expected Content-Type header to be application/x-www-form-urlencoded")
		}

		// Return successful response
		response := Response{
			Code:       200,
			Status:     "success",
			ResponseID: "test-id",
			Message:    "success",
			Data:       json.RawMessage(`{"test": "data"}`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Parse server URL to get host and port
	serverURL := server.URL[7:] // Remove http://
	host := "127.0.0.1"
	port := 80
	for i, c := range serverURL {
		if c == ':' {
			host = serverURL[:i]
			remaining := serverURL[i+1:]
			// Parse port
			var err error
			if port, err = strconv.Atoi(remaining); err != nil {
				port = 80
			}
			break
		}
	}

	target := &Target{
		Host:         host,
		Port:         port,
		Scheme:       "http",
		AuthMethod:   "basic",
		Username:     "user",
		Password:     "pass",
		ValidateCert: false,
		Timeout:      30,
	}

	// This is a partial test - we can't easily test the full Request function
	// without significant mocking infrastructure
	// But we can test the components

	// Test formatURL
	url := formatURL(target, "/test/endpoint")
	expectedURL := fmt.Sprintf("http://%s:%d/test/endpoint", host, port)
	if url != expectedURL {
		t.Errorf("Expected %s, got %s", expectedURL, url)
	}
}

func TestNewHTTPClient(t *testing.T) {
	target := &Target{
		ValidateCert: true,
		Timeout:      45,
	}

	client := newHTTPClient(target)

	if client == nil {
		t.Error("Expected client to be created")
	}

	if client.Timeout != 45*time.Second {
		t.Errorf("Expected timeout 45s, got %v", client.Timeout)
	}

	// Test with cert validation disabled
	target.ValidateCert = false
	client = newHTTPClient(target)

	if client == nil {
		t.Error("Expected client to be created")
	}
}

func TestFormatURL(t *testing.T) {
	target := &Target{
		Scheme: "https",
		Host:   "example.com",
		Port:   443,
	}

	url := formatURL(target, "/api/v1/test")
	expected := "https://example.com:443/api/v1/test"

	if url != expected {
		t.Errorf("Expected %s, got %s", expected, url)
	}

	// Test with HTTP
	target.Scheme = "http"
	target.Port = 80
	url = formatURL(target, "/test")
	expected = "http://example.com:80/test"

	if url != expected {
		t.Errorf("Expected %s, got %s", expected, url)
	}
}

func TestSetHeaders(t *testing.T) {
	// Test basic auth
	req := httptest.NewRequest("GET", "http://example.com", nil)
	target := &Target{
		AuthMethod: "basic",
		Username:   "testuser",
		Password:   "testpass",
	}

	setHeaders(req, target)

	if req.Header.Get("Accept") != "application/json" {
		t.Error("Expected Accept header to be application/json")
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Error("Expected Content-Type header to be application/x-www-form-urlencoded")
	}

	username, password, ok := req.BasicAuth()
	if !ok {
		t.Error("Expected basic auth to be set")
	}
	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", username)
	}
	if password != "testpass" {
		t.Errorf("Expected password 'testpass', got %s", password)
	}

	// Test API key auth
	req = httptest.NewRequest("GET", "http://example.com", nil)
	target = &Target{
		AuthMethod: "key",
		Key:        "test-api-key",
	}

	setHeaders(req, target)

	if req.Header.Get("X-API-Key") != "test-api-key" {
		t.Errorf("Expected X-API-Key header to be 'test-api-key', got %s", req.Header.Get("X-API-Key"))
	}
}

func TestExecuteAndParse(t *testing.T) {
	// Test successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Code:       200,
			Status:     "success",
			ResponseID: "test-response-id",
			Message:    "Operation successful",
			Data:       json.RawMessage(`{"key": "value"}`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := executeAndParse(client, req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Code != 200 {
		t.Errorf("Expected code 200, got %d", resp.Code)
	}
	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %s", resp.Status)
	}
	if resp.ResponseID != "test-response-id" {
		t.Errorf("Expected response ID 'test-response-id', got %s", resp.ResponseID)
	}

	// Test error response (non-200 status)
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Code:    400,
			Status:  "error",
			Message: "Bad request",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server2.Close()

	req2, err := http.NewRequest("GET", server2.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = executeAndParse(client, req2)
	if err == nil {
		t.Error("Expected error for non-200 status code")
	}

	// Test invalid JSON response
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server3.Close()

	req3, err := http.NewRequest("GET", server3.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = executeAndParse(client, req3)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestResponse(t *testing.T) {
	// Test Response struct
	data := json.RawMessage(`{"test": "data"}`)
	response := Response{
		Code:       200,
		Status:     "success",
		ResponseID: "test-id",
		Message:    "success message",
		Data:       data,
	}

	if response.Code != 200 {
		t.Errorf("Expected code 200, got %d", response.Code)
	}
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %s", response.Status)
	}
	if response.ResponseID != "test-id" {
		t.Errorf("Expected response ID 'test-id', got %s", response.ResponseID)
	}
	if response.Message != "success message" {
		t.Errorf("Expected message 'success message', got %s", response.Message)
	}
	if string(response.Data) != `{"test": "data"}` {
		t.Errorf("Expected data '{\"test\": \"data\"}', got %s", string(response.Data))
	}
}

func TestRequestFull(t *testing.T) {
	tests := []struct {
		name           string
		target         *Target
		method         string
		endpoint       string
		serverResponse string
		serverStatus   int
		expectedError  string
		expectedCode   int
	}{
		{
			name: "successful_basic_auth_request",
			target: &Target{
				Host:         "test.com",
				Port:         443,
				Scheme:       "https",
				Username:     "testuser",
				Password:     "testpass",
				AuthMethod:   "basic",
				ValidateCert: false,
				Timeout:      30,
			},
			method:       "GET",
			endpoint:     "/api/v1/test",
			serverStatus: 200,
			serverResponse: `{
				"code": 200,
				"status": "ok",
				"response_id": "test123",
				"message": "success",
				"data": {"key": "value"}
			}`,
			expectedCode: 200,
		},
		{
			name: "successful_api_key_request",
			target: &Target{
				Host:         "test.com",
				Port:         443,
				Scheme:       "https",
				Key:          "test-api-key",
				AuthMethod:   "key",
				ValidateCert: true,
				Timeout:      30,
			},
			method:       "POST",
			endpoint:     "/api/v1/create",
			serverStatus: 200,
			serverResponse: `{
				"code": 200,
				"status": "created",
				"response_id": "create123",
				"message": "resource created",
				"data": {"id": 42}
			}`,
			expectedCode: 200,
		},
		{
			name: "api_error_response",
			target: &Target{
				Host:         "test.com",
				Port:         80,
				Scheme:       "http",
				Username:     "user",
				Password:     "pass",
				AuthMethod:   "basic",
				ValidateCert: false,
				Timeout:      10,
			},
			method:       "GET",
			endpoint:     "/api/v1/error",
			serverStatus: 200,
			serverResponse: `{
				"code": 400,
				"status": "error",
				"response_id": "error123",
				"message": "Bad request",
				"data": null
			}`,
			expectedError: "received non-200 status code 400: Bad request",
		},
		{
			name: "invalid_json_response",
			target: &Target{
				Host:         "test.com",
				Port:         8080,
				Scheme:       "http",
				Username:     "user",
				Password:     "pass",
				AuthMethod:   "basic",
				ValidateCert: false,
				Timeout:      5,
			},
			method:         "GET",
			endpoint:       "/api/v1/invalid",
			serverStatus:   200,
			serverResponse: `{"invalid": json}`,
			expectedError:  "error unmarshalling response body:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, r.Method)
				}
				if r.URL.Path != tt.endpoint {
					t.Errorf("Expected path %s, got %s", tt.endpoint, r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("Accept") != "application/json" {
					t.Error("Expected Accept header to be application/json")
				}
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Error("Expected Content-Type header to be application/x-www-form-urlencoded")
				}

				// Verify authentication
				switch tt.target.AuthMethod {
				case "basic":
					username, password, ok := r.BasicAuth()
					if !ok {
						t.Error("Expected basic auth")
					}
					if username != tt.target.Username || password != tt.target.Password {
						t.Errorf("Expected credentials %s:%s, got %s:%s",
							tt.target.Username, tt.target.Password, username, password)
					}
				case "key":
					if r.Header.Get("X-API-Key") != tt.target.Key {
						t.Errorf("Expected API key %s, got %s", tt.target.Key, r.Header.Get("X-API-Key"))
					}
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Parse server URL to update target
			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("Failed to parse server URL: %v", err)
			}

			port, err := strconv.Atoi(serverURL.Port())
			if err != nil {
				t.Fatalf("Failed to parse server port: %v", err)
			}

			// Update target with test server details
			tt.target.Host = serverURL.Hostname()
			tt.target.Port = port
			tt.target.Scheme = serverURL.Scheme

			// Execute request
			response, err := Request(tt.target, tt.method, tt.endpoint)

			// Check error expectations
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			// Check success expectations
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("Expected response, got nil")
				return
			}

			if response.Code != tt.expectedCode {
				t.Errorf("Expected code %d, got %d", tt.expectedCode, response.Code)
			}
		})
	}
}

func TestRequestNetworkError(t *testing.T) {
	target := &Target{
		Host:         "nonexistent.invalid.domain.test",
		Port:         443,
		Scheme:       "https",
		Username:     "user",
		Password:     "pass",
		AuthMethod:   "basic",
		ValidateCert: false,
		Timeout:      1, // Short timeout to fail fast
	}

	_, err := Request(target, "GET", "/api/test")
	if err == nil {
		t.Error("Expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Expected network error, got: %v", err)
	}
}

func TestRequestInvalidMethod(t *testing.T) {
	target := &Target{
		Host:         "test.com",
		Port:         443,
		Scheme:       "https",
		Username:     "user",
		Password:     "pass",
		AuthMethod:   "basic",
		ValidateCert: false,
		Timeout:      30,
	}

	// Test with invalid HTTP method
	_, err := Request(target, "INVALID\nMETHOD", "/api/test")
	if err == nil {
		t.Error("Expected error for invalid method, got nil")
	}
	if !strings.Contains(err.Error(), "error creating request") {
		t.Errorf("Expected request creation error, got: %v", err)
	}
}
