# SSE Bridge

Relay de Server-Sent Events entre el backend de Genix y las pestañas del
navegador. Existe por una sola razón: **Lambda no puede sostener un stream** ni
recibir la respuesta del navegador dentro de la misma invocación, y el agente
necesita ambas cosas (empuja eventos de chat y además pide al navegador que
ejecute comandos y espera el resultado).

Este proceso corre en un servidor normal (EC2/VPS), mantiene la conexión con el
navegador y hace de punto de encuentro en las dos direcciones.

No tiene lógica de negocio ni conexión a base de datos. Los mensajes son JSON
opaco y **no se bufferiza nada**: un mensaje para una pestaña desconectada se
descarta.

```
navegador                     bridge                        backend (Lambda)
   |--- GET /sse?tab= --------->| registra el canal
   |<-- data:{bridgeReady} -----| handshake
   |                            |<--- POST /publish ---------| evento (no bloquea)
   |<-- data:{agentStatus} -----|
   |                            |<--- POST /rpc -------------| comando (BLOQUEA)
   |<-- data:{ID:7,navigate} ---|
   |--- POST /in {ID:7,...} --->|
   |                            |---- 200 {Kind,Payload} --->| request() retorna
```

Diseño completo: `../PLAN_SSE_BRIDGE.md`.

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `GET` | `/sse?ch=<token>` | token de sesión | Abre el stream. Primer frame `{"Type":"bridgeReady"}`, keepalive `: ping` cada 20s. |
| `POST` | `/in?ch=<token>` | token de sesión | Respuesta del navegador `{ID,Type,Payload}`. Despierta al `/rpc` que espera ese `ID`. |
| `POST` | `/publish` | HMAC de servicio | `{Channel,Message,WaitMs}` → `{Delivered}`. No bloquea. |
| `POST` | `/rpc` | HMAC de servicio | `{Channel,ID,Message,TimeoutMs,WaitMs}` → `{Kind,Payload}`. Bloquea hasta la respuesta. |
| `GET` | `/health` | — | `{Ok,Channels,UptimeSeconds}`. |

## Token de canal

Un canal es una pestaña, y se nombra con un solo string (`channel.go`):

```
bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 bytes aleatorios (tab)
token = base64url(bytes), sin padding
```

Para ids normales son **11 caracteres** (`7/42` → `Byo3bFBobzE`); el tab son 6
bytes = 8 caracteres si se mira suelto. Los varints son lo que lo mantiene
corto: un company id en decimal cuesta 6-7 caracteres él solo.

El decodificador **rechaza codificaciones no canónicas** (un varint alargado
representa el mismo número con otros bytes). Eso hace que el token sea
biyectivo con la terna, que es lo que permite usarlo directamente como clave del
registro: dos strings distintas nunca pueden nombrar el mismo canal.

**Es un identificador, no una credencial.** El navegador sigue probando quién es
con su token de sesión, y el bridge comprueba que la identidad *dentro* del
token de canal coincida con la autenticada (`resolveClientChannel`). Sin ese
cruce, editar el company id del token sería conectarse al stream de otro tenant.

El formato está triplicado en `channel.go`, `backend/agent/channel.go` y
`frontend/core/agent/channel.ts`; los vectores de `channel_vectors_test.go`
fijan que las tres implementaciones coincidan byte a byte.

## Autenticación

Ambos lados usan `SECRET_PHRASE` de `credentials.json` — no hay ningún secreto
nuevo que aprovisionar.

- **Navegador**: `Authorization: Bearer <token>`, el mismo token de sesión que ya
  emite el backend. Es autocontenido (payload colbin + HMAC), así que el bridge
  verifica la identidad sin tocar ScyllaDB. El bridge **no** evalúa accesos: eso
  ya lo hizo el backend al aceptar el turno.
- **Backend**: `X-Bridge-Auth: <unix_ts>.<hex(hmac_sha256(SECRET_PHRASE, "sse-bridge:v1|"+unix_ts))>`,
  con ventana de ±300s.

## Configuración

| Variable | Default | Descripción |
|---|---|---|
| `GENIX_CREDENTIALS_FILE` | `../credentials.json`, `./credentials.json` | De dónde leer `SECRET_PHRASE`. |
| `SECRET_PHRASE` | — | Sobrescribe el valor del archivo (para desplegar sin `credentials.json`). |
| `SSE_BRIDGE_PORT` | `14012` | Puerto de escucha. |
| `SSE_BRIDGE_VERBOSE` | — | `1` para loguear cada mensaje entregado. |

Del lado de quienes lo usan, `credentials.json` lleva `SSE_BRIDGE_URL` con la URL
pública de este proceso. El backend solo lo usa cuando corre en Lambda; el
frontend solo cuando el endpoint seleccionado es el Lambda. Si `SSE_BRIDGE_URL`
falta o es igual a `LAMBDA_URL`, no hay bridge y el backend sirve su propio
`/agent/stream`.

## Compilar y correr

```bash
cd sse_bridge
go build -o sse_bridge .
GENIX_CREDENTIALS_FILE=/etc/genix/credentials.json ./sse_bridge

go test ./...     # auth, fan-out, correlación de rpc, timeouts, reconexión
```

Compilación cruzada para un servidor arm64 (mismo patrón que el backend):

```bash
GOOS=linux GOARCH=arm64 go build -o sse_bridge_arm64 .
```

### systemd

```ini
# /etc/systemd/system/genix-sse-bridge.service
[Unit]
Description=Genix SSE Bridge
After=network.target

[Service]
Type=simple
User=ubuntu
Environment=GENIX_CREDENTIALS_FILE=/etc/genix/credentials.json
ExecStart=/usr/local/bin/genix/sse_bridge
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

### nginx

El bridge habla HTTP plano; el TLS lo termina nginx, igual que el backend. Los
tres ajustes marcados abajo no son opcionales: sin ellos el stream se entrega a
tirones o se corta.

```nginx
location / {
    proxy_pass http://127.0.0.1:14012;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;          # imprescindible: si no, los eventos se acumulan
    proxy_read_timeout 3600s;     # imprescindible: un stream ocioso no debe morir
    proxy_send_timeout 3600s;
}
```

## Probar a mano

```bash
BASE=http://localhost:14012
TOKEN=<token de sesión de una pestaña logueada>
CH=<el ?ch= que manda esa pestaña; míralo en la pestaña de red>

# 1. Abrir el stream (deja esta terminal abierta; verás bridgeReady y los pings)
curl -N -H "Authorization: Bearer $TOKEN" "$BASE/sse?ch=$CH"

# 2. Firmar una llamada de servicio y publicar un evento
TS=$(date +%s)
SECRET=$(grep -oP '"SECRET_PHRASE"\s*:\s*"\K[^"]+' ../credentials.json)
SIG="$TS.$(printf 'sse-bridge:v1|%s' "$TS" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $2}')"

curl -s -X POST "$BASE/publish" -H "X-Bridge-Auth: $SIG" \
  -d "{\"Channel\":\"$CH\",\"Message\":{\"Type\":\"agentStatus\",\"Payload\":{\"Label\":\"hola\"}}}"
```

Tiene que ser el **mismo** token de canal en los dos pasos: si difiere en un
carácter es otro canal y `Delivered` saldrá `false`. Para fabricar uno a mano,
`EncodeChannelToken(company, user, tab)` en un test es lo más rápido.
