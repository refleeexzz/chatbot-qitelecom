package security

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	
	"github.com/rs/zerolog/log"
)

// SecurityMiddleware provides security-related HTTP middleware
type SecurityMiddleware struct {
	config     *SecurityConfig
	rateLimiter *RateLimiter
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	clients map[string]*clientBucket
	mutex   sync.RWMutex
	rpm     int
	burst   int
}

type clientBucket struct {
	tokens    int
	lastRefill time.Time
}

// NewSecurityMiddleware creates a new security middleware instance
func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	var rateLimiter *RateLimiter
	if config.RateLimitEnabled {
		rateLimiter = &RateLimiter{
			clients: make(map[string]*clientBucket),
			rpm:     config.RateLimitRPM,
			burst:   config.RateLimitBurst,
		}
	}
	
	return &SecurityMiddleware{
		config:      config,
		rateLimiter: rateLimiter,
	}
}

// SecurityHeaders adds security headers to responses
func (sm *SecurityMiddleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; img-src 'self' data:; connect-src 'self'")
		
		// HTTPS enforcement
		if sm.config.ForceHTTPS && r.Header.Get("X-Forwarded-Proto") != "https" && r.TLS == nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			// In production, this should redirect to HTTPS
			// For development, we'll just add the header
		}
		
		next.ServeHTTP(w, r)
	})
}

// CORS handles Cross-Origin Resource Sharing with secure defaults
func (sm *SecurityMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		if origin != "" && sm.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(sm.config.AllowedOrigins) == 1 && sm.config.AllowedOrigins[0] == "*" {
			// Only allow wildcard in development
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RateLimit implements rate limiting per IP address
func (sm *SecurityMiddleware) RateLimit(next http.Handler) http.Handler {
	if !sm.config.RateLimitEnabled || sm.rateLimiter == nil {
		return next
	}
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := sm.getClientIP(r)
		
		if !sm.rateLimiter.Allow(clientIP) {
			log.Warn().Str("ip", clientIP).Msg("Rate limit exceeded")
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimit limits the size of incoming requests
func (sm *SecurityMiddleware) RequestSizeLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, sm.config.MaxRequestSizeBytes)
		next.ServeHTTP(w, r)
	})
}

// HTTPSRedirect redirects HTTP requests to HTTPS
func (sm *SecurityMiddleware) HTTPSRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sm.config.ForceHTTPS && r.Header.Get("X-Forwarded-Proto") != "https" && r.TLS == nil {
			redirectURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
			http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the origin is in the allowed list
func (sm *SecurityMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range sm.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

// getClientIP extracts the real client IP from request headers
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check for forwarded IP in common headers
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fallback to remote address
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// Allow checks if a request from the given IP is allowed by the rate limiter
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	bucket, exists := rl.clients[clientIP]
	if !exists {
		bucket = &clientBucket{
			tokens:    rl.burst - 1, // Use one token
			lastRefill: now,
		}
		rl.clients[clientIP] = bucket
		return true
	}
	
	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed.Minutes()) * rl.rpm / 60
	
	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.burst {
			bucket.tokens = rl.burst
		}
		bucket.lastRefill = now
	}
	
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// CleanupExpiredClients removes old client buckets (should be called periodically)
func (rl *RateLimiter) CleanupExpiredClients() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	for ip, bucket := range rl.clients {
		// Remove buckets not accessed for 1 hour
		if now.Sub(bucket.lastRefill) > time.Hour {
			delete(rl.clients, ip)
		}
	}
}