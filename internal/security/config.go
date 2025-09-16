package security

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	// Server security
	ForceHTTPS           bool
	AllowedOrigins       []string
	MaxRequestSizeBytes  int64
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	
	// Rate limiting
	RateLimitEnabled     bool
	RateLimitRPM         int // Requests per minute per IP
	RateLimitBurst       int
	
	// Session security  
	SessionTimeout       time.Duration
	SecureCookies        bool
	
	// Input validation
	MaxMessageLength     int
	MaxUserIDLength      int
	
	// External services
	GoogleSheetsID       string
	RedisPassword        string
	TLSCertFile          string
	TLSKeyFile           string
}

// LoadSecurityConfig loads security configuration from environment variables
func LoadSecurityConfig() (*SecurityConfig, error) {
	config := &SecurityConfig{
		// Default secure values
		ForceHTTPS:           getEnvBool("FORCE_HTTPS", true),
		AllowedOrigins:       getEnvStringSlice("ALLOWED_ORIGINS", []string{"https://localhost:8081"}),
		MaxRequestSizeBytes:  getEnvInt64("MAX_REQUEST_SIZE_BYTES", 1024*1024), // 1MB
		ReadTimeout:          getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:         getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:          getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		
		RateLimitEnabled:     getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRPM:         getEnvInt("RATE_LIMIT_RPM", 60), // 60 requests per minute
		RateLimitBurst:       getEnvInt("RATE_LIMIT_BURST", 10),
		
		SessionTimeout:       getEnvDuration("SESSION_TIMEOUT", time.Hour),
		SecureCookies:        getEnvBool("SECURE_COOKIES", true),
		
		MaxMessageLength:     getEnvInt("MAX_MESSAGE_LENGTH", 1000),
		MaxUserIDLength:      getEnvInt("MAX_USER_ID_LENGTH", 100),
		
		GoogleSheetsID:       os.Getenv("GOOGLE_SHEETS_ID"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		TLSCertFile:          os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:           os.Getenv("TLS_KEY_FILE"),
	}
	
	// Validate required fields
	if config.GoogleSheetsID == "" {
		return nil, fmt.Errorf("GOOGLE_SHEETS_ID environment variable is required")
	}
	
	// Validate allowed origins
	if len(config.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("at least one allowed origin must be specified")
	}
	
	// Validate TLS configuration if HTTPS is forced
	if config.ForceHTTPS && (config.TLSCertFile == "" || config.TLSKeyFile == "") {
		// Allow for development, but warn
		fmt.Println("WARNING: HTTPS forced but no TLS certificates provided. Using self-signed for development.")
	}
	
	return config, nil
}

// Utility functions for environment variable parsing
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	return strings.Split(value, ",")
}