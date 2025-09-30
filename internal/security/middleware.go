// Package security implementa middlewares de segurança e rate limiting para handlers HTTP.		
package security

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// rateLimiter implementa um rate limiter simples baseado em IP.
type rateLimiter struct {
	mu       sync.Mutex
	counts   map[string]int
	resetAt  time.Time
	limit    int
	interval time.Duration
}

// newRateLimiter cria uma nova instância de rateLimiter.
func newRateLimiter(limit int, interval time.Duration) *rateLimiter {
	return &rateLimiter{
		counts:   make(map[string]int),
		limit:    limit,
		interval: interval,
		resetAt:  time.Now().Add(interval),
	}
}

// NewGlobalRateLimiter cria um rate limiter global com janela de 1 minuto.
func NewGlobalRateLimiter(perMinute int) *rateLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return newRateLimiter(perMinute, time.Minute)
}

// allow verifica se o IP pode continuar fazendo requisições dentro do limite.
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

// SecurityConfig define limites de requisição e tamanho do corpo aceito.
type SecurityConfig struct {
	BodyLimitBytes int
	RatePerMinute  int
}

// LoadConfig carrega limites de segurança a partir das variáveis de ambiente.
func LoadConfig() SecurityConfig {
	cfg := SecurityConfig{
		BodyLimitBytes: 4096,
		RatePerMinute:  60,
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

// WrapHandler aplica body limit, rate limiting e headers de segurança ao handler HTTP.
func WrapHandler(h http.Handler, cfg SecurityConfig, rl *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.BodyLimitBytes))

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !rl.allow(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-XSS-Protection", "0")

		h.ServeHTTP(w, r)
	})
}
