package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"leadprojectarrumado/internal/security"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	excelize "github.com/xuri/excelize/v2"
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

// ImportRowResult descreve o resultado da tentativa de importar uma linha da planilha
type ImportRowResult struct {
	Line   int    `json:"line"`
	Name   string `json:"name,omitempty"`
	Phone  string `json:"phone"`
	Email  string `json:"email,omitempty"`
	Result string `json:"result"`           // inserted | skipped | error
	Reason string `json:"reason,omitempty"` // motivo quando pulado ou erro
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

	log.Info().Str("nome", input.Nome).Str("telefone", security.MaskPhone(input.Telefone)).Msg("Contato adicionado")

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

	log.Info().Str("telefone", security.MaskPhone(input.Telefone)).Msg("Contato removido")

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

	log.Info().Str("telefone", security.MaskPhone(input.Telefone)).Bool("opt_in", input.OptIn).Msg("Opt-in atualizado")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Opt-in atualizado com sucesso",
	})
}

// HandleImportContacts aceita um upload de planilha (CSV) e importa contatos para a tabela leads.
// O endpoint espera um multipart/form-data com campo 'file'. Formato suportado: CSV com colunas nome, telefone, email (nomes de colunas são case-insensitive e podem estar em português/inglês).
func (h *AdminHandler) HandleImportContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	// Limitar tamanho aceitável (por exemplo 8MB)
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Erro ao processar upload: " + err.Error()})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Campo 'file' é obrigatório"})
		return
	}
	defer file.Close()

	// Ler conteúdo em memória (aceitável para uploads pequenos)
	buf := make([]byte, 0)
	tmp := make([]byte, 4096)
	for {
		n, err := file.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			break
		}
	}

	var records [][]string

	// Detectar XLSX (arquivo ZIP que começa com PK) e usar excelize; caso contrário tentar CSV
	if len(buf) >= 2 && buf[0] == 'P' && buf[1] == 'K' {
		f, err := excelize.OpenReader(bytes.NewReader(buf))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Erro ao ler arquivo XLSX: " + err.Error()})
			return
		}
		// Pegar primeira sheet
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "XLSX sem planilha"})
			return
		}
		rows, err := f.GetRows(sheets[0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Erro ao ler linhas XLSX: " + err.Error()})
			return
		}
		records = rows
	} else {
		// Tentar parsear como CSV
		reader := csv.NewReader(bytes.NewReader(buf))
		reader.TrimLeadingSpace = true
		recs, err := reader.ReadAll()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Erro ao ler CSV: " + err.Error()})
			return
		}
		records = recs
	}

	if len(records) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Arquivo vazio"})
		return
	}

	// Detectar índice das colunas (nome, telefone, email)
	header := records[0]
	idxName, idxPhone, idxEmail := -1, -1, -1
	for i, hcol := range header {
		lc := strings.ToLower(strings.TrimSpace(hcol))
		switch lc {
		case "nome", "name", "full name", "full_name":
			idxName = i
		case "telefone", "phone", "telefone celular", "celular", "phone_number":
			idxPhone = i
		case "email", "e-mail":
			idxEmail = i
		}
	}

	if idxPhone == -1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Arquivo CSV deve conter coluna de telefone (ex: telefone, phone)"})
		return
	}

	inserted := 0
	skipped := 0
	var results []ImportRowResult

	// Preparar statement de inserção
	insertQuery := `INSERT INTO leads (nome, telefone, email, tipo, opt_in, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := h.db.Prepare(insertQuery)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao preparar statement de inserção")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Erro interno"})
		return
	}
	defer stmt.Close()

	now := time.Now()

	for i := 1; i < len(records); i++ {
		row := records[i]
		if idxPhone >= len(row) {
			skipped++
			results = append(results, ImportRowResult{Line: i + 1, Result: "skipped", Reason: "coluna telefone ausente"})
			continue
		}
		rawPhone := strings.TrimSpace(row[idxPhone])
		phone := normalizePhone(rawPhone)
		if phone == "" {
			skipped++
			name := ""
			if idxName >= 0 && idxName < len(row) {
				name = strings.TrimSpace(row[idxName])
			}
			email := ""
			if idxEmail >= 0 && idxEmail < len(row) {
				email = strings.TrimSpace(row[idxEmail])
			}
			results = append(results, ImportRowResult{Line: i + 1, Name: name, Phone: rawPhone, Email: email, Result: "skipped", Reason: "telefone inválido"})
			continue
		}

		name := ""
		if idxName >= 0 && idxName < len(row) {
			name = strings.TrimSpace(row[idxName])
		}
		email := ""
		if idxEmail >= 0 && idxEmail < len(row) {
			email = strings.TrimSpace(row[idxEmail])
		}

		// Verificar duplicado por telefone antes de inserir
		var exists int
		err = h.db.QueryRow("SELECT COUNT(1) FROM leads WHERE telefone = ?", phone).Scan(&exists)
		if err != nil {
			log.Warn().Err(err).Msg("Erro ao verificar duplicado")
			skipped++
			results = append(results, ImportRowResult{Line: i + 1, Name: name, Phone: phone, Email: email, Result: "error", Reason: "erro ao verificar duplicado: " + err.Error()})
			continue
		}
		if exists > 0 {
			skipped++
			results = append(results, ImportRowResult{Line: i + 1, Name: name, Phone: phone, Email: email, Result: "skipped", Reason: "duplicado"})
			continue
		}

		// Inserir no banco
		_, err := stmt.Exec(name, phone, email, "Lead", false, now)
		if err != nil {
			log.Warn().Err(err).Msg("Falha ao inserir contato, pulando")
			skipped++
			results = append(results, ImportRowResult{Line: i + 1, Name: name, Phone: phone, Email: email, Result: "error", Reason: "erro ao inserir: " + err.Error()})
			continue
		}
		inserted++
		results = append(results, ImportRowResult{Line: i + 1, Name: name, Phone: phone, Email: email, Result: "inserted"})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"inserted": inserted,
		"skipped":  skipped,
		"results":  results,
	})
}

// normalizePhone normaliza números removendo tudo que não for dígito e garantindo DDI 55 quando apropriado.
func normalizePhone(raw string) string {
	digits := make([]rune, 0, len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	s := string(digits)
	if s == "" {
		return ""
	}
	// Se já começa com 55 e tem tamanho razoável
	if strings.HasPrefix(s, "55") && len(s) >= 12 {
		return s
	}
	// Se tem 11 dígitos (DDD + número) -> prefixar 55
	if len(s) == 11 {
		return "55" + s
	}
	// Se tem 10 dígitos (sem 9 em celular), também prefixar 55
	if len(s) == 10 {
		return "55" + s
	}
	// Se tem 9 dígitos (número local) assume DDD ausente -> não importar
	if len(s) < 10 {
		return ""
	}
	// Caso geral: se tiver entre 11-13, tentar prefixar 55 se faltar
	if len(s) >= 11 && !strings.HasPrefix(s, "55") {
		return "55" + s
	}
	return s
}
