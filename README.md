# QI TELECOM - Sistema de Atendimento Inteligente

Este projeto é um sistema completo de atendimento para provedores de internet, com chatbot integrado, interface web responsiva e integração com Google Sheets.

## Funcionalidades

## Tecnologias Utilizadas

## Como usar
1. Clone o repositório
2. Configure as credenciais do Google Sheets (`credentials.json`)
3. Configure as variáveis de ambiente do WhatsApp Cloud API (opcional, para integração WhatsApp):
	- `WHATSAPP_VERIFY_TOKEN`: Token de verificação do webhook (definido no painel Meta)
	- `WHATSAPP_PHONE_ID`: ID do número do WhatsApp Cloud API
	- `WHATSAPP_TOKEN`: Token de acesso (Bearer) da API do WhatsApp
4. Execute o backend Go
5. Acesse a interface web em `index.html` ou via servidor

## Sessões de Usuário (Isolamento de Conversa)

O endpoint `/chatbot` agora suporta isolamento por sessão automaticamente.

Ordem de resolução do identificador de sessão:
1. Campo `user_id` no JSON da requisição
2. Header `X-Session-ID`
3. Cookie `qid` (gerado automaticamente se ausente)
4. Geração automática (UUID) caso nenhum seja fornecido

Exemplo de requisição inicial (sem `user_id`):
```bash
curl -X POST http://localhost:8081/chatbot \
	-H "Content-Type: application/json" \
	-d '{"message":"oi"}'
```
Resposta incluirá:
```json
{ "response": "...", "session_id": "<UUID>" }
```

Use esse `session_id` nas próximas chamadas:
```bash
curl -X POST http://localhost:8081/chatbot \
	-H "Content-Type: application/json" \
	-H "X-Session-ID: <UUID>" \
	-d '{"message":"1"}'
```

Ou envie diretamente em `user_id`:
```bash
{ "user_id":"<UUID>", "message":"menu" }
```


## Integração com WhatsApp Cloud API

O sistema suporta integração direta com o WhatsApp Cloud API (Meta). Basta configurar as variáveis de ambiente e apontar o webhook no painel Meta para:

```
https://<SEU_DOMÍNIO>/webhook/whatsapp
```

**Variáveis obrigatórias:**
- `WHATSAPP_VERIFY_TOKEN`: Token de verificação do webhook (definido no painel Meta)
- `WHATSAPP_PHONE_ID`: ID do número do WhatsApp Cloud API
- `WHATSAPP_TOKEN`: Token de acesso (Bearer) da API do WhatsApp

**Como funciona:**
- O webhook recebe mensagens de texto do WhatsApp, processa usando o mesmo fluxo do chatbot e responde automaticamente.
- O número do remetente é usado como `user_id` para isolar sessões.
- O endpoint `/webhook/whatsapp` aceita POST (mensagens) e GET (validação Meta).

**Testando:**
1. Configure o webhook no painel Meta (WhatsApp Cloud API) para apontar para seu endpoint público.
2. Envie mensagens para o número do WhatsApp Cloud API.
3. O bot responderá automaticamente conforme o fluxo do chatbot.

**Dica:**
Se o telefone do usuário já está disponível no remetente, o fluxo de planos pode pular a etapa de pedir telefone.

---

## Fluxo de Planos (Atualizado)

O fluxo de contratação/upgrade de planos agora coleta também o **telefone/WhatsApp** para facilitar o contato do time comercial e foi incluída uma coluna adicional na aba `Página3` da planilha.

Ordem do fluxo:
1. Usuário escolhe opção `2` (Planos e Serviços)
2. Responde se é cliente atual (SIM/NÃO)
3. Informa plano atual (se for cliente)
4. Seleciona ou digita o plano desejado
5. Informa **Nome Completo**
6. Informa **Telefone/WhatsApp** (ex: 44999998888 ou (44) 99999-8888)
7. Sistema salva na `Página3` com colunas:
	- DATA/HORA
	- NOME COMPLETO
	- SITUAÇÃO CLIENTE
	- PLANO ATUAL
	- PLANO DESEJADO
	- TELEFONE
	- OBSERVAÇÕES (Ex: "Interesse em: X | Plano atual: Y")

Importante: A função `SavePlans` foi alterada para receber o telefone. Caso já exista dados antigos, apenas a nova coluna será adicionada (não apaga anteriores).

Exemplo de resposta final mostrada ao usuário:
```
🎉 Dados Registrados com Sucesso!

Nome: João da Silva
Situação: Cliente Atual
Plano Interesse: QI FIBRA PREMIUM (MELHOR)
Telefone: (44)99999-8888

Próximos Passos:
Nossa equipe comercial entrará em contato em até 24 horas para finalizar!
```

Se integrar com WhatsApp, o telefone pode já vir do remetente e preencher automaticamente esta etapa (adaptável no código adicionando verificação antes de perguntar o telefone).

- O sistema pode ser adaptado para outros provedores ou fluxos de atendimento.
---
Desenvolvido por Kauan Botura (dev) e Ronan Moreira (liderança do projeto)
