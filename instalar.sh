#!/bin/bash

# Script de instalación automática para Nodos Guard Pro 7 (GuardPro)
echo "--------------------------------------------------------"
echo "🛰️ Iniciando Instalación Automática de Nodo Guard Pro 7"
echo "--------------------------------------------------------"

# 1. Detectar el entorno (Ubuntu tradicional o Android/Termux)
if [ -d "$HOME/.termux" ] || [ -f "/system/build.prop" ]; then
    echo "🤖 Entorno detectado: Android / Termux"
    PAQUETES="golang git curl"
    MANEJADOR="pkg"
else
    echo "🐧 Entorno detectado: Linux / Ubuntu"
    PAQUETES="golang-go git curl"
    MANEJADOR="sudo apt"
fi

# 2. Actualizar el sistema e instalar dependencias si faltan
echo "🔄 Actualizando repositorios e instalando dependencias básicas..."
if [ "$MANEJADOR" == "pkg" ]; then
    pkg update && pkg upgrade -y
    pkg install $PAQUETES -y
else
    sudo apt update
    sudo apt install $PAQUETES -y
fi

# 3. Configurar la carpeta del nodo limpio
echo "📁 Preparando directorios del proyecto..."
mkdir -p $HOME/guardpro-clean
cd $HOME/guardpro-clean

# 4. Clonar el repositorio limpio (Asegúrate de cambiar este enlace por tu repositorio real más adelante)
echo "📦 Descargando el código fuente oficial de GuardPro..."
# Nota: Durante tu desarrollo manual, puedes omitir este paso clonando directamente tu carpeta local.
  git clone https://github.com/roonm89/guardpro-node .

# 5. Leer los parámetros que el usuario envió al comando
TIPO_NODO="validador" # Por defecto se asigna como validador
for i in "$@"
do
case $i in
    --tipo=*)
    TIPO_NODO="${i#*=}"
    shift
    ;;
    *)
    ;;
esac
done

echo "⚙️ Configuración completada con éxito."
echo "🚀 Iniciando el nodo Guard Pro 7 en modo: [$TIPO_NODO]..."
echo "--------------------------------------------------------"

# 6. Ejecutar el código Go con los parámetros dinámicos correspondientes
# Nota: Si el archivo main.go ya está en la carpeta, lo arranca de inmediato
if [ -f "main.go" ]; then
    go run main.go --tipo=$TIPO_NODO
else
    # Si estás haciendo una prueba en tu misma laptop, copiamos tu main.go actual
    cp $HOME/guardpro-node/main.go . 2>/dev/null
    go run main.go --tipo=$TIPO_NODO
fi
