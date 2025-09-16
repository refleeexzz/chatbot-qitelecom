package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"leadprojectarrumado/internal/security"
)

type ChatbotHandler struct {
	service   ChatbotService
	validator *security.InputValidator
}

type ChatbotService interface {
	ProcessMessage(userID, message string) (string, error)
}

type ChatRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// Cria um novo handler do chatbot
func NewChatbotHandler(service ChatbotService, securityConfig *security.SecurityConfig) *ChatbotHandler {
	return &ChatbotHandler{
		service:   service,
		validator: security.NewInputValidator(securityConfig),
	}
}

// Processa requisições do chatbot
func (h *ChatbotHandler) HandleChatbot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Método não permitido",
		})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "JSON inválido",
		})
		return
	}

	// Validate and sanitize input
	cleanUserID, cleanMessage, err := h.validator.ValidateAndSanitizeUserInput(req.UserID, req.Message)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", security.SanitizeForLog(req.UserID)).
			Str("message", security.SanitizeForLog(req.Message)).
			Msg("Input validation failed")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Entrada inválida: " + err.Error(),
		})
		return
	}

	// Processar mensagem com dados limpos
	response, err := h.service.ProcessMessage(cleanUserID, cleanMessage)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", security.SanitizeForLog(cleanUserID)).
			Msg("Erro ao processar mensagem")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Erro interno do servidor",
		})
		return
	}

	// Resposta de sucesso
	json.NewEncoder(w).Encode(ChatResponse{
		Response: response,
	})
}

// Serve arquivos estáticos
func (h *ChatbotHandler) HandleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "index.html")
		return
	}

	// Determinar Content-Type baseado na extensão
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}

	// Servir arquivo estático
	filename := strings.TrimPrefix(r.URL.Path, "/")
	http.ServeFile(w, r, filename)
}

// Endpoint de saúde para monitoramento
func (h *ChatbotHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "qibot-chatbot",
	})
}
