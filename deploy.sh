#!/bin/bash
source ~/.bashrc

AWS_PROFILE="ivanjoz"
AWS_S3="gerp-v2-frontend"
FUNCTION_NAME="jobfinder6-p-app"

# Ensure we are in the root directory
cd "$(dirname "$0")"

echo "=== GENIX DEPLOYMENT & UTILS ==="
echo "Seleccione acciones a realizar (ej: '123'):"
echo "Publicar Código ----------------"
echo "[1] Frontend (Main + Store -> docs/)"
echo "[2] Backend (AWS Cloud)"
echo "[4] Backup Lib (S3 Binary)"
echo "Ejecutar Proceso ---------------"
echo "[5] Recrear Tablas (Backend)"
echo "[6] Poblar Estructuras (Backend)"
echo "[7] Inspeccionar/Compilar Backend"
echo "Local Development --------------"
echo "[8] Serve Local Build (docs/)"
read ACCIONES

# Check if we need git pull
if [[ $ACCIONES =~ [1234] ]]; then
    echo "Obteniendo los últimos cambios del repositorio (GIT PULL)..."
    git pull
fi

export PATH=$HOME/.nvm/versions/node/v20.16.0/bin:$PATH

GO_PATH="go"
if [ -x /usr/local/go/bin/go ]; then
    GO_PATH="/usr/local/go/bin/go"
fi

# PUBLICAR FRONTEND
if [[ $ACCIONES == *"1"* ]]; then
    echo "=== PUBLICANDO FRONTEND ==="
    echo "Generando frontend (Main + Store) en carpeta 'docs' para GitHub Pages..."
    
    cd ./frontend
    # Usamos publish que ya integra build-all.js (Main + Store) y postbuild.js (copy to docs)
    bun run publish
    cd ..
    
    echo "✅ El deploy frontend finalizado en ./docs!"
fi

# SERVE LOCAL BUILD
if [[ $ACCIONES == *"8"* ]]; then
    echo "=== SIRVIENDO FRONTEND LOCAL (docs/) ==="
    if [ ! -d "./docs" ]; then
        echo "❌ La carpeta 'docs' no existe. Ejecute el paso [1] primero."
    else
        echo "Iniciando servidor local en http://localhost:3000..."
        # Usamos bun x para ejecutar serve (configurado via docs/serve.json para multi-SPA)
        bun x serve ./docs -l 3000
    fi
fi

# PUBLICAR BACKEND
if [[ $ACCIONES == *"2"* ]]; then
    echo "=== PUBLICANDO BACKEND ==="
    cd ./cloud
    $GO_PATH run . accion=1
    cd ..
    echo "✅ El deploy backend finalizado!"
fi

# PUBLICAR DB BACKUP BINARY
if [[ $ACCIONES == *"4"* ]]; then
    echo "=== PUBLICANDO DB-BACKUP ==="
    cd ./db-backup
    GOOS=linux GOARCH=arm64 go build -ldflags '-s -w' .
    aws --profile $AWS_PROFILE s3 cp ./db-backup s3://$AWS_S3/_bin/db-backup.bin
    cd ..
    echo "✅ El deploy del ejecutable finalizó."
fi

# RECREAR TABLAS
if [[ $ACCIONES == *"5"* ]]; then
    echo "=== RECREANDO TABLAS ==="
    cd ./backend
    $GO_PATH run . fn-homologate
    cd ..
fi

# INSPECCIONAR BACKEND
if [[ $ACCIONES == *"7"* ]]; then
    echo "=== COMPILANDO BACKEND ==="
    cd ./backend
    $GO_PATH build -v .
    # gsa app (assuming this is a local tool)
    cd ..
fi

echo "Finalizado!. Presione cualquier tecla para salir"
read -n 1

