# 🚀 Deploy do QIBOT na VPS Hostinger

## 📋 Pré-requisitos na VPS

Antes de começar, sua VPS precisa ter:
- ✅ Docker instalado
- ✅ Docker Compose instalado
- ✅ Acesso SSH
- ✅ Porta 8081 liberada no firewall

---

## 🔧 Passo 1: Instalar Docker na VPS Hostinger

### Conectar via SSH:
```bash
ssh root@SEU_IP_VPS
```

### Instalar Docker (Ubuntu/Debian):
```bash
# Atualizar sistema
apt update && apt upgrade -y

# Instalar dependências
apt install -y apt-transport-https ca-certificates curl software-properties-common

# Adicionar repositório Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

# Instalar Docker
apt update
apt install -y docker-ce docker-ce-cli containerd.io

# Instalar Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Verificar instalação
docker --version
docker-compose --version
```

---

## 📦 Passo 2: Enviar Projeto para a VPS

### Opção A: Via SCP (do seu PC para VPS)

No **PowerShell do seu PC**:

```powershell
# Compactar projeto
Compress-Archive -Path C:\Users\Usuario\Documents\LEADPROJECT\* -DestinationPath C:\Users\Usuario\Documents\LEADPROJECT.zip

# Enviar para VPS
scp C:\Users\Usuario\Documents\LEADPROJECT.zip root@SEU_IP_VPS:/root/
```

Na **VPS** (via SSH):

```bash
# Descompactar
cd /root
unzip LEADPROJECT.zip -d /opt/qibot
cd /opt/qibot

# Dar permissões
chmod +x *.sh 2>/dev/null
```

### Opção B: Via Git (RECOMENDADO)

No **seu PC**:
```bash
# Já está no GitHub: chatbot-qitelecom
```

Na **VPS**:
```bash
# Clonar repositório
cd /opt
git clone https://github.com/refleeexzz/chatbot-qitelecom.git qibot
cd qibot
```

---

## 🔐 Passo 3: Configurar Variáveis de Ambiente

Na **VPS**, crie/edite o arquivo `.env`:

```bash
nano /opt/qibot/.env
```

Cole o conteúdo (use seus tokens reais):

```env
GOOGLE_API_KEY=AIzaSyD4WxOTKAUrCswxQvdlHNFmWIDRpp-Zagg
PORT=8081
REDIS_HOST=redis
REDIS_PORT=6379
BODY_LIMIT_BYTES=8192
RATE_LIMIT_PER_MINUTE=120

# Meta WhatsApp Business API
META_APP_ID=1876520162929628
META_APP_SECRET=7d20047b2253c6311a47317408e86358
WHATSAPP_VERIFY_TOKEN=euamojesuscristodenazare
WHATSAPP_PHONE_ID=780150178518900
WHATSAPP_TOKEN=EAAaqr2W5q9wBPihOTAS0Uoe7UHKIGJBsQvNDg1fyJdz8Yw3IoPVPi9AsLsc6iScQzrUAAbwrJoPAC3uiIAjDxEpFkOSaLqoDfLF9ugoVTCXEvzeMlZCi4sxsNNWJBIBRr1Sy84cbl284JmEvkkoEha0RjluqElbKxHIObZAArSOAHLrop7JpzZCMEGTlmatufbAKIoiMh0WYp4WE9AWODeN4X6anajbZAo5obsRrOLgzgsCZBFQlRxjnVq7PgZAAZDZD

# Gemini AI
GENERATIVE_MODEL=models/gemini-2.5-pro

# Admin
ADMIN_USERNAME=QItelecom
ADMIN_PASSWORD=@Redeqi.com.br

# Datadog (opcional - pode deixar)
DD_API_KEY=73826956f44d8ef4ebb832927e90aef4
DD_SITE=us5.datadoghq.com
DD_ENV=production
DD_SERVICE=qibot-chatbot
DD_VERSION=1.0.0
DD_AGENT_HOST=localhost
DD_TRACE_AGENT_PORT=8126
```

Salvar: `Ctrl+O` → `Enter` → `Ctrl+X`

---

## 🚀 Passo 4: Subir os Containers

Na **VPS**:

```bash
cd /opt/qibot

# Construir e iniciar containers
docker-compose up -d --build

# Ver logs
docker-compose logs -f
```

Para sair dos logs: `Ctrl+C`

---

## 🌐 Passo 5: Configurar Firewall e Domínio

### Liberar porta 8081:

```bash
# UFW (Ubuntu)
ufw allow 8081/tcp
ufw reload

# OU iptables
iptables -A INPUT -p tcp --dport 8081 -j ACCEPT
iptables-save
```

### Acessar o sistema:

```
http://SEU_IP_VPS:8081
```

---

## 🔒 Passo 6: Configurar HTTPS com Nginx (OPCIONAL mas RECOMENDADO)

### Instalar Nginx:

```bash
apt install -y nginx certbot python3-certbot-nginx
```

### Configurar Nginx como Proxy Reverso:

```bash
nano /etc/nginx/sites-available/qibot
```

Cole:

```nginx
server {
    listen 80;
    server_name seu-dominio.com;

    location / {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Ativar configuração:

```bash
ln -s /etc/nginx/sites-available/qibot /etc/nginx/sites-enabled/
nginx -t
systemctl restart nginx
```

### Configurar SSL (HTTPS):

```bash
certbot --nginx -d seu-dominio.com
```

Agora acesse via HTTPS:
```
https://seu-dominio.com
```

---

## 🔄 Comandos Úteis na VPS

### Ver status dos containers:
```bash
docker-compose ps
```

### Ver logs em tempo real:
```bash
docker-compose logs -f qibot
```

### Reiniciar containers:
```bash
docker-compose restart
```

### Parar containers:
```bash
docker-compose down
```

### Atualizar código (se usar Git):
```bash
cd /opt/qibot
git pull
docker-compose down
docker-compose up -d --build
```

### Ver uso de recursos:
```bash
docker stats
```

---

## 🎯 Configurar Webhook do WhatsApp

Após deploy, configure o webhook na Meta:

1. Acesse: https://developers.facebook.com/apps
2. Vá em **WhatsApp** → **Configuração**
3. Em **Webhook**, clique em **Editar**
4. Configure:
   - **URL de retorno de chamada**: `https://seu-dominio.com/webhook`
   - **Token de verificação**: `euamojesuscristodenazare`
5. Marque: `messages` e `messaging_postbacks`
6. Clique em **Verificar e salvar**

---

## 🛠️ Manutenção e Monitoramento

### Backup do banco de dados:

```bash
# Criar backup
docker cp qibot:/app/leads.db /root/backup/leads-$(date +%Y%m%d).db

# Agendar backup diário (crontab)
crontab -e

# Adicionar linha:
0 3 * * * docker cp qibot:/app/leads.db /root/backup/leads-$(date +\%Y\%m\%d).db
```

### Monitorar logs:

```bash
# Logs das últimas 100 linhas
docker-compose logs --tail=100 qibot

# Buscar erros
docker-compose logs qibot | grep -i error
```

### Atualizar sistema:

```bash
cd /opt/qibot
git pull origin main
docker-compose down
docker-compose up -d --build
```

---

## 🆘 Solução de Problemas

### Container não inicia:
```bash
# Ver logs detalhados
docker-compose logs qibot

# Verificar se as portas estão em uso
netstat -tulpn | grep 8081
```

### Erro de permissão:
```bash
# Ajustar permissões
chmod -R 755 /opt/qibot
chown -R root:root /opt/qibot
```

### Limpar tudo e começar de novo:
```bash
docker-compose down -v
docker system prune -a
docker-compose up -d --build
```

---

## 📊 Checklist de Deploy

- [ ] VPS criada na Hostinger
- [ ] Docker e Docker Compose instalados
- [ ] Projeto enviado para VPS
- [ ] Arquivo `.env` configurado
- [ ] Containers rodando (`docker-compose ps`)
- [ ] Porta 8081 liberada no firewall
- [ ] Sistema acessível via IP
- [ ] (Opcional) Nginx configurado
- [ ] (Opcional) SSL/HTTPS configurado
- [ ] Webhook do WhatsApp configurado
- [ ] Backup agendado

---

## 📞 URLs Após Deploy

Substitua `SEU_IP_VPS` ou `seu-dominio.com`:

- 🏠 **Página Inicial**: `http://SEU_IP_VPS:8081/`
- 🔐 **Login Admin**: `http://SEU_IP_VPS:8081/admin_login.html`
- 📊 **Dashboard**: `http://SEU_IP_VPS:8081/admin_dashboard.html`
- 🔗 **Webhook WhatsApp**: `http://SEU_IP_VPS:8081/webhook`

---

**Desenvolvido por Kauan Botura** 🚀  
**Deploy Guide - Hostinger VPS**
