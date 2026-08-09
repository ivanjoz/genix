# PLAN — Migración de `credentials.json` a `config.toml`

> **Especificación de implementación.** Escrita para que otro agente la ejecute sin volver a
> investigar el repositorio. Cada sección lleva rutas y números de línea reales, y el código
> nuevo va literal. Los números de línea son del estado del repo al escribir el plan: verificar
> con `rg` antes de editar, no aplicar por offset.

---

## 0. Decisiones cerradas

| # | Decisión | Valor |
|---|---|---|
| 1 | Estructura del TOML | **Secciones agrupadas** (`[db]`, `[aws]`, …) |
| 2 | `EnvStruct` de Go | **Se queda plana.** Se parsea a un `fileConfig` anidado y se mapea. Los 213 call sites de `core.Env.X` **no se tocan** |
| 3 | Escritura desde Python | **Edición quirúrgica de línea**, sin dependencias, conservando comentarios |
| 4 | Variable de entorno | `GENIX_CREDENTIALS_FILE` → **`GENIX_CONFIG_FILE`** |
| 5 | Python en los VPS | Última versión ⇒ **`tomllib` (stdlib, 3.11+) es seguro**. Sin parser vendorizado |
| 6 | Segundo entorno | `credentials.1.json` → **`config.1.toml`**, se migra también |

**Sin compatibilidad hacia atrás** (`AGENTS.md`): no queda ningún camino que lea `credentials.json`.
Nada de "si existe el .json úsalo". Se migra y se borra.

---

## 1. Contexto imprescindible antes de tocar nada

### 1.1 El archivo tiene tres caminos de consumo, no uno

1. **Lectura directa del archivo** — backend local, deployer, scripts de Python/Bun/shell.
2. **Comprimido dentro de una variable de entorno de Lambda.** `scripts/deployer/lambda_env.go:49-76`
   lee los **bytes crudos** del archivo, los pasa por zstd + base64 url-safe y los publica como
   `CONFIG`. `backend/core/security.go:202-210` los revierte y los parsea.
   **Consecuencia dura: el payload de `CONFIG` pasa a ser TOML.** El backend queda con un solo
   parser. Si se cambia sólo uno de los dos lados, toda invocación de Lambda falla al arrancar.
3. **Regeneración a `.env` para el frontend** — `frontend/scripts/setup-env.js` produce
   `PUBLIC_*`; el frontend nunca lee el archivo de config.

### 1.2 La ganancia real

El JSON simula comentarios con claves falsas que ningún parser interpreta:

```json
"BACKEND_PROVIDER:OPTIONS": "aws|cloudflare|none",
"SSE_BRIDGE_URL:DOC": "Relay SSE (sse_bridge/) para el agente cuando el backend corre en Lambda...",
"GENIXSEARCH_URL:DOC": "Endpoint del servicio de busqueda (host:puerto)...",
```

En TOML son comentarios `#` reales. **Por eso la decisión 3 es una restricción y no un detalle:**
un escritor que reserialice el archivo borraría los comentarios en cada `configure_db.py` y
anularía la migración a las dos semanas.

### 1.3 Regla de TOML que rompe el archivo si se ignora

Tras un encabezado `[[tabla]]`, **toda clave suelta posterior pertenece a esa tabla**.
`[[endpoints]]` y `[[servers]]` van **obligatoriamente al final**. Esto aplica al archivo a mano,
al generador de la migración y al escritor de Python (ver §5.1, que sólo añade secciones `[x]`
nunca claves sueltas al final).

---

## 2. Contrato de claves — tabla canónica

Fuente de verdad de toda la migración. `—` = no existe en ese lado.

### 2.1 Nivel raíz

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Otros consumidores |
|---|---|---|---|
| `APP_NAME` | `app_name` | `APP_NAME` | cloud, deployer, p2p |
| `IS_LOCAL` | `is_local` | `IS_LOCAL` | — |
| `ENVIROMENT` | `environment` | `ENVIROMENT` *(se corrige el typo sólo en el TOML)* | — |
| `ADMIN_PASSWORD` | `admin_password` | `ADMIN_PASSWORD` | — |
| `SECRET_PHRASE` | `secret_phrase` | `SECRET_PHRASE` | sse_bridge |
| `ADMIN_EMAIL` | `admin_email` | — | **ninguno** (ver §2.9) |
| `GITHUB_ACCOUNT` | `github_account` | — | `set-github-frontend-vars.ts` |

### 2.2 `[providers]`

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Valores |
|---|---|---|---|
| `BACKEND_PROVIDER` | `providers.backend` | `BACKEND_PROVIDER` | `aws` \| `cloudflare` \| `none` |
| `CDN_PROVIDER` | `providers.cdn` | `CDN_PROVIDER` | `aws` \| `cloudflare` |
| `MODEL_PROVIDER` | `providers.model` | `MODEL_PROVIDER` | `meta` \| `openrouter` \| vacío |

### 2.3 `[db]`

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Tipo |
|---|---|---|---|
| `DB_HOST` | `db.host` | `DB_HOST` | string |
| `DB_PORT` | `db.port` | `DB_PORT` | int32 |
| `DB_NAME` | `db.name` | `DB_NAME` | string |
| `DB_USER` | `db.user` | `DB_USER` | string |
| `DB_PASSWORD` | `db.password` | `DB_PASSWORD` | string |
| `DB_DISABLE_SSL` | `db.disable_ssl` | `DB_DISABLE_SSL` | bool |
| `MAX_CLUSTERING_KEY` | `db.max_clustering_key` | `MAX_CLUSTERING_KEY` | int32 |

### 2.4 `[server]`

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Consumidor |
|---|---|---|---|
| `SERVER_PORT` | `server.port` | `SERVER_PORT` | backend, `configure_server.py` |
| `NGINX_DOMAIN` | `server.nginx_domain` | — | `configure_server.py` |
| `NGINX_PROCESS` | `server.nginx_process` | — | `configure_server.py` |

### 2.5 `[aws]` / `[cloudflare]` / `[frontend]`

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Otros |
|---|---|---|---|
| `AWS_REGION` | `aws.region` | `AWS_REGION` | cloud, p2p, db-backup, ai |
| `AWS_PROFILE` | `aws.profile` | `AWS_PROFILE` | cloud, p2p, db-backup, ai |
| `DEPLOYMENT_BUCKET` | `aws.deployment_bucket` | — | cloud |
| `LAMBDA_IAM_ROLE` | `aws.lambda_iam_role` | — | cloud |
| `LAMBDA_URL` | `aws.lambda_url` | — | `setup-env.js`, `set-github-*`, cloudformation |
| `S3_BUCKET` | `aws.s3_bucket` | `S3_BUCKET` | db-backup |
| `FRONTEND_BUCKET` | `aws.frontend_bucket` | — | cloud (vacío ⇒ `<app_name>-frontend`) |
| `SAGEMAKER_IAM_ROLE` | `aws.sagemaker_iam_role` | — | `ai/*.py` |
| `SAGEMAKER_S3_OUTPUT` | `aws.sagemaker_s3_output` | — | `ai/*.py` |
| `HUGGING_FACE_TOKEN` | `aws.hugging_face_token` | — | `ai/*.py` |
| `CLOUDFLARE_ACCOUNT` | `cloudflare.account` | `CLOUDFLARE_ACCOUNT` | cloud, deployer |
| `CLOUDFLARE_TOKEN` | `cloudflare.token` | `CLOUDFLARE_TOKEN` | cloud, deployer |
| `CLOUDFLARE_BUCKET` | `cloudflare.bucket` | `CLOUDFLARE_BUCKET` | cloud, deployer |
| `CLOUDFLARE_DATABASE_ID` | `cloudflare.database_id` | `CLOUDFLARE_DATABASE_ID` | — |
| `FRONTEND_CDN` | `frontend.cdn_url` | `FRONTEND_CDN` | cloud, deployer, setup-env |
| `APP_URL` | `frontend.app_url` | — | backend (enlace del correo de registro) |
| `ZONE_NAME` | `frontend.zone_name` | `ZONE_NAME` | setup-env, set-github-* |
| `WEBPAGE_RENDERER_URL` | `frontend.webpage_renderer_url` | `WEBPAGE_RENDERER_URL` | cloud, deployer |

### 2.6 `[smtp]` / `[agent]` / `[search]` / `[sse_bridge]`

| Clave JSON | Ruta TOML | Campo `EnvStruct` | Otros |
|---|---|---|---|
| `SMTP_HOST` | `smtp.host` | `SMTP_HOST` | — |
| `SMTP_PORT` | `smtp.port` | `SMTP_PORT` | — |
| `SMTP_EMAIL` | `smtp.email` | `SMTP_EMAIL` | — |
| `SMTP_USER` | `smtp.user` | `SMTP_USER` | — |
| `SMTP_PASSWORD` | `smtp.password` | `SMTP_PASSWORD` | — |
| `DEFAULT_MODEL` | `agent.default_model` | `DEFAULT_MODEL` | — |
| `META_KEY` | `agent.meta_key` | `META_KEY` | — |
| `OPENROUTER_KEY` | `agent.openrouter_key` | `OPENROUTER_KEY` | — |
| `GENIXSEARCH_URL` | `search.url` | `GENIXSEARCH_URL` | **escrito** por `configure_db.py` |
| `GENIXSEARCH_PASSWORD` | `search.password` | `GENIXSEARCH_PASSWORD` | **escrito** por `configure_db.py` |
| `SSE_BRIDGE_URL` | `sse_bridge.url` | `SSE_BRIDGE_URL` | setup-env, set-github-*, configure_sse_bridge |
| `SSE_BRIDGE_APIKEY` | `sse_bridge.apikey` | — | sse_bridge, **escrito** por configure_sse_bridge |
| `SSE_BRIDGE_PORT` | `sse_bridge.port` | — | configure_sse_bridge |

### 2.7 `[signaling]` (módulo `p2p/`) y `[logs]`

| Clave JSON | Ruta TOML | Campo |
|---|---|---|
| `SIGNALING_SOCKET` | `signaling.socket` | `p2p Config.SignalingSocket` |
| `SIGNALING_ENDPOINT` | `signaling.endpoint` | `p2p Config.SignalingEndpoint` |
| `SIGNALING_API_KEY` | `signaling.api_key` | `p2p Config.ApiKey` |
| `SIGNALING_APP_NAME` | `signaling.app_name` | `p2p Config.SignalingAppName` |
| `SIGNALING_STACK_NAME` | `signaling.stack_name` | `p2p Config.StackName` |
| `LAMBDA_FUNCTION_NAME` | `signaling.lambda_function_name` | `p2p Config.LambdaFunctionNameActual` |
| `AWS_ACCOUNT` | `signaling.aws_account` | `p2p Config.AWSAccount` |
| `LOGS_FULL` | `logs.full` | `LOGS_FULL` |
| `LOGS_DEBUG` | `logs.debug` | `LOGS_DEBUG` |
| `LOGS_ONLY_SAVE` | `logs.only_save` | `LOGS_ONLY_SAVE` |

### 2.8 Tablas de array — **siempre al final del archivo**

| Clave JSON | Ruta TOML | Campos |
|---|---|---|
| `ENPOINTS` *(typo)* | `[[endpoints]]` | `name`, `route` |
| `SERVERS` | `[[servers]]` | `host`, `user`, `bin`, `arch` |

`ENPOINTS` sólo lo leen `frontend/scripts/setup-env.js:17` y `scripts/set-github-frontend-vars.ts:97`.
La variable que consume el frontend ya se llama `PUBLIC_ENDPOINTS`, así que corregir el typo **no
toca `frontend/core/env.ts`**.

### 2.9 Claves que NO van al TOML

- **Derivadas o de entorno en runtime**, nunca del archivo: `APP_CODE`, `IS_PROD`, `IS_SERVERLESS`,
  `LAMBDA_RESPONSE_STREAMING`, `LAMBDA_NAME`, `DYNAMO_TABLE`, `API_ROUTE`, `TMP_DIR`, `USUARIO_ID`,
  `REQ_IP`, `REQ_ID`, `REQ_PARAMS`, `REQ_USER_AGENT`, `REQ_PATH`, `REQ_LAMBDA_ID`.
  **No añadirlas a `fileConfig`**: siguen calculándose igual en `PopulateVariables()`.
- **`db-backup/`**: `IS_PRODUCTION`, `SCYLLA_DATA`, `KEYSPACE`, `BACKUP_MAIN_DIR` se fijan en
  `db-backup/main.go:96-102`, no salen del archivo. De él sólo toma `aws.profile`, `aws.region`,
  `aws.s3_bucket`.
- **`ADMIN_EMAIL`**: presente en el JSON pero **sin ningún consumidor en el repo**. Se conserva
  como `admin_email` para no perder el dato; no se le añade campo en Go.
- `*:OPTIONS` y `*:DOC`: desaparecen, se convierten en comentarios `#`.

---

## 3. `config.example.toml` — plantilla de referencia

Este es el layout exacto que debe generar la migración y que se versiona como ejemplo
(con valores de relleno). Los comentarios son parte del entregable.

```toml
# Configuración de Genix. Copie este archivo a config.toml y complete los valores.
# config.toml está en .gitignore: nunca se versiona.

app_name       = "genix"
is_local       = true
environment    = ""
admin_email    = "admin@example.com"
admin_password = "admin_password"
secret_phrase  = "cambie_esta_frase"
github_account = "mi-cuenta"

# ─── Proveedores ────────────────────────────────────────────────────────────
[providers]
backend = "none"        # aws | cloudflare | none  — espejo de datos: DynamoDB, D1 o ninguno
cdn     = "aws"         # aws | cloudflare         — object storage y origen público de assets
model   = "openrouter"  # meta | openrouter        — LLM del agente; sólo se exige su key

# ─── Base de datos (ScyllaDB / Cassandra) ───────────────────────────────────
[db]
host        = "111.111.111.111"
port        = 9042
name        = "genix"
user        = "cassandra"
password    = "my_strong_password"
disable_ssl = false
# max_clustering_key_restrictions_per_query del nodo. 0 usa el default del ORM (100).
max_clustering_key = 100

# ─── Servidor propio (systemd + Nginx) ──────────────────────────────────────
[server]
# port debe coincidir con la mitad de puerto de nginx_process. 0 usa 3589.
port          = 14010
nginx_domain  = "api.example.com"
nginx_process = "127.0.0.1:14010"

# ─── AWS ────────────────────────────────────────────────────────────────────
[aws]
region            = "us-east-1"
profile           = "default"
deployment_bucket = "deploys-bucket"
lambda_iam_role   = "arn:aws:iam::000000000000:role/lambda-role"
lambda_url        = ""
s3_bucket         = ""
frontend_bucket   = ""   # vacío = "<app_name>-frontend"
# SageMaker — sólo los usan los scripts de ai/
sagemaker_iam_role  = ""
sagemaker_s3_output = ""
hugging_face_token  = ""

# ─── Cloudflare ─────────────────────────────────────────────────────────────
[cloudflare]
account     = ""
token       = ""
bucket      = ""   # vacío = "<app_name>-files"
database_id = ""

# ─── Frontend / CDN público ─────────────────────────────────────────────────
[frontend]
cdn_url   = ""
# Origen público de la app web, sin barra final. Lo usa el correo de registro.
app_url   = ""
zone_name = ""
# Artefacto webpage-renderer.zip publicado por CI. Vacío = el default de
# core.DefaultWebpageRendererURL, que debe seguir igual al de cloud/webpage-renderer.go.
webpage_renderer_url = ""

# ─── SMTP ───────────────────────────────────────────────────────────────────
[smtp]
host     = "email-smtp.us-east-1.amazonaws.com"
port     = 587
email    = ""
user     = ""
password = ""

# ─── Agente / LLM ───────────────────────────────────────────────────────────
[agent]
# default_model debe existir en el proveedor elegido en providers.model.
# Vacío = el default de compilación de backend/agent/llm.
default_model  = ""
meta_key       = ""
openrouter_key = ""

# ─── GenixSearch ────────────────────────────────────────────────────────────
# Los escribe 'configure_db.py search' con la IP alcanzable del host.
# url admite "host:puerto" o "esquema://host:puerto" (el esquema se ignora).
# Vacío = 127.0.0.1:14446.
[search]
url      = ""
password = ""

# ─── SSE Bridge ─────────────────────────────────────────────────────────────
# Relay que sostiene el stream del agente cuando el backend corre en Lambda, que no
# puede mantener una conexión abierta. url igual a aws.lambda_url (o vacía) = sin
# bridge: el backend sirve su propio /agent/stream.
[sse_bridge]
url    = ""
apikey = ""
port   = 14012

# ─── Signaling (p2p/) ───────────────────────────────────────────────────────
[signaling]
socket               = ""
endpoint             = ""
api_key              = ""
app_name             = ""   # vacío = "<app_name>-signaling"
stack_name           = ""   # vacío = "<app_name>-signaling"
lambda_function_name = ""
aws_account          = ""

# ─── Logs ───────────────────────────────────────────────────────────────────
[logs]
full      = false
debug     = false
only_save = false

# ════════════════════════════════════════════════════════════════════════════
# TABLAS DE ARRAY — SIEMPRE AL FINAL DEL ARCHIVO.
# En TOML, tras un [[header]] toda clave suelta pertenece a esa tabla: añadir
# cualquier clave simple debajo de este punto la asigna al último [[...]].
# ════════════════════════════════════════════════════════════════════════════

# Endpoints ofrecidos en el selector de servidor del login.
[[endpoints]]
name  = "Principal"
route = "https://mi-backend.example.com/"

# Destinos del deploy a VPS (scripts/deploy_vps.go).
[[servers]]
host = "127.0.0.1"
user = "ubuntu"
bin  = "/usr/local/bin/genix/genix_app"
arch = "arm64"
```

---

## 4. Go — 6 módulos

### 4.0 Dependencia

```bash
go get github.com/pelletier/go-toml/v2   # en cada módulo
```

Elegida porque `toml.Unmarshal(data, &v) error` tiene **firma idéntica** a `json.Unmarshal`: en la
mayoría de sitios el cambio es una palabra y el import. Módulos afectados: `backend/`, `cloud/`,
`scripts/`, `p2p/`, `db-backup/`, `sse_bridge/`.

> Los campos se enlazan por tag `toml:"..."`. **Poner el tag siempre, explícito**, aunque el
> emparejamiento por defecto acertaría: una clave mal escrita queda en su cero-valor y falla en
> runtime, no al compilar.

### 4.1 `backend/core/security.go` — el cambio central

Es el archivo que define el contrato; hacer este primero.

**(a) Añadir `fileConfig` justo después de `EnvStruct` (tras la línea 144).** Refleja las secciones
de §2. Existe sólo para parsear:

