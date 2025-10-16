# ========================================
#  SCRIPT PARA ENVIAR PROJETO PARA VPS
# ========================================

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  ENVIANDO PROJETO PARA VPS HOSTINGER" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Configurações
$VPS_IP = Read-Host "Digite o IP da sua VPS"
$VPS_USER = Read-Host "Digite o usuário SSH (padrão: root)"
if ([string]::IsNullOrEmpty($VPS_USER)) { $VPS_USER = "root" }

$PROJECT_PATH = "C:\Users\Usuario\Documents\LEADPROJECT"
$TEMP_ZIP = "$env:TEMP\qibot-deploy.zip"
$VPS_DIR = "/opt/qibot"

Write-Host ""
Write-Host "📦 Compactando projeto..." -ForegroundColor Yellow

# Arquivos para incluir
$filesToInclude = @(
    "docker-compose.yml",
    "Dockerfile",
    "go.mod",
    "go.sum",
    "main.go",
    ".env",
    "credentials.json",
    "*.html",
    "internal/*",
    "*.sh"
)

# Criar arquivo zip
if (Test-Path $TEMP_ZIP) { Remove-Item $TEMP_ZIP }
Compress-Archive -Path "$PROJECT_PATH\*" -DestinationPath $TEMP_ZIP -Force

Write-Host "✅ Projeto compactado: $TEMP_ZIP" -ForegroundColor Green
Write-Host ""

Write-Host "📤 Enviando para VPS..." -ForegroundColor Yellow
Write-Host "   Conectando em: $VPS_USER@$VPS_IP" -ForegroundColor Gray
Write-Host ""

# Enviar via SCP
scp $TEMP_ZIP "${VPS_USER}@${VPS_IP}:/tmp/qibot-deploy.zip"

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Arquivo enviado com sucesso!" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "📂 Descompactando na VPS..." -ForegroundColor Yellow
    
    # Executar comandos na VPS
    $commands = @"
mkdir -p $VPS_DIR
cd $VPS_DIR
unzip -o /tmp/qibot-deploy.zip
chmod +x *.sh
rm /tmp/qibot-deploy.zip
echo '✅ Projeto descompactado em $VPS_DIR'
"@
    
    # Executar via SSH
    $commands | ssh "${VPS_USER}@${VPS_IP}" "bash -s"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "=========================================" -ForegroundColor Green
        Write-Host "  PROJETO ENVIADO COM SUCESSO!" -ForegroundColor Green
        Write-Host "=========================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "Próximos passos:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "1. Conecte na VPS:" -ForegroundColor White
        Write-Host "   ssh $VPS_USER@$VPS_IP" -ForegroundColor Gray
        Write-Host ""
        Write-Host "2. Verifique/edite o arquivo .env:" -ForegroundColor White
        Write-Host "   cd $VPS_DIR && nano .env" -ForegroundColor Gray
        Write-Host ""
        Write-Host "3. Execute o deploy:" -ForegroundColor White
        Write-Host "   ./deploy-vps.sh" -ForegroundColor Gray
        Write-Host ""
    }
} else {
    Write-Host "❌ Erro ao enviar arquivo!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Verifique:" -ForegroundColor Yellow
    Write-Host "  - IP da VPS está correto?" -ForegroundColor Gray
    Write-Host "  - Consegue conectar via SSH?" -ForegroundColor Gray
    Write-Host "  - Usuário e senha estão corretos?" -ForegroundColor Gray
}

# Limpar arquivo temporário
Remove-Item $TEMP_ZIP -ErrorAction SilentlyContinue

Write-Host ""
Read-Host "Pressione ENTER para sair"
