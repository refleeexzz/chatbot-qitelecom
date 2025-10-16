#!/bin/bash

echo "========================================="
echo "  DEPLOY QIBOT - VPS HOSTINGER"
echo "========================================="
echo ""

# Definir diretório
PROJECT_DIR="/opt/qibot"

# Criar diretório se não existir
mkdir -p $PROJECT_DIR
cd $PROJECT_DIR

echo "📂 Diretório do projeto: $PROJECT_DIR"
echo ""

# Verificar se .env existe
if [ ! -f ".env" ]; then
    echo "⚠️  ATENÇÃO: Arquivo .env não encontrado!"
    echo ""
    echo "Crie o arquivo .env com suas configurações antes de continuar."
    echo "Execute: nano .env"
    echo ""
    exit 1
fi

echo "✅ Arquivo .env encontrado"
echo ""

# Parar containers existentes (se houver)
echo "🛑 Parando containers existentes..."
docker-compose down 2>/dev/null

# Construir e iniciar
echo "🔨 Construindo e iniciando containers..."
docker-compose up -d --build

# Verificar status
echo ""
echo "📊 Status dos containers:"
docker-compose ps

echo ""
echo "========================================="
echo "  DEPLOY CONCLUÍDO!"
echo "========================================="
echo ""
echo "Para ver os logs em tempo real:"
echo "  docker-compose logs -f qibot"
echo ""
echo "Para verificar status:"
echo "  docker-compose ps"
echo ""
echo "Acesse o sistema em:"
echo "  http://$(curl -s ifconfig.me):8081"
echo ""
