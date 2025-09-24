package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"

	"leadprojectarrumado/internal/ai"
	"leadprojectarrumado/internal/handlers"
	"leadprojectarrumado/internal/security"
	"leadprojectarrumado/internal/services"
	"leadprojectarrumado/internal/sheets"

	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)
func main() {
	// 📋 Configurar logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerologlog.Logger = zerologlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 🔑 Carregar variáveis de ambiente
	if err := godotenv.Load("C:/Users/Uaurio/Documents/LEADPROJECT/.env"); err != nil {
		zerologlog.Warn().Err(err).Msg("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// ▶️ Iniciar Datadog tracer (APM)
	agentHost := os.Getenv("DD_AGENT_HOST")
	if agentHost == "" {
		agentHost = "localhost"
	}
	agentPort := os.Getenv("DD_TRACE_AGENT_PORT")
	if agentPort == "" {
		agentPort = "8126"
	}
	tracer.Start(
		tracer.WithAgentAddr(fmt.Sprintf("%s:%s", agentHost, agentPort)),
		tracer.WithServiceName("qibot-chatbot"),
		tracer.WithEnv(os.Getenv("DD_ENV")),
		tracer.WithRuntimeMetrics(),
	)
	defer tracer.Stop()

	// 🗄️ Configurar banco de dados SQLite
	db, err := setupDatabase()
	if err != nil {
		zerologlog.Fatal().Err(err).Msg("Erro ao configurar banco de dados")
	}
	defer db.Close()

	// 📊 Configurar cliente Google Sheets
	sheetsClient, err := sheets.NewClient()
	if err != nil {
		zerologlog.Fatal().Err(err).Msg("Erro ao configurar Google Sheets")
	}
	
	// 🤖 Configurar cliente IA Gemini
	aiClient, err := ai.NewClient()
	if err != nil {
		zerologlog.Warn().Err(err).Msg("IA Gemini não disponível")
	} else {
		zerologlog.Info().Msg("Gemini habilitado.")
	}

	// 🔴 Configurar Redis
	redisClient := setupRedis()
	defer redisClient.Close()

	// ⚙️ Configurar serviços
	chatbotService := services.NewChatbotService(redisClient, db, sheetsClient, aiClient)

	// 🚪 Configurar handlers
	chatbotHandler := handlers.NewChatbotHandler(chatbotService)

	// 🌐 Configurar rotas
	setupRoutes(chatbotHandler)

	// 🚀 Iniciar servidor
	startServer()
}

func setupDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "leads.db")
	if err != nil {
		return nil, err
	}

	// Criar tabela se não existir
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS leads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nome TEXT NOT NULL,
			telefone TEXT,
			email TEXT,
			tipo TEXT DEFAULT 'Lead',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func setupRedis() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Testar conexão (não fatal se falhar)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		zerologlog.Warn().Err(err).Msg("Redis não disponível - algumas funcionalidades podem não funcionar")
		return client // Retorna mesmo sem conexão para desenvolvimento
	}

	zerologlog.Info().Msg("Redis conectado com sucesso")
	return client
}

func setupRoutes(chatbotHandler *handlers.ChatbotHandler) {
	cfg := security.LoadConfig()
	rl := security.NewGlobalRateLimiter(cfg.RatePerMinute)

	// Wrappear handlers com Datadog tracing
	tracedChatbot := httptrace.WrapHandler(http.HandlerFunc(chatbotHandler.HandleChatbot), "qibot-chatbot", "/chatbot")
	tracedHealth := httptrace.WrapHandler(http.HandlerFunc(chatbotHandler.HandleHealth), "qibot-chatbot", "/health")

	http.Handle("/chatbot", security.WrapHandler(tracedChatbot, cfg, rl))
	http.Handle("/health", security.WrapHandler(tracedHealth, cfg, rl))
	http.HandleFunc("/", chatbotHandler.HandleStatic) // página estática sem wrappers

	// WhatsApp webhook handler
	whatsappHandler := handlers.NewWhatsAppWebhookHandler(chatbotHandler.Service())
	http.HandleFunc("/webhook/whatsapp", whatsappHandler.HandleWhatsAppWebhook)
}

func startServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Canal para capturar sinais do sistema
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Iniciar servidor em goroutine
	go func() {
		zerologlog.Info().Msgf("🚀 QIBOT rodando em http://localhost:8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zerologlog.Fatal().Err(err).Msg("Erro ao iniciar servidor")
		}
	}()

	// Aguardar sinal de parada
	<-stop
	zerologlog.Info().Msg("🛑 Parando servidor...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		zerologlog.Error().Err(err).Msg("Erro ao parar servidor")
	} else {
		zerologlog.Info().Msg("✅ Servidor parado com sucesso")
	}
}

//Copyright 2025 Kauan Botura
