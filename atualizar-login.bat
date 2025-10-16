@echo off
echo =========================================
echo  ATUALIZANDO admin_login.html NO CONTAINER
echo =========================================
echo.
echo Copiando arquivo para o container...
docker cp admin_login.html qibot:/app/admin_login.html

if %errorlevel% equ 0 (
    echo.
    echo =========================================
    echo  ARQUIVO ATUALIZADO COM SUCESSO!
    echo =========================================
    echo.
    echo O arquivo foi copiado para o container.
    echo Nao precisa reiniciar o Docker!
    echo.
    echo Acesse: http://192.168.0.13:8081/admin_login.html
    echo Pressione Ctrl+F5 no navegador para limpar o cache.
) else (
    echo.
    echo ERRO ao copiar arquivo!
    echo Verifique se o container "qibot" esta rodando.
)
echo.
pause
