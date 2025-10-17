package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

var (
	// ErrSessionNotFound indica que o ID de sessão não existe ou expirou.
	ErrSessionNotFound = errors.New("session not found")
)

// SessionManager gerencia sessões persistidas no Redis.
// Ele abstrai criação, validação e destruição de sessões admin.
type SessionManager struct {
	redis    *redis.Client
	prefix   string
	ttl      time.Duration
	fallback atomic.Bool

	mu     sync.RWMutex
	memory map[string]memorySession
}

type memorySession struct {
	username string
	expires  time.Time
}

// NewSessionManager cria um gerenciador com TTL configurável via ADMIN_SESSION_TTL.
// Aceita durações em formatos reconhecidos por time.ParseDuration, ex: "24h", "30m".
func NewSessionManager(redisClient *redis.Client) *SessionManager {
	ttl := 24 * time.Hour
	if v := os.Getenv("ADMIN_SESSION_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			ttl = parsed
		}
	}

	return &SessionManager{
		redis:  redisClient,
		prefix: "admin:session:",
		ttl:    ttl,
		memory: make(map[string]memorySession),
	}
}

// TTL retorna a duração configurada para as sessões.
func (m *SessionManager) TTL() time.Duration {
	return m.ttl
}

// Create gera uma nova sessão e a persiste no Redis.
func (m *SessionManager) Create(ctx context.Context, username string) (string, error) {
	id, err := generateSessionID()
	if err != nil {
		return "", err
	}
	if m.useRedis() {
		if err := m.redis.Set(ctx, m.key(id), username, m.ttl).Err(); err == nil {
			return id, nil
		} else {
			m.enableFallback(err)
		}
	}

	m.storeInMemory(id, username)
	return id, nil
}

// Validate verifica se a sessão existe e atualiza o TTL.
func (m *SessionManager) Validate(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrSessionNotFound
	}

	if m.useRedis() {
		key := m.key(sessionID)
		username, err := m.redis.Get(ctx, key).Result()
		if err == nil {
			if err := m.redis.Expire(ctx, key, m.ttl).Err(); err != nil {
				log.Warn().Err(err).Msg("Falha ao renovar TTL da sessão no Redis")
			}
			return username, nil
		}
		if err == redis.Nil {
			return "", ErrSessionNotFound
		}
		m.enableFallback(err)
	}

	return m.validateFromMemory(sessionID)
}

// Destroy remove a sessão do Redis.
func (m *SessionManager) Destroy(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	if m.useRedis() {
		if err := m.redis.Del(ctx, m.key(sessionID)).Err(); err == nil {
			return nil
		} else {
			m.enableFallback(err)
		}
	}

	m.deleteFromMemory(sessionID)
	return nil
}

func (m *SessionManager) key(id string) string {
	return fmt.Sprintf("%s%s", m.prefix, id)
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *SessionManager) useRedis() bool {
	return m.redis != nil && !m.fallback.Load()
}

func (m *SessionManager) enableFallback(err error) {
	if m.fallback.CompareAndSwap(false, true) {
		log.Warn().Err(err).Msg("Redis indisponível para sessões; alternando para armazenamento em memória")
	}
}

func (m *SessionManager) storeInMemory(id, username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[id] = memorySession{
		username: username,
		expires:  time.Now().Add(m.ttl),
	}
}

func (m *SessionManager) validateFromMemory(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.memory[id]
	if !ok {
		return "", ErrSessionNotFound
	}
	if time.Now().After(session.expires) {
		delete(m.memory, id)
		return "", ErrSessionNotFound
	}

	session.expires = time.Now().Add(m.ttl)
	m.memory[id] = session
	return session.username, nil
}

func (m *SessionManager) deleteFromMemory(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.memory, id)
}
