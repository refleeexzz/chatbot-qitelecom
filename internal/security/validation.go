package security

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// InputValidator provides input validation and sanitization
type InputValidator struct {
	config *SecurityConfig
}

// NewInputValidator creates a new input validator
func NewInputValidator(config *SecurityConfig) *InputValidator {
	return &InputValidator{config: config}
}

// ValidateAndSanitizeUserInput validates and sanitizes user input
func (iv *InputValidator) ValidateAndSanitizeUserInput(userID, message string) (string, string, error) {
	// Validate user ID
	cleanUserID, err := iv.validateUserID(userID)
	if err != nil {
		return "", "", fmt.Errorf("invalid user ID: %w", err)
	}
	
	// Validate and sanitize message
	cleanMessage, err := iv.validateAndSanitizeMessage(message)
	if err != nil {
		return "", "", fmt.Errorf("invalid message: %w", err)
	}
	
	return cleanUserID, cleanMessage, nil
}

// validateUserID validates the user ID
func (iv *InputValidator) validateUserID(userID string) (string, error) {
	if userID == "" {
		return "anonymous", nil
	}
	
	// Check length
	if len(userID) > iv.config.MaxUserIDLength {
		return "", fmt.Errorf("user ID too long (max %d characters)", iv.config.MaxUserIDLength)
	}
	
	// Check for valid characters (alphanumeric, dash, underscore)
	if !isValidUserID(userID) {
		return "", fmt.Errorf("user ID contains invalid characters")
	}
	
	// Sanitize
	return html.EscapeString(strings.TrimSpace(userID)), nil
}

// validateAndSanitizeMessage validates and sanitizes the message
func (iv *InputValidator) validateAndSanitizeMessage(message string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("message cannot be empty")
	}
	
	// Check UTF-8 validity
	if !utf8.ValidString(message) {
		return "", fmt.Errorf("message contains invalid UTF-8 characters")
	}
	
	// Check length
	if len(message) > iv.config.MaxMessageLength {
		return "", fmt.Errorf("message too long (max %d characters)", iv.config.MaxMessageLength)
	}
	
	// Trim whitespace
	message = strings.TrimSpace(message)
	if message == "" {
		return "", fmt.Errorf("message cannot be empty after trimming")
	}
	
	// Check for suspicious patterns
	if err := iv.checkSuspiciousPatterns(message); err != nil {
		return "", err
	}
	
	// Sanitize HTML entities but preserve formatting
	message = html.EscapeString(message)
	
	return message, nil
}

// checkSuspiciousPatterns checks for potentially malicious patterns
func (iv *InputValidator) checkSuspiciousPatterns(message string) error {
	lowercaseMsg := strings.ToLower(message)
	
	// SQL injection patterns
	sqlPatterns := []string{
		"' or '1'='1",
		"' or 1=1",
		"union select",
		"drop table",
		"delete from",
		"insert into",
		"update set",
		"<script",
		"javascript:",
		"data:text/html",
		"eval(",
		"expression(",
		"@import",
	}
	
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowercaseMsg, pattern) {
			return fmt.Errorf("message contains suspicious pattern: %s", pattern)
		}
	}
	
	// Check for excessive repeated characters (possible DoS attempt)
	if hasExcessiveRepetition(message) {
		return fmt.Errorf("message contains excessive character repetition")
	}
	
	return nil
}

// isValidUserID checks if user ID contains only valid characters
func isValidUserID(userID string) bool {
	// Allow alphanumeric characters, dashes, underscores, and dots
	matched, _ := regexp.MatchString("^[a-zA-Z0-9._-]+$", userID)
	return matched
}

// hasExcessiveRepetition checks for excessive character repetition
func hasExcessiveRepetition(message string) bool {
	if len(message) < 10 {
		return false
	}
	
	consecutiveCount := 1
	maxConsecutive := 10 // Allow max 10 consecutive identical characters
	
	for i := 1; i < len(message); i++ {
		if message[i] == message[i-1] {
			consecutiveCount++
			if consecutiveCount > maxConsecutive {
				return true
			}
		} else {
			consecutiveCount = 1
		}
	}
	
	return false
}

// SanitizeForLog sanitizes strings for safe logging
func SanitizeForLog(input string) string {
	// Remove newlines and control characters for log safety
	input = regexp.MustCompile(`[\r\n\t\x00-\x1f]`).ReplaceAllString(input, " ")
	
	// Limit length
	if len(input) > 100 {
		input = input[:97] + "..."
	}
	
	return html.EscapeString(input)
}

// SanitizeHTML provides basic HTML sanitization
func SanitizeHTML(input string) string {
	// For now, we just escape HTML. In the future, we could use a proper
	// HTML sanitization library like bluemonday
	return html.EscapeString(input)
}

// ValidateEnvironmentVariable validates environment variables
func ValidateEnvironmentVariable(name, value string) error {
	if value == "" {
		return fmt.Errorf("environment variable %s is required but not set", name)
	}
	
	// Check for basic patterns that might indicate issues
	if strings.Contains(value, "\n") || strings.Contains(value, "\r") {
		return fmt.Errorf("environment variable %s contains newline characters", name)
	}
	
	// Validate specific types
	switch name {
	case "GOOGLE_SHEETS_ID":
		if !isValidGoogleSheetsID(value) {
			return fmt.Errorf("invalid Google Sheets ID format")
		}
	case "GOOGLE_API_KEY":
		if !isValidGoogleAPIKey(value) {
			return fmt.Errorf("invalid Google API key format")
		}
	}
	
	return nil
}

// isValidGoogleSheetsID validates Google Sheets ID format
func isValidGoogleSheetsID(id string) bool {
	// Google Sheets IDs are typically 44 characters long and contain alphanumeric characters, dashes, and underscores
	if len(id) < 30 || len(id) > 60 {
		return false
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", id)
	return matched
}

// isValidGoogleAPIKey validates Google API key format
func isValidGoogleAPIKey(key string) bool {
	// Google API keys are typically 39 characters starting with AIza
	if len(key) < 30 || len(key) > 50 {
		return false
	}
	return strings.HasPrefix(key, "AIza")
}