@echo off
echo =========================================
echo  ATUALIZANDO TODOS OS ARQUIVOS HTML
echo =========================================
echo.
echo Copiando arquivos para o container...

docker cp index.html qibot:/app/index.html
docker cp admin_login.html qibot:/app/admin_login.html
docker cp admin_dashboard.html qibot:/app/admin_dashboard.html
docker cp admin_contacts.html qibot:/app/admin_contacts.html
docker cp bulk_messages.html qibot:/app/bulk_messages.html

echo.
echo =========================================
echo  ARQUIVOS ATUALIZADOS COM SUCESSO!
echo =========================================
echo.
echo Todos os arquivos HTML foram copiados.
echo Nao precisa reiniciar o Docker!
echo.
echo Pressione Ctrl+F5 no navegador para limpar o cache.
echo.
pause
