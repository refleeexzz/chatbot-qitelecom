package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"leadprojectarrumado/internal/security"
	"leadprojectarrumado/internal/whatsapp"

	"github.com/rs/zerolog/log"
)

// isAllDigits retorna true se a string contém apenas dígitos.
func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type BulkMessageHandler struct {
	db             *sql.DB
	whatsappClient *whatsapp.Client
}

type BulkMessageRequest struct {
	Message      string   `json:"message"`
	Phones       []string `json:"phones,omitempty"`         // Lista específica de telefones
	SendToAll    bool     `json:"send_to_all,omitempty"`    // Enviar para todos os leads
	FilterByType string   `json:"filter_by_type,omitempty"` // Filtrar por tipo (Lead, Cliente, etc)
}

type BulkMessageResponse struct {
	Status  string   `json:"status"`
	Sent    int      `json:"sent"`
	Failed  int      `json:"failed"`
	Total   int      `json:"total"`
	Errors  []string `json:"errors,omitempty"`
	Message string   `json:"message"`
}

func NewBulkMessageHandler(db *sql.DB) *BulkMessageHandler {
	return &BulkMessageHandler{
		db:             db,
		whatsappClient: whatsapp.NewClient(),
	}
}

// HandleBulkMessage processa envio em lote de mensagens
func (h *BulkMessageHandler) HandleBulkMessage(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Método não permitido. Use POST.",
		})
		return
	}

	var req BulkMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "JSON inválido",
		})
		return
	}

	// Validação de tipos e sanitização
	if len(req.Message) > 1024 {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Mensagem muito longa",
		})
		return
	}

	for _, phone := range req.Phones {
		if len(phone) < 10 || len(phone) > 15 || !isAllDigits(phone) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Telefone inválido detectado",
			})
			return
		}
	}

	// Validar mensagem
	if strings.TrimSpace(req.Message) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "A mensagem não pode estar vazia",
		})
		return
	}

	var phones []string

	// Determinar lista de telefones
	if len(req.Phones) > 0 {
		// Usar lista fornecida
		phones = req.Phones
	} else if req.SendToAll {
		// Buscar todos os leads do banco
		var err error
		phones, err = h.getPhonesFromDatabase(req.FilterByType)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao buscar telefones do banco")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Erro ao buscar contatos do banco de dados",
			})
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Você precisa fornecer 'phones' ou marcar 'send_to_all' como true",
		})
		return
	}

	if len(phones) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Nenhum telefone encontrado para envio",
		})
		return
	}

	log.Info().Int("total", len(phones)).Msg("Iniciando envio em lote")

	// Enviar mensagens
	sent, failed, errors := h.whatsappClient.SendBulkMessages(phones, req.Message)

	response := BulkMessageResponse{
		Status:  "success",
		Sent:    sent,
		Failed:  failed,
		Total:   len(phones),
		Errors:  errors,
		Message: fmt.Sprintf("Disparo concluído: %d enviadas, %d falhas", sent, failed),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LeadContact representa um contato com nome e telefone
type LeadContact struct {
	Nome     string
	Telefone string
}

// getPhonesFromDatabase busca telefones dos leads no banco de dados
// COMPLIANCE: Apenas contatos com opt_in = true para evitar spam
func (h *BulkMessageHandler) getPhonesFromDatabase(filterType string) ([]string, error) {
	var phones []string

	// CRÍTICO: Adicionar filtro opt_in = 1 para compliance com Meta WhatsApp
	query := "SELECT DISTINCT telefone FROM leads WHERE telefone IS NOT NULL AND telefone != '' AND opt_in = 1"
	args := []interface{}{}

	if filterType != "" {
		query += " AND tipo = ?"
		args = append(args, filterType)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var phone string
		if err := rows.Scan(&phone); err != nil {
			log.Warn().Err(err).Msg("Erro ao escanear telefone")
			continue
		}
		// Limpar e validar telefone
		phone = strings.TrimSpace(phone)
		if phone != "" {
			phones = append(phones, phone)
		}
	}

	return phones, nil
}

// getLeadContactsFromDatabase busca nome e telefone dos leads no banco de dados
// COMPLIANCE: Apenas contatos com opt_in = true para evitar spam
func (h *BulkMessageHandler) getLeadContactsFromDatabase(filterType string) ([]LeadContact, error) {
	var contacts []LeadContact

	// CRÍTICO: Adicionar filtro opt_in = 1 para compliance com Meta WhatsApp
	query := "SELECT nome, telefone FROM leads WHERE telefone IS NOT NULL AND telefone != '' AND opt_in = 1"
	args := []interface{}{}

	if filterType != "" {
		query += " AND tipo = ?"
		args = append(args, filterType)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var contact LeadContact
		if err := rows.Scan(&contact.Nome, &contact.Telefone); err != nil {
			log.Warn().Err(err).Msg("Erro ao escanear contato")
			continue
		}
		// Limpar e validar
		contact.Nome = strings.TrimSpace(contact.Nome)
		contact.Telefone = strings.TrimSpace(contact.Telefone)
		if contact.Telefone != "" {
			contacts = append(contacts, contact)
		}
	}

	return contacts, nil
}

// HandleGetOptInStats retorna estatísticas de opt-in para o admin
func (h *BulkMessageHandler) HandleGetOptInStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	filterType := r.URL.Query().Get("type")

	var stats struct {
		Total      int `json:"total"`
		OptedIn    int `json:"opted_in"`
		NotOptedIn int `json:"not_opted_in"`
	}

	// Contar total
	query := "SELECT COUNT(*) FROM leads WHERE telefone IS NOT NULL AND telefone != ''"
	args := []interface{}{}
	if filterType != "" {
		query += " AND tipo = ?"
		args = append(args, filterType)
	}
	h.db.QueryRow(query, args...).Scan(&stats.Total)

	// Contar opt-in
	query = "SELECT COUNT(*) FROM leads WHERE telefone IS NOT NULL AND telefone != '' AND opt_in = 1"
	args = []interface{}{}
	if filterType != "" {
		query += " AND tipo = ?"
		args = append(args, filterType)
	}
	h.db.QueryRow(query, args...).Scan(&stats.OptedIn)

	stats.NotOptedIn = stats.Total - stats.OptedIn

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// HandleBulkTemplateMessage processa envio em lote de template messages
func (h *BulkMessageHandler) HandleBulkTemplateMessage(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("⚡ HandleBulkTemplateMessage CHAMADO")

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
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Método não permitido. Use POST.",
		})
		return
	}

	var req struct {
		TemplateName   string   `json:"template_name"`
		Language       string   `json:"language,omitempty"`         // Código do idioma: pt_BR, en_US, es_ES, etc
		Parameters     []string `json:"parameters,omitempty"`       // Parâmetros para substituir {{1}}, {{2}}, etc
		UseContactName bool     `json:"use_contact_name,omitempty"` // Se true, {{1}} será substituído pelo nome do contato
		HeaderImageURL string   `json:"header_image_url,omitempty"` // URL pública da imagem para header do template
		Phones         []string `json:"phones,omitempty"`
		SendToAll      bool     `json:"send_to_all,omitempty"`
		FilterByType   string   `json:"filter_by_type,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "JSON inválido",
		})
		return
	}

	// Validação de tipos e sanitização
	if len(req.TemplateName) > 128 {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Nome do template muito longo",
		})
		return
	}

	for _, phone := range req.Phones {
		if len(phone) < 10 || len(phone) > 15 || !isAllDigits(phone) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Telefone inválido detectado",
			})
			return
		}
	}

	// Validar template name
	if strings.TrimSpace(req.TemplateName) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "O nome do template não pode estar vazio",
		})
		return
	}

	var phones []string

	// Determinar lista de telefones
	if len(req.Phones) > 0 {
		phones = req.Phones
	} else if req.SendToAll {
		var err error
		phones, err = h.getPhonesFromDatabase(req.FilterByType)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao buscar telefones do banco")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Erro ao buscar contatos do banco de dados",
			})
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Você precisa fornecer 'phones' ou marcar 'send_to_all' como true",
		})
		return
	}

	if len(phones) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Nenhum telefone encontrado para envio (apenas contatos com opt-in)",
		})
		return
	}

	// Se language não foi fornecido, usar pt_BR como padrão
	if req.Language == "" {
		req.Language = "pt_BR"
	}

	var sent, failed int
	var errors []string

	// Se use_contact_name = true E send_to_all = true, buscar nomes do banco
	if req.UseContactName && req.SendToAll {
		contacts, err := h.getLeadContactsFromDatabase(req.FilterByType)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao buscar contatos do banco")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Erro ao buscar contatos do banco de dados",
			})
			return
		}

		if len(contacts) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(BulkMessageResponse{
				Status:  "error",
				Message: "Nenhum contato encontrado para envio (apenas contatos com opt-in)",
			})
			return
		}

		log.Info().
			Int("total", len(contacts)).
			Str("template", req.TemplateName).
			Str("language", req.Language).
			Str("header_image_url", req.HeaderImageURL).
			Bool("personalized", true).
			Msg("Iniciando envio em lote de templates PERSONALIZADOS")

		// Enviar template personalizado para cada contato
		for _, contact := range contacts {
			// Criar parâmetros personalizados: nome do contato + outros parâmetros fornecidos
			personalParams := []string{contact.Nome}
			if len(req.Parameters) > 0 {
				personalParams = append(personalParams, req.Parameters...)
			}

			err := h.whatsappClient.SendTemplateMessageWithMedia(contact.Telefone, req.TemplateName, req.Language, personalParams, req.HeaderImageURL)
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("Erro ao enviar para %s (%s): %v", contact.Nome, security.MaskPhone(contact.Telefone), err))
				log.Error().Err(err).Str("phone", security.MaskPhone(contact.Telefone)).Str("name", contact.Nome).Msg("Falha ao enviar template")
			} else {
				sent++
				log.Debug().Str("phone", security.MaskPhone(contact.Telefone)).Str("name", contact.Nome).Msg("Template enviado com sucesso")
			}
		}

		response := BulkMessageResponse{
			Status:  "success",
			Sent:    sent,
			Failed:  failed,
			Total:   len(contacts),
			Errors:  errors,
			Message: fmt.Sprintf("Disparo de templates PERSONALIZADOS concluído: %d enviadas, %d falhas", sent, failed),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Envio padrão (sem personalização de nome)
	if len(phones) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BulkMessageResponse{
			Status:  "error",
			Message: "Nenhum telefone encontrado para envio (apenas contatos com opt-in)",
		})
		return
	}

	log.Info().
		Int("total", len(phones)).
		Str("template", req.TemplateName).
		Str("language", req.Language).
		Msg("Iniciando envio em lote de templates")

	// Enviar templates
	sent, failed, errors = h.whatsappClient.SendBulkTemplateMessagesWithMedia(phones, req.TemplateName, req.Language, req.Parameters, req.HeaderImageURL)

	response := BulkMessageResponse{
		Status:  "success",
		Sent:    sent,
		Failed:  failed,
		Total:   len(phones),
		Errors:  errors,
		Message: fmt.Sprintf("Disparo de templates concluído: %d enviadas, %d falhas", sent, failed),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
