package security

import (
	"os"
	"testing"
	"time"
)

func TestLoadSecurityConfig(t *testing.T) {
	// Set required environment variable
	os.Setenv("GOOGLE_SHEETS_ID", "test_sheet_id")
	defer os.Unsetenv("GOOGLE_SHEETS_ID")

	config, err := LoadSecurityConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test defaults
	if config.ForceHTTPS != true {
		t.Errorf("Expected ForceHTTPS to be true by default")
	}

	if config.RateLimitRPM != 60 {
		t.Errorf("Expected RateLimitRPM to be 60, got %d", config.RateLimitRPM)
	}

	if config.MaxMessageLength != 1000 {
		t.Errorf("Expected MaxMessageLength to be 1000, got %d", config.MaxMessageLength)
	}

	if config.GoogleSheetsID != "test_sheet_id" {
		t.Errorf("Expected GoogleSheetsID to be 'test_sheet_id', got %s", config.GoogleSheetsID)
	}
}

func TestLoadSecurityConfigMissingRequired(t *testing.T) {
	// Make sure GOOGLE_SHEETS_ID is not set
	os.Unsetenv("GOOGLE_SHEETS_ID")

	_, err := LoadSecurityConfig()
	if err == nil {
		t.Fatal("Expected error for missing GOOGLE_SHEETS_ID")
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
		defaultVal bool
	}{
		{"true", true, false},
		{"false", false, true},
		{"", true, true}, // Should use default
		{"invalid", false, false}, // Should use default on parse error
	}

	for _, test := range tests {
		if test.envValue != "" {
			os.Setenv("TEST_BOOL", test.envValue)
		} else {
			os.Unsetenv("TEST_BOOL")
		}

		result := getEnvBool("TEST_BOOL", test.defaultVal)
		if result != test.expected {
			t.Errorf("For env value '%s' with default %v, expected %v, got %v",
				test.envValue, test.defaultVal, test.expected, result)
		}
	}

	os.Unsetenv("TEST_BOOL")
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		envValue string
		expected time.Duration
		defaultVal time.Duration
	}{
		{"30s", 30 * time.Second, time.Minute},
		{"5m", 5 * time.Minute, time.Second},
		{"", time.Hour, time.Hour}, // Should use default
		{"invalid", time.Minute, time.Minute}, // Should use default on parse error
	}

	for _, test := range tests {
		if test.envValue != "" {
			os.Setenv("TEST_DURATION", test.envValue)
		} else {
			os.Unsetenv("TEST_DURATION")
		}

		result := getEnvDuration("TEST_DURATION", test.defaultVal)
		if result != test.expected {
			t.Errorf("For env value '%s' with default %v, expected %v, got %v",
				test.envValue, test.defaultVal, test.expected, result)
		}
	}

	os.Unsetenv("TEST_DURATION")
}

func TestGetEnvStringSlice(t *testing.T) {
	tests := []struct {
		envValue string
		expected []string
		defaultVal []string
	}{
		{"a,b,c", []string{"a", "b", "c"}, []string{"default"}},
		{"single", []string{"single"}, []string{"default"}},
		{"", []string{"default"}, []string{"default"}}, // Should use default
	}

	for _, test := range tests {
		if test.envValue != "" {
			os.Setenv("TEST_SLICE", test.envValue)
		} else {
			os.Unsetenv("TEST_SLICE")
		}

		result := getEnvStringSlice("TEST_SLICE", test.defaultVal)
		if len(result) != len(test.expected) {
			t.Errorf("For env value '%s', expected length %d, got %d",
				test.envValue, len(test.expected), len(result))
			continue
		}

		for i, v := range result {
			if v != test.expected[i] {
				t.Errorf("For env value '%s', expected %v, got %v",
					test.envValue, test.expected, result)
				break
			}
		}
	}

	os.Unsetenv("TEST_SLICE")
}