package main

import (
	"context"
	"database/sql"
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
)

func main() {
	// 📋 Configurar logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerologlog.Logger = zerologlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 🔑 Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		zerologlog.Warn().Err(err).Msg("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// 🔒 Carregar configuração de segurança
	securityConfig, err := security.LoadSecurityConfig()
	if err != nil {
		zerologlog.Fatal().Err(err).Msg("Erro ao carregar configuração de segurança")
	}
	zerologlog.Info().Msg("Configuração de segurança carregada")

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
	redisClient := setupRedis(securityConfig)
	defer redisClient.Close()

	// ⚙️ Configurar serviços
	chatbotService := services.NewChatbotService(redisClient, db, sheetsClient, aiClient)

	// 🚪 Configurar handlers
	chatbotHandler := handlers.NewChatbotHandler(chatbotService, securityConfig)

	// 🛡️ Configurar middleware de segurança
	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	// 🌐 Configurar rotas
	setupRoutes(chatbotHandler, securityMiddleware)

	// 🚀 Iniciar servidor
	startServer(securityConfig)
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

func setupRedis(securityConfig *security.SecurityConfig) *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     securityConfig.RedisPassword,
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

func setupRoutes(chatbotHandler *handlers.ChatbotHandler, securityMiddleware *security.SecurityMiddleware) {
	// Aplicar middleware de segurança
	mux := http.NewServeMux()
	
	// Configurar rotas
	mux.HandleFunc("/chatbot", chatbotHandler.HandleChatbot)
	mux.HandleFunc("/health", chatbotHandler.HandleHealth)
	mux.HandleFunc("/", chatbotHandler.HandleStatic)
	
	// Aplicar middleware em ordem (do último para o primeiro na cadeia)
	handler := securityMiddleware.SecurityHeaders(mux)
	handler = securityMiddleware.CORS(handler)
	handler = securityMiddleware.RateLimit(handler)
	handler = securityMiddleware.RequestSizeLimit(handler)
	handler = securityMiddleware.HTTPSRedirect(handler)
	
	http.Handle("/", handler)
}

func startServer(securityConfig *security.SecurityConfig) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		ReadTimeout:  securityConfig.ReadTimeout,
		WriteTimeout: securityConfig.WriteTimeout,
		IdleTimeout:  securityConfig.IdleTimeout,
	}

	// Canal para capturar sinais do sistema
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Iniciar servidor em goroutine
	go func() {
		zerologlog.Info().Msgf("🚀 QIBOT rodando em %s", server.Addr)
		
		// Tentar iniciar com HTTPS se certificados estiverem disponíveis
		if securityConfig.TLSCertFile != "" && securityConfig.TLSKeyFile != "" {
			zerologlog.Info().Msg("Iniciando servidor HTTPS")
			if err := server.ListenAndServeTLS(securityConfig.TLSCertFile, securityConfig.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				zerologlog.Fatal().Err(err).Msg("Erro ao iniciar servidor HTTPS")
			}
		} else {
			zerologlog.Warn().Msg("Iniciando servidor HTTP (desenvolvimento apenas)")
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zerologlog.Fatal().Err(err).Msg("Erro ao iniciar servidor")
			}
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
