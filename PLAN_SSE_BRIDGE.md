# PLAN — SSE Bridge (EC2) para el agente en Lambda

Lambda no puede sostener un stream SSE por turno ni recibir la respuesta del
navegador dentro de la misma invocación. El bridge es un proceso Go diminuto en
EC2/VPS cuyo único trabajo es: **mantener la conexión abierta con el navegador y
ser el punto de encuentro (rendezvous) entre el Lambda y esa pestaña**.

El bridge no sabe nada del agente, del chat ni de la base de datos. Solo mueve
JSON opaco entre dos partes y correlaciona respuestas por `ID`.

## Identidad del canal

Un canal = una pestaña, nombrado por un **token de canal** único
(`sse_bridge/channel.go`, `backend/agent/channel.go`,
`frontend/core/agent/channel.ts`):

```
bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 bytes aleatorios (tab)
token = base64url(bytes), sin padding      →  "Byo3bFBobzE" (11 chars)
```

- Los varints son lo que lo mantienen corto: el company id en decimal cuesta
  6-7 caracteres él solo.
- El tab son 6 bytes = **8 caracteres** base64url, 48 bits de entropía. Sigue
  existiendo suelto en `sessionStorage["__agent_tab_id"]` porque es la clave del
  historial local (`chat_history.idb`), que no debe partirse al cambiar de
  company.
- El decodificador rechaza varints no canónicos, así que el token es biyectivo
  con la terna y puede usarse como clave del registro sin normalizar.

**Es un identificador, no una credencial.** El navegador sigue mandando su token
de sesión (autocontenido: `colbin` + HMAC de `SECRET_PHRASE`,
`backend/core/usuario-accesos.go:143`, validable sin DB), y tanto el bridge
(`resolveClientChannel`) como el handler del turno (`PostAgentTurn`) comprueban
que el company/user *dentro* del token de canal coincidan con los del token de
sesión validado. Sin ese cruce, editar el token sería direccionar la pestaña de
otro tenant.

Última conexión gana (misma regla que `registerClient` hoy): al reconectar, el
stream anterior recibe `{"Type":"replaced"}` y se cierra.

## Flujo de un turno con el bridge

```
navegador                    bridge (EC2)                    Lambda
   |--- GET /sse?ch= ----------->|  registra el canal
   |<-- data:{bridgeReady} ------|  handshake: ya es alcanzable
   |                             |
   |--- POST /api/p-agent-turn ---------------------------->|  arranca el turn
   |                             |<-- POST /publish {status} -|
   |<-- data:{agentStatus} ------|                            |
   |                             |<-- POST /rpc {ID:7,navigate}  (BLOQUEA)
   |<-- data:{ID:7,navigate} ----|                            |
   |--- POST /in?ch= {ID:7,…} -->|                            |
   |                             |--- 200 {result} ---------->|  request() retorna
   |                             |<-- POST /publish {reply} --|
   |<-- data:{agentReply} -------|                            |
   |<-- 200 {Ok:true} --------------------------------------|  el turn termina
```

Puntos clave:

- El **turn es un POST normal que bloquea hasta terminar**; su respuesta ya no es
  el stream, es un `{Ok:true}` final. Los eventos incrementales salen por el
  stream.
- `POST /rpc` es lo que hace posible el RPC inverso: el Lambda queda bloqueado en
  una llamada HTTP mientras el navegador ejecuta el comando y responde.
- **Stream permanente de inicio ocioso**: no se abre en el boot de la app. Se
  abre la primera vez que hace falta (al abrir el chat o justo antes del primer
  turn) y a partir de ahí se mantiene, con reconexión exponencial.
- **Handshake**: el primer frame que manda el servidor es `{"Type":"bridgeReady"}`,
  emitido *después* de registrar el canal. El cliente no manda el turn hasta
  recibirlo, así que el Lambda nunca publica contra un canal inexistente.
  Como segunda red, `/publish` y `/rpc` aceptan `WaitMs` para esperar a que el
  canal aparezca.

### Unificación de los dos modos

En vez de mantener dos transportes distintos (turn-streaming en VPS + bridge en
Lambda), los dos modos pasan a tener **la misma forma**: stream permanente +
turn como POST plano. Solo cambia a qué host se conecta cada uno.

