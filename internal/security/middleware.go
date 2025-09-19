package security

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// rateLimiter mantém contagem simples em memória por IP.
type rateLimiter struct {
	mu       sync.Mutex
	counts   map[string]int
	resetAt  time.Time
	limit    int
	interval time.Duration
}

func newRateLimiter(limit int, interval time.Duration) *rateLimiter {
	return &rateLimiter{
		counts:   make(map[string]int),
		limit:    limit,
		interval: interval,
		resetAt:  time.Now().Add(interval),
	}
}

// NewGlobalRateLimiter expõe um construtor simplificado para uso externo (minutos como janela).
func NewGlobalRateLimiter(perMinute int) *rateLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return newRateLimiter(perMinute, time.Minute)
}

func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Now().After(r.resetAt) {
		r.counts = make(map[string]int)
		r.resetAt = time.Now().Add(r.interval)
	}
	r.counts[key]++
	return r.counts[key] <= r.limit
}

// SecurityConfig contém parâmetros configuráveis.
type SecurityConfig struct {
	BodyLimitBytes int
	RatePerMinute  int
}

// LoadConfig carrega limites de env, com defaults seguros.
func LoadConfig() SecurityConfig {
	cfg := SecurityConfig{
		BodyLimitBytes: 4096, // 4KB default
		RatePerMinute:  60,   // 60 req/min por IP
	}
	if v := os.Getenv("BODY_LIMIT_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BodyLimitBytes = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_PER_MINUTE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RatePerMinute = n
		}
	}
	return cfg
}

// WrapHandler aplica body limit, rate limiting e security headers.
func WrapHandler(h http.Handler, cfg SecurityConfig, rl *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limitar tamanho do corpo
		r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.BodyLimitBytes))

		// Descobrir IP (X-Forwarded-For ignorado para segurança básica local)
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !rl.allow(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Security Headers básicos
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-XSS-Protection", "0")

		h.ServeHTTP(w, r)
	})
}
