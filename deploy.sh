#!/bin/bash
source ~/.bashrc

AWS_PROFILE="ivanjoz"
AWS_S3="gerp-v2-frontend"
FUNCTION_NAME="jobfinder6-p-app"

# Ensure we are in the root directory
cd "$(dirname "$0")"

echo "=== GENIX DEPLOYMENT & UTILS ==="
echo "Seleccione acciones a realizar separadas por espacio o coma (ej: '1 2 3'):"
echo "Publicar Código ----------------"
echo "[1] Frontend (Main + Store -> docs/)"
echo "[2] Backend (AWS Cloud)"
echo "[3] Backend (VPS)"
echo "[4] Backup Lib (S3 Binary)"
echo "Ejecutar Proceso ---------------"
echo "[5] Desplegar Tablas (Backend)"
echo "[6] Desplegar: Tablas, Datos Iniciales, Cloudflare Worker"
echo "[7] Inspeccionar/Compilar Backend"
echo "[10] Deploy Cloudflare Worker"
echo "[11] Deploy Company Webpage"
echo "[12] Sincronizar Catálogo de Imágenes"
echo "Infraestructura ----------------"
echo "[9] Desplegar Infraestructura"
echo "Local Development --------------"
echo "[8] Serve Local Build (docs/)"

INTERACTIVE=0
if [ "$#" -gt 0 ]; then
    ACTION_INPUT="$1"
    COMPANY_ID_ARGUMENT="${2:-}"
else
    INTERACTIVE=1
    read -r ACTION_INPUT
    COMPANY_ID_ARGUMENT=""
fi

# Exact action tokens prevent option 11 from also matching option 1.
read -r -a ACTIONS <<< "${ACTION_INPUT//,/ }"
has_action() {
    local expected_action="$1"
    local selected_action
    for selected_action in "${ACTIONS[@]}"; do
        if [ "$selected_action" = "$expected_action" ]; then
            return 0
        fi
    done
    return 1
}

# Check if we need git pull
if has_action "1" || has_action "2" || has_action "3" || has_action "4"; then
    echo "Obteniendo los últimos cambios del repositorio (GIT PULL)..."
    git pull
fi

export PATH=$HOME/.nvm/versions/node/v20.16.0/bin:$PATH

GO_PATH="go"
if [ -x /usr/local/go/bin/go ]; then
    GO_PATH="/usr/local/go/bin/go"
fi

deploy_tables() {
    echo "=== RECREANDO TABLAS ==="
    # Refresh controllers.generated.go so fn-homologate sees every current table struct.
    echo "--- Regenerando controllers.generated.go ---"
    (cd scripts && "$GO_PATH" run . generate_controllers) || return 1
    (cd backend && "$GO_PATH" run . fn-homologate) || return 1
}

# PUBLICAR FRONTEND
if has_action "1"; then
    echo "=== PUBLICANDO FRONTEND ==="
    echo "Generando bundle comprimido (frontend.zip) en carpeta 'docs'..."
    
    cd ./frontend
    # Usamos publish que ya integra build-all.js (Main + Store) y postbuild.js (zip only bundled)
    bun run publish
    cd ..
    
    echo "✅ El bundle frontend.zip ha sido generado en ./docs!"
    echo "💡 Recuerde hacer git add docs/frontend.zip y push para activar el deploy."
fi

# SERVE LOCAL BUILD
if has_action "8"; then
    echo "=== SIRVIENDO FRONTEND LOCAL (frontend/build/) ==="
    if [ ! -d "./frontend/build" ]; then
        echo "❌ La carpeta './frontend/build' no existe. Ejecute el paso [1] primero."
    else
        echo "Iniciando servidor local en http://localhost:3000..."
        # Usamos bun x para ejecutar serve (configurado via frontend/build/serve.json para multi-SPA)
        bun x serve ./frontend/build -l 3000
    fi
fi

# PUBLICAR BACKEND
if has_action "2"; then
    echo "=== PUBLICANDO BACKEND ==="
    cd ./cloud
    $GO_PATH run . accion=1
    cd ..
    echo "✅ El deploy backend finalizado!"
fi

# PUBLICAR BACKEND (VPS)
if has_action "3"; then
    echo "=== PUBLICANDO BACKEND (VPS) ==="
    cd ./scripts
    $GO_PATH run . deploy_vps
    cd ..
    echo "✅ El deploy VPS finalizado!"
fi

# PUBLICAR DB BACKUP BINARY
if has_action "4"; then
    echo "=== PUBLICANDO DB-BACKUP ==="
    cd ./db-backup
    GOOS=linux GOARCH=arm64 go build -ldflags '-s -w' .
    aws --profile $AWS_PROFILE s3 cp ./db-backup s3://$AWS_S3/_bin/db-backup.bin
    cd ..
    echo "✅ El deploy del ejecutable finalizó."
fi

# RECREAR TABLAS
if has_action "5" && ! has_action "6"; then
    deploy_tables || exit 1
fi

# DESPLEGAR TABLAS, DATOS INICIALES Y CLOUDFLARE WORKER
if has_action "6"; then
    deploy_tables || exit 1
    echo "=== POBLANDO DATOS INICIALES ==="
    (cd backend && "$GO_PATH" run . fn-init) || exit 1
    echo "=== DESPLEGANDO CLOUDFLARE WORKER ==="
    (cd backend && "$GO_PATH" run . fn-deploy-cloudflare-worker) || exit 1
fi

# DESPLEGAR SOLO CLOUDFLARE WORKER
if has_action "10"; then
    echo "=== DESPLEGANDO CLOUDFLARE WORKER ==="
    (cd backend && "$GO_PATH" run . fn-deploy-cloudflare-worker) || exit 1
fi

# DESPLEGAR INFRAESTRUCTURA
if has_action "9"; then
    echo "=== DESPLEGANDO INFRAESTRUCTURA ==="
    cd ./cloud
    $GO_PATH run . accion=3
    cd ..
    echo "✅ El deploy de infraestructura finalizó!"
fi

# INSPECCIONAR BACKEND
if has_action "7"; then
    echo "=== COMPILANDO BACKEND ==="
    cd ./backend
    $GO_PATH build -v .
    # gsa app (assuming this is a local tool)
    cd ..
fi

# DESPLEGAR WEBPAGE DE UNA EMPRESA
if has_action "11"; then
    COMPANY_ID="$COMPANY_ID_ARGUMENT"
    if [ -z "$COMPANY_ID" ]; then
        echo "Ingrese CompanyID:"
        read -r COMPANY_ID
    fi
    if [[ ! "$COMPANY_ID" =~ ^[1-9][0-9]*$ ]]; then
        echo "❌ CompanyID debe ser un entero positivo."
        exit 1
    fi

    echo "=== DESPLEGANDO WEBPAGE DE COMPANY ID $COMPANY_ID ==="
    (cd backend && "$GO_PATH" run . fn-deploy-company-webpage "$COMPANY_ID") || exit 1
fi

# SINCRONIZAR CATÁLOGO DE IMÁGENES
if has_action "12"; then
    echo "=== SINCRONIZANDO CATÁLOGO DE IMÁGENES ==="
    (cd backend && "$GO_PATH" run . fn-sync-image-assets) || exit 1
fi

if [ "$INTERACTIVE" -eq 1 ]; then
    echo "Finalizado!. Presione cualquier tecla para salir"
    read -r -n 1
else
    echo "Finalizado!"
fi
