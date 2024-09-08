#!/bin/bash
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

#PUBLICAR FRONTEND
if [[ $ACCIONES == *"1"* ]]; then

    echo "=== PUBLICANDO FRONTEND ==="
    echo "Enviando archivos a S3: $AWS_S3"

    S3_SYNC="s3 sync ./frontend/.output/public"
    S3_CP="s3 cp ./frontend/.output/public"

    if [[ $ACCIONES != *"x"* ]]; then
       npm run build --prefix ./frontend
    fi

    aws --profile $AWS_PROFILE $S3_SYNC/assets s3://$AWS_S3/assets --size-only --exclude "*" --include "*.js"  --content-type application/javascript --delete
    aws --profile $AWS_PROFILE $S3_SYNC/assets s3://$AWS_S3/assets --size-only --exclude "*" --include "*.css" --content-type text/css --delete
    aws --profile $AWS_PROFILE $S3_SYNC/_build s3://$AWS_S3/_build --size-only --exclude "*" --include "*.js"  --content-type application/javascript --delete
    aws --profile $AWS_PROFILE $S3_SYNC/_build s3://$AWS_S3/_build --size-only --exclude "*" --include "*.css" --content-type text/css --delete
    aws --profile $AWS_PROFILE $S3_CP/sw.js s3://$AWS_S3/sw.js --content-type application/javascript
    aws --profile $AWS_PROFILE $S3_CP/manifest.webmanifest s3://$AWS_S3/manifest.webmanifest --content-type application/json
    aws --profile $AWS_PROFILE $S3_CP/index.html s3://$AWS_S3/index.html --content-type text/html

    if [[ $ACCIONES == *"3"* ]]; then
       aws --profile $AWS_PROFILE $S3_SYNC/images s3://$AWS_S3/images --size-only
       # aws --profile $AWS_PROFILE $S3_SYNC/icons s3://$AWS_S3/icons --size-only
       aws --profile $AWS_PROFILE $S3_SYNC/libs s3://$AWS_S3/libs --size-only
    fi

    echo "El deploy frontend finalizado!"

fi

#PUBLICAR BACKEND
if [[ $ACCIONES == *"2"* ]]; then

    echo "=== PUBLICANDO BACKEND ==="

    cd ./cloud
    go run . accion=1

    echo "El deploy backend-node finalizado!"

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
    go run . fn-homologate

fi

if [[ $ACCIONES == *"7"* ]]; then

    echo "=== COMPILANDO BACKEND ==="

    cd ./backend
    go build -v .
    gsa app

fi

echo "Finalizado!. Presione cualquier tecla para salir"
read
kill -9 $PPID
