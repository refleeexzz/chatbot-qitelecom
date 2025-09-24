// Package services implementa a lógica de negócio do chatbot, incluindo fluxos de atendimento, integração com IA e persistência.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// ChatbotService implementa o fluxo de atendimento do chatbot, integrando Redis, banco de dados, Google Sheets e IA.
type ChatbotService struct {
	redis  *redis.Client
	db     *sql.DB
	sheets SheetsClient
	ai     AIClient
}

const planList = `• *QI FIBRA BASIC*
  300 Mega + QI TV PLAY + IPV6

• *QI FIBRA PREMIUM*
  600 Mega + QI TV PLAY + IPV6 + QUALIDADE QI

• *QI FIBRA PREMIUM (MELHOR)*
  650 Mega + QI TV PLAY + IPV6 + PARAMOUNT + WATCH TV

• *QI FIBRA PREMIUM TOP*
  700 Mega + QI TV PLAY + IPV6 + PARAMOUNT + WATCH TV`

// SheetsClient define interface para persistência de dados em Google Sheets.
type SheetsClient interface {
	SaveSupport(nome, problema, descricao, status string) error
	SavePlans(nome, situacao, planoAtual, planoDesejado, telefone, observacoes string) error
	SaveFeedback(nome, tipoAtendimento, feedback, sugestoes string) error
}

// AIClient define interface para geração de respostas automáticas por IA.
type AIClient interface {
	GenerateResponse(problema string) (string, error)
	GenerateFreeResponse(pergunta string) (string, error)
}

// UserData armazena o estado da sessão do usuário durante o atendimento.
type UserData struct {
	Nome               string `json:"nome"`
	Problema           string `json:"problema"`
	Descricao          string `json:"descricao"`
	PlanoAtual         string `json:"plano_atual"`
	PlanoDesejado      string `json:"plano_desejado"`
	Situacao           string `json:"situacao"`
	Telefone           string `json:"telefone"`
	TentativasIA       int    `json:"tentativas_ia"`
	TipoAtendimento    string `json:"tipo_atendimento"`
	AguardandoFeedback bool   `json:"aguardando_feedback"`
	UltimaAtividade    int64  `json:"ultima_atividade"`
}

// NewChatbotService cria instância do serviço de chatbot.
// NewChatbotService cria uma nova instância do serviço de chatbot.
func NewChatbotService(redis *redis.Client, db *sql.DB, sheets SheetsClient, ai AIClient) *ChatbotService {
	return &ChatbotService{
		redis:  redis,
		db:     db,
		sheets: sheets,
		ai:     ai,
	}
}

// ProcessMessage roteia a mensagem do usuário conforme o estado atual da sessão.
func (s *ChatbotService) ProcessMessage(userID, message string) (string, error) {
	userData := s.getUserData(userID)
	now := time.Now().Unix()
	if userData.UltimaAtividade > 0 && now-userData.UltimaAtividade > 600 {
		ctx := context.Background()
		s.redis.Del(ctx, "chat:"+userID)
		s.redis.Del(ctx, "data:"+userID)
		userData = UserData{}
	}
	userData.UltimaAtividade = now
	s.setUserData(userID, userData)
	ctx := context.Background()

	msgLower := strings.ToLower(strings.TrimSpace(message))
	if msgLower == "oi" || msgLower == "menu" {
		return s.showMainMenu(userID)
	}

	state, _ := s.redis.Get(ctx, "chat:"+userID).Result()
	if state == "" {
		return s.showMainMenu(userID)
	}

	switch state {
	case "menu":
		return s.handleMenuSelection(userID, message)
	case "support_name":
		return s.handleSupportName(userID, message)
	case "support_problem":
		return s.handleSupportProblem(userID, message)
	case "support_ia":
		return s.handleSupportIA(userID, message)
	case "support_feedback":
		return s.handleSupportFeedback(userID, message)
	case "plans_client_check":
		return s.handlePlansClientCheck(userID, message)
	case "plans_current":
		return s.handlePlansCurrent(userID, message)
	case "plans_name":
		return s.handlePlansName(userID, message)
	case "plans_phone":
		return s.handlePlansPhone(userID, message)
	case "plans_selection":
		return s.handlePlansSelection(userID, message)
	case "ai_free":
		return s.handleFreeAI(userID, message)
	default:
		return s.showMainMenu(userID)
	}
}

// showMainMenu reinicia o estado e retorna o menu principal do chatbot.
func (s *ChatbotService) showMainMenu(userID string) (string, error) {
	ctx := context.Background()

	s.redis.Del(ctx, "chat:"+userID)
	s.redis.Del(ctx, "data:"+userID)

	s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)

	return `*QI TELECOM | Menu Principal 🛰️*

Bem-vindo ao QIChatBot!
Digite apenas o *número* da opção desejada:

[1] Suporte Técnico
    - Problemas com internet, modem ou instalação

[2] Planos e Serviços
    - Conhecer planos ou solicitar upgrade

[3] Boleto e Financeiro
    - Segunda via e questões financeiras

[4] Assistente Livre
    - Chat livre para qualquer dúvida

Digite sua opção (1-4):`, nil
}

// handleMenuSelection processa a escolha do menu principal pelo usuário.
func (s *ChatbotService) handleMenuSelection(userID, message string) (string, error) {
	ctx := context.Background()
	option := strings.TrimSpace(message)

	switch option {
	case "1":
		s.redis.Set(ctx, "chat:"+userID, "support_name", time.Hour)
		userData := UserData{TipoAtendimento: "Suporte Técnico", TentativasIA: 0}
		s.setUserData(userID, userData)
		return "🔧 *Suporte Técnico Selecionado*\n\nPara melhor atendê-lo, preciso do seu *nome completo*:", nil

	case "2":
		s.redis.Set(ctx, "chat:"+userID, "plans_client_check", time.Hour)
		userData := UserData{TipoAtendimento: "Planos e Serviços"}
		s.setUserData(userID, userData)
		return "📋 *Planos e Serviços*\n\nVocê já é cliente QI TELECOM? Responda *SIM* ou *NÃO*.\n\n(Após responder, mostrarei as opções de planos.)", nil

	case "3":
		return s.showBoletoInfo(userID)

	case "4":
		s.redis.Set(ctx, "chat:"+userID, "ai_free", time.Hour)
		userData := UserData{TipoAtendimento: "IA Livre"}
		s.setUserData(userID, userData)
		return "🤖 *Assistente Livre Ativado*\n\nAgora você pode fazer qualquer pergunta que quiser! Estou aqui para ajudar.", nil

	default:
		return s.showMainMenu(userID)
	}
}

// showBoletoInfo retorna informações financeiras e canais de contato.
func (s *ChatbotService) showBoletoInfo(userID string) (string, error) {
	ctx := context.Background()
	s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)

	return `💰 *Boleto e Financeiro*

Para *segunda via* ou dúvidas financeiras, utilize os canais oficiais:

*Unidade / Responsável*
Francisco Alves: Av. Brigadeiro Faria Lima 703 - Centro | (44) 3643-1736

Iporã: Rua Katsuo Nakata 1115 - Centro | (44) 98402-7130 / (44) 3199-9115

Palotina: Aldir Pedron 1319 - Centro | (44) 3649-1486

Terra Roxa: Av. da Saudade 369 - Centro | (44) 3645-3257

⚠️ *Aplicativo de boletos em desenvolvimento. Em breve novidades.*

Digite MENU para voltar ao menu principal.`, nil
}

// handleSupportName armazena o nome do usuário e avança para o próximo passo do suporte.
func (s *ChatbotService) handleSupportName(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	userData.Nome = strings.TrimSpace(message)
	s.setUserData(userID, userData)

	s.redis.Set(ctx, "chat:"+userID, "support_problem", time.Hour)
	return fmt.Sprintf("Obrigado, %s! 👋\n\nAgora, descreva detalhadamente o problema técnico que você está enfrentando:", userData.Nome), nil
}

// handleSupportProblem armazena o problema relatado e inicia o suporte técnico.
func (s *ChatbotService) handleSupportProblem(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	userData.Problema = strings.TrimSpace(message)
	userData.Descricao = message
	s.setUserData(userID, userData)

	s.redis.Set(ctx, "chat:"+userID, "support_ia", time.Hour)
	return s.startTechnicalSupport(userID, message)
}

// startTechnicalSupport inicia o atendimento técnico, usando IA se disponível.
func (s *ChatbotService) startTechnicalSupport(userID, problema string) (string, error) {
	userData := s.getUserData(userID)
	userData.TentativasIA = 1
	s.setUserData(userID, userData)

	prompt := fmt.Sprintf(`Você é um técnico especializado em internet, modem e instalações da QI TELECOM. 
		Analise o problema relatado pelo cliente e forneça uma solução técnica detalhada e prática.
		O nome do cliente é: %s
		PROBLEMA: %s
   
		Forneça:
		1. Diagnóstico provável
		2. Solução passo a passo 
		3. Se não funcionar, próximos passos
   
		Seja técnico mas didático, lembrando que você está se relacionando com pessoas leigas no assunto. Não repita o problema ou o nome do cliente na resposta.`, userData.Nome, problema)

	if s.ai != nil {
		response, err := s.ai.GenerateResponse(prompt)
		if err == nil {
			return fmt.Sprintf("🔧 Analise Técnica - Tentativa 1/5\n\n%s\n\n---\nIsso resolveu seu problema?\n- Digite SIM se resolveu\n- Digite NAO se não resolveu", response), nil
		}
		log.Printf("IA indisponível para suporte técnico: %v", err)
	}

	return "🔧 Analise Técnica - Tentativa 1/5\n\nVamos diagnosticar seu problema passo a passo:\n\n1️⃣ Verifique as conexões - Confirme se todos os cabos estão bem conectados\n2️⃣ Reinicie o modem - Desligue por 30 segundos e ligue novamente\n3️⃣ Teste a velocidade - Use speedtest.net para verificar\n\nIsso resolveu seu problema?\n- Digite SIM se resolveu\n- Digite NAO se não resolveu", nil
}

// continueTechnicalSupport gera novas tentativas de solução técnica para o problema do usuário.
func (s *ChatbotService) continueTechnicalSupport(userID string, tentativa int, problema string) (string, error) {
	prompt := fmt.Sprintf(`Esta é a tentativa %d/5 de resolver este problema técnico. 
	Problema anterior: %s
	
	Forneça uma solução DIFERENTE e mais avançada. Seja mais específico e didatico para uma pessoa leiga. tente ser direto ao ponto, sem muita escrita.`, tentativa, problema)

	if s.ai != nil {
		response, err := s.ai.GenerateResponse(prompt)
		if err == nil {
			return fmt.Sprintf("🔧 *Nova Análise Técnica - Tentativa %d/5*\n\n%s\n\n---\n*Isso resolveu seu problema?*\n- Digite *SIM* se resolveu\n- Digite *NÃO* se não resolveu", tentativa, response), nil
		}
		log.Printf("IA indisponível para tentativa %d: %v", tentativa, err)
	}

	defaultSolutions := []string{
		"🔧 *Verificação de DNS*\n\n1️⃣ Altere o DNS para 177.39.208.2 e 177.39.208.3\n2️⃣ Limpe o cache DNS: `ipconfig /flushdns`\n3️⃣ Teste novamente",
		"🔧 *Verificação de Portas*\n\n1️⃣ Teste diferentes portas Ethernet\n2️⃣ Verifique se o cabo não está danificado\n3️⃣ Teste com outro dispositivo",
		"🔧 *Verificação de Sinal*\n\n1️⃣ Verifique atenuação da linha\n2️⃣ Confirme se não há interferências\n3️⃣ Teste isoladamente sem outros equipamentos",
	}

	solutionIndex := (tentativa - 2) % len(defaultSolutions)
	return fmt.Sprintf("%s\n\n*Isso resolveu seu problema?*\n- Digite *SIM* se resolveu\n- Digite *NÃO* se não resolveu", defaultSolutions[solutionIndex]), nil
}

// handlePlansClientCheck identifica se o usuário é cliente atual ou novo e direciona o fluxo.
func (s *ChatbotService) handlePlansClientCheck(userID, message string) (string, error) {
	ctx := context.Background()
	response := strings.ToLower(strings.TrimSpace(message))
	userData := s.getUserData(userID)

	if response == "sim" {
		userData.Situacao = "Cliente Atual"
		s.setUserData(userID, userData)
		s.redis.Set(ctx, "chat:"+userID, "plans_current", time.Hour)
		return "👤 *Cliente Atual Identificado*\n\nQual seu *plano atual*? Digite exatamente uma das opções abaixo:\n\n" + planList, nil
	}

	if response == "não" || response == "nao" {
		userData.Situacao = "Novo Cliente"
		userData.PlanoAtual = "Nenhum"
		s.setUserData(userID, userData)
		s.redis.Set(ctx, "chat:"+userID, "plans_selection", time.Hour)
		return "🆕 *Novo Cliente - Bem-vindo!*\n\nPerfeito! Qual plano desperta seu interesse?\n\n" + planList, nil
	}

	return "Por favor, responda *SIM* ou *NÃO*.", nil
}

// handlePlansCurrent armazena o plano atual informado pelo usuário.
func (s *ChatbotService) handlePlansCurrent(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	userData.PlanoAtual = strings.TrimSpace(message)
	s.setUserData(userID, userData)

	s.redis.Set(ctx, "chat:"+userID, "plans_selection", time.Hour)
	return fmt.Sprintf("📋 *Plano Atual: %s*\n\nGostaria de fazer *upgrade*? Veja nossas opções superiores:\n\n%s", userData.PlanoAtual, planList), nil
}

// handlePlansSelection armazena o plano desejado e avança para coleta de dados do usuário.
func (s *ChatbotService) handlePlansSelection(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	userData.PlanoDesejado = strings.TrimSpace(message)
	s.setUserData(userID, userData)

	if strings.ToLower(userData.PlanoDesejado) == "manter atual" {
		s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)
		return "✅ *Entendido!*\n\nVocê optou por manter seu plano atual. Se mudar de ideia, estaremos aqui!\n\nDigite *MENU* para voltar ao menu principal.", nil
	}

	s.redis.Set(ctx, "chat:"+userID, "plans_name", time.Hour)
	return "📝 *Dados para Contato*\n\nPara avançar, preciso do seu *nome completo*:", nil
}

// handlePlansName armazena o nome do usuário e coleta telefone, se necessário.
func (s *ChatbotService) handlePlansName(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	userData.Nome = strings.TrimSpace(message)

	if userData.Telefone == "" && len(userID) >= 10 && len(userID) <= 15 && isAllDigits(userID) {
		userData.Telefone = userID
	}
	s.setUserData(userID, userData)

	if userData.Telefone != "" {
		observacoes := fmt.Sprintf("Interesse em: %s | Plano atual: %s", userData.PlanoDesejado, userData.PlanoAtual)
		s.sheets.SavePlans(userData.Nome, userData.Situacao, userData.PlanoAtual, userData.PlanoDesejado, userData.Telefone, observacoes)
		s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)
		return fmt.Sprintf("🎉 *Dados Registrados com Sucesso!*\n\n*Nome*: %s\n*Situação*: %s\n*Plano Interesse*: %s\n*Telefone*: %s\n\n📞 *Próximos Passos*:\nNossa equipe comercial entrará em contato em até 24 horas para finalizar!\n\nDigite *MENU* para voltar ao menu principal.", userData.Nome, userData.Situacao, userData.PlanoDesejado, userData.Telefone), nil
	}

	s.redis.Set(ctx, "chat:"+userID, "plans_phone", time.Hour)
	return "📞 Agora informe um *telefone/WhatsApp* para contato (somente números ou formato (XX) XXXXX-XXXX):", nil
}

// isAllDigits retorna true se a string contém apenas dígitos.
func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// handlePlansPhone armazena o telefone informado e finaliza o fluxo de planos.
func (s *ChatbotService) handlePlansPhone(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)
	telefone := strings.TrimSpace(message)
	telefone = strings.ReplaceAll(telefone, " ", "")
	userData.Telefone = telefone
	s.setUserData(userID, userData)

	observacoes := fmt.Sprintf("Interesse em: %s | Plano atual: %s", userData.PlanoDesejado, userData.PlanoAtual)
	s.sheets.SavePlans(userData.Nome, userData.Situacao, userData.PlanoAtual, userData.PlanoDesejado, userData.Telefone, observacoes)

	s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)

	return fmt.Sprintf("🎉 *Dados Registrados com Sucesso!*\n\n*Nome*: %s\n*Situação*: %s\n*Plano Interesse*: %s\n*Telefone*: %s\n\n📞 *Próximos Passos*:\nNossa equipe comercial entrará em contato em até 24 horas para finalizar!\n\nDigite *MENU* para voltar ao menu principal.", userData.Nome, userData.Situacao, userData.PlanoDesejado, userData.Telefone), nil
}

// handleFreeAI processa perguntas livres para a IA.
func (s *ChatbotService) handleFreeAI(userID, message string) (string, error) {
	if strings.ToLower(strings.TrimSpace(message)) == "menu" {
		return s.showMainMenu(userID)
	}

	if s.ai != nil {
		response, err := s.ai.GenerateFreeResponse(message)
		if err == nil {
			return fmt.Sprintf("🤖 %s\n\n---\n*Digite *MENU* para voltar ao menu principal*", response), nil
		}
	}

	return "🤖 Desculpe, não consegui processar sua pergunta no momento. Tente novamente ou digite *MENU* para voltar ao menu principal.", nil
}

// handleSupportIA processa a resposta do usuário sobre a resolução do problema técnico.
func (s *ChatbotService) handleSupportIA(userID, message string) (string, error) {
	ctx := context.Background()
	response := strings.ToLower(strings.TrimSpace(message))
	userData := s.getUserData(userID)

	if response == "sim" {
		s.sheets.SaveSupport(userData.Nome, userData.Problema, userData.Descricao, "Resolvido pela IA")
		userData.AguardandoFeedback = false
		s.setUserData(userID, userData)
		s.redis.Set(ctx, "chat:"+userID, "support_feedback", time.Hour)
		return "🎉 *Ótimo! Problema resolvido!*\n\nPoderia nos dar um *feedback/opinião* sobre nosso atendimento? (Ex: Excelente, Bom, Regular...)", nil
	}

	if response == "não" || response == "nao" {
		userData.TentativasIA++
		if userData.TentativasIA >= 5 {
			s.sheets.SaveSupport(userData.Nome, userData.Problema, userData.Descricao, "Encaminhado para Técnico Humano")
			userData.AguardandoFeedback = false
			s.setUserData(userID, userData)
			s.redis.Set(ctx, "chat:"+userID, "support_feedback", time.Hour)
			return "🚨 *Encaminhamento para Técnico Especializado*\n\n📅 Prazo: 24-48 horas\n📞 Entraremos em contato.\n\nAntes de finalizar, poderia avaliar nosso atendimento? (Ex: Excelente, Bom, Regular...)", nil
		}
		s.setUserData(userID, userData)
		return s.continueTechnicalSupport(userID, userData.TentativasIA, userData.Problema)
	}

	return "Por favor, responda apenas *SIM* ou *NÃO* para que eu possa ajudá-lo melhor.", nil
}

// handleSupportFeedback armazena feedback e sugestões do usuário após o atendimento.
func (s *ChatbotService) handleSupportFeedback(userID, message string) (string, error) {
	ctx := context.Background()
	userData := s.getUserData(userID)

	if !userData.AguardandoFeedback {
		feedback := strings.TrimSpace(message)
		userData.Problema = feedback
		userData.AguardandoFeedback = true
		s.setUserData(userID, userData)

		return "💭 *Obrigado pela avaliação!*\n\nPara finalizar, tem alguma *sugestão* ou *comentário* para melhorarmos nosso atendimento?\n\n*(Digite sua sugestão ou 'NÃO' se não tiver)*", nil
	}

	sugestoes := strings.TrimSpace(message)
	if strings.ToLower(sugestoes) == "não" || strings.ToLower(sugestoes) == "nao" {
		sugestoes = ""
	}
	avaliacao := userData.Problema
	s.sheets.SaveFeedback(userData.Nome, userData.TipoAtendimento, avaliacao, sugestoes)

	s.redis.Set(ctx, "chat:"+userID, "menu", time.Hour)
	return "🙏 *Feedback registrado com sucesso!* \n\nSua opinião é muito importante para melhorarmos nossos serviços.\n\nDigite *MENU* para voltar ao menu principal.", nil
}

// getUserData lê o estado do usuário do Redis.
func (s *ChatbotService) getUserData(userID string) UserData {
	ctx := context.Background()
	data, err := s.redis.Get(ctx, "data:"+userID).Result()
	if err != nil {
		return UserData{}
	}

	var userData UserData
	json.Unmarshal([]byte(data), &userData)
	return userData
}

// setUserData grava o estado do usuário no Redis.
func (s *ChatbotService) setUserData(userID string, userData UserData) {
	ctx := context.Background()
	data, _ := json.Marshal(userData)
	s.redis.Set(ctx, "data:"+userID, data, time.Hour)
}

//Copyright 2025 Kauan Botura
