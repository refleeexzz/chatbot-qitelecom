package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func init() {
	log.Info().Msg("🔐 auth.go: Arquivo carregado - VerifyAdminToken disponível")
}

type AuthHandler struct {
	username string
	password string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewAuthHandler() *AuthHandler {
	// Usar variáveis de ambiente ou valores padrão
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin" // padrão para desenvolvimento
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123" // padrão para desenvolvimento
	}

	return &AuthHandler{
		username: username,
		password: password,
	}
}

// HandleLogin processa login de administrador
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Método não permitido",
		})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "JSON inválido",
		})
		return
	}

	// Verificar credenciais com timing-safe comparison
	usernameMatch := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.password)) == 1

	if usernameMatch && passwordMatch {
		// Gerar token simples (em produção, use JWT)
		token := generateSimpleToken(req.Username)

		// Criar cookie de sessão
		http.SetCookie(w, &http.Cookie{
			Name:     "admin_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // true em produção com HTTPS
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		log.Info().Str("username", req.Username).Msg("Login de administrador bem-sucedido")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: true,
			Token:   token,
			Message: "Login bem-sucedido",
		})
	} else {
		log.Warn().Str("username", req.Username).Msg("Tentativa de login falhou")

		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Usuário ou senha incorretos",
		})
	}
}

// HandleLogout faz logout do administrador
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("🚪 Logout solicitado - removendo cookie admin_token")

	// Remover cookie de várias formas para garantir
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // true se usar HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0), // Expirar imediatamente
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logout realizado com sucesso",
	})

	log.Info().Msg("✅ Logout concluído - cookie removido")
}

// HandleAdminDashboard serve a página de administração
func (h *AuthHandler) HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// Verificar se está autenticado
	cookie, err := r.Cookie("admin_token")
	if err != nil || cookie.Value == "" {
		// Redirecionar para login
		http.ServeFile(w, r, "admin_login.html")
		return
	}

	// Servir dashboard
	http.ServeFile(w, r, "admin_dashboard.html")
}

// generateSimpleToken gera um token simples (use JWT em produção)
func generateSimpleToken(username string) string {
	// Em produção, use JWT com assinatura
	return "admin_" + username + "_" + time.Now().Format("20060102150405")
}

// VerifyAdminToken middleware para verificar autenticação
func VerifyAdminToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info().Str("path", r.URL.Path).Msg("🔒 VerifyAdminToken: Verificando autenticação")

		cookie, err := r.Cookie("admin_token")
		if err != nil || cookie.Value == "" {
			log.Warn().Str("path", r.URL.Path).Msg("🚫 VerifyAdminToken: Cookie não encontrado")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Não autenticado. Faça login primeiro.",
			})
			return
		}

		// Verificar se token é válido (deve começar com "admin_" e ter tamanho mínimo)
		if len(cookie.Value) < 20 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Token inválido - comprimento insuficiente",
			})
			return
		}

		// Verificar formato do token
		if len(cookie.Value) < 6 || cookie.Value[:6] != "admin_" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Token inválido - formato incorreto",
			})
			return
		}

		next(w, r)
	}
}
