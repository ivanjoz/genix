# Deployment Options

Este proyecto se puede desplegar en AWS Lambda + una base de datos ScyllaDB en un VPS/EC2

## Instalación de Scylla


## Parámetros de configuración

`deploy.sh` usa `config.toml` por defecto. Cuando existe `config.1.toml`, primero pide
elegir el environment con botones y exporta su ruta como `GENIX_CONFIG_FILE`. Las acciones de
frontend, backend, VPS, infraestructura y variables de GitHub leen esa misma selección; las
actualizaciones de `aws.lambda_url` y `frontend.cdn_url` también se escriben en el archivo elegido.

`providers.backend` accepts `aws`, `cloudflare`, or `none`. Use `none` for a self-hosted VPS
that keeps auth and tenant master data only in the primary ScyllaDB database.


## Lambda Deployment + DynamoDB + CDN

Set `providers.backend` to `aws` to deploy Lambda + DynamoDB. Set `providers.cdn` independently:
`aws` provisions the S3 + CloudFront frontend, while `cloudflare` provisions an R2 public origin.

La infraestructura AWS es una plantilla de CloudFormation, `cloud/template.yml`, desplegada
por la herramienta Go en `cloud/`. No hay CDK: ni Node, ni `npx`, ni bootstrap stack.

Se ejecuta desde `cloud/` con `go run . <accion>`:

| Acción | Qué hace |
|---|---|
| `1` | Compila el backend y actualiza el código de las dos Lambdas. |
| `2` | Comprime el archivo de credenciales seleccionado y lo publica como variables de entorno de las Lambdas. |
| `3` | Compila, sube el `.zip` a S3 y crea o actualiza el stack de CloudFormation. |

La acción `13` de `deploy.sh` hace lo mismo que la `2` pero con AWS CLI, sin compilar nada ni
esperar una tecla: reinyecta `CONFIG` (credenciales zstd + base64 url-safe),
`APP_CODE` y `LAMBDA_RESPONSE_STREAMING` en `<APP_NAME>-backend` y `<APP_NAME>-backend_2`, y las
variables sueltas de Cloudflare en `<APP_NAME>-webpage-renderer`. Las Lambdas que no existan en el
stack se omiten con un aviso.

### Qué crea la plantilla

Todo se nombra a partir de `app_name` de `config.toml` (el stack es `<APP_NAME>-stack`):

- Bucket S3 del frontend + CloudFront con OAI. HTML, `sw.js`, `registerSW.js` y
  `*.webmanifest` se sirven sin caché; el resto con la política `CachingOptimized`. Los 403 y
  404 se reescriben a `/index.html` para el enrutado del SPA.
- Dos Lambdas ARM64 sobre `provided.al2023` que corren el mismo binario: `<APP_NAME>-backend`
  (192 MB) y `<APP_NAME>-backend_2` (2048 MB), cada una con su Function URL pública.
- Regla de EventBridge que invoca la Lambda pequeña cada 10 minutos con `{"body":"{\"exec\":10}"}`.
  El backend detecta ese prefijo y ejecuta las filas pendientes de `cron_actions` (el equivalente
  en Lambda del `StartCronWatcher` que solo corre en el VPS).
- Tabla DynamoDB `<APP_NAME>-db` con 5 GSIs y TTL.

El bucket y la tabla son `Retain`: sobreviven al borrado del stack. Un stack posterior no
podrá recrear esos nombres mientras los huérfanos existan, así que hay que borrarlos a mano
o usar un `APP_NAME` nuevo.

La plantilla no crea recursos IAM; consume el rol existente de `aws.lambda_iam_role`.

### La tabla DynamoDB la gobierna CloudFormation

`MainTable` tiene un solo dueño: la plantilla. `DynamoORM.Init()` (`backend/cloud/orm-dynamodb.go`)
**solo comprueba** que la tabla exista y devuelve un error si no está; ya no la crea.

Antes la creaban los dos. Ejecutar `./deploy.sh "6 9"` hacía que `fn-init` la crease segundos
antes que CloudFormation, y el stack entero abortaba con `AlreadyExists`. Como consecuencia:

**la acción 9 debe correr antes que la 5 o la 6.** El propio `deploy.sh` ya coloca el bloque de
infraestructura antes que el de tablas, así que `./deploy.sh "6 9"` funciona en una sola
invocación; el orden solo importa si las lanzas por separado.

