# Claudio - SSH Tunnel Manager con Telegram Bot

<!-- LLM-METADATA: SSH tunnel manager with Telegram bot integration for remote access -->
<!-- VERSION: 1.0 -->
<!-- LAST_UPDATE: 2025-08-12 -->

## Contexto Rápido

Aplicación Go que crea túneles SSH públicos y permite ejecutar scripts remotamente vía comandos de Telegram.

## Características

- 🔐 **Túnel SSH automático** usando ngrok o localtunnel
- 📱 **Notificaciones Telegram** de estado y errores
- 🤖 **Ejecución remota de scripts** con comandos `!<script>`
- 🔄 **Auto-actualización** del repositorio y reinicio automático
- 📊 **Monitoreo** de estado SSH y túneles
- 🔑 **Comando SSH listo** para conexión inmediata
- ⚙️ **Servicio systemd** para auto-inicio con el SO

## Configuración

### Variables de Entorno

```bash
export TELEGRAM_BOT_TOKEN="tu_bot_token"
export TELEGRAM_CHAT_ID="tu_chat_id"
export SSH_PORT="22"                    # Opcional, default: 22
export TUNNEL_SERVICE="ngrok"           # Opcional: ngrok|localtunnel
```

### Dependencias

```bash
# Para ngrok
wget https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz
tar xvf ngrok-v3-stable-linux-amd64.tgz
sudo mv ngrok /usr/local/bin/

# Para localtunnel (alternativa)
npm install -g localtunnel
```

## Instalación y Uso

### Ejecución Manual

```bash
# 1. Compilar
go build -o claudio .

# 2. Ejecutar
./claudio

# 3. En Telegram, envía comandos como:
# !test      - Ejecuta scripts/test.sh
# !status    - Muestra estado del sistema
# !update    - Actualiza y reinicia automáticamente
```

### Instalación como Servicio (Recomendado)

```bash
# 1. Compilar
go build -o claudio .

# 2. Copiar archivo de servicio
sudo cp claudio.service /etc/systemd/system/

# 3. Configurar variables de entorno en .env
echo "TELEGRAM_BOT_TOKEN=tu_bot_token" > .env
echo "TELEGRAM_CHAT_ID=tu_chat_id" >> .env

# 4. Habilitar y iniciar servicio
sudo systemctl daemon-reload
sudo systemctl enable claudio.service
sudo systemctl start claudio.service

# 5. Verificar estado
sudo systemctl status claudio.service

# 6. Ver logs
sudo journalctl -u claudio.service -f
```

## Scripts Disponibles

### Scripts Incluidos
- **test** - Información del sistema (memoria, disco, uptime)
- **status** - Estado SSH y túneles activos  
- **update** - Actualización automática y reinicio

### Crear Scripts Personalizados

1. Crear archivo en `./scripts/nombre.sh`
2. Hacer ejecutable: `chmod +x scripts/nombre.sh`  
3. Ejecutar desde Telegram: `!nombre`

```bash
#!/bin/bash
# Ejemplo: scripts/deploy.sh
echo "Desplegando aplicación..."
# Tu código aquí
```

## Comandos Frecuentes

### Ejecución Manual
```bash
# Compilar y ejecutar
go build -o claudio . && ./claudio

# Ver logs en tiempo real
tail -f claudio.out

# Verificar proceso corriendo
ps aux | grep claudio

# Matar proceso
pkill claudio

# Actualización manual
bash scripts/update.sh
```

### Gestión del Servicio
```bash
# Controlar servicio
sudo systemctl start claudio.service
sudo systemctl stop claudio.service
sudo systemctl restart claudio.service
sudo systemctl status claudio.service

# Ver logs
sudo journalctl -u claudio.service -f
sudo journalctl -u claudio.service --since today

# Deshabilitar servicio
sudo systemctl disable claudio.service
```

## Seguridad

- ✅ Solo acepta comandos del `TELEGRAM_CHAT_ID` configurado
- ✅ Scripts ejecutados con permisos del usuario actual
- ✅ Logs de todas las ejecuciones
- ⚠️ **IMPORTANTE**: Revisar scripts antes de ejecutar desde Telegram

## Troubleshooting

### SSH no detectado
```bash
# Verificar SSH corriendo
sudo systemctl status ssh
sudo systemctl start ssh

# En WSL, puede requerir:
sudo service ssh start
```

### Ngrok no funciona
```bash
# Verificar instalación
which ngrok
ngrok version

# Configurar authtoken (requerido)
ngrok authtoken tu_authtoken
```

### Bot no responde
```bash
# Verificar variables
echo $TELEGRAM_BOT_TOKEN
echo $TELEGRAM_CHAT_ID

# Test manual del bot
curl -X GET "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getMe"
```

---
*Creado: 2025-08-12 | Versión: 1.0*