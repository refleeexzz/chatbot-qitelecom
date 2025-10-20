package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

type AdminHandler struct {
	db    *sql.DB
	redis *redis.Client
}

type ContactInfo struct {
	Phone      string `json:"phone"`
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`
	Type       string `json:"type,omitempty"`
	OptIn      bool   `json:"opt_in"`
	CreatedAt  string `json:"created_at,omitempty"`
	Source     string `json:"source"` // "database" ou "redis"
	LastActive string `json:"last_active,omitempty"`
}

type AdminResponse struct {
	TotalContacts    int           `json:"total_contacts"`
	DatabaseContacts int           `json:"database_contacts"`
	RedisKeys        int           `json:"redis_keys"`
	Contacts         []ContactInfo `json:"contacts"`
	Message          string        `json:"message,omitempty"`
}

func NewAdminHandler(db *sql.DB, redisClient *redis.Client) *AdminHandler {
	return &AdminHandler{
		db:    db,
		redis: redisClient,
	}
}

// HandleGetContacts retorna todos os contatos do banco e Redis
func (h *AdminHandler) HandleGetContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	var contacts []ContactInfo

	// 1. Buscar contatos do banco de dados
	dbContacts, err := h.getContactsFromDatabase()
	if err != nil {
		log.Error().Err(err).Msg("Erro ao buscar contatos do banco")
	} else {
		contacts = append(contacts, dbContacts...)
	}

	// 2. Buscar chaves do Redis
	redisContacts, err := h.getContactsFromRedis()
	if err != nil {
		log.Error().Err(err).Msg("Erro ao buscar contatos do Redis")
	} else {
		contacts = append(contacts, redisContacts...)
	}

	response := AdminResponse{
		TotalContacts:    len(contacts),
		DatabaseContacts: len(dbContacts),
		RedisKeys:        len(redisContacts),
		Contacts:         contacts,
		Message:          "Contatos carregados com sucesso",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getContactsFromDatabase busca contatos do SQLite
func (h *AdminHandler) getContactsFromDatabase() ([]ContactInfo, error) {
	var contacts []ContactInfo

	query := `SELECT nome, telefone, email, tipo, opt_in, created_at FROM leads ORDER BY created_at DESC`
	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var contact ContactInfo
		var createdAt time.Time
		var name, phone, email, contactType sql.NullString
		var optIn bool

		err := rows.Scan(&name, &phone, &email, &contactType, &optIn, &createdAt)
		if err != nil {
			log.Warn().Err(err).Msg("Erro ao escanear contato")
			continue
		}

		contact.Name = name.String
		contact.Phone = phone.String
		contact.Email = email.String
		contact.Type = contactType.String
		contact.OptIn = optIn
		contact.CreatedAt = createdAt.Format("02/01/2006 15:04")
		contact.Source = "database"

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// getContactsFromRedis busca chaves do Redis
func (h *AdminHandler) getContactsFromRedis() ([]ContactInfo, error) {
	var contacts []ContactInfo
	ctx := context.Background()

	// Buscar todas as chaves de usuários (padrão: user:* ou session:*)
	keys, err := h.redis.Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		// Pegar TTL da chave para ver quando expira
		ttl, _ := h.redis.TTL(ctx, key).Result()

		contact := ContactInfo{
			Phone:  key,
			Source: "redis",
			Type:   "Session/Cache",
		}

		if ttl > 0 {
			contact.LastActive = ttl.String() + " restantes"
		} else if ttl == -1 {
			contact.LastActive = "Sem expiração"
		}

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// HandleAddContact adiciona um novo contato ao banco
func (h *AdminHandler) HandleAddContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	var input struct {
		Nome     string `json:"nome"`
		Telefone string `json:"telefone"`
		Email    string `json:"email"`
		Tipo     string `json:"tipo"`
		OptIn    bool   `json:"opt_in"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Formato JSON inválido",
		})
		return
	}

	// Validações
	if input.Nome == "" || input.Telefone == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Nome e telefone são obrigatórios",
		})
		return
	}

	// Inserir no banco
	query := `INSERT INTO leads (nome, telefone, email, tipo, opt_in, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := h.db.Exec(query, input.Nome, input.Telefone, input.Email, input.Tipo, input.OptIn, time.Now())
	if err != nil {
		log.Error().Err(err).Msg("Erro ao adicionar contato")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Erro ao adicionar contato: " + err.Error(),
		})
		return
	}

	log.Info().Str("nome", input.Nome).Str("telefone", input.Telefone).Msg("Contato adicionado")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Contato adicionado com sucesso",
	})
}

// HandleDeleteContact remove um contato do banco
func (h *AdminHandler) HandleDeleteContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "DELETE" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	var input struct {
		Telefone string `json:"telefone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Formato JSON inválido",
		})
		return
	}

	if input.Telefone == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Telefone é obrigatório",
		})
		return
	}

	// Deletar do banco
	query := `DELETE FROM leads WHERE telefone = ?`
	result, err := h.db.Exec(query, input.Telefone)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao deletar contato")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Erro ao deletar contato: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Contato não encontrado",
		})
		return
	}

	log.Info().Str("telefone", input.Telefone).Msg("Contato removido")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Contato removido com sucesso",
	})
}

// HandleUpdateOptIn atualiza o status de opt-in de um contato
func (h *AdminHandler) HandleUpdateOptIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	var input struct {
		Telefone string `json:"telefone"`
		OptIn    bool   `json:"opt_in"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Formato JSON inválido",
		})
		return
	}

	if input.Telefone == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Telefone é obrigatório",
		})
		return
	}

	// Atualizar opt-in no banco
	query := `UPDATE leads SET opt_in = ? WHERE telefone = ?`
	result, err := h.db.Exec(query, input.OptIn, input.Telefone)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao atualizar opt-in")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Erro ao atualizar opt-in: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Contato não encontrado",
		})
		return
	}

	log.Info().Str("telefone", input.Telefone).Bool("opt_in", input.OptIn).Msg("Opt-in atualizado")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Opt-in atualizado com sucesso",
	})
}