### Streaming de respuesta

Las Function URL usan `InvokeMode: RESPONSE_STREAM`, que evita la expansión base64 del +33% y
sube el techo de respuesta de 6 MB a 20 MB. El handler de Go elige la forma de su respuesta
según `LAMBDA_RESPONSE_STREAMING`, así que **esa variable y el `InvokeMode` deben coincidir
siempre**: si se desajustan falla toda petición. Ambos se definen juntos en `template.yml`, y
la acción `2` los reescribe desde la constante `lambdaResponseStreamingFlag` de `cloud/main.go`
(`UpdateFunctionConfiguration` reemplaza el entorno completo, no lo fusiona).

### Después de un despliegue

La acción `3` imprime los outputs del stack y escribe `BackendUrl` en el `aws.lambda_url` del
archivo de configuración seleccionado, avisando del valor anterior si cambió. Si `aws.lambda_url` apuntaba a un
dominio propio en vez de a una Function URL, la herramienta lo advierte: hay que reapuntar ese
dominio a la nueva URL o restaurar el valor anterior a mano.

`frontend.cdn_url` no se actualiza solo. Cópialo del output `FrontendDistributionDomain` cuando el
dominio de CloudFront cambie, y vuelve a subir el frontend al bucket nuevo.

### El agente necesita el SSE Bridge

Lambda no puede sostener un stream abierto ni recibir la respuesta del navegador dentro de la
misma invocación, y el agente necesita las dos cosas. Con el backend en Lambda hay que desplegar
además `fareward/` en un servidor normal — el bridge es una de sus dos mitades, junto al
credit rate limiter — y poner su URL pública en `sse_bridge.url` de `config.toml`
(`sudo python3 scripts/configure.py 38`; detalles en
`scripts/configure/CONFIGURE_FAREWARD.md`).

Ese host necesita `internal_apikey` y `secret_phrase` idénticos a los del backend, y las tablas
ya desplegadas: el rate limiter sale con error si no puede leer `credit_usage`, y al ser un solo
proceso eso también deja el bridge sin arrancar.

Si `sse_bridge.url` falta o es igual a `aws.lambda_url`, el chat del agente queda inoperativo en el
endpoint Lambda; todo lo demás de la app funciona igual. En self-host no hace falta: ese proceso
sirve su propio `/agent/stream`, y `scripts/configure.py 238` instala Fareward sin validar esa
URL ni configurar su sitio de HAProxy cuando se elige Backend mode `1` o `2`.

## Self-host Deployment + DynamoDB + S3

El proyecto debe ser compilado y el archivo "app" y el "config.toml" debe ser subido en el mismo folder.

Creación de servicio en systemd

nano /etc/systemd/system/genix.service

Configuracion
```TOML
[Unit]
Description=Genix Backend
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root
ExecStart=/root/app
Restart=always
RestartSec=5
StandardOutput=append:/var/log/genix.log
StandardError=append:/var/log/genix.log

[Install]
WantedBy=multi-user.target
```

Luego ejecutar:
systemctl daemon-reload
systemctl enable genix
systemctl start genix

Revisar que el servicio esté ejecutándose:

systemctl status genix

Revisar lso logs:

tail -f /var/log/genix.log

Follow the logs:

journalctl -u genix.service -f

### Edge: HAProxy + certificate + fareward SPOE

The edge is installed by `scripts/configure/configure_server.py` (mode `1` full, or `3` edge-only),
not by hand: HAProxy 3.2 from the official repository, the Let's Encrypt certificate through
`certbot --standalone` behind HAProxy, HTTP/3, and, when `[spoe].socket` is set, the fareward SPOE
agent that refuses requests on their headers before the body is read. Nginx is no longer used; an
existing one is stopped and disabled by the script. Details:
[`scripts/configure/CONFIGURE_SERVER.md`](scripts/configure/CONFIGURE_SERVER.md).

```bash
sudo python3 scripts/configure.py 28   # then choose Backend mode 1
sudo haproxy -c -f /etc/haproxy/haproxy.cfg
journalctl -u haproxy -f              # access log, with the fareward verdict {status|reason|error}
```
