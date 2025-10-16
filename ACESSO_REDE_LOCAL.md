# 🌐 Como Acessar o QIBOT em Rede Local

## 📍 Informações do Servidor

- **IP do Servidor**: `192.168.0.13`
- **Porta**: `8081`
- **URL Principal**: `http://192.168.0.13:8081`

---

## 🚀 Passo a Passo Rápido

### **1️⃣ Liberar o Firewall (OBRIGATÓRIO)**

**Execute como Administrador:**

1. Localize o arquivo: **`liberar-firewall.bat`**
2. Clique com **botão direito** → **"Executar como Administrador"**
3. Aguarde a mensagem de sucesso

### **2️⃣ Testar se Está Funcionando**

Execute o arquivo: **`testar-conexao.bat`**

Se aparecer "Status HTTP: 200", está tudo OK! ✅

### **3️⃣ Acessar de Outro Computador**

No navegador do **outro computador** (na mesma rede), digite:

```
http://192.168.0.13:8081
```

---

## 🔗 URLs Disponíveis

| Página | URL |
|--------|-----|
| 🏠 **Página Inicial** | `http://192.168.0.13:8081/` |
| 🔐 **Login Admin** | `http://192.168.0.13:8081/admin_login.html` |
| 📊 **Dashboard** | `http://192.168.0.13:8081/admin_dashboard.html` |
| 📱 **Disparos em Lote** | `http://192.168.0.13:8081/bulk_messages.html` |
| 👥 **Gerenciar Contatos** | `http://192.168.0.13:8081/admin_contacts.html` |

---

## 🔧 Configuração Manual do Firewall

Se preferir configurar manualmente:

### **Opção A: PowerShell (como Administrador)**

```powershell
New-NetFirewallRule -DisplayName "QIBOT Server" -Direction Inbound -LocalPort 8081 -Protocol TCP -Action Allow -Profile Any
```

### **Opção B: Interface Gráfica**

1. Pressione **Win + R**, digite `wf.msc` e pressione Enter
2. Clique em **"Regras de Entrada"** (lado esquerdo)
3. Clique em **"Nova Regra"** (lado direito)
4. Selecione **"Porta"** → Próximo
5. **TCP** → Portas locais específicas: **8081** → Próximo
6. **"Permitir a conexão"** → Próximo
7. Marque **todos os perfis** (Domínio, Privado, Público) → Próximo
8. Nome: **QIBOT Server** → Concluir

---

## ❓ Solução de Problemas

### **Problema: Não consigo acessar de outro PC**

✅ **Checklist:**
- [ ] O firewall foi liberado? (executou `liberar-firewall.bat` como Admin?)
- [ ] O Docker está rodando? (execute `docker-compose ps`)
- [ ] Ambos os PCs estão na **mesma rede Wi-Fi/Ethernet**?
- [ ] O IP está correto? (confira com `ipconfig`)

### **Como verificar o IP do servidor:**

No PowerShell:
```powershell
ipconfig
```

Procure o **"Adaptador Ethernet"** ou **"Wi-Fi"** e veja o **Endereço IPv4**.

### **Testar conectividade do outro PC:**

No outro computador, abra PowerShell e execute:

```powershell
Test-NetConnection -ComputerName 192.168.0.13 -Port 8081
```

**Resultado esperado:**
```
TcpTestSucceeded : True
```

Se aparecer **False**, o firewall está bloqueando.

---

## 🔒 Credenciais de Acesso

Configuradas no arquivo `.env`:

- **Usuário**: `QItelecom`
- **Senha**: `@Redeqi.com.br`

---

## 📋 Comandos Úteis

### **Ver status do Docker:**
```powershell
docker-compose ps
```

### **Ver logs em tempo real:**
```powershell
docker-compose logs -f qibot
```

### **Reiniciar o servidor:**
```powershell
docker-compose restart
```

### **Parar o servidor:**
```powershell
docker-compose down
```

### **Iniciar o servidor:**
```powershell
docker-compose up -d
```

---

## ⚠️ Importante

- ✅ O servidor está acessível apenas na **rede local** (LAN)
- ❌ **NÃO está acessível pela internet** (isso é bom para segurança!)
- 🔐 Certifique-se de estar em uma **rede confiável**

---

## 🆘 Ainda com problemas?

1. Execute `testar-conexao.bat` e tire print da tela
2. No outro PC, tente acessar e veja qual erro aparece
3. Verifique os logs: `docker-compose logs qibot`

---

**Desenvolvido por Kauan Botura** 🚀  
**Projeto: QIBOT - QITelecom** 📱