```go
// fileConfig refleja la forma por secciones de config.toml. Sólo existe para el parseo:
// el resto del backend sigue leyendo core.Env, que es plano, y así los 213 puntos de uso
// de core.Env.X no cambian. Todo campo nuevo del archivo se añade aquí Y en el bloque de
// asignación de PopulateVariables, o queda silenciosamente en su cero-valor.
type fileConfig struct {
	AppName       string `toml:"app_name"`
	IsLocal       bool   `toml:"is_local"`
	Environment   string `toml:"environment"`
	AdminPassword string `toml:"admin_password"`
	SecretPhrase  string `toml:"secret_phrase"`

	Providers struct {
		Backend string `toml:"backend"`
		CDN     string `toml:"cdn"`
		Model   string `toml:"model"`
	} `toml:"providers"`

	DB struct {
		Host             string `toml:"host"`
		Port             int32  `toml:"port"`
		Name             string `toml:"name"`
		User             string `toml:"user"`
		Password         string `toml:"password"`
		DisableSSL       bool   `toml:"disable_ssl"`
		MaxClusteringKey int32  `toml:"max_clustering_key"`
	} `toml:"db"`

	Server struct {
		Port int32 `toml:"port"`
	} `toml:"server"`

	AWS struct {
		Region   string `toml:"region"`
		Profile  string `toml:"profile"`
		S3Bucket string `toml:"s3_bucket"`
	} `toml:"aws"`

	Cloudflare struct {
		Account    string `toml:"account"`
		Token      string `toml:"token"`
		Bucket     string `toml:"bucket"`
		DatabaseID string `toml:"database_id"`
	} `toml:"cloudflare"`

	Frontend struct {
		CDNURL             string `toml:"cdn_url"`
		AppURL             string `toml:"app_url"`
		ZoneName           string `toml:"zone_name"`
		WebpageRendererURL string `toml:"webpage_renderer_url"`
	} `toml:"frontend"`

	SMTP struct {
		Host     string `toml:"host"`
		Port     int32  `toml:"port"`
		Email    string `toml:"email"`
		User     string `toml:"user"`
		Password string `toml:"password"`
	} `toml:"smtp"`

	Agent struct {
		DefaultModel  string `toml:"default_model"`
		MetaKey       string `toml:"meta_key"`
		OpenRouterKey string `toml:"openrouter_key"`
	} `toml:"agent"`

	Search struct {
		URL      string `toml:"url"`
		Password string `toml:"password"`
	} `toml:"search"`

	SSEBridge struct {
		URL string `toml:"url"`
	} `toml:"sse_bridge"`

	Logs struct {
		Full     bool `toml:"full"`
		Debug    bool `toml:"debug"`
		OnlySave bool `toml:"only_save"`
	} `toml:"logs"`
}

// applyToEnv vuelca el archivo por secciones sobre la Env plana. Es el único punto donde
// las dos formas se tocan.
func (file *fileConfig) applyToEnv(env *EnvStruct) {
	env.APP_NAME = file.AppName
	env.IS_LOCAL = file.IsLocal
	env.ENVIROMENT = file.Environment
	env.ADMIN_PASSWORD = file.AdminPassword
	env.SECRET_PHRASE = file.SecretPhrase

	env.BACKEND_PROVIDER = file.Providers.Backend
	env.CDN_PROVIDER = file.Providers.CDN
	env.MODEL_PROVIDER = file.Providers.Model

	env.DB_HOST = file.DB.Host
	env.DB_PORT = file.DB.Port
	env.DB_NAME = file.DB.Name
	env.DB_USER = file.DB.User
	env.DB_PASSWORD = file.DB.Password
	env.DB_DISABLE_SSL = file.DB.DisableSSL
	env.MAX_CLUSTERING_KEY = file.DB.MaxClusteringKey

	env.SERVER_PORT = file.Server.Port

	env.AWS_REGION = file.AWS.Region
	env.AWS_PROFILE = file.AWS.Profile
	env.S3_BUCKET = file.AWS.S3Bucket

	env.CLOUDFLARE_ACCOUNT = file.Cloudflare.Account
	env.CLOUDFLARE_TOKEN = file.Cloudflare.Token
	env.CLOUDFLARE_BUCKET = file.Cloudflare.Bucket
	env.CLOUDFLARE_DATABASE_ID = file.Cloudflare.DatabaseID

	env.FRONTEND_CDN = file.Frontend.CDNURL
	env.ZONE_NAME = file.Frontend.ZoneName
	env.WEBPAGE_RENDERER_URL = file.Frontend.WebpageRendererURL

	env.SMTP_HOST = file.SMTP.Host
	env.SMTP_PORT = file.SMTP.Port
	env.SMTP_EMAIL = file.SMTP.Email
	env.SMTP_USER = file.SMTP.User
	env.SMTP_PASSWORD = file.SMTP.Password

	env.DEFAULT_MODEL = file.Agent.DefaultModel
	env.META_KEY = file.Agent.MetaKey
	env.OPENROUTER_KEY = file.Agent.OpenRouterKey

	env.GENIXSEARCH_URL = file.Search.URL
	env.GENIXSEARCH_PASSWORD = file.Search.Password

	env.SSE_BRIDGE_URL = file.SSEBridge.URL

	env.LOGS_FULL = file.Logs.Full
	env.LOGS_DEBUG = file.Logs.Debug
	env.LOGS_ONLY_SAVE = file.Logs.OnlySave
}
```

**(b) `PopulateVariables()` (líneas 154-257).** Cambios puntuales, el resto de la función queda igual:

- L160: `os.Getenv("GENIX_CREDENTIALS_FILE")` → `os.Getenv("GENIX_CONFIG_FILE")`; renombrar la
  variable local a `configuredConfigPath`.
- L172: `[]string{parentPath + "/credentials.json", wd + "/credentials.json"}` →
  `[]string{parentPath + "/config.toml", wd + "/config.toml"}`.
- L192: mensaje → `"Seteando config.toml desde:"`.
- L199: panic → `"Archivo config.toml no encontrado. Configure GENIX_CONFIG_FILE o suba el archivo al directorio esperado."`
- L212-216: sustituir el `json.Unmarshal(variablesBytes, &Env)` por:

```go
	// Env se aloja aquí porque hasta ahora la creaba el propio json.Unmarshal sobre el puntero.
	Env = &EnvStruct{}
	parsedFile := fileConfig{}
	if err := toml.Unmarshal(variablesBytes, &parsedFile); err != nil {
		fmt.Println("Error parsing config.toml:", err)
		return
	}
	parsedFile.applyToEnv(Env)
```

> **Trampa a no pasar por alto:** `var Env *EnvStruct` es un puntero nil y hoy lo inicializa
> `json.Unmarshal(..., &Env)` (unmarshal sobre `**EnvStruct`). Con el nuevo flujo hay que
> asignarlo explícitamente o el primer acceso hace panic por nil.

- L218: `"Credenciales .json Parseadas:: "` → `"config.toml parseado:: "`.
- L244-248: el comentario sobre `LAMBDA_RESPONSE_STREAMING` cita `credentials.json`; actualizar.
- Eliminar el import `encoding/json` si ya no se usa en el archivo (verificar: `GetAwsConfig` y
  el resto del archivo pueden seguir usándolo).

**(c)** Comentarios de `EnvStruct` (L126, L138) que nombran `credentials.json` / claves viejas.

### 4.2 `cloud/` (módulo aparte)

**`cloud/main.go`**

- L16-40: `DeployParams` pasa a anidado con tags `toml:`. Los campos que el resto de `cloud/`
  consume por nombre plano (`params.APP_NAME`, `params.AWS_PROFILE`, …) se usan en varios archivos
  del módulo — **aquí sí conviene anidar de verdad**, la estructura es pequeña; ajustar los usos
  en `cloudformation.go`, `webpage-renderer.go` y demás (`rg 'params\.' cloud/`).
  `S3_COMPILED_PATH` y `FRONTEND_BUCKET` son derivados, no llevan tag de archivo.
- L61-68: `GetCredentialsPath()` → `GetConfigPath()`, env var `GENIX_CONFIG_FILE`,
  fallback `GetBaseWD() + "/config.toml"`. Actualizar los 3 puntos de llamada.
- L106-118 y L250-259: `json.Unmarshal` → `toml.Unmarshal`, mensajes.
- L120-147: las validaciones de `BACKEND_PROVIDER` / `CDN_PROVIDER` / bucket pasan a los campos
  anidados; **la lógica no cambia**.

**`cloud/cloudformation.go` L243-282** — reescribe `LAMBDA_URL` por regex de texto para no perder
formato ni orden. Misma técnica, nueva forma:

