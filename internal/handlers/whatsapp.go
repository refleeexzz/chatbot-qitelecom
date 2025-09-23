// Package handlers contém os handlers HTTP do sistema, incluindo integração com o WhatsApp Cloud API.
package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// WhatsAppWebhookHandler lida com requisições do webhook do WhatsApp Cloud API.
type WhatsAppWebhookHandler struct {
	service ChatbotService
}

// NewWhatsAppWebhookHandler cria um novo handler para o webhook do WhatsApp.
func NewWhatsAppWebhookHandler(service ChatbotService) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{service: service}
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

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
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
					SendWhatsAppMessage(from, response)
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

// SendWhatsAppMessage envia uma mensagem de texto para um usuário via WhatsApp Cloud API.
func SendWhatsAppMessage(to, message string) error {
	phoneID := os.Getenv("WHATSAPP_PHONE_ID")
	token := os.Getenv("WHATSAPP_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", phoneID)

	fmt.Println("[WHATSAPP] phoneID:", phoneID)
	fmt.Println("[WHATSAPP] token (first 8 chars):", func() string {
		if len(token) > 8 {
			return token[:8] + "..."
		} else {
			return token
		}
	}())
	fmt.Println("[WHATSAPP] url:", url)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": message},
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyResp, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("[WHATSAPP] response status:", resp.StatusCode)
	fmt.Println("[WHATSAPP] response body:", string(bodyResp))
	return nil
}
