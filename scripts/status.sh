#!/bin/bash
# Muestra el estado del servicio SSH y túneles activos

echo "🔐 Estado del servicio SSH:"
if systemctl is-active --quiet ssh 2>/dev/null || service ssh status >/dev/null 2>&1; then
    echo "✅ SSH está corriendo"
else
    echo "❌ SSH no está activo"
fi

echo ""
echo "🌐 Túneles activos:"
ps aux | grep -E "(ngrok|lt)" | grep -v grep || echo "No hay túneles detectados"

echo ""
echo "🔌 Puertos en uso:"
ss -tuln | grep :22 || netstat -tuln 2>/dev/null | grep :22 || echo "Puerto SSH no listado"