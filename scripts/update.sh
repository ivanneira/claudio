#!/bin/bash
# Script de actualización automática para Claudio
# Actualiza el repositorio, recompila y reinicia el servicio

set -e  # Salir si hay algún error

REPO_DIR="/home/ineira/codigo/claudio"
BINARY_NAME="claudio"
LOG_FILE="$REPO_DIR/update.log"

# Función para logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "🔄 Iniciando proceso de actualización..."

# Verificar que estamos en el directorio correcto
if [ ! -f "$REPO_DIR/main.go" ]; then
    log "❌ Error: No se encuentra main.go en $REPO_DIR"
    exit 1
fi

cd "$REPO_DIR"

# Buscar y matar el proceso actual si existe
log "🔍 Buscando proceso en ejecución..."
PID=$(pgrep -f "$BINARY_NAME" || true)
if [ -n "$PID" ]; then
    log "⏹️  Deteniendo proceso existente (PID: $PID)"
    kill -TERM "$PID" 2>/dev/null || kill -KILL "$PID" 2>/dev/null
    sleep 2
fi

# Actualizar desde git si es un repositorio git
if [ -d ".git" ]; then
    log "📥 Actualizando desde repositorio..."
    git fetch origin
    
    # Verificar si hay cambios
    BEHIND=$(git rev-list HEAD..origin/main --count 2>/dev/null || echo "0")
    if [ "$BEHIND" -gt "0" ]; then
        log "📦 Aplicando $BEHIND nuevos commits..."
        git pull origin main
    else
        log "ℹ️  Repositorio ya está actualizado"
    fi
else
    log "⚠️  No es un repositorio git, omitiendo actualización"
fi

# Hacer backup del binario actual si existe
if [ -f "$BINARY_NAME" ]; then
    log "💾 Respaldando binario actual..."
    cp "$BINARY_NAME" "${BINARY_NAME}.backup.$(date +%s)"
fi

# Compilar la nueva versión
log "🔨 Compilando nueva versión..."
if go build -o "$BINARY_NAME" .; then
    log "✅ Compilación exitosa"
else
    log "❌ Error en compilación"
    # Restaurar backup si existe
    if [ -f "${BINARY_NAME}.backup"* ]; then
        LATEST_BACKUP=$(ls -t ${BINARY_NAME}.backup* | head -1)
        cp "$LATEST_BACKUP" "$BINARY_NAME"
        log "🔄 Restaurado backup: $LATEST_BACKUP"
    fi
    exit 1
fi

# Hacer el binario ejecutable
chmod +x "$BINARY_NAME"

# Limpiar backups antiguos (mantener solo los 3 más recientes)
ls -t ${BINARY_NAME}.backup* 2>/dev/null | tail -n +4 | xargs rm -f 2>/dev/null || true

# Reiniciar el servicio en segundo plano
log "🚀 Reiniciando servicio..."
nohup ./"$BINARY_NAME" > claudio.out 2>&1 &
NEW_PID=$!

# Verificar que el nuevo proceso está corriendo
sleep 2
if kill -0 "$NEW_PID" 2>/dev/null; then
    log "✅ Servicio reiniciado exitosamente (PID: $NEW_PID)"
    echo "✅ Actualización completada exitosamente"
    echo "🆔 Nuevo PID: $NEW_PID"
    echo "📄 Ver logs: tail -f $REPO_DIR/claudio.out"
else
    log "❌ Error: El servicio no se pudo reiniciar"
    echo "❌ Error en el reinicio del servicio"
    exit 1
fi

log "🎉 Proceso de actualización completado"