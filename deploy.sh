#!/bin/bash
source ~/.bashrc

AWS_PROFILE="ivanjoz"
AWS_S3="gerp-v2-frontend"
FUNCTION_NAME="jobfinder6-p-app"

# Ensure we are in the root directory
cd "$(dirname "$0")"

# Select credentials before showing or executing any deployment action.
DEFAULT_CREDENTIALS_FILE="$PWD/credentials.json"
ALTERNATE_CREDENTIALS_FILE="$PWD/credentials.1.json"
SELECTED_CREDENTIALS_FILE="$DEFAULT_CREDENTIALS_FILE"

if [ -f "$ALTERNATE_CREDENTIALS_FILE" ]; then
    while true; do
        echo "Choose Environment: 0 | 1"
        echo "[0] credentials.json"
        echo "[1] credentials.1.json"
        read -r -n 1 SELECTED_ENVIRONMENT
        echo

        if [ "$SELECTED_ENVIRONMENT" = "0" ]; then
            SELECTED_CREDENTIALS_FILE="$DEFAULT_CREDENTIALS_FILE"
            break
        fi
        if [ "$SELECTED_ENVIRONMENT" = "1" ]; then
            SELECTED_CREDENTIALS_FILE="$ALTERNATE_CREDENTIALS_FILE"
            break
        fi

        echo "❌ Seleccione 0 o 1."
    done
fi

if [ ! -f "$SELECTED_CREDENTIALS_FILE" ]; then
    echo "❌ No se encontró el archivo de credenciales: $SELECTED_CREDENTIALS_FILE"
    exit 1
fi

# Every child process, including all three steps in action 6, uses this exact file.
export GENIX_CREDENTIALS_FILE="$SELECTED_CREDENTIALS_FILE"
echo "✅ Environment seleccionado: $(basename "$GENIX_CREDENTIALS_FILE")"

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
echo "[13] Actualizar Variables de Entorno de las Lambdas (AWS CLI)"
echo "[14] Configurar Variables Frontend en GitHub"
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

# Equivalente a 'cloud accion=2' pero con AWS CLI, sin compilar ni esperar una tecla.
# UpdateFunctionConfiguration reemplaza el entorno completo (no lo fusiona), así que cada
# Lambda recibe aquí TODAS sus variables o las que falten se borran de la función desplegada.
update_lambda_env_vars() {
    local required_tool
    for required_tool in jq zstd aws base64; do
        if ! command -v "$required_tool" > /dev/null 2>&1; then
            echo "❌ Falta la herramienta requerida: $required_tool"
            return 1
        fi
    done

    local app_name aws_profile aws_region
    app_name=$(jq -r '.APP_NAME // ""' "$GENIX_CREDENTIALS_FILE") || return 1
    aws_profile=$(jq -r '.AWS_PROFILE // ""' "$GENIX_CREDENTIALS_FILE") || return 1
    aws_region=$(jq -r '.AWS_REGION // ""' "$GENIX_CREDENTIALS_FILE") || return 1

    if [ -z "$app_name" ] || [ -z "$aws_profile" ] || [ -z "$aws_region" ]; then
        echo "❌ APP_NAME, AWS_PROFILE y AWS_REGION son requeridos en $(basename "$GENIX_CREDENTIALS_FILE")."
        return 1
    fi

    echo "Perfil: $aws_profile | Región: $aws_region | App: $app_name"

    # CONFIG viaja zstd + base64 url-safe (/ → _, + → -, = → ~): es exactamente lo que
    # MakeB64UrlDecode + DecompressZstd deshacen en backend/core/security.go.
    # El '-' del segundo set va escapado o tr lo lee como rango '_' a '~' y corrompe el valor.
    local config_value
    config_value=$(zstd -q -c "$GENIX_CREDENTIALS_FILE" | base64 -w 0 | tr '/+=' '_\-~') || return 1
    if [ -z "$config_value" ]; then
        echo "❌ No se pudo comprimir el archivo de credenciales."
        return 1
    fi

    # APP_CODE lleva guion bajo porque el backend deduce IS_PROD con Contains(APP_CODE, "_prd"),
    # y LAMBDA_RESPONSE_STREAMING debe coincidir con el InvokeMode de template.yml. Ambos valores
    # son los mismos que las constantes de cloud/main.go.
    local backend_env_json
    backend_env_json=$(jq -n --arg config "$config_value" \
        '{Variables: {APP_CODE: "gerp_prd", CONFIG: $config, LAMBDA_RESPONSE_STREAMING: "1"}}') || return 1

    local lambda_name failed=0
    for lambda_name in "${app_name}-backend" "${app_name}-backend_2"; do
        apply_lambda_env "$aws_profile" "$aws_region" "$lambda_name" "$backend_env_json" || failed=1
    done

    # La Lambda de render recibe las credenciales sueltas (no el CONFIG comprimido) para no
    # obligar al handler de Node a descomprimir zstd.
    local frontend_cdn cloudflare_account cloudflare_token cloudflare_bucket renderer_zip_url
    frontend_cdn=$(jq -r '.FRONTEND_CDN // ""' "$GENIX_CREDENTIALS_FILE")
    cloudflare_account=$(jq -r '.CLOUDFLARE_ACCOUNT // ""' "$GENIX_CREDENTIALS_FILE")
    cloudflare_token=$(jq -r '.CLOUDFLARE_TOKEN // ""' "$GENIX_CREDENTIALS_FILE")
    cloudflare_bucket=$(jq -r '.CLOUDFLARE_BUCKET // ""' "$GENIX_CREDENTIALS_FILE")
    renderer_zip_url=$(jq -r '.WEBPAGE_RENDERER_URL // ""' "$GENIX_CREDENTIALS_FILE")
    # Mismos defaults que cloud/main.go y cloud/webpage-renderer.go.
    [ -z "$cloudflare_bucket" ] && cloudflare_bucket="${app_name}-files"
    [ -z "$renderer_zip_url" ] && renderer_zip_url="https://genix-dev.un.pe/webpage-renderer.zip"

    if [ -z "$frontend_cdn" ]; then
        echo "⚠️  FRONTEND_CDN vacío: se omite la Lambda de render."
    else
        local renderer_env_json
        renderer_env_json=$(jq -n \
            --arg zip "$renderer_zip_url" \
            --arg cdn "$frontend_cdn" \
            --arg account "$cloudflare_account" \
            --arg token "$cloudflare_token" \
            --arg bucket "$cloudflare_bucket" \
            '{Variables: {RENDERER_ZIP_URL: $zip, FRONTEND_CDN: $cdn, CLOUDFLARE_ACCOUNT: $account, CLOUDFLARE_TOKEN: $token, CLOUDFLARE_BUCKET: $bucket}}') || return 1
        apply_lambda_env "$aws_profile" "$aws_region" "${app_name}-webpage-renderer" "$renderer_env_json" || failed=1
    fi

    return $failed
}

# Salta las Lambdas inexistentes (el stack puede no tener la _2 ni el renderer) en vez de
# abortar el resto de la actualización.
apply_lambda_env() {
    local aws_profile="$1" aws_region="$2" lambda_name="$3" env_json="$4"

    if ! aws --profile "$aws_profile" --region "$aws_region" lambda get-function-configuration \
        --function-name "$lambda_name" > /dev/null 2>&1; then
        echo "⚠️  Lambda no encontrada, se omite: $lambda_name"
        return 0
    fi

    echo "--- Actualizando variables de $lambda_name ---"
    if ! aws --profile "$aws_profile" --region "$aws_region" lambda update-function-configuration \
        --function-name "$lambda_name" \
        --environment "$env_json" \
        --query 'LastModified' --output text; then
        echo "❌ Error al actualizar las variables de $lambda_name"
        return 1
    fi
}

# CONFIGURAR VARIABLES PÚBLICAS DEL FRONTEND EN GITHUB ACTIONS
if has_action "14"; then
    echo "=== CONFIGURANDO VARIABLES FRONTEND EN GITHUB ==="
    bun run ./scripts/set-github-frontend-vars.ts || exit 1
fi

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
    (cd ./cloud && "$GO_PATH" run . accion=1) || exit 1
    echo "✅ El deploy backend finalizado!"
fi

# PUBLICAR BACKEND (VPS)
if has_action "3"; then
    echo "=== PUBLICANDO BACKEND (VPS) ==="
    (cd ./scripts && "$GO_PATH" run . deploy_vps) || exit 1
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

# DESPLEGAR INFRAESTRUCTURA
# Va antes que las tablas a propósito: CloudFormation es el dueño de la tabla DynamoDB, y
# fn-init (acción 6) ahora falla si no existe. Así "6 9" en una sola invocación funciona.
if has_action "9"; then
    echo "=== DESPLEGANDO INFRAESTRUCTURA ==="
    (cd ./cloud && "$GO_PATH" run . accion=3) || exit 1
    echo "✅ El deploy de infraestructura finalizó!"
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

# ACTUALIZAR VARIABLES DE ENTORNO DE LAS LAMBDAS
if has_action "13"; then
    echo "=== ACTUALIZANDO VARIABLES DE ENTORNO DE LAS LAMBDAS ==="
    update_lambda_env_vars || exit 1
    echo "✅ Variables de entorno actualizadas!"
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
