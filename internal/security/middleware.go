// Package security implementa middlewares de segurança e rate limiting para handlers HTTP.
package security

import (
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
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
		// Forçar redirecionamento para HTTPS se configurado e requisição não for segura
		if strings.EqualFold(os.Getenv("FORCE_HTTPS"), "true") && !isMetaWebhookRequest(r) {
			// Detectar se requisição já veio por HTTPS (via TLS ou cabeçalhos de proxy)
			secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") || strings.EqualFold(r.Header.Get("X-Forwarded-Ssl"), "on")
			fwd := strings.ToLower(r.Header.Get("Forwarded"))
			if strings.Contains(fwd, "proto=https") {
				secure = true
			}
			if !secure {
				// Montar URL https equivalente
				target := "https://" + r.Host + r.URL.RequestURI()
				// Usar 307 para preservar método em POSTs
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				return
			}
		}
		// Verificar se é webhook da Meta (WhatsApp/Facebook)
		isMetaWebhook := isMetaWebhookRequest(r)

		// Body limit diferenciado: webhooks da Meta podem ter payloads maiores
		if isMetaWebhook {
			r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.BodyLimitBytes*4)) // 4x maior para Meta
		} else {
			r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.BodyLimitBytes))
		}

		// Rate limiting apenas para requisições que não são da Meta
		if !isMetaWebhook {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !rl.allow(ip) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-XSS-Protection", "0")
		// Forçar HSTS se configurado via ENV (apenas em produção com HTTPS)
		if strings.EqualFold(os.Getenv("FORCE_HSTS"), "true") {
			// Max-age=63072000 (~2 anos), includeSubDomains e preload opcional
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		h.ServeHTTP(w, r)
	})
}

// isMetaWebhookRequest verifica se a requisição é um webhook da Meta/WhatsApp.
// Checa User-Agent e presença de assinatura X-Hub-Signature-256.
func isMetaWebhookRequest(r *http.Request) bool {
	// Verificar path do webhook
	if strings.HasPrefix(r.URL.Path, "/webhook/whatsapp") {
		return true
	}

	// Verificar User-Agent da Meta
	userAgent := strings.ToLower(r.Header.Get("User-Agent"))
	if strings.Contains(userAgent, "facebookexternalua") ||
		strings.Contains(userAgent, "whatsapp") {
		return true
	}

	// Verificar assinatura X-Hub-Signature-256 (presente em webhooks da Meta)
	signature := r.Header.Get("X-Hub-Signature-256")
	return signature != ""
}

// BlockPathPrefixes returns a middleware that denies requests whose cleaned URL path matches any of the
// provided prefixes. Use this to prevent exposure of sensitive folders like /data or /app/data.
func BlockPathPrefixes(prefixes ...string) func(http.Handler) http.Handler {
	normalized := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		clean := path.Clean(prefix)
		if !strings.HasPrefix(clean, "/") {
			clean = "/" + clean
		}
		normalized = append(normalized, clean)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path
			if requestPath == "" {
				requestPath = "/"
			}

			cleaned := path.Clean(requestPath)
			if !strings.HasPrefix(cleaned, "/") {
				cleaned = "/" + cleaned
			}

			for _, prefix := range normalized {
				if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
					http.NotFound(w, r)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MaskPhone replaces the middle digits of a phone number with asterisks for logging.
// Ex: 5511999887766 -> 55119*****766
func MaskPhone(phone string) string {
	s := strings.TrimSpace(phone)
	// keep any non-digit structure but mask digits in the middle
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	n := len(digits)
	if n <= 6 {
		// too short to meaningfully mask
		return s
	}
	// mask from index 3 to n-3
	for i := 3; i < n-3; i++ {
		digits[i] = '*'
	}
	// reconstruct using original non-digit separators
	var out strings.Builder
	di := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out.WriteRune(digits[di])
			di++
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
