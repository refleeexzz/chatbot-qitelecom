# 📝 INFORMAÇÕES IMPORTANTES - QIBOT DEPLOY

## 🔐 Credenciais e Tokens

### Admin do Sistema
```
Usuário: QItelecom
Senha: @Redeqi.com.br
```

### Webhook WhatsApp
```
Token de Verificação: euamojesuscristodenazare
```

---

## 📋 Arquivos Criados para Deploy

### No Windows (seu PC):
1. **`enviar-para-vps.ps1`** - Script para enviar projeto para VPS
2. **`DEPLOY_HOSTINGER.md`** - Guia completo de deploy
3. **`DEPLOY_RAPIDO.txt`** - Guia rápido de referência

### Para a VPS (Linux):
1. **`install-docker-vps.sh`** - Instala Docker e Docker Compose
2. **`deploy-vps.sh`** - Faz deploy do projeto

---

## 🚀 Processo de Deploy Simplificado

### ETAPA 1: No seu PC (Windows)
```powershell
# Execute no PowerShell
.\enviar-para-vps.ps1
```

Isso vai:
- Compactar o projeto
- Enviar para a VPS via SCP
- Descompactar no diretório `/opt/qibot`

### ETAPA 2: Na VPS (via SSH)
```bash
# Conectar
ssh root@SEU_IP_VPS

# Ir para o diretório
cd /opt/qibot

# Instalar Docker (apenas primeira vez)
chmod +x install-docker-vps.sh
./install-docker-vps.sh

# Fazer deploy
chmod +x deploy-vps.sh
./deploy-vps.sh
```

### ETAPA 3: Configurar Webhook
1. Acesse: https://developers.facebook.com/apps
2. Configure webhook com a URL da sua VPS

---

## 🔗 URLs Importantes

### Desenvolvimento Local
```
http://localhost:8081
http://192.168.0.13:8081  (rede local)
```

### Produção (VPS)
```
http://SEU_IP_VPS:8081
https://seu-dominio.com  (com Nginx + SSL)
```

### Endpoints API
```
/webhook                  - Webhook WhatsApp
/chatbot                  - API do Chatbot
/api/bulk-message         - Disparo em lote
/api/bulk-template        - Templates em lote
/api/admin/contacts       - Gerenciar contatos
```

---

## 📊 Portas Utilizadas

| Serviço | Porta | Descrição |
|---------|-------|-----------|
| QIBOT   | 8081  | Aplicação principal |
| Redis   | 6379  | Cache (apenas interna) |
| Nginx   | 80    | HTTP (se configurado) |
| Nginx   | 443   | HTTPS (se configurado) |

---

## 🗂️ Estrutura de Diretórios na VPS

```
/opt/qibot/
├── docker-compose.yml
├── Dockerfile
├── .env
├── credentials.json
├── main.go
├── go.mod
├── go.sum
├── *.html
├── internal/
│   ├── ai/
│   ├── handlers/
│   ├── security/
│   ├── services/
│   ├── sheets/
│   └── whatsapp/
├── install-docker-vps.sh
└── deploy-vps.sh
```

---

## 🔄 Comandos de Manutenção

### Ver status
```bash
docker-compose ps
docker-compose logs -f qibot
```

### Reiniciar
```bash
docker-compose restart
```

### Atualizar (se usar Git)
```bash
cd /opt/qibot
git pull
docker-compose down
docker-compose up -d --build
```

### Backup
```bash
# Banco de dados
docker cp qibot:/app/leads.db ~/backup-leads-$(date +%Y%m%d).db

# Logs
docker-compose logs qibot > ~/logs-$(date +%Y%m%d).txt
```

---

## ⚠️ Pontos de Atenção

### Segurança
- ✅ Troque a senha do admin após primeiro acesso
- ✅ Use HTTPS em produção (Nginx + Certbot)
- ✅ Configure firewall (UFW)
- ✅ Mantenha tokens do .env seguros
- ✅ Não commite .env no Git

### Performance
- ✅ VPS mínima: 1GB RAM, 1 vCPU
- ✅ Recomendado: 2GB RAM, 2 vCPU
- ✅ Monitore uso: `docker stats`

### Backup
- ✅ Faça backup diário do banco de dados
- ✅ Configure cron para backup automático
- ✅ Guarde credentials.json em local seguro

---

## 📞 Suporte

### Logs para Debug
```bash
# Ver todos os logs
docker-compose logs

# Últimas 100 linhas
docker-compose logs --tail=100 qibot

# Seguir logs em tempo real
docker-compose logs -f qibot

# Buscar erros
docker-compose logs qibot | grep -i error
```

### Comandos de Diagnóstico
```bash
# Testar conectividade
curl http://localhost:8081

# Ver processos Docker
docker ps -a

# Ver uso de recursos
docker stats

# Testar Redis
docker exec redis redis-cli ping
```

---

## 🎯 Checklist Pós-Deploy

- [ ] Containers rodando (`docker-compose ps`)
- [ ] Sistema acessível via IP
- [ ] Admin login funcionando
- [ ] Webhook configurado na Meta
- [ ] Teste de envio de mensagem OK
- [ ] Firewall configurado
- [ ] Backup agendado
- [ ] (Opcional) Domínio configurado
- [ ] (Opcional) HTTPS configurado

---

**Desenvolvido por Kauan Botura**  
**Projeto: QIBOT - QITelecom** 📱🚀
