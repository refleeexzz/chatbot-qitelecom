package security

import (
	"os"
	"strings"
	"testing"
)

func TestInputValidator(t *testing.T) {
	// Set up test config
	os.Setenv("GOOGLE_SHEETS_ID", "test_sheet_id")
	defer os.Unsetenv("GOOGLE_SHEETS_ID")
	
	config, err := LoadSecurityConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		userID      string
		message     string
		expectError bool
		description string
	}{
		{
			name:        "Valid input",
			userID:      "user123",
			message:     "Hello, I need help",
			expectError: false,
			description: "Normal valid input should pass",
		},
		{
			name:        "Empty user ID",
			userID:      "",
			message:     "Hello",
			expectError: false,
			description: "Empty user ID should default to anonymous",
		},
		{
			name:        "Empty message",
			userID:      "user123",
			message:     "",
			expectError: true,
			description: "Empty message should be rejected",
		},
		{
			name:        "Whitespace only message",
			userID:      "user123",
			message:     "   \n\t   ",
			expectError: true,
			description: "Whitespace-only message should be rejected",
		},
		{
			name:        "SQL injection attempt",
			userID:      "user123",
			message:     "' OR '1'='1",
			expectError: true,
			description: "SQL injection patterns should be blocked",
		},
		{
			name:        "XSS attempt",
			userID:      "user123",
			message:     "<script>alert('xss')</script>",
			expectError: true,
			description: "XSS patterns should be blocked",
		},
		{
			name:        "Long message",
			userID:      "user123",
			message:     string(make([]byte, 2000)), // Exceeds max length
			expectError: true,
			description: "Messages exceeding max length should be rejected",
		},
		{
			name:        "Long user ID",
			userID:      string(make([]byte, 200)), // Exceeds max length
			message:     "Hello",
			expectError: true,
			description: "User IDs exceeding max length should be rejected",
		},
		{
			name:        "Invalid user ID characters",
			userID:      "user@#$%",
			message:     "Hello",
			expectError: true,
			description: "Invalid characters in user ID should be rejected",
		},
		{
			name:        "Excessive repetition",
			userID:      "user123",
			message:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaa", // More than 10 consecutive chars
			expectError: true,
			description: "Excessive character repetition should be blocked",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := validator.ValidateAndSanitizeUserInput(test.userID, test.message)
			
			if test.expectError && err == nil {
				t.Errorf("Expected error but got none. %s", test.description)
			}
			
			if !test.expectError && err != nil {
				t.Errorf("Expected no error but got: %v. %s", err, test.description)
			}
		})
	}
}

func TestIsValidUserID(t *testing.T) {
	tests := []struct {
		userID   string
		expected bool
	}{
		{"user123", true},
		{"user_123", true},
		{"user-123", true},
		{"user.123", true},
		{"User123", true},
		{"user@123", false},   // @ not allowed
		{"user#123", false},   // # not allowed
		{"user 123", false},   // space not allowed
		{"user/123", false},   // / not allowed
		{"", false},           // empty not allowed for this test
	}

	for _, test := range tests {
		t.Run(test.userID, func(t *testing.T) {
			result := isValidUserID(test.userID)
			if result != test.expected {
				t.Errorf("For userID '%s', expected %v, got %v", test.userID, test.expected, result)
			}
		})
	}
}

func TestHasExcessiveRepetition(t *testing.T) {
	tests := []struct {
		message  string
		expected bool
	}{
		{"hello world", false},
		{"aaaaaaaaaa", false},          // exactly 10 chars, should be ok
		{"aaaaaaaaaaa", true},          // 11 consecutive chars
		{"hello aaaaaaaaaaaa world", true}, // embedded excessive repetition
		{"abcdefgh", false},            // no repetition
		{"aabbccdd", false},            // short repetitions
		{"", false},                    // empty string
		{"a", false},                   // single char
	}

	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			result := hasExcessiveRepetition(test.message)
			if result != test.expected {
				t.Errorf("For message '%s', expected %v, got %v", test.message, test.expected, result)
			}
		})
	}
}

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text\nwith\nnewlines", "text with newlines"},
		{"text\twith\ttabs", "text with tabs"},
		{"text\rwith\rcarriage", "text with carriage"},
		{string(make([]byte, 200)), string(make([]byte, 97)) + "..."}, // long text truncated
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
	}

	for _, test := range tests {
		t.Run(test.input[:min(len(test.input), 20)], func(t *testing.T) {
			result := SanitizeForLog(test.input)
			if len(test.input) > 100 {
				// For long inputs, just check truncation
				if len(result) != 100 || !strings.HasSuffix(result, "...") {
					t.Errorf("Expected truncated string of length 100 ending with '...', got length %d: %s", len(result), result)
				}
			} else if result != test.expected {
				t.Errorf("For input '%s', expected '%s', got '%s'", test.input, test.expected, result)
			}
		})
	}
}

func TestValidateEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"GOOGLE_SHEETS_ID", "1iUElxVPVqqBqAUq-9rXRjhSTAo94Quqt9-0KIUgNgOA", false},
		{"GOOGLE_SHEETS_ID", "short", true}, // too short
		{"GOOGLE_SHEETS_ID", "", true},      // empty
		{"GOOGLE_API_KEY", "AIzaSyDummy1234567890123456789012345678", false},
		{"GOOGLE_API_KEY", "invalid_key", true}, // doesn't start with AIza
		{"OTHER_VAR", "value\nwith\nnewline", true}, // contains newlines
		{"OTHER_VAR", "normal_value", false},
	}

	for _, test := range tests {
		t.Run(test.name+"_"+test.value[:min(len(test.value), 10)], func(t *testing.T) {
			err := ValidateEnvironmentVariable(test.name, test.value)
			
			if test.expectError && err == nil {
				t.Errorf("Expected error for %s='%s' but got none", test.name, test.value)
			}
			
			if !test.expectError && err != nil {
				t.Errorf("Expected no error for %s='%s' but got: %v", test.name, test.value, err)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}