@echo off
echo =========================================
echo  TESTANDO ACESSO AO QIBOT
echo =========================================
echo.
echo IP Local: 192.168.0.13
echo Porta: 8081
echo.
echo [1/3] Verificando se o Docker esta rodando...
docker-compose ps

echo.
echo [2/3] Testando conexao local (localhost)...
curl http://localhost:8081 -s -o nul -w "Status HTTP: %%{http_code}\n"

echo.
echo [3/3] Verificando se a porta esta aberta...
netstat -an | findstr ":8081"

echo.
echo =========================================
echo  RESULTADO
echo =========================================
echo.
echo Se aparecer "Status HTTP: 200" acima, o servidor esta OK!
echo.
echo Para acessar de OUTRO COMPUTADOR na mesma rede:
echo   http://192.168.0.13:8081
echo.
echo Se nao funcionar do outro PC:
echo   1. Execute "liberar-firewall.bat" como ADMINISTRADOR
echo   2. Verifique se ambos os PCs estao na mesma rede
echo.
pause
