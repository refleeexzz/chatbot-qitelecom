#!/bin/bash

echo "========================================="
echo "  INSTALANDO DOCKER NA VPS HOSTINGER"
echo "========================================="
echo ""

# Verificar se já está instalado
if command -v docker &> /dev/null; then
    echo "✅ Docker já está instalado!"
    docker --version
else
    echo "📦 Instalando Docker..."
    
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
    
    echo "✅ Docker instalado com sucesso!"
fi

# Verificar Docker Compose
if command -v docker-compose &> /dev/null; then
    echo "✅ Docker Compose já está instalado!"
    docker-compose --version
else
    echo "📦 Instalando Docker Compose..."
    
    # Instalar Docker Compose
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    
    echo "✅ Docker Compose instalado com sucesso!"
fi

# Iniciar Docker
systemctl start docker
systemctl enable docker

echo ""
echo "========================================="
echo "  INSTALAÇÃO CONCLUÍDA!"
echo "========================================="
echo ""
echo "Versões instaladas:"
docker --version
docker-compose --version
echo ""
