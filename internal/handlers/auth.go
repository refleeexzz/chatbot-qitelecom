package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"leadprojectarrumado/internal/security"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "qibot_session"
)

type contextKey string

const (
	ctxAdminUser contextKey = "admin.user"
)

// AuthHandler centraliza login/logout do painel admin usando sessões no Redis.
type AuthHandler struct {
	username     string
	password     string
	passwordHash string
	session      *security.SessionManager
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Redirect string `json:"redirect,omitempty"`
}

// NewAuthHandler cria um handler configurado a partir das variáveis de ambiente.
func NewAuthHandler(session *security.SessionManager) *AuthHandler {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	passwordHash := os.Getenv("ADMIN_PASSWORD_HASH")

	if password == "" && passwordHash == "" {
		password = "admin123"
		log.Warn().Msg("ADMIN_PASSWORD não definido - usando valor padrão inseguro para desenvolvimento")
	}

	return &AuthHandler{
		username:     username,
		password:     password,
		passwordHash: passwordHash,
		session:      session,
	}
}

// HandleLogin autentica o administrador e cria uma nova sessão persistida.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	loginReq, viaForm, err := parseLoginRequest(r)
	if err != nil {
		log.Warn().Err(err).Msg("Falha ao interpretar requisição de login")
		respondAuthFailure(w, r, viaForm, "Credenciais inválidas", http.StatusBadRequest)
		return
	}

	if !h.validateCredentials(loginReq.Username, loginReq.Password) {
		log.Warn().Str("username", loginReq.Username).Msg("Tentativa de login rejeitada")
		respondAuthFailure(w, r, viaForm, "Usuário ou senha incorretos", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sessionID, err := h.session.Create(ctx, loginReq.Username)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao criar sessão de admin")
		respondAuthFailure(w, r, viaForm, "Não foi possível iniciar a sessão", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(h.session.TTL()),
		MaxAge:   int(h.session.TTL().Seconds()),
	})

	log.Info().Str("username", loginReq.Username).Msg("Login admin bem-sucedido")

	if viaForm {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{Success: true, Redirect: "/admin"})
}

// HandleLogout encerra a sessão atual.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := readSessionID(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.session.Destroy(ctx, sessionID); err != nil && !errors.Is(err, security.ErrSessionNotFound) {
		log.Warn().Err(err).Msg("Falha ao remover sessão durante logout")
	}

	expireSessionCookie(w, shouldUseSecureCookie(r))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Logout realizado com sucesso",
	})
}

// RequireSession garante que a sessão esteja válida antes de executar o handler protegido.
func (h *AuthHandler) RequireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := readSessionID(r)
		if err != nil {
			log.Warn().Str("path", r.URL.Path).Msg("Sessão ausente - redirecionando para login")
			h.handleUnauthorized(w, r, "missing")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		username, err := h.session.Validate(ctx, sessionID)
		if err != nil {
			if errors.Is(err, security.ErrSessionNotFound) {
				log.Info().Str("path", r.URL.Path).Msg("Sessão expirada - limpando cookie")
				expireSessionCookie(w, shouldUseSecureCookie(r))
				h.handleUnauthorized(w, r, "expired")
			} else {
				log.Error().Err(err).Msg("Erro ao validar sessão")
				h.handleUnauthorized(w, r, "error")
			}
			return
		}

		ctxWithUser := context.WithValue(r.Context(), ctxAdminUser, username)
		next(w, r.WithContext(ctxWithUser))
	}
}

// HandleAdminGateway entrega a página de login ou o dashboard conforme a sessão.
func (h *AuthHandler) HandleAdminGateway(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	sessionID, err := readSessionID(r)
	if err != nil || sessionID == "" {
		log.Info().Str("path", r.URL.Path).Msg("Visitante sem sessão - servindo login")
		http.ServeFile(w, r, "admin_login.html")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	username, err := h.session.Validate(ctx, sessionID)
	if err != nil {
		if errors.Is(err, security.ErrSessionNotFound) {
			log.Info().Msg("Sessão expirada - redirecionando para login")
		} else {
			log.Error().Err(err).Msg("Erro ao validar sessão ao carregar dashboard")
		}
		expireSessionCookie(w, shouldUseSecureCookie(r))
		http.ServeFile(w, r, "admin_login.html")
		return
	}

	log.Info().Str("username", username).Msg("Sessão válida - servindo dashboard")
	http.ServeFile(w, r, "admin_dashboard.html")
}

// AdminUserFromContext retorna o usuário autenticado armazenado no contexto.
func AdminUserFromContext(r *http.Request) (string, bool) {
	val := r.Context().Value(ctxAdminUser)
	if val == nil {
		return "", false
	}
	username, ok := val.(string)
	return username, ok
}

func (h *AuthHandler) validateCredentials(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	if username != h.username {
		return false
	}

	if h.passwordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(h.passwordHash), []byte(password)) == nil
	}

	return subtle.ConstantTimeCompare([]byte(password), []byte(h.password)) == 1
}

func parseLoginRequest(r *http.Request) (LoginRequest, bool, error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))

	if strings.Contains(ct, "application/json") {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// fallback para formulário se possível
			if err := r.ParseForm(); err != nil {
				return LoginRequest{}, false, err
			}
			return LoginRequest{
				Username: r.FormValue("username"),
				Password: r.FormValue("password"),
			}, len(r.PostForm) > 0, nil
		}
		return req, false, nil
	}

	if err := r.ParseForm(); err != nil {
		return LoginRequest{}, true, err
	}

	return LoginRequest{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}, true, nil
}

func readSessionID(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func respondAuthFailure(w http.ResponseWriter, r *http.Request, viaForm bool, message string, status int) {
	if viaForm {
		http.Redirect(w, r, "/admin_login.html?error=1", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(LoginResponse{Success: false, Message: message})
}

func (h *AuthHandler) handleUnauthorized(w http.ResponseWriter, r *http.Request, reason string) {
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{Success: false, Message: "Não autenticado"})
		return
	}

	target := "/admin_login.html"
	if reason == "expired" {
		target += "?session=expired"
	} else if reason == "error" {
		target += "?error=1"
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func expireSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func shouldUseSecureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	if strings.EqualFold(r.Header.Get("X-Forwarded-Ssl"), "on") {
		return true
	}
	fwd := strings.ToLower(r.Header.Get("Forwarded"))
	if strings.Contains(fwd, "proto=https") {
		return true
	}
	if strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") {
		return true
	}
	return false
}
