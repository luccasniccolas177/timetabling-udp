#!/bin/bash
# Script para verificar si las instancias de cátedras están en el mismo bloque

echo "🔍 Verificando distribución de instancias de cátedras..."
echo ""

# Ejecutar el programa y capturar output
./bin/timetabling 2>&1 > /tmp/timetabling_output.txt

# Buscar algunas cátedras con frecuencia > 1
echo "Ejemplo 1: CIT1000-L1 (debería tener W1, W2, W3)"
grep -E "CIT1000-L1-W[123]" /tmp/timetabling_output.txt | head -5

echo ""
echo "Ejemplo 2: CBM1000-L1 (debería tener W1, W2)"
grep -E "CBM1000-L1-W[12]" /tmp/timetabling_output.txt | head -5

echo ""
echo "Ejemplo 3: CBF1000-L1 (debería tener W1, W2)"
grep -E "CBF1000-L1-W[12]" /tmp/timetabling_output.txt | head -5

echo ""
echo "📊 Resumen: ¿Están en el mismo bloque?"
