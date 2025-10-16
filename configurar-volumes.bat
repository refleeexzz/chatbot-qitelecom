@echo off
echo =========================================
echo  CONFIGURANDO VOLUMES PARA HTML
echo =========================================
echo.
echo Esta opcao permite editar os arquivos HTML
echo e ver as mudancas IMEDIATAMENTE, sem precisar
echo copiar ou fazer rebuild!
echo.
echo Reiniciando containers com nova configuracao...
docker-compose down
docker-compose up -d

echo.
echo =========================================
echo  CONFIGURACAO CONCLUIDA!
echo =========================================
echo.
echo Agora voce pode editar qualquer arquivo HTML
echo e apenas dar F5 no navegador para ver as mudancas!
echo.
echo Arquivos mapeados:
echo  - index.html
echo  - admin_login.html
echo  - admin_dashboard.html
echo  - admin_contacts.html
echo  - bulk_messages.html
echo.
pause
