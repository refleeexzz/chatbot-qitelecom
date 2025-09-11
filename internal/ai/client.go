package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Client struct {
	model *genai.GenerativeModel
}

// Cria um novo cliente da IA Gemini
func NewClient() (*Client, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY não configurada")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente Gemini: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")

	return &Client{model: model}, nil
}

// Gera resposta da IA para problemas técnicos
func (c *Client) GenerateResponse(problema string) (string, error) {
	if c.model == nil {
		return generateTechFallback(problema), nil
	}

	ctx := context.Background()
	prompt := fmt.Sprintf(`Como assistente técnico especializado, resolva este problema de forma clara e prática:

Problema: %s

Forneça uma solução objetiva em até 200 palavras, incluindo:
- Diagnóstico do problema
- Passos para resolver
- Dicas de prevenção

Seja direto e útil, lembrando que você pode estar lidando com pessoas leigas no assunto.`, problema)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("Erro na IA Gemini (técnico): %v", err)
		return generateTechFallback(problema), nil
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}

	return generateTechFallback(problema), nil
}

// Gera resposta livre da IA
func (c *Client) GenerateFreeResponse(pergunta string) (string, error) {
	if c.model == nil {
		return generateFreeFallback(), nil
	}

	ctx := context.Background()
	prompt := fmt.Sprintf(`Responda de forma útil e amigável em português:

Pergunta: %s

Seja informativo, claro e conciso (máximo 250 palavras).`, pergunta)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("Erro na IA Gemini (livre): %v", err)
		return generateFreeFallback(), nil
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}

	return generateFreeFallback(), nil
}

// Fallbacks caso a IA não esteja disponível
func generateTechFallback(problema string) string {
	problema = strings.ToLower(problema)

	switch {
	case strings.Contains(problema, "internet") || strings.Contains(problema, "wifi"):
		return "🌐 **Problema de Internet/WiFi**\n\n" +
			"**Soluções rápidas:**\n" +
			"1. Reinicie o roteador (desligue 30s e ligue)\n" +
			"2. Verifique se outros dispositivos conectam\n" +
			"3. Teste conexão com cabo ethernet\n" +
			"4. Entre em contato com sua operadora se persistir\n\n" +
			"**Prevenção:** Mantenha firmware do roteador atualizado."

	case strings.Contains(problema, "lento") || strings.Contains(problema, "travando"):
		return "🐌 **Sistema Lento/Travando**\n\n" +
			"**Soluções:**\n" +
			"1. Feche programas desnecessários\n" +
			"2. Reinicie o computador\n" +
			"3. Verifique espaço em disco (mín. 15% livre)\n" +
			"4. Execute antivírus\n\n" +
			"**Prevenção:** Limpeza semanal e evite muitos programas simultâneos."

	case strings.Contains(problema, "senha") || strings.Contains(problema, "login"):
		return "🔐 **Problema de Login/Senha**\n\n" +
			"**Soluções:**\n" +
			"1. Use 'Esqueci minha senha' no sistema\n" +
			"2. Verifique se Caps Lock está desligado\n" +
			"3. Tente navegador em modo privado\n" +
			"4. Limpe cookies do navegador\n\n" +
			"**Prevenção:** Use gerenciador de senhas seguro."

	default:
		return "🛠️ **Suporte Técnico**\n\n" +
			"Recebemos seu problema e nossa equipe técnica irá analisar.\n\n" +
			"**Soluções gerais:**\n" +
			"1. Reinicie o dispositivo\n" +
			"2. Verifique conexão com internet\n" +
			"3. Atualize o navegador/aplicativo\n\n" +
			"**Em breve entraremos em contato com uma solução específica!**"
	}
}

func generateFreeFallback() string {
	return "🤖 **Assistente IA Temporariamente Indisponível**\n\n" +
		"Desculpe, nosso assistente de IA está temporariamente fora do ar para manutenção.\n\n" +
		"**Alternativas:**\n" +
		"• Para suporte técnico: Use a opção 'Suporte Técnico' no menu\n" +
		"• Para planos: Use a opção 'Planos' no menu\n" +
		"• Para fatura: Use a opção 'Fatura' no menu\n\n" +
		"Agradecemos sua compreensão! 🙏"
}
