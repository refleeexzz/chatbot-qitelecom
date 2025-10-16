# 📱 Sistema de Disparos em Lote - QIBOT

Sistema completo para envio de mensagens em massa via WhatsApp Business API.

## 🚀 Como Usar

### 1. **Acessar a Interface**
Abra no navegador: `http://localhost:8081/bulk-messages`

### 2. **Preencher o Formulário**

#### **Mensagem**
- Digite a mensagem que será enviada
- Máximo de 1000 caracteres
- O contador mostra quantos caracteres você já usou

#### **Destinatários**
Escolha uma das opções:

- **Enviar para todos os contatos**: Envia para todos que estão no banco de dados
- **Apenas Leads**: Filtra apenas contatos marcados como "Lead"
- **Apenas Clientes**: Filtra apenas contatos marcados como "Cliente"
- **Números específicos**: Digite números manualmente

#### **Números Específicos (opcional)**
Se escolher "Números específicos":
- Digite os números no formato internacional: `5511999999999`
- Separe múltiplos números com vírgula
- Exemplo: `5511999999999, 5511888888888, 5521777777777`

### 3. **Enviar**
- Clique em "🚀 Enviar Mensagens"
- Aguarde o processamento (pode levar alguns minutos)
- Você verá um resumo com quantas mensagens foram enviadas

## ⚙️ Como Funciona

### **Rate Limiting Automático**
- O sistema envia **1 mensagem a cada 1.5 segundos**
- Isso evita bloqueios da Meta (limite é ~80 msg/min)
- Para 100 contatos, leva aproximadamente 2-3 minutos

### **Logs**
Você pode acompanhar os envios nos logs do Docker:
```powershell
docker compose logs -f qibot
```

### **Formato dos Números**
- **Correto**: `5511999999999` (código do país + DDD + número)
- **Incorreto**: `11999999999` (sem código do país)
- **Incorreto**: `+55 11 99999-9999` (com formatação)

## 📊 Endpoints da API

### **POST /api/bulk-message**
Envia mensagens em lote

**Payload 1 - Todos os contatos:**
```json
{
  "message": "Olá! Temos uma promoção especial para você.",
  "send_to_all": true
}
```

**Payload 2 - Filtrar por tipo:**
```json
{
  "message": "Mensagem apenas para leads",
  "send_to_all": true,
  "filter_by_type": "Lead"
}
```

**Payload 3 - Números específicos:**
```json
{
  "message": "Mensagem para números específicos",
  "phones": ["5511999999999", "5511888888888"]
}
```

**Resposta de Sucesso:**
```json
{
  "status": "success",
  "sent": 45,
  "failed": 2,
  "total": 47,
  "errors": ["Erro ao enviar para 5511999999999: ..."],
  "message": "Disparo concluído: 45 enviadas, 2 falhas"
}
```

## 🔐 Segurança

### **Autenticação (Recomendado para Produção)**
Para proteger o endpoint em produção, adicione autenticação:

1. Crie um middleware de autenticação
2. Valide um token no header `Authorization`
3. Aplique ao endpoint `/api/bulk-message`

### **CORS**
O endpoint permite requisições de qualquer origem (`Access-Control-Allow-Origin: *`)
- Em produção, configure domínios específicos

## 🛠️ Variáveis de Ambiente Necessárias

```env
WHATSAPP_TOKEN=seu_token_aqui
WHATSAPP_PHONE_ID=seu_phone_id_aqui
```

Essas variáveis já devem estar configuradas no seu `.env`.

## 📝 Exemplos de Uso

### **Via Interface Web**
1. Acesse `http://localhost:8081/bulk-messages`
2. Digite: "Promoção especial! 50% OFF em todos os planos."
3. Escolha: "Enviar para todos os contatos"
4. Clique em "Enviar Mensagens"

### **Via API (PowerShell)**
```powershell
Invoke-RestMethod -Uri http://localhost:8081/api/bulk-message `
  -Method Post `
  -Headers @{ "Content-Type" = "application/json" } `
  -Body '{ 
    "message": "Olá! Temos novidades para você.",
    "send_to_all": true
  }'
```

### **Via API (cURL)**
```bash
curl -X POST http://localhost:8081/api/bulk-message \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Mensagem de teste",
    "phones": ["5511999999999"]
  }'
```

## ⚠️ Avisos Importantes

### **Compliance**
- ✅ Envie apenas para usuários que consentiram (opt-in)
- ❌ Não envie spam ou mensagens não solicitadas
- ⚖️ Respeite a LGPD e termos da Meta

### **Limites da Meta**
- **80 mensagens/minuto** (comercial)
- **1000 conversas/dia** (depende do tier da sua conta)
- Ultrapassar limites pode resultar em bloqueio temporário

### **Boas Práticas**
- Teste com poucos números primeiro
- Monitore os logs para erros
- Não envie mensagens promocionais fora do horário comercial
- Ofereça sempre opção de opt-out

## 🐛 Troubleshooting

### **Erro: "WHATSAPP_TOKEN não configurado"**
- Verifique se as variáveis estão no `.env`
- Recrie o container: `docker compose up -d --force-recreate`

### **Erro 403 da API WhatsApp**
- Token expirado ou inválido
- Gere um novo token no Meta Business Manager

### **Mensagens não chegam**
- Verifique se o número está no formato internacional
- Confirme que o número tem WhatsApp ativo
- Veja logs: `docker compose logs qibot`

### **Erro: "Nenhum telefone encontrado"**
- Banco de dados vazio
- Contatos sem telefone preenchido
- Use "Números específicos" e digite manualmente

## 📞 Suporte

Para dúvidas ou problemas:
1. Verifique os logs: `docker compose logs -f qibot`
2. Teste o endpoint `/health` para confirmar que o serviço está ativo
3. Revise a documentação da Meta WhatsApp Business API

---

**Desenvolvido com ❤️ para QIBOT**  
Copyright 2025 Kauan Botura
