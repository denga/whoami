package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"Valid integer", "TEST_INT", "8080", 80, 8080},
		{"Invalid integer", "TEST_INT", "invalid", 80, 80},
		{"Empty value", "TEST_INT", "", 80, 80},
		{"Unset env var", "UNSET_VAR", "", 80, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.envKey, tt.envValue); err != nil {
					t.Fatalf("Failed to set environment variable: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.envKey); err != nil {
						t.Logf("Failed to unset environment variable: %v", err)
					}
				}()
			}

			result := getEnvAsInt(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRealIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.1"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name:       "Fall back to remote address",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name:       "Invalid remote address",
			headers:    map[string]string{},
			remoteAddr: "invalid-addr",
			expected:   "invalid-addr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := getRealIP(req)
			if result != tt.expected {
				t.Errorf("getRealIP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetLocalIPs(t *testing.T) {
	ips := getLocalIPs()

	// We can't predict the exact IPs, but we can check the format
	for _, ip := range ips {
		parts := strings.Split(ip, ".")
		if len(parts) != 4 {
			t.Errorf("Invalid IP format: %s", ip)
		}

		for _, part := range parts {
			if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
				t.Errorf("Invalid IP octet: %s in %s", part, ip)
			}
		}
	}
}

func TestRootHandler(t *testing.T) {
	// Set up config for testing
	originalConfig := config
	config = Config{Port: 8080, Name: "test-server", Verbose: false}
	defer func() { config = originalConfig }()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	rr := httptest.NewRecorder()

	rootHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	body := rr.Body.String()

	// Check that response contains expected fields
	expectedFields := []string{
		"Hostname:",
		"Name: test-server",
		"IP:",
		"RemoteAddr:",
		"Host:",
		"URL: /",
		"Method: GET",
		"RealIP: 192.168.1.1",
		"Protocol:",
		"OS:",
		"Architecture:",
		"Runtime:",
		"Time:",
		"Version:",
		"Headers:",
		"User-Agent: test-agent",
		"Environment:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected field '%s' not found in response", field)
		}
	}
}

// validateRequestInfo validates common RequestInfo fields.
func validateRequestInfo(t *testing.T, info *RequestInfo) {
	t.Helper()

	if len(info.Environment) == 0 {
		t.Error("Expected environment variables to be populated")
	}
	if info.OS == "" {
		t.Error("Expected OS to be populated")
	}
	if info.Architecture == "" {
		t.Error("Expected Architecture to be populated")
	}
	if info.Runtime == "" {
		t.Error("Expected Runtime to be populated")
	}
	if info.Time == "" {
		t.Error("Expected Time to be populated")
	}
	if info.Version == "" {
		t.Error("Expected Version to be populated")
	}
}

