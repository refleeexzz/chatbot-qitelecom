# QI TELECOM - Sistema de Atendimento Inteligente

Este projeto é um sistema completo de atendimento para provedores de internet, com chatbot integrado, interface web responsiva e integração com Google Sheets.

## Funcionalidades
- Chatbot para atendimento automatizado (suporte técnico, planos, financeiro e dúvidas gerais)
- Interface web moderna e responsiva
- Coleta de feedback do cliente
- Integração com Google Sheets para registro de atendimentos, planos e feedbacks
- Gerenciamento de sessões e histórico de atendimento
- Suporte a múltiplos usuários simultâneos

## Tecnologias Utilizadas
- Go (backend)
- HTML, CSS, JavaScript (frontend)
- Google Sheets API
- Redis (gerenciamento de sessão)
- Docker (opcional)

## Como usar
1. Clone o repositório
2. Configure as credenciais do Google Sheets (`credentials.json`)
3. Execute o backend Go
4. Acesse a interface web em `index.html` ou via servidor

## Observações
- Não suba arquivos sensíveis como `credentials.json` ou `.env` em repositórios públicos.
- O sistema pode ser adaptado para outros provedores ou fluxos de atendimento.

---
Desenvolvido por Kauan Botura (dev) e Ronan Moreira (liderança do projeto)