Esto **borra** código en lugar de añadirlo: el streaming de `turn.go`, el `sink`
de `AgentSession` y `readTurnStream` del frontend desaparecen. El stream ocioso
local (`/agent/stream`) ya existe — deja de ser opt-in y vuelve a ser el canal de
eventos, que es lo que era antes de `PLAN_AGENT_TURN_STREAM.md`.

## API del bridge

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `GET` | `/sse?ch=<token>` | token de usuario | Abre el stream. Responde `text/event-stream`, primer frame `{"Type":"bridgeReady"}`, keepalive `: ping` cada 20s. |
| `POST` | `/in?ch=<token>` | token de usuario | Mensaje navegador→backend `{ID,Type,Payload}`. Si hay un `/rpc` esperando ese `ID`, lo despierta. Si no, se descarta (se loguea). |
| `POST` | `/publish` | HMAC de servicio | `{Channel,Message,WaitMs}` → empuja `Message` al stream. Responde `{Delivered:bool}`. No bloquea. |
| `POST` | `/rpc` | HMAC de servicio | `{Channel,ID,Message,TimeoutMs,WaitMs}` → empuja y **bloquea** hasta la respuesta del navegador o timeout. Responde `{Kind,Payload}`. |
| `GET` | `/health` | — | `{Ok,Channels,Clients,Uptime}`. |

- Sin buffer: si el canal no está conectado, `/publish` responde
  `{Delivered:false}` y `/rpc` responde 409 de inmediato (decisión ya tomada:
  los mensajes se descartan, el cliente se resincroniza por la API normal).
- CORS abierto para los dos endpoints de cliente (`GET /sse`, `POST /in`),
  igual que el backend hoy.

### Auth de servicio (Lambda → bridge)

Header `X-Bridge-Auth: <unix_ts>.<hex(hmac_sha256(SECRET_PHRASE, "sse-bridge:v1|"+unix_ts))>`,
con ventana de ±300s. Comparación en tiempo constante. Sin secreto nuevo que
distribuir: el Lambda y el bridge ya comparten `credentials.json`.

### Auth de cliente (navegador → bridge)

`Authorization: Bearer <token>` — el mismo token que ya manda el frontend.
El bridge lo decodifica con `colbin` y recomputa el HMAC exactamente como
`ComputeUsuarioTokenHash`. Sin acceso a Scylla: el bridge no evalúa *accesos*,
solo identidad (el Lambda ya evaluó permisos al aceptar el turn).

`EventSource` no permite headers → el stream se consume con `fetch` +
`ReadableStream`, que es lo que ya hace `frontend/libs/sse-client.ts`.

## Proyecto Go

```
sse_bridge/
  go.mod              módulo propio (module genix/sse_bridge), deps: colbin
  main.go             flags/env, servidor HTTP, /health, apagado ordenado
  config.go           lee credentials.json (mismo lookup que el backend) + env vars
  auth.go             validación de token de usuario + HMAC de servicio
  channel.go          códec del token de canal (varints + base64url)
  channels.go         registro de canales, fan-out, tabla de RPC pendientes
  handlers.go         /sse, /in, /publish, /rpc
  README.md           despliegue, systemd, nginx, pruebas con curl
```

Sin dependencias del módulo `backend/`: es un binario aparte que se despliega
solo. Las dos piezas compartidas se replican con un comentario apuntando al
original: el formato del token de sesión (colbin + HMAC) en `auth.go`, y el
códec del token de canal en `channel.go` — este último con vectores de prueba
(`channel_vectors_test.go`) que fijan que las tres implementaciones (bridge,
backend, frontend) produzcan los mismos bytes.

Config: `SSE_BRIDGE_PORT` (default **14012**, siguiendo 14008/14010/14446),
`GENIX_CREDENTIALS_FILE` para el `credentials.json`.

## Wiring de configuración

**`credentials.json`** — nueva clave opcional:

```json
"SSE_BRIDGE_URL": "https://genix-sse.un.pe/"
```

- **Frontend** (`frontend/scripts/setup-env.js`):
  `PUBLIC_SSE_BRIDGE_URL = credentials.SSE_BRIDGE_URL || credentials.LAMBDA_URL`
  → expuesto en `Env.SSE_BRIDGE_URL` (`frontend/core/env.ts`).
- **Backend** (`backend/core/security.go`, `EnvStruct`): campo `SSE_BRIDGE_URL`.
  Vacío = sin bridge (comportamiento actual, SSE nativo).

Regla de selección, idéntica en ambos extremos para que no puedan discrepar:

