# QI TELECOM - Sistema de Atendimento Inteligente

Este projeto é um sistema completo de atendimento para provedores de internet, com chatbot integrado, interface web responsiva e integração com Google Sheets.

## Funcionalidades
- Chatbot para atendimento automatizado (suporte técnico, planos, financeiro e dúvidas gerais)
- Interface web moderna e responsiva
- Coleta de feedback do cliente
- Integração com Google Sheets para registro de atendimentos, planos e feedbacks
- Gerenciamento de sessões e histórico de atendimento
- Suporte a múltiplos usuários simultâneos
- **Sistema de segurança robusto** com validação de entrada, rate limiting e headers de segurança

## Tecnologias Utilizadas
- Go (backend)
- HTML, CSS, JavaScript (frontend)
- Google Sheets API
- Redis (gerenciamento de sessão)
- Docker (opcional)

## Recursos de Segurança

### 🛡️ Segurança Implementada
- **Validação e sanitização de entrada** - Proteção contra XSS e injection
- **Rate limiting** - Proteção contra ataques DDoS
- **CORS configurável** - Controle de origem para requisições
- **Headers de segurança** - CSP, XSS Protection, e outros
- **Timeouts configuráveis** - Proteção contra ataques de timeout
- **Limites de tamanho de requisição** - Proteção contra overflow
- **Gerenciamento seguro de secrets** - Variáveis de ambiente
- **Logs seguros** - Sanitização de dados sensíveis

### 🔧 Configuração de Segurança
As configurações de segurança são controladas por variáveis de ambiente:

```bash
# Copie o arquivo de exemplo e configure
cp .env.example .env
# Edite .env com suas configurações
```

**Variáveis importantes:**
- `GOOGLE_SHEETS_ID` - ID da planilha Google (obrigatório)
- `ALLOWED_ORIGINS` - Origens permitidas para CORS
- `RATE_LIMIT_RPM` - Limite de requisições por minuto
- `FORCE_HTTPS` - Forçar uso de HTTPS
- `MAX_REQUEST_SIZE_BYTES` - Tamanho máximo de requisição

## Como usar

### Instalação Rápida
1. Clone o repositório
```bash
git clone https://github.com/refleeexzz/chatbot-qitelecom.git
cd chatbot-qitelecom
```

2. Configure as variáveis de ambiente
```bash
cp .env.example .env
# Edite .env com suas configurações
```

3. Configure as credenciais do Google Sheets (`credentials.json`)

4. Execute com Docker (recomendado)
```bash
docker-compose up -d
```

OU execute localmente:
```bash
go run main.go
```

5. Acesse a interface web em `http://localhost:8081`

### Configuração de Produção

Para ambiente de produção, recomendamos:

1. **HTTPS obrigatório:**
```bash
FORCE_HTTPS=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

2. **CORS restritivo:**
```bash
ALLOWED_ORIGINS=https://yourdomain.com
```

3. **Rate limiting agressivo:**
```bash
RATE_LIMIT_RPM=30
RATE_LIMIT_BURST=5
```

4. **Redis com senha:**
```bash
REDIS_PASSWORD=seu_password_forte
```

## Observações de Segurança

⚠️ **IMPORTANTE:**
- Não suba arquivos sensíveis como `credentials.json` ou `.env` em repositórios públicos
- Use HTTPS em produção
- Configure CORS adequadamente para seu domínio
- Use senhas fortes para Redis
- Monitore logs de segurança
- Mantenha dependências atualizadas

## Monitoramento

O sistema inclui:
- Endpoint de saúde em `/health`
- Logs estruturados com níveis de segurança
- Métricas de rate limiting nos logs

---
Desenvolvido por Kauan Botura (dev) e Ronan Moreira (liderança do projeto)