```go
// Reemplazo de texto y no re-serialización: el archivo se mantiene a mano y sus comentarios
// son la razón de ser del formato TOML.
// Ancla en la clave de la sección [aws]; en el archivo sólo existe una lambda_url.
var lambdaUrlInTomlPattern = regexp.MustCompile(`(?m)^(\s*lambda_url\s*=\s*")([^"]*)(")`)
```

Renombrar `SyncLambdaUrlInCredentials` → `SyncLambdaUrlInConfig` y sus mensajes (L219, L245-281).
**Verificar** que ninguna otra clave del archivo se llame `lambda_url`; hoy es única.

### 4.3 `scripts/` (módulo del deployer)

**`scripts/deployer/main.go`**

- L23-24: `credentials.json` → `config.toml`, `credentials.1.json` → `config.1.toml`.
- L39 y L67: `GENIX_CREDENTIALS_FILE` → `GENIX_CONFIG_FILE`.
- L151-162 `readBackendProvider`: parsea `providers.backend`:

```go
	var parsed struct {
		Providers struct {
			Backend string `toml:"backend"`
		} `toml:"providers"`
	}
	if err := toml.Unmarshal(content, &parsed); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Providers.Backend))
```

- Renombrar `credentialsFile` → `configFile` en `main`, en `deployContext` (`actions.go:31`) y en
  todos sus usos; mensaje de L58 → "archivo de configuración".

**`scripts/deployer/lambda_env.go`** — el más delicado, casa con §4.1.

- L30-39: `lambdaCredentials` → anidado con tags `toml:` (`app_name`, `aws.profile`, `aws.region`,
  `frontend.cdn_url`, `cloudflare.account`, `cloudflare.token`, `cloudflare.bucket`,
  `frontend.webpage_renderer_url`).
- L55: `json.Unmarshal` → `toml.Unmarshal`.
- L67: `compressToUrlSafeBase64(credentialsContent)` — **no cambia**, sigue comprimiendo los bytes
  crudos del archivo, que ahora son TOML. Actualizar el comentario de L26-28 para decir que el
  payload es TOML y que `backend/core/security.go` lo parsea con el mismo parser.
- L149 `json.Marshal(environment)`: **se queda en JSON.** Es el payload del `--environment` de la
  AWS CLI, no tiene nada que ver con el archivo de config. No tocar.

**`scripts/deploy_vps.go`** L28-54: `Credentials{ Servers []ServerCredentials }` → tag
`toml:"servers"` con `host`/`user`/`bin`/`arch` en minúscula; env var, fallback `../config.toml`,
`json.Unmarshal` → `toml.Unmarshal`, mensaje de L54 (`SERVERS` → `servers`).

**`scripts/deployer/tui.go`** L49, L219, L229: renombrar parámetros `credentialsPath`/`credentialsFile`
y el comentario "credentials.json es el entorno por defecto" → `config.toml`.

### 4.4 `sse_bridge/config.go`

- L31-34: `credentialsSubset` → `configSubset` anidado. **Conservar la semántica de los dos
  nombres**: `sse_bridge.apikey` si existe, si no `secret_phrase` (L50-53). Es lo que permite que
  un host del bridge lleve un archivo mínimo y una máquina de desarrollo el archivo completo.

```go
type configSubset struct {
	SecretPhrase string `toml:"secret_phrase"`
	SSEBridge    struct {
		ApiKey string `toml:"apikey"`
	} `toml:"sse_bridge"`
}
```

- L75-95 `readCredentialsFile` → `readConfigFile`: env `GENIX_CONFIG_FILE`, candidatos
  `"../config.toml", "config.toml"`, `toml.Unmarshal`.
- L67: mensaje de error → `"…agréguelo como apikey en la sección [sse_bridge] de config.toml (GENIX_CONFIG_FILE) o expórtelo como variable de entorno"`.
- `SSE_BRIDGE_APIKEY` y `SSE_BRIDGE_PORT` como **variables de entorno** (L55-64) siguen igual: son
  el override de systemd, no claves del archivo.

### 4.5 `p2p/config/config.go` — aquí se borra código

TOML fija las claves, así que toda la capa de normalización de nombres sobra:

- **Borrar** `rawConfig` (L28), `normalizeKey` (L57-59), `getValueFromRaw` (L62-73).
- **Borrar** L111-168 completo (los 10 bloques `getValueFromRaw`) y sustituir por un
  `toml.Unmarshal` directo sobre `Config` con tags `toml:` según §2.7 (todo bajo `[signaling]`
  salvo `app_name` y `aws.profile`/`aws.region`).
- L75-97 `configPath()`: busca `credentials.json` hacia arriba → `config.toml`.
- L100-179 `Load()`: mensajes; la validación de `APP_NAME` requerido y el campo derivado
  `LambdaFunctionName` (L176) se conservan.
- L181-210 `LoadWithEnv()`: **sin cambios**, los overrides son variables de entorno.
- Nota: `Config` conserva su forma plana con nombres Go (`AWSProfile`, `SignalingSocket`), sólo
  cambian los tags. Sus consumidores (`p2p/deploy/`, `p2p/homelab_server/`) no se tocan.

### 4.6 `db-backup/main.go` L58-102

- L70/L72: `credentials.json` → `config.toml`.
- L90: `json.Unmarshal(variablesBytes, &Env)` → `toml.Unmarshal` sobre un struct anidado local con
  `aws.profile`, `aws.region`, `aws.s3_bucket`, volcado a `Env` (mismo patrón que §4.1, pero con
  3 campos).
- L86/L92: mensajes.
- **No tocar** L96-102: `KEYSPACE`, `IS_PRODUCTION`, `SCYLLA_DATA` y `BACKUP_MAIN_DIR` se fijan en
  código y no vienen del archivo.

### 4.7 Sólo texto (strings de error y comentarios)

Ningún cambio de lógica; actualizar el nombre del archivo y de la clave citada:

`backend/exec/init.go` (L23, L39, L249) · `backend/main.go` (L236, L250) ·
`backend/cloud/orm-core.go` (L46, L59) · `backend/cloud/s3.go` (L54) ·
`backend/cloud/orm-sqlite.go` (L48) · `backend/cloud/webpage_renderer.go` (L125) ·
`backend/exec/cloudflare_assets.go` (L142, L144) · `backend/agent/bridge.go` (L66) ·
`backend/agent/llm/client.go` (L4, L63, L85, L244, L257, L266) ·
`backend/agent/llm/models.go` (L15, L17) · `backend/genix-orm/scylla/text_search/driver.go` (L33, L47)

Ejemplo de la forma esperada — `backend/cloud/orm-core.go:46`:

```go
return nil, errors.New("providers.backend en config.toml no está definido o es inválido (debe ser 'aws', 'cloudflare' o 'none')")
```

`backend/agent/llm/client.go:63` merece atención: el campo `KeyName` guarda **el nombre de la clave
de credenciales** por proveedor (`META_KEY` / `OPENROUTER_KEY`) y se interpola en el error de L266.
Actualizar sus valores a `agent.meta_key` / `agent.openrouter_key` para que el mensaje siga
señalando algo que existe en el archivo.

---

## 5. Python — 7 scripts

Lectura: `tomllib` (stdlib, binario, `tomllib.load(open(path,'rb'))`). Decisión 5 confirma 3.11+ en
los VPS. Ojo: **`tomllib` abre en binario**, `json.load` abría en texto.

### 5.1 Escritor quirúrgico — helper nuevo compartido

Crear `scripts/toml_config.py`. Lo importan `configure_db.py`, `configure_server.py` y
`configure_sse_bridge.py`.

```python
"""Lectura y escritura puntual de config.toml sin dependencias externas.

El escritor edita líneas en su sitio en vez de reserializar el archivo: los comentarios
son la razón de ser del formato TOML aquí, y un round-trip con tomllib los borraría en
cada ejecución de configure_db.py.
"""
import re
import tomllib
from pathlib import Path


def read_config(config_path):
    """Devuelve el archivo parseado como dict anidado. {} si no existe."""
    config_file = Path(config_path)
    if not config_file.exists():
        return {}
    with open(config_file, "rb") as config_handle:
        return tomllib.load(config_handle)


def get_config_value(config_data, dotted_key, default=None):
    """Lee 'seccion.clave' de un dict anidado. Evita .get().get() encadenados."""
    current_value = config_data
    for key_part in dotted_key.split("."):
        if not isinstance(current_value, dict) or key_part not in current_value:
            return default
        current_value = current_value[key_part]
    return current_value


def format_toml_value(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    escaped_value = str(value).replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped_value}"'


def set_config_values(config_path, value_updates):
    """Escribe {'seccion.clave': valor} en config.toml conservando el resto del archivo.

    Reemplaza la clave dentro de su sección si ya está; si la sección existe pero la clave
    no, la inserta al final de esa sección; si la sección no existe, añade un bloque
    [seccion] al final del archivo.

    Nunca añade claves sueltas al final: en TOML quedarían dentro del último [[endpoints]]
    o [[servers]], que por diseño van al final del archivo.
    """
    config_file = Path(config_path)
    file_lines = config_file.read_text(encoding="utf-8").splitlines()

    for dotted_key, new_value in value_updates.items():
        section_name, _, key_name = dotted_key.rpartition(".")
        formatted_value = format_toml_value(new_value)
        new_line = f"{key_name} = {formatted_value}"

        section_header_pattern = re.compile(r"^\s*\[([^\[\]]+)\]\s*$")
        key_pattern = re.compile(rf"^\s*{re.escape(key_name)}\s*=")

        current_section = ""
        key_line_index = None
        section_last_line_index = None

        for line_index, line_text in enumerate(file_lines):
            header_match = section_header_pattern.match(line_text)
            if header_match:
                current_section = header_match.group(1).strip()
                continue
            if line_text.strip().startswith("[["):
                current_section = "\x00"  # tabla de array: nunca es destino de escritura
                continue
            if current_section != section_name:
                continue
            # Última línea con contenido de la sección: ahí se inserta si falta la clave.
            if line_text.strip():
                section_last_line_index = line_index
            if key_pattern.match(line_text):
                key_line_index = line_index

        if key_line_index is not None:
            file_lines[key_line_index] = new_line
        elif section_last_line_index is not None:
            file_lines.insert(section_last_line_index + 1, new_line)
        elif section_name:
            file_lines.extend(["", f"[{section_name}]", new_line])
        else:
            raise ValueError(f"No se puede añadir la clave raíz {key_name} al final del archivo")

    config_file.write_text("\n".join(file_lines) + "\n", encoding="utf-8")
```

**Tests obligatorios para este helper** (nuevo `scripts/tests/test_toml_config.py`):

1. Reemplazar una clave existente **conserva los comentarios** del archivo, incluidos los de la
   propia sección.
2. Añadir una clave a una sección existente la deja **dentro** de la sección, no al final.
3. Añadir una sección nueva a un archivo que termina en `[[servers]]` **no** contamina la tabla
   de array (re-parsear con `tomllib` y comprobar que `servers[0]` no ganó claves).
4. Valores con `"` y `\` sobreviven el round-trip.
5. Los enteros se escriben sin comillas (`port = 14010`, no `"14010"`).

### 5.2 `scripts/configure_db.py`

- L42: `PROJECT_CREDENTIALS_FILE = PROJECT_ROOT_DIRECTORY / "credentials.json"` →
  `PROJECT_CONFIG_FILE = PROJECT_ROOT_DIRECTORY / "config.toml"`.
- L555-575 `load_project_credentials()` → `read_config` + validación; mensaje de L561 →
  "Create it from config.example.toml and set db.password/db.port".
- L577-591 `save_project_credentials()` → llama a `set_config_values`. **Conservar el `chown` a
  `SUDO_UID`/`SUDO_GID` de L584-589**: el script corre bajo sudo y sin eso el archivo del repo
  queda de root.
- L593-613 `resolve_scylla_credentials()`: `.get("DB_PASSWORD")` → `get_config_value(data, "db.password")`,
  ídem `db.name`, `db.port`.
- L1318-1380: `GENIXSEARCH_URL` → `search.url`, `GENIXSEARCH_PASSWORD` → `search.password`.
- L1421-1432: el dict de actualizaciones pasa a claves con punto:
  `{"search.url": …, "search.password": …}`. **Conservar la regla de L1421-1422**: sólo se escriben
  los valores que el script generó; los ya presentes son decisión del operador.

### 5.3 `scripts/configure_server.py`

- L117-132 `detect_repository_credentials_path()` → `detect_repository_config_path()`, `config.toml`.
- L135-157 `load_project_credentials()` → `read_config`. Mantener el contrato de **archivo ausente
  ⇒ `{}`, no error** (L138-140): un host sólo-Nginx sin clon completo teclea los valores. El
  `except json.JSONDecodeError` pasa a `tomllib.TOMLDecodeError`, y **se conserva el fallo duro**
  ante archivo ilegible (L150-152): prompt + guardado sobreescribiría un archivo real roto.
- L253-270 `resolve_credential_value()`: la firma pasa a recibir la clave con punto; los mensajes
  de L263-270 deben nombrar la clave nueva (`server.nginx_domain`, …).
- L282-357: `NGINX_DOMAIN` → `server.nginx_domain`, `NGINX_PROCESS` → `server.nginx_process`,
  `SERVER_PORT` → `server.port`.
- L359-401 `persist_prompted_credentials()`: `json.dumps` → `set_config_values`. **Conservar**
  el prompt de confirmación (L370), el guard de `isatty` (L366) y el bloque `chown`+`chmod 0600`
  de L390-399 para archivo recién creado.
