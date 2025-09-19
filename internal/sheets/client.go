package sheets

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	SpreadsheetID = "1iUElxVPVqqBqAUq-9rXRjhSTAo94Quqt9-0KIUgNgOA"
)

type Client struct {
	service *sheets.Service
	ctx     context.Context
}

// Cria um novo cliente do Google Sheets
func NewClient() (*Client, error) {
	ctx := context.Background()

	fmt.Println("\nConectando ao Google Sheets...")

	// Autentica usando o arquivo credentials.json
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Não foi possível ler o arquivo de credenciais (credentials.json): %v", err)
	}

	config, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Não foi possível processar o arquivo de credenciais: %v", err)
	}
	sheetsClient := config.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(sheetsClient))
	if err != nil {
		log.Fatalf("Não foi possível criar o serviço do Sheets: %v", err)
	}

	client := &Client{
		service: srv,
		ctx:     ctx,
	}

	// Formatar as planilhas
	client.formatSheets()

	log.Println("Google Sheets conectado e formatado com sucesso!")

	return client, nil
}

// Cria cabeçalhos das três páginas
func (c *Client) formatSheets() {
	c.formatFeedbackSheet()
	c.formatSupportSheet()
	c.formatPlansSheet()
}

// Cabeçalhos da Página1 (Feedback)
func (c *Client) formatFeedbackSheet() {
	// Não limpar a planilha para não apagar dados existentes

	// Cabeçalhos da Página1 (Feedback)
	headers := [][]interface{}{
		{"DATA/HORA", "NOME COMPLETO", "TIPO DE ATENDIMENTO", "AVALIAÇÃO", "SUGESTÕES/OBSERVAÇÕES"},
	}

	valueRange := &sheets.ValueRange{
		Values: headers,
	}

	// Inserir cabeçalhos
	c.service.Spreadsheets.Values.Update(SpreadsheetID, "Página1!A1:E1", valueRange).
		ValueInputOption("RAW").
		Do()

	log.Println("Página1 (Feedback) formatada com cabeçalhos")
}

// Cabeçalhos da Página2 (Suporte)
func (c *Client) formatSupportSheet() {
	// Não limpar a planilha para não apagar dados existentes

	// Cabeçalhos da Página2 (Suporte)
	headers := [][]interface{}{
		{"DATA/HORA", "NOME COMPLETO", "PROBLEMA RELATADO", "DESCRIÇÃO DETALHADA", "STATUS RESOLUÇÃO"},
	}

	valueRange := &sheets.ValueRange{
		Values: headers,
	}

	// Inserir cabeçalhos
	c.service.Spreadsheets.Values.Update(SpreadsheetID, "Página2!A1:E1", valueRange).
		ValueInputOption("RAW").
		Do()

	log.Println("Página2 (Suporte Técnico) formatada com cabeçalhos")
}

// Cabeçalhos da Página3 (Planos)
func (c *Client) formatPlansSheet() {
	// Não limpar a planilha para não apagar dados existentes

	// Cabeçalhos da Página3 (Planos)
	headers := [][]interface{}{
		{"DATA/HORA", "NOME COMPLETO", "SITUAÇÃO CLIENTE", "PLANO ATUAL", "PLANO DESEJADO", "TELEFONE", "OBSERVAÇÕES"},
	}

	valueRange := &sheets.ValueRange{
		Values: headers,
	}

	// Inserir cabeçalhos
	c.service.Spreadsheets.Values.Update(SpreadsheetID, "Página3!A1:G1", valueRange).
		ValueInputOption("RAW").
		Do()

	log.Println("Página3 (Planos) formatada com cabeçalhos")
}

// Salva dados de suporte técnico na Página2
func (c *Client) SaveSupport(nome, problema, descricao, status string) error {
	logger := logrus.WithFields(logrus.Fields{
		"operation": "SaveSupport",
		"user":      nome,
		"timestamp": time.Now(),
	})
	logger.Info("Salvando dados de suporte")

	// Adiciona timestamp brasileiro
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	values := [][]interface{}{
		{timestamp, nome, problema, descricao, status},
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Append(SpreadsheetID, "Página2!A:E", valueRange).
		ValueInputOption("RAW").
		Do()

	if err != nil {
		log.Printf("Erro ao salvar suporte: %v", err)
		return err
	}

	log.Println("Suporte salvo com sucesso!")
	return nil
}

// Salva dados de consulta de planos na Página3
func (c *Client) SavePlans(nome, situacao, planoAtual, planoDesejado, telefone, observacoes string) error {
	log.Printf("Salvando planos: %s, %s, %s, %s, %s, %s", nome, situacao, planoAtual, planoDesejado, telefone, observacoes)

	// Adiciona timestamp brasileiro
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	values := [][]interface{}{
		{timestamp, nome, situacao, planoAtual, planoDesejado, telefone, observacoes},
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Append(SpreadsheetID, "Página3!A:G", valueRange).
		ValueInputOption("RAW").
		Do()

	if err != nil {
		log.Printf("Erro ao salvar planos: %v", err)
		return err
	}

	log.Println("Planos salvos com sucesso!")
	return nil
}

// Salva dados de feedback na Página1
func (c *Client) SaveFeedback(nome, tipoAtendimento, feedback, sugestoes string) error {
	log.Printf("Salvando feedback: %s, %s, %s, %s", nome, tipoAtendimento, feedback, sugestoes)

	// Adiciona timestamp brasileiro
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	values := [][]interface{}{
		{timestamp, nome, tipoAtendimento, feedback, sugestoes},
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Append(SpreadsheetID, "Página1!A:E", valueRange).
		ValueInputOption("RAW").
		Do()

	if err != nil {
		log.Printf("Erro ao salvar feedback: %v", err)
		return err
	}

	log.Println("Feedback salvo com sucesso!")
	return nil
}
