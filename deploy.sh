#!/bin/bash
source ~/.bashrc

AWS_PROFILE="ivanjoz"
AWS_S3="gerp-v2-frontend"
FUNCTION_NAME="jobfinder6-p-app"
PUBLICAR_ASSETS=""

echo "Seleccione acciones a realizar: (Es posible escoger más de 1. Ejemplo: '123')"
echo "Publicar Código ----------------"
echo "[1] Frontend [2] Backend [3] Frontend (Assets) [4] Backup Lib"
echo "Ejecutar Proceso ---------------"
echo "[5] Recrear Tablas [6] Poblar Estructuras [7] Inspeccionar Backend"
read ACCIONES

echo "Obteniendo los últimos cambios del repositorio (GIT PULL)..."

if [[ $ACCIONES == *"1"* || $ACCIONES == *"2"* || $ACCIONES == *"3"* ]]; then
    git pull
fi

echo "Usando AWS Profile: $AWS_PROFILE"
export PATH=$HOME/.nvm/versions/node/v20.16.0/bin:$PATH

GO_PATH="go"
if [ -x /usr/local/go/bin/go ]; then
    GO_PATH="/usr/local/go/bin/go"
fi

#PUBLICAR FRONTEND
if [[ $ACCIONES == *"1"* ]]; then

    echo "=== PUBLICANDO FRONTEND ==="
    echo "Generando frontend a docs para su deploy en .github"

    if [[ $ACCIONES != *"x"* ]]; then
       npm run build --prefix ./frontend
    fi

    cd ./frontend
    node build.js

    echo "El deploy frontend finalizado!"
fi

#PUBLICAR BACKEND
if [[ $ACCIONES == *"2"* ]]; then

    echo "=== PUBLICANDO BACKEND ==="

    cd ./cloud
    $GO_PATH run . accion=1

    echo "El deploy backend finalizado!"

fi

#PUBLICAR DB BACKUP BINARY
if [[ $ACCIONES == *"4"* ]]; then

    echo "=== PUBLICANDO DB-BACKUP ==="

    cd ./db-backup
    GOOS=linux GOARCH=arm64 go build -ldflags '-s -w' .

    aws --profile $AWS_PROFILE s3 cp ./db-backup s3://$AWS_S3/_bin/db-backup.bin
    echo "El deploy del ejecutable finalizó."

fi

#PUBLICAR DB BACKUP BINARY
if [[ $ACCIONES == *"5"* ]]; then

    echo "=== RECREANDO TABLAS ==="

    cd ./backend
    $GO_PATH run . fn-homologate

fi

if [[ $ACCIONES == *"7"* ]]; then

    echo "=== COMPILANDO BACKEND ==="

    cd ./backend
    $GO_PATH build -v .
    gsa app

fi

echo "Finalizado!. Presione cualquier tecla para salir"
read
kill -9 $PPID
