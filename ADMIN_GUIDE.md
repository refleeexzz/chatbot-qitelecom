# 🔐 Sistema de Administração QIBOT

## 📋 Visão Geral

O QIBOT agora possui um painel de administração completo e unificado que permite:
- **Autenticação segura** com login e logout
- **Gerenciamento de contatos** (visualizar, adicionar, remover, buscar)
- **Disparos em lote** de mensagens WhatsApp
- **Estatísticas** em tempo real

## 🚀 Acesso

### URL Principal
```
http://localhost:8081/admin
```

### Credenciais Padrão
- **Usuário:** `QItelecom`
- **Senha:** `@Redeqi.com.br`

> ⚠️ **IMPORTANTE:** Estas credenciais estão configuradas no arquivo `.env`. Altere-as conforme necessário!

## 🔑 Configuração de Credenciais

### Variáveis de Ambiente

Configure as credenciais de administrador através de variáveis de ambiente:

```bash
ADMIN_USERNAME=seu_usuario_aqui
ADMIN_PASSWORD=sua_senha_segura_aqui
```

### No arquivo `.env`
```env
ADMIN_USERNAME=QItelecom
ADMIN_PASSWORD=@Redeqi.com.br
```

### No Docker Compose
```yaml
services:
  qibot:
    environment:
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=senha_producao_123
```

## 📱 Funcionalidades

### 1. **Dashboard Principal**
- Estatísticas em tempo real
- Total de contatos
- Total de leads e clientes
- Acesso rápido às funcionalidades

### 2. **Gerenciar Contatos**

#### ➕ Adicionar Contato
- **Nome** (obrigatório)
- **Telefone** (obrigatório, formato: 5511999999999)
- **Email** (opcional)
- **Tipo** (Lead ou Cliente)

#### 🔍 Buscar Contatos
- Busca em tempo real
- Filtra por nome, telefone ou email
- Resultados instantâneos

#### 📋 Listar Contatos
- Tabela organizada
- Badges coloridos por tipo
- Data de cadastro
- Ações rápidas

#### 🗑️ Remover Contato
- Confirmação antes de remover
- Remoção permanente do banco de dados
- Feedback visual de sucesso

### 3. **Disparos em Lote**

#### 📱 Enviar Mensagens
- Mensagem personalizada (até 1000 caracteres)
- Contador de caracteres em tempo real
- Seleção de destinatários:
  - **Todos os contatos**
  - **Apenas Leads**
  - **Apenas Clientes**

#### ⏱️ Rate Limiting
- Intervalo de 1,5 segundos entre mensagens
- ~40 mensagens por minuto (seguro para WhatsApp)
- Previne bloqueios da API

## 🔒 Segurança

### Autenticação
- Cookie HttpOnly (`admin_token`)
- Sessão válida por 24 horas
- Logout limpa o cookie imediatamente

### Proteção de Rotas
Todas as rotas de administração são protegidas:
- `/api/admin/contacts` - ✅ Requer autenticação
- `/api/admin/contacts/add` - ✅ Requer autenticação
- `/api/admin/contacts/delete` - ✅ Requer autenticação

### Comparação Timing-Safe
- Validação de credenciais com `subtle.ConstantTimeCompare`
- Previne ataques de timing

## 🌐 Endpoints da API

### Autenticação

#### `POST /api/admin/login`
**Request:**
```json
{
  "username": "QItelecom",
  "password": "@Redeqi.com.br"
}
```

**Response (Sucesso):**
```json
{
  "success": true,
  "token": "admin_QItelecom_20250126123045",
  "message": "Login bem-sucedido"
}
```

**Response (Erro):**
```json
{
  "success": false,
  "message": "Usuário ou senha incorretos"
}
```

#### `POST /api/admin/logout`
**Response:**
```json
{
  "success": true
}
```

---

### Gerenciamento de Contatos

#### `GET /api/admin/contacts`
🔒 **Requer autenticação**

**Response:**
```json
{
  "total_contacts": 15,
  "database_contacts": 15,
  "redis_keys": 0,
  "contacts": [
    {
      "name": "João Silva",
      "phone": "5511999999999",
      "email": "joao@example.com",
      "type": "Lead",
      "created_at": "26/01/2025 12:30",
      "source": "database"
    }
  ],
  "message": "Contatos carregados com sucesso"
}
```

#### `POST /api/admin/contacts/add`
🔒 **Requer autenticação**

**Request:**
```json
{
  "nome": "Maria Santos",
  "telefone": "5511988888888",
  "email": "maria@example.com",
  "tipo": "Cliente"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Contato adicionado com sucesso"
}
```

#### `DELETE /api/admin/contacts/delete`
🔒 **Requer autenticação**

**Request:**
```json
{
  "telefone": "5511999999999"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Contato removido com sucesso"
}
```

---

### Disparos em Lote

#### `POST /api/bulk-message`
**Request:**
```json
{
  "message": "Olá! Temos uma promoção especial para você!",
  "send_to_all": false,
  "filter_by_type": "Lead"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "3 mensagens enviadas com sucesso",
  "sent_count": 3
}
```

## 🎨 Interface do Usuário

### Design Responsivo
- Layout adaptável para desktop, tablet e mobile
- Cards informativos com estatísticas
- Sistema de abas para organização

### Feedback Visual
- Alertas de sucesso (verde)
- Alertas de erro (vermelho)
- Estados de loading
- Confirmações de ações críticas

### Busca em Tempo Real
- Filtragem instantânea
- Sem necessidade de recarregar página
- Resultados enquanto digita

## 🚀 Deploy

### 1. Build da Imagem Docker
```bash
docker build -t qibot:latest .
```

### 2. Executar com Docker Compose
```bash
docker compose up -d
```

### 3. Configurar Variáveis de Ambiente
Edite o arquivo `.env`:
```env
# Admin Credentials
ADMIN_USERNAME=QItelecom
ADMIN_PASSWORD=@Redeqi.com.br

# Other configs
PORT=8081
GOOGLE_API_KEY=sua_chave_api
WHATSAPP_TOKEN=seu_token_whatsapp
WHATSAPP_PHONE_ID=seu_phone_id
```

### 4. Rebuild após mudanças
```bash
docker compose down
docker compose up --build -d
```

## 📝 Logs

### Visualizar logs do container
```bash
docker compose logs -f qibot
```

### Eventos importantes logados:
- ✅ Login de administrador bem-sucedido
- ⚠️ Tentativas de login falhadas
- ℹ️ Contato adicionado
- ℹ️ Contato removido
- 📱 Mensagens em lote enviadas

## 🔧 Troubleshooting

### Problema: "Não autenticado"
**Solução:** Faça login em `/admin` com as credenciais configuradas

### Problema: "Contato não encontrado"
**Solução:** Verifique se o número de telefone está correto (formato: 5511999999999)

### Problema: "Erro ao enviar mensagens"
**Solução:** 
- Verifique se `WHATSAPP_TOKEN` e `WHATSAPP_PHONE_ID` estão configurados
- Confirme que a API do WhatsApp está ativa

### Problema: Cookie não persiste
**Solução:** 
- Verifique se o navegador aceita cookies
- Em produção, configure `Secure: true` no cookie (requer HTTPS)

## 🔐 Boas Práticas de Segurança

1. **Altere as credenciais padrão imediatamente**
2. **Use senhas fortes** (mínimo 12 caracteres, letras, números, símbolos)
3. **Não compartilhe credenciais** em código ou repositórios
4. **Use HTTPS em produção**
5. **Configure logs para monitorar acessos**
6. **Considere implementar 2FA** (autenticação em dois fatores)
7. **Implemente JWT** em vez de tokens simples para produção
8. **Configure rate limiting** para endpoints de admin

## 📊 Banco de Dados

### Estrutura da Tabela `leads`
```sql
CREATE TABLE IF NOT EXISTS leads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    telefone TEXT,
    email TEXT,
    tipo TEXT DEFAULT 'Lead',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### Backup do Banco
```bash
# Backup
docker compose exec qibot sqlite3 /app/leads.db .dump > backup.sql

# Restore
cat backup.sql | docker compose exec -T qibot sqlite3 /app/leads.db
```

## 🌟 Próximas Melhorias

- [ ] Implementar JWT para tokens mais seguros
- [ ] Adicionar edição de contatos
- [ ] Exportar contatos para CSV/Excel
- [ ] Histórico de disparos
- [ ] Agendamento de mensagens
- [ ] Templates de mensagens
- [ ] Autenticação em dois fatores (2FA)
- [ ] Múltiplos usuários administradores
- [ ] Logs de auditoria detalhados

## 📞 Suporte

Para problemas ou dúvidas sobre o sistema de administração, verifique:
1. Logs do container: `docker compose logs qibot`
2. Documentação do WhatsApp Business API
3. Issues no repositório do projeto

---

**Desenvolvido com ❤️ para QIBOT**
