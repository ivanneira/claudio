#!/bin/bash
# Script de prueba - muestra información del sistema

echo "🖥️  Sistema: $(uname -a)"
echo "📅 Fecha: $(date)"
echo "⏰ Uptime: $(uptime)"
echo "💾 Memoria:"
free -h
echo "💽 Espacio en disco:"
df -h / | tail -1