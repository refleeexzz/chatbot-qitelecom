package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ChatbotHandler struct {
	service ChatbotService
}

type ChatbotService interface {
	ProcessMessage(userID, message string) (string, error)
}

type ChatRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ChatResponse struct {
	Response  string `json:"response"`
	Error     string `json:"error,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// Cria um novo handler do chatbot
func NewChatbotHandler(service ChatbotService) *ChatbotHandler {
	return &ChatbotHandler{service: service}
}

// Processa requisições do chatbot
func (h *ChatbotHandler) HandleChatbot(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Método não permitido",
		})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{Error: "JSON inválido"})
		return
	}

	// Resolver / criar session ID
	sessionID := strings.TrimSpace(req.UserID)
	if sessionID == "" {
		// Tenta header X-Session-ID
		sessionID = strings.TrimSpace(r.Header.Get("X-Session-ID"))
	}
	if sessionID == "" {
		// Tenta cookie
		if c, err := r.Cookie("qid"); err == nil {
			sessionID = c.Value
		}
	}
	if sessionID == "" {
		// Gera novo
		newID, err := uuid.NewRandom()
		if err != nil {
			newID = uuid.Must(uuid.NewRandom())
		}
		sessionID = newID.String()
		// Set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "qid",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // colocar true em produção com HTTPS
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})
	}
	req.UserID = sessionID
	if strings.TrimSpace(req.Message) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{Error: "Mensagem não pode estar vazia", SessionID: sessionID})
		return
	}

	// Processar mensagem
	response, err := h.service.ProcessMessage(req.UserID, req.Message)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao processar mensagem")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{Error: "Erro interno do servidor", SessionID: sessionID})
		return
	}

	// Resposta de sucesso
	json.NewEncoder(w).Encode(ChatResponse{Response: response, SessionID: sessionID})
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
