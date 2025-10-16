@echo off
echo =========================================
echo  LIBERANDO FIREWALL PARA QIBOT
echo =========================================
echo.
echo Porta: 8081
echo IP do Servidor: 192.168.0.13
echo.
echo Removendo regras antigas (se existirem)...
netsh advfirewall firewall delete rule name="QIBOT Server" >nul 2>&1
netsh advfirewall firewall delete rule name="QIBOT Server - Porta 8081" >nul 2>&1

echo.
echo Criando nova regra no firewall...
netsh advfirewall firewall add rule name="QIBOT Server" dir=in action=allow protocol=TCP localport=8081 profile=any

echo.
echo =========================================
echo  FIREWALL LIBERADO COM SUCESSO!
echo =========================================
echo.
echo Para acessar de outro computador na MESMA REDE, use:
echo.
echo   http://192.168.0.13:8081
echo.
echo URLs disponiveis:
echo   - Login Admin: http://192.168.0.13:8081/admin_login.html
echo   - Dashboard:   http://192.168.0.13:8081/admin_dashboard.html
echo.
pause