func TestAPIHandler(t *testing.T) {
	// Set up config for testing
	originalConfig := config
	config = Config{Port: 8080, Name: "test-server", Verbose: false}
	defer func() { config = originalConfig }()

	req := httptest.NewRequest("POST", "/api?test=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Real-IP", "10.0.0.1")

	rr := httptest.NewRecorder()
	apiHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != ContentTypeJSON {
		t.Errorf("Expected Content-Type %s, got %s", ContentTypeJSON, contentType)
	}

	var info RequestInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	// Verify specific values
	if info.Name != "test-server" {
		t.Errorf("Expected Name 'test-server', got '%s'", info.Name)
	}
	if info.URL != "/api?test=value" {
		t.Errorf("Expected URL '/api?test=value', got '%s'", info.URL)
	}
	if info.Method != "POST" {
		t.Errorf("Expected Method 'POST', got '%s'", info.Method)
	}
	if info.RealIP != "10.0.0.1" {
		t.Errorf("Expected RealIP '10.0.0.1', got '%s'", info.RealIP)
	}
	if info.Headers["User-Agent"] != "test-agent" {
		t.Errorf("Expected User-Agent 'test-agent', got '%s'", info.Headers["User-Agent"])
	}

	// Validate common fields
	validateRequestInfo(t, &info)
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != ContentTypeJSON {
		t.Errorf("Expected Content-Type %s, got %s", ContentTypeJSON, contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}

	if response["time"] == "" {
		t.Error("Expected time to be populated")
	}

	if response["version"] == "" {
		t.Error("Expected version to be populated")
	}
}

func TestFormatAsText(t *testing.T) {
	info := &RequestInfo{
		Hostname:     "test-host",
		Name:         "test-name",
		IP:           []string{"192.168.1.1", "10.0.0.1"},
		RemoteAddr:   "192.168.1.100:1234",
		Host:         "example.com",
		URL:          "/test?param=value",
		Method:       "GET",
		RealIP:       "192.168.1.100",
		Protocol:     "HTTP/1.1",
		Headers:      map[string]string{"User-Agent": "test-agent", "Accept": "text/html"},
		Environment:  map[string]string{"PATH": "/usr/bin", "HOME": "/home/user"},
		OS:           "linux",
		Architecture: "amd64",
		Runtime:      "go1.21.0",
		Time:         "2023-01-01T12:00:00Z",
		Version:      "dev",
	}

	result := formatAsText(info)

	expectedLines := []string{
		"Hostname: test-host",
		"Name: test-name",
		"IP: 192.168.1.1, 10.0.0.1",
		"RemoteAddr: 192.168.1.100:1234",
		"Host: example.com",
		"URL: /test?param=value",
		"Method: GET",
		"RealIP: 192.168.1.100",
		"Protocol: HTTP/1.1",
		"OS: linux",
		"Architecture: amd64",
		"Runtime: go1.21.0",
		"Time: 2023-01-01T12:00:00Z",
		"Version: dev",
		"Headers:",
		"  Accept: text/html",
		"  User-Agent: test-agent",
		"Environment:",
		"  HOME: /home/user",
		"  PATH: /usr/bin",
	}

	for _, expectedLine := range expectedLines {
		if !strings.Contains(result, expectedLine) {
			t.Errorf("Expected line '%s' not found in formatted text", expectedLine)
		}
	}
}

func TestGetRequestInfo(t *testing.T) {
	// Set up config for testing
	originalConfig := config
	config = Config{Port: 8080, Name: "test-server", Verbose: false}
	defer func() { config = originalConfig }()

	req := httptest.NewRequest("PUT", "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Accept", ContentTypeJSON)
	req.RemoteAddr = "192.168.1.100:1234"

	info := getRequestInfo(req)

	if info.Name != "test-server" {
		t.Errorf("Expected Name 'test-server', got '%s'", info.Name)
	}

	if info.URL != "/test?param=value" {
		t.Errorf("Expected URL '/test?param=value', got '%s'", info.URL)
	}

	if info.Method != "PUT" {
		t.Errorf("Expected Method 'PUT', got '%s'", info.Method)
	}

	if info.RemoteAddr != "192.168.1.100:1234" {
		t.Errorf("Expected RemoteAddr '192.168.1.100:1234', got '%s'", info.RemoteAddr)
	}

	if info.Headers["User-Agent"] != "test-agent" {
		t.Errorf("Expected User-Agent 'test-agent', got '%s'", info.Headers["User-Agent"])
	}

	if info.Headers["Accept"] != ContentTypeJSON {
		t.Errorf("Expected Accept '%s', got '%s'", ContentTypeJSON, info.Headers["Accept"])
	}

	if len(info.IP) == 0 {
		t.Error("Expected at least one IP address")
	}

	// Validate common fields
	validateRequestInfo(t, info)
}

// Benchmark tests
func BenchmarkRootHandler(b *testing.B) {
	config = Config{Port: 8080, Name: "bench-test", Verbose: false}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "benchmark-agent")

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rootHandler(rr, req)
	}
}

func BenchmarkAPIHandler(b *testing.B) {
	config = Config{Port: 8080, Name: "bench-test", Verbose: false}

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("User-Agent", "benchmark-agent")

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		apiHandler(rr, req)
	}
}

func BenchmarkGetLocalIPs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getLocalIPs()
	}
}
