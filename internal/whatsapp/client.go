package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"leadprojectarrumado/internal/security"

	"github.com/rs/zerolog/log"
)

type Client struct {
	AccessToken string
	PhoneID     string
	httpClient  *http.Client
}

type SendMessageRequest struct {
	MessagingProduct string      `json:"messaging_product"`
	To               string      `json:"to"`
	Type             string      `json:"type"`
	Text             TextContent `json:"text"`
}

type TextContent struct {
	Body string `json:"body"`
}

type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type SendTemplateRequest struct {
	MessagingProduct string          `json:"messaging_product"`
	To               string          `json:"to"`
	Type             string          `json:"type"`
	Template         TemplateContent `json:"template"`
}

type TemplateContent struct {
	Name       string              `json:"name"`
	Language   TemplateLanguage    `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

type TemplateLanguage struct {
	Code string `json:"code"`
}

type TemplateComponent struct {
	Type       string              `json:"type"`
	Parameters []TemplateParameter `json:"parameters,omitempty"`
}

type TemplateParameter struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Image *MediaParameter `json:"image,omitempty"`
}

type MediaParameter struct {
	Link string `json:"link"`
}

func NewClient() *Client {
	return &Client{
		AccessToken: security.GetSecret("WHATSAPP_TOKEN"),
		PhoneID:     security.GetSecret("WHATSAPP_PHONE_ID"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// normalizePhoneNumber normaliza números de telefone brasileiros
// Remove o 9º dígito extra dos celulares que têm 13 dígitos
// Formato esperado: 5544998526479 (13 dígitos) -> 554498526479 (12 dígitos)
func normalizePhoneNumber(phone string) string {
	// Remover caracteres não numéricos
	re := regexp.MustCompile(`[^0-9]`)
	phone = re.ReplaceAllString(phone, "")

	// Se o número tem 13 dígitos E começa com 55 (Brasil)
	if len(phone) == 13 && strings.HasPrefix(phone, "55") {
		// Extrair partes: 55 + DDD (2 dígitos) + restante
		countryCode := phone[0:2] // "55"
		areaCode := phone[2:4]    // DDD (ex: "44")
		ninthDigit := phone[4:5]  // 9º dígito (deve ser "9")
		number := phone[5:13]     // 8 dígitos restantes

		// Se o 5º dígito é "9", remover (formato antigo brasileiro)
		if ninthDigit == "9" {
			normalized := countryCode + areaCode + number
			log.Debug().
				Str("original", phone).
				Str("normalized", normalized).
				Msg("📞 Número normalizado (removido 9 extra)")
			return normalized
		}
	}

	// Se já tem 12 dígitos ou outro formato, retornar como está
	return phone
}

// SendMessage envia uma mensagem de texto para um número do WhatsApp
// Implementa retry com backoff exponencial para erros 429 (Too Many Requests)
func (c *Client) SendMessage(to, message string) error {
	if c.AccessToken == "" || c.PhoneID == "" {
		return fmt.Errorf("WHATSAPP_TOKEN ou WHATSAPP_PHONE_ID não configurados")
	}

	// Normalizar número de telefone
	to = normalizePhoneNumber(to)

	// Retry configuration
	maxRetries := 5
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.sendMessageAttempt(to, message)

		// Se não houve erro, retorna sucesso
		if err == nil {
			return nil
		}

		// Se é erro 429, tenta novamente com backoff exponencial
		if isRateLimitError(err) && attempt < maxRetries {
			// Backoff exponencial: 1s, 2s, 4s, 8s, 16s
			delay := baseDelay * time.Duration(1<<attempt)
			log.Warn().
				Str("phone", to).
				Int("attempt", attempt+1).
				Dur("retry_in", delay).
				Msg("Rate limit atingido, aguardando para retry")
			time.Sleep(delay)
			continue
		}

		// Outros erros ou último retry falhou
		return err
	}

	return fmt.Errorf("máximo de tentativas atingido após rate limit")
}

// sendMessageAttempt tenta enviar uma mensagem (uma tentativa)
func (c *Client) sendMessageAttempt(to, message string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", c.PhoneID)

	payload := SendMessageRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text: TextContent{
			Body: message,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao criar payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar requisição: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &WhatsAppError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var response SendMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Warn().Err(err).Msg("Erro ao parsear resposta da API WhatsApp")
	}

	return nil
}

// WhatsAppError representa um erro da API do WhatsApp
type WhatsAppError struct {
	StatusCode int
	Message    string
}

func (e *WhatsAppError) Error() string {
	return fmt.Sprintf("erro na API WhatsApp (status %d): %s", e.StatusCode, e.Message)
}

// isRateLimitError verifica se é erro 429 (Too Many Requests)
func isRateLimitError(err error) bool {
	if waErr, ok := err.(*WhatsAppError); ok {
		return waErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

// SendTemplateMessage envia uma mensagem de template (para fora da janela de 24h)
// Implementa retry com backoff exponencial para erros 429 (Too Many Requests)
// languageCode deve ser um código de idioma válido como: pt_BR, en_US, es_ES, etc.
func (c *Client) SendTemplateMessage(to, templateName, languageCode string, params []string) error {
	return c.SendTemplateMessageWithMedia(to, templateName, languageCode, params, "")
}

// SendTemplateMessageWithMedia envia template com suporte a imagem no header
// headerImageURL: URL pública da imagem para o header (deixe vazio se não tiver header)
func (c *Client) SendTemplateMessageWithMedia(to, templateName, languageCode string, params []string, headerImageURL string) error {
	if c.AccessToken == "" || c.PhoneID == "" {
		return fmt.Errorf("WHATSAPP_TOKEN ou WHATSAPP_PHONE_ID não configurados")
	}

	// Normalizar número de telefone
	to = normalizePhoneNumber(to)

	// Se languageCode estiver vazio, usar pt_BR como padrão
	if languageCode == "" {
		languageCode = "pt_BR"
	}

	// Retry configuration
	maxRetries := 5
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.sendTemplateMessageAttempt(to, templateName, languageCode, params, headerImageURL)

		// Se não houve erro, retorna sucesso
		if err == nil {
			return nil
		}

		// Se é erro 429, tenta novamente com backoff exponencial
		if isRateLimitError(err) && attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<attempt)
			log.Warn().
				Str("phone", to).
				Int("attempt", attempt+1).
				Dur("retry_in", delay).
				Msg("Rate limit atingido no template, aguardando para retry")
			time.Sleep(delay)
			continue
		}

		// Outros erros ou último retry falhou
		return err
	}

	return fmt.Errorf("máximo de tentativas atingido após rate limit")
}

// sendTemplateMessageAttempt tenta enviar um template (uma tentativa)
func (c *Client) sendTemplateMessageAttempt(to, templateName, languageCode string, params []string, headerImageURL string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", c.PhoneID)

	// Log de debug para verificar parâmetros recebidos
	log.Debug().
		Str("phone", to).
		Str("template", templateName).
		Str("language", languageCode).
		Strs("params", params).
		Str("header_image_url", headerImageURL).
		Msg("🔍 Preparando envio de template")

	// Construir componentes
	var components []TemplateComponent

	// IMPORTANTE: Se a imagem do header for FIXA no template (não variável),
	// NÃO enviar componente header. Só enviar se for uma variável {{1}} no header.
	// A maioria dos templates tem imagem fixa, então só enviamos header se
	// headerImageURL for fornecido E o template tiver header com variável de imagem.
	if headerImageURL != "" {
		log.Info().Str("image_url", headerImageURL).Msg("📸 Adicionando imagem VARIÁVEL no header do template")
		components = append(components, TemplateComponent{
			Type: "header",
			Parameters: []TemplateParameter{
				{
					Type: "image",
					Image: &MediaParameter{
						Link: headerImageURL,
					},
				},
			},
		})
	}

	// Se tiver parâmetros de texto, adicionar componente BODY
	if len(params) > 0 {
		var parameters []TemplateParameter
		for _, param := range params {
			parameters = append(parameters, TemplateParameter{
				Type: "text",
				Text: param,
			})
		}
		components = append(components, TemplateComponent{
			Type:       "body",
			Parameters: parameters,
		})
	}

	payload := SendTemplateRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "template",
		Template: TemplateContent{
			Name: templateName,
			Language: TemplateLanguage{
				Code: languageCode,
			},
			Components: components,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao criar payload: %w", err)
	}

	// Log do payload completo para debug
	log.Debug().
		Str("phone_number_id", c.PhoneID).
		Str("to", to).
		RawJSON("payload", data).
		Msg("📤 Enviando requisição para WhatsApp API")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar requisição: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Log da resposta da API para template
	log.Debug().
		Int("status_code", resp.StatusCode).
		Str("response_body", string(body)).
		Msg("📥 Resposta da WhatsApp API (Template)")

	if resp.StatusCode != http.StatusOK {
		return &WhatsAppError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var response SendMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Warn().Err(err).Msg("Erro ao parsear resposta da API WhatsApp")
	}

	return nil
}

// SendBulkMessages envia mensagens para múltiplos destinatários com rate limiting
func (c *Client) SendBulkMessages(phones []string, message string) (sent int, failed int, errors []string) {
	// Meta permite ~80 mensagens por minuto para evitar bloqueios
	// Usamos 1.5s entre mensagens = ~40 msg/min para ser conservador
	rateLimitDelay := 1500 * time.Millisecond

	for _, phone := range phones {
		if phone == "" {
			continue
		}

		err := c.SendMessage(phone, message)
		if err != nil {
			failed++
			errMsg := fmt.Sprintf("Erro ao enviar para %s: %v", phone, err)
			errors = append(errors, errMsg)
			log.Error().Str("phone", phone).Err(err).Msg("Falha ao enviar mensagem")
		} else {
			sent++
			log.Info().Str("phone", phone).Msg("Mensagem enviada com sucesso")
		}

		// Rate limiting entre mensagens
		time.Sleep(rateLimitDelay)
	}

	return sent, failed, errors
}

// SendBulkTemplateMessages envia template messages para múltiplos destinatários
func (c *Client) SendBulkTemplateMessages(phones []string, templateName, languageCode string, params []string) (sent int, failed int, errors []string) {
	return c.SendBulkTemplateMessagesWithMedia(phones, templateName, languageCode, params, "")
}

// SendBulkTemplateMessagesWithMedia envia template messages com imagem no header para múltiplos destinatários
func (c *Client) SendBulkTemplateMessagesWithMedia(phones []string, templateName, languageCode string, params []string, headerImageURL string) (sent int, failed int, errors []string) {
	rateLimitDelay := 1500 * time.Millisecond

	for _, phone := range phones {
		if phone == "" {
			continue
		}

		err := c.SendTemplateMessageWithMedia(phone, templateName, languageCode, params, headerImageURL)
		if err != nil {
			failed++
			errMsg := fmt.Sprintf("Erro ao enviar template para %s: %v", phone, err)
			errors = append(errors, errMsg)
			log.Error().Str("phone", phone).Err(err).Msg("Falha ao enviar template")
		} else {
			sent++
			log.Info().Str("phone", phone).Str("template", templateName).Msg("Template enviado com sucesso")
		}

		// Rate limiting entre mensagens
		time.Sleep(rateLimitDelay)
	}

	return sent, failed, errors
}