- Frontend: si el endpoint API seleccionado es `PUBLIC_LAMBDA_URL` **y**
  `PUBLIC_SSE_BRIDGE_URL` difiere de él → el stream va al bridge. En cualquier
  otro caso (local, VPS) → SSE nativo contra el endpoint seleccionado.
- Backend: si `IS_SERVERLESS` **y** `SSE_BRIDGE_URL` no está vacío → transporte
  por bridge. Si no → `clientConn` local como hoy.

## Cambios en el backend (`backend/agent/`)

`ws.go` ya concentra todo el transporte en dos funciones: `cc.push` (servidor→
navegador) y `request` (RPC con espera). Se introduce una interfaz mínima:

```go
// agentTransport abstrae de dónde cuelga la pestaña: una conexión SSE local
// (VPS) o el bridge remoto (Lambda). Nada más del paquete cambia.
type agentTransport interface {
    Push(tab string, data []byte) error
    Request(ctx context.Context, tab, cmdType string, payload, out any) error
}
```

- `localTransport` = el código actual de `clientConn` (sin cambios de conducta).
- `bridgeTransport` (nuevo, `bridge.go`) = `POST /publish` y `POST /rpc`.
- `AgentSession.sendJSON` y `request()` pasan por la interfaz.

**Ruta del turn:** `POST p-agent-turn` en `ModuleHandlers`, la misma en los dos
modos (reemplaza a `POST /agent/turn`). El prefijo `p-` solo significa "sin
requisito de *acceso*" en `mainHandler`; el handler llama a `core.CheckUser`
directamente y rechaza cualquier petición sin token válido. Motivo: en
`main-handlers.go:143-170` un POST sin entrada en `access_list.yml` se rechaza
para todo usuario que no sea el ID 1, y el chat del agente debe seguir
disponible para cualquier usuario logueado (hoy no tiene gate alguno).
`companyID/userID` del canal salen del token validado, no del body.

**Limitación conocida (fuera de alcance):** `chatSessions` es un mapa en memoria;
en Lambda dos turnos consecutivos pueden caer en instancias distintas y perder
el estado en vivo de la sesión. El historial persiste en Scylla
(`chat_store.go`), así que la conversación no se pierde, pero el
`inFlight` anti-doble-turno solo protege dentro de una instancia.

## Cambios en el frontend (`frontend/core/agent/sse.ts`)

- `agentHttpBase()` se desdobla: `agentTurnBase()` (endpoint API seleccionado,
  para `POST` del turn) y `agentStreamBase()` (bridge o endpoint, según la regla
  de arriba).
- El stream vuelve a ser permanente cuando hay bridge: se abre con `fetch` +
  `ReadableStream` (reusando el parser de `libs/sse-client.ts`) mandando
  `Authorization`, con reconexión exponencial y rotación de tab ante `replaced`.
- `postIn` apunta a `agentStreamBase()/in` (el bridge es quien tiene al waiter).
- El despacho de mensajes (`ID > 0` → `runCommand` → `postIn`; `ID == 0` →
  `chatListeners`) no cambia.

## Fases

1. **Bridge + config** — proyecto Go completo, `SSE_BRIDGE_URL` en credenciales,
   `EnvStruct`, `setup-env.js`, `Env`. Verificable solo con `curl`.
2. **Transporte del backend** — interfaz + `bridgeTransport` + turn serverless.
3. **Frontend** — selección de base, stream permanente autenticado.

## Verificación

1. `go build ./...` y `go vet ./...` en `sse_bridge/` y en `backend/`.
2. Tests del bridge (`go test ./...`): auth HMAC (válido/expirado/alterado),
   token de usuario (hash correcto/incorrecto), fan-out, correlación de `/rpc`,
   timeout de `/rpc`, canal ausente.
3. Manual con `curl`: abrir `/sse`, publicar, ver el frame; lanzar `/rpc`,
   responder con `/in`, ver que retorna.
4. `npx svelte-check` sin errores nuevos sobre los 12 preexistentes.
5. Manual end-to-end contra el Lambda: un turno multi-herramienta ("ve a
   productos y busca X") con estados incrementales y navegación real.

## Fuera de alcance

- Buffer/replay de eventos (`Last-Event-ID`) — decidido: se descartan.
- TLS en el bridge (va detrás de nginx, como el backend en VPS).
- Migrar el SSE de métricas del sistema (`config/system_metrics_sse.go`) — es
  local-only por diseño.
- Escalado horizontal del bridge (un proceso, estado en memoria).
