// Package handlers contém os handlers HTTP do sistema, incluindo integração com o WhatsApp Cloud API.
package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"leadprojectarrumado/internal/security"
	"net/http"
	"os"
	"strings"
	"time"
)

// (removed old simple WhatsAppWebhookHandler declaration)

// NewWhatsAppWebhookHandler cria um novo handler para o webhook do WhatsApp.
// Sender is an interface used to send messages to the WhatsApp API. It allows
// tests to inject a mock sender to avoid real network calls.
type Sender interface {
	Send(to, message string) error
}

// httpSender is the production sender that calls the real SendWhatsAppMessage helper.
type httpSender struct{}

func (s httpSender) Send(to, message string) error {
	// Hardened sender: timeout, retries, and masked logging.
	phoneID := security.GetSecret("WHATSAPP_PHONE_ID")
	token := security.GetSecret("WHATSAPP_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", phoneID)

	// Mask token for logs (show only first 4 chars if present)
	maskedToken := ""
	if len(token) > 4 {
		maskedToken = token[:4] + "..."
	} else {
		maskedToken = token
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": message},
	}
	b, _ := json.Marshal(payload)

	// Use a client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Retry loop with exponential backoff
	attempts := 3
	backoff := 200 * time.Millisecond
	var lastErr error
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest("POST", url, bytes.NewReader(b))
		if err != nil {
			lastErr = err
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		// Log minimal info
		fmt.Println("[WHATSAPP] url:", url)
		fmt.Println("[WHATSAPP] token (masked):", maskedToken)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			fmt.Println("[WHATSAPP] send attempt", i+1, "error:", err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		defer resp.Body.Close()

		bodyResp, _ := io.ReadAll(resp.Body)
		fmt.Println("[WHATSAPP] response status:", resp.StatusCode)
		fmt.Println("[WHATSAPP] response body:", string(bodyResp))

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		// for non-2xx, capture error and retry (up to attempts)
		lastErr = fmt.Errorf("non-2xx status: %d", resp.StatusCode)
		time.Sleep(backoff)
		backoff *= 2
	}
	return lastErr
}

// WhatsAppWebhookHandler handles incoming webhook requests from WhatsApp.
type WhatsAppWebhookHandler struct {
	service ChatbotService
	sender  Sender
}

// NewWhatsAppWebhookHandler creates a handler with an injected Sender.
func NewWhatsAppWebhookHandler(service ChatbotService, sender Sender) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{service: service, sender: sender}
}

// NewWhatsAppWebhookHandlerDefault is a convenience constructor that uses the
// default HTTP sender (SendWhatsAppMessage) for production.
func NewWhatsAppWebhookHandlerDefault(service ChatbotService) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{service: service, sender: httpSender{}}
}

// WhatsAppWebhookPayload representa o payload recebido do webhook do WhatsApp Cloud API.
type WhatsAppWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// HandleWhatsAppWebhook processa requisições GET (validação) e POST (mensagens) do webhook do WhatsApp.
func (h *WhatsAppWebhookHandler) HandleWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	// Validação do webhook pelo Meta (GET)
	// Processamento de mensagens recebidas (POST)
	// Para cada mensagem recebida, processa com o fluxo do chatbot e responde via API do WhatsApp
	if r.Method == "GET" {
		mode := r.URL.Query().Get("hub.mode")
		verifyToken := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")
		envToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
		fmt.Println("[WEBHOOK] mode:", mode)
		fmt.Println("[WEBHOOK] verifyToken (req):", verifyToken)
		fmt.Println("[WEBHOOK] verifyToken (env):", envToken)
		fmt.Println("[WEBHOOK] challenge:", challenge)
		if mode == "subscribe" && verifyToken == envToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden: token mismatch or mode error"))
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read raw body bytes for HMAC verification and JSON parsing
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Determine app secret (support WHATSAPP_APP_SECRET or META_APP_SECRET)
	appSecret := security.GetSecret("WHATSAPP_APP_SECRET")
	if appSecret == "" {
		appSecret = security.GetSecret("META_APP_SECRET")
	}

	// Determine if verification should be enforced (fail-closed)
	forceVerify := false
	if strings.ToLower(os.Getenv("FORCE_VERIFY")) == "true" {
		forceVerify = true
	}
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("ENV"))
	}
	if env == "production" {
		forceVerify = true
	}

	if appSecret != "" {
		sigHeader := r.Header.Get("X-Hub-Signature-256")
		if sigHeader == "" {
			// Missing signature
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing X-Hub-Signature-256 header"))
			return
		}

		// Header format is "sha256=hex"
		parts := strings.SplitN(sigHeader, "=", 2)
		if len(parts) != 2 || parts[0] != "sha256" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid X-Hub-Signature-256 header format"))
			return
		}
		sigHex := parts[1]
		sigBytes, err := hex.DecodeString(sigHex)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid signature hex"))
			return
		}

		mac := hmac.New(sha256.New, []byte(appSecret))
		mac.Write(body)
		expected := mac.Sum(nil)

		if len(expected) != len(sigBytes) || subtle.ConstantTimeCompare(expected, sigBytes) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid webhook signature"))
			return
		}
	} else {
		if forceVerify {
			// In enforced mode, secret missing is a server misconfiguration
			fmt.Println("[WEBHOOK] HMAC verification required but no secret configured (WHATSAPP_APP_SECRET or META_APP_SECRET)")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server misconfiguration: webhook verification required but secret missing"))
			return
		}
		// If secret not set and not enforcing, skip verification but log a warning
		fmt.Println("[WEBHOOK] No WHATSAPP_APP_SECRET/META_APP_SECRET set — skipping X-Hub-Signature verification")
	}

	var payload WhatsAppWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				from := msg.From
				text := msg.Text.Body
				if strings.TrimSpace(text) == "" {
					continue
				}
				response, err := h.service.ProcessMessage(from, text)
				if err == nil {
					// Use injected sender to allow mocking in tests and avoid
					// real network calls during unit tests.
					if h.sender != nil {
						h.sender.Send(from, response)
					} else {
						// Fallback to helper if sender unexpectedly nil
						SendWhatsAppMessage(from, response)
					}
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

// SendWhatsAppMessage envia uma mensagem de texto para um usuário via WhatsApp Cloud API.
func SendWhatsAppMessage(to, message string) error {
	// For backward compatibility keep helper but use hardened httpSender
	return httpSender{}.Send(to, message)
}