- L1126-1145 `build_main_service_contents()`: la unit de systemd emite
  `Environment=GENIX_CONFIG_FILE={repository_config_path}`, y el comentario de L1139 cita
  `server.port` / `server.nginx_process`.
- L1276: el comentario "el backend lee credentials.json a través de GENIX_CREDENTIALS_FILE".

### 5.4 `scripts/configure_sse_bridge.py`

Importa varios helpers de `configure_server.py` (L40-48) — actualizar los nombres importados.

- L85-117 `resolve_bridge_domain()`: `SSE_BRIDGE_URL` → `sse_bridge.url`.
- L119-134 `resolve_bridge_port()`: `SSE_BRIDGE_PORT` → `sse_bridge.port`.
- L136-161 `store_bridge_api_key()`: `set_config_values(path, {"sse_bridge.apikey": key})`.
  **Conservar `chown` + `chmod 0600`** (L154-160).
- L164-180 `resolve_bridge_api_key()`: **conservar el fallback `sse_bridge.apikey` → `secret_phrase`**,
  que es el mismo contrato de dos nombres que §4.4.
- L489-501: la unit emite `GENIX_CONFIG_FILE`; el comentario de L500 ("/etc/systemd/system es
  world-readable y credentials.json no") se actualiza al nombre nuevo.
- L587-603 `warn_if_credentials_are_unreadable()`: mensajes.

### 5.5 `scripts/recovery.py` L44-82

Sólo lectura. Rutas candidatas → `config.toml`, `json.load` → `tomllib.load` (modo binario),
`DB_USER`/`DB_PASSWORD`/`DB_PORT` → `db.user`/`db.password`/`db.port`. Mensaje de L56.

### 5.6 `ai/` — 4 scripts que la primera pasada no vio

Todos hacen `json.load` sobre `credentials.json` del directorio padre:

| Archivo | Líneas | Claves |
|---|---|---|
| `ai/launch_sagemaker_train.py` | 22-27, 33, 49-50, 98, 187 | `aws.sagemaker_iam_role`, `aws.sagemaker_s3_output`, `aws.hugging_face_token`, `aws.profile`, `aws.region` |
| `ai/launch_sagemaker_convert.py` | 22-28, 32, 48, 88 | ídem |
| `ai/follow_logs.py` | 11-13 | `aws.profile`, `aws.region` |
| `ai/view_training_logs.py` | 17-19 | `aws.profile`, `aws.region` |

`launch_sagemaker_*.py` aceptan hoy `SAGEMAKER_ROLE` **o** `SAGEMAKER_IAM_ROLE`
(`config.get("SAGEMAKER_ROLE", config.get("SAGEMAKER_IAM_ROLE"))`). En el TOML queda **una sola**
clave, `aws.sagemaker_iam_role`; eliminar el doble nombre.

### 5.7 Tests existentes

`scripts/tests/test_configure_server.py` (L64, L120-161, L503) y
`scripts/tests/test_configure_sse_bridge.py` (L114-217): las fixtures pasan de `json.dumps(...)` a
texto TOML literal, y los asserts de `json.loads(path.read_text())` a `tomllib`. El assert de
`test_configure_sse_bridge.py:217` espera `Environment=GENIX_CREDENTIALS_FILE=...` → `GENIX_CONFIG_FILE`.

---

## 6. Bun / JS

Ambos scripts corren bajo Bun (`frontend/package.json:18` usa `bun scripts/build-all.js`;
`scripts/deployer/actions.go:110` invoca `bun run ./scripts/set-github-frontend-vars.ts`), que trae
**`Bun.TOML.parse()`** incorporado. **Sin dependencias nuevas.**

**`frontend/scripts/setup-env.js` L6-40**

```js
  const configuredConfigPath = process.env.GENIX_CONFIG_FILE;
  const configPath = configuredConfigPath
    ? path.resolve(configuredConfigPath)
    : path.resolve(process.cwd(), '..', 'config.toml');
  // ...
  const config = Bun.TOML.parse(fs.readFileSync(configPath, 'utf-8'));
  const serializedPublicEndpoints = JSON.stringify(
    Array.isArray(config.endpoints) ? config.endpoints : []
  );
  const envContent = [
    `VITE_PROXY_PORT=${process.env.GENIX_PROXY_PORT || '3572'}`,
    `PUBLIC_LAMBDA_URL=${config.aws?.lambda_url || ''}`,
    `PUBLIC_SSE_BRIDGE_URL=${config.sse_bridge?.url || config.aws?.lambda_url || ''}`,
    `PUBLIC_FRONTEND_CDN=${config.frontend?.cdn_url || ''}`,
    `PUBLIC_ZONE_NAME=${config.frontend?.zone_name || ''}`,
    `PUBLIC_ENDPOINTS=${serializedPublicEndpoints}`
  ].join('\n') + '\n';
```

Los nombres `PUBLIC_*` de salida **no cambian**: `frontend/core/env.ts` y `frontend/app.html` no se
tocan. Conservar el fallback de `sse_bridge.url` a `aws.lambda_url` (sin bridge) y el
comportamiento de archivo ausente ⇒ warning y `return false`, no excepción (L40-41).

**`scripts/set-github-frontend-vars.ts` L9-107**

- L20-22: ruta + `GENIX_CONFIG_FILE`, `config.toml`.
- L92: `JSON.parse(...)` → `Bun.TOML.parse(await Bun.file(configPath).text())`.
- L9 `ProjectCredentials` → re-tipar por secciones (`aws?: { lambda_url?: unknown }`, etc.).
- L97-107: `credentials.ENPOINTS` → `config.endpoints`; `LAMBDA_URL` → `aws.lambda_url`;
  `SSE_BRIDGE_URL` → `sse_bridge.url`; `FRONTEND_CDN` → `frontend.cdn_url`;
  `ZONE_NAME` → `frontend.zone_name`; `GITHUB_ACCOUNT` → `github_account`.
- Los mensajes de error de `requireNonEmptyString` / `requireHttpUrl` reciben el nombre de la clave;
  pasarles el nombre con punto para que el error señale algo localizable en el archivo.

---

## 7. Shell

**`connect_db.sh`** — `jq` no lee TOML. Reemplazar L4-24; se elimina de paso el `dnf install jq`:

```bash
CONFIG_FILE="config.toml"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: $CONFIG_FILE not found in the current folder."
    exit 1
fi

# tomllib es stdlib desde Python 3.11 y ya es requisito de scripts/.
read_config_value() {
    python3 -c "import tomllib,sys; print(tomllib.load(open('$CONFIG_FILE','rb'))['db']['$1'])"
}

SCYLLA_HOST=$(read_config_value host)
SCYLLA_PORT=$(read_config_value port)
SCYLLA_USER=$(read_config_value user)
SCYLLA_PASS=$(read_config_value password)
```

**`app.sh:38`** y **`deploy.sh`**: menciones en comentarios. `app.sh` está marcado como deprecado en
`scripts/DEPLOYER.md`; **actualizar la mención, no reescribirlo**.

---

## 8. Archivos de proyecto

- **`.gitignore`**: `credentials.json*` y `credentials.*.json*` → `config.toml*` y `config.*.toml*`;
  `!credentials.example.json` → `!config.example.toml`. **Hacerlo antes de generar los
  `config.toml` reales**, que llevan secretos.
- **`credentials.example.json` → `config.example.toml`** (contenido de §3). Borrar el JSON.
- **`cloud/template.yml`**: 5 menciones en comentarios.
- **Basura de la raíz, borrar al final**: `credentials.json`, `credentials.1.json`,
  `credentials.json.backup`, `credentials.json.tmp`.
- `p2p/.gitignore` y `p2p/deploy/.gitignore` tienen su propia mención a `credentials.json`.

---

## 9. Documentación

Sólo texto. `AGENTS.md` · `DEPLOYMENT.md` · `README.md` ·
`scripts/{DEPLOYER,CONFIGURE_DB,CONFIGURE_SERVER,CONFIGURE_SSE_BRIDGE,SET_GITHUB_FRONTEND_VARS,SCRIPTS}.md` ·
`sse_bridge/README.md` · `p2p/{README,config/README,homelab_server/README}.md` ·
`backend/docs/ORM_DATABASE_QUERY.md` · `backend/agent/{PLAN_META_PROVIDER,AGENTIC_LOOP_DESIGN}.md` ·
`backend/cloud/PLAN_CLOUD_SCHEMA_DERIVATION.md` · `ai/{QUICK_START,README_TRAINING}.md`.

`AGENTS.md` describe `CONFIGURE_DB.md` como "escribiendo `GENIXSEARCH_URL`/`GENIXSEARCH_PASSWORD` en
credentials.json" → `search.url` / `search.password` en `config.toml`.

---

## 10. Script de migración (paso 1, de un solo uso)

`scripts/migrate_config_to_toml.py`. Evita transcribir secretos a mano y garantiza que los dos
entornos salgan idénticos en estructura.

- Entrada: `credentials.json` y `credentials.1.json` (decisión 6).
- Salida: `config.toml` y `config.1.toml`.
- Usa la tabla de §2 como diccionario `{"DB_HOST": "db.host", ...}` literal.
- Emite las secciones **en el orden de §3**, con sus comentarios, y `[[endpoints]]` / `[[servers]]`
  al final.
- `ENPOINTS` → `[[endpoints]]`; los ítems conservan `name`/`route` en minúscula.
- Una clave del JSON sin destino en el diccionario ⇒ **abortar con error listándola**, no
  descartarla en silencio. Es la red de seguridad contra una clave olvidada.
- Ignora explícitamente `*:OPTIONS` y `*:DOC`.
- **Verificación al final**: re-parsear el TOML generado con `tomllib` y comparar valor a valor
  contra el JSON de origen usando el mismo diccionario. Cualquier diferencia aborta.
- Se borra del repo al completar el paso 9.

---

## 11. Orden de ejecución

El orden importa: el contrato de `CONFIG` (§1.1) obliga a que backend y deployer cambien juntos.

| # | Paso | Verificación |
|---|---|---|
| 1 | `.gitignore` + `config.example.toml` | `git status` no muestra ningún `config.toml` |
| 2 | `scripts/migrate_config_to_toml.py`; generar `config.toml` y `config.1.toml` | la auto-verificación del §10 pasa |
| 3 | `scripts/toml_config.py` + `scripts/tests/test_toml_config.py` | los 5 tests de §5.1 pasan |
| 4 | **backend** (`core/security.go` + §4.7) | `cd backend && go build ./...`; arranca en local e imprime `Seteando config.toml desde: …` y conecta a Scylla |
| 5 | **deployer + cloud** (§4.2, §4.3) | `go build ./...` en `cloud/` y `scripts/`; `deploy.sh` abre el TUI y lista los dos entornos |
| 6 | `sse_bridge`, `p2p`, `db-backup` (§4.4-4.6) | `go build ./...` en cada módulo |
| 7 | Python (§5.2-5.7) | `python3 -m unittest discover scripts/tests` |
| 8 | Bun + shell (§6, §7) | `bun run scripts/set-github-frontend-vars.ts` (dry-run) y revisar el `.env` generado; `connect_db.sh` conecta |
| 9 | Docs (§9); borrar `credentials*.json*` y el script de migración | `rg -i 'credentials\.json'` no devuelve nada |

### Verificación de extremo a extremo (obligatoria, no opcional)

`deploy.sh` → **"Actualizar Variables"** contra la Lambda, y luego una petición real al backend
desplegado. Es el único camino que ejercita el par comprimir-TOML (`lambda_env.go`) /
descomprimir-TOML (`security.go`). Compilar los dos módulos **no** detecta un desajuste ahí.

---

## 12. Riesgos y trampas concretas

| Riesgo | Por qué muerde | Mitigación |
|---|---|---|
| **`Env` nil tras quitar `json.Unmarshal(&Env)`** | `var Env *EnvStruct` lo inicializaba el propio unmarshal sobre el doble puntero; el nuevo flujo no | `Env = &EnvStruct{}` explícito antes de `applyToEnv` (§4.1b) |
| **Clave olvidada en `applyToEnv`** | Queda en su cero-valor: **compila y arranca**, falla en runtime con un string vacío | Recorrer §2 campo a campo contra `EnvStruct`; el §10 lo cubre para el archivo, no para el mapeo |
| **Backend nuevo + Lambda con `CONFIG` en JSON** | Toda invocación falla al arrancar | Pasos 4 y 5 en la misma sesión, seguidos del redeploy del §11 |
| **Servidores con systemd apuntando a `GENIX_CREDENTIALS_FILE`** | La unit exporta una variable que ya nadie lee; el backend cae al path search y **no encuentra** el archivo | Re-ejecutar `configure_server.py` y `configure_sse_bridge.py` en cada host: regeneran la unit |
| **Clave suelta añadida tras `[[endpoints]]`** | TOML la asigna a la última tabla de array; el valor desaparece sin error de parseo | El escritor del §5.1 sólo añade bloques `[seccion]`; test 3 de §5.1 |
| **`json.Marshal` de `lambda_env.go:149`** | Es el `--environment` de la AWS CLI, no el archivo de config | **No tocar.** Se documenta aquí para que no se convierta por arrastre |
| **`DB_PORT`/`SMTP_PORT`/`server.port` escritos como string** | TOML sí distingue `14010` de `"14010"`; Go falla al parsear el int32 | `format_toml_value` del §5.1 no comilla ints; test 5 |
| **Doble nombre `SAGEMAKER_ROLE` en `ai/`** | Al colapsar a una clave, un archivo que usaba el nombre viejo deja de resolver el rol | La migración mapea ambos a `aws.sagemaker_iam_role` |
| **Secretos versionados** | `config.toml` lleva contraseñas de DB y tokens | Paso 1 antes del paso 2, sin excepción |
