# Deployment Options

Este proyecto se puede desplegar en AWS Lambda + una base de datos ScyllaDB en un VPS/EC2

## Instalación de Scylla


## Parámetros de configuración


## Lambda Deployment + DynamoDB + S3

La infraestructura AWS es una plantilla de CloudFormation, `cloud/template.yml`, desplegada
por la herramienta Go en `cloud/`. No hay CDK: ni Node, ni `npx`, ni bootstrap stack.

Se ejecuta desde `cloud/` con `go run . <accion>`:

| Acción | Qué hace |
|---|---|
| `1` | Compila el backend y actualiza el código de las dos Lambdas. |
| `2` | Comprime `credentials.json` y lo publica como variables de entorno de las Lambdas. |
| `3` | Compila, sube el `.zip` a S3 y crea o actualiza el stack de CloudFormation. |

### Qué crea la plantilla

Todo se nombra a partir de `APP_NAME` de `credentials.json` (el stack es `<APP_NAME>-stack`):

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

La plantilla no crea recursos IAM; consume el rol existente de `LAMBDA_IAM_ROLE`.

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

La acción `3` imprime los outputs del stack y escribe `BackendUrl` en el `LAMBDA_URL` de
`credentials.json`, avisando del valor anterior si cambió. Si `LAMBDA_URL` apuntaba a un
dominio propio en vez de a una Function URL, la herramienta lo advierte: hay que reapuntar ese
dominio a la nueva URL o restaurar el valor anterior a mano.

`FRONTEND_CDN` no se actualiza solo. Cópialo del output `FrontendDistributionDomain` cuando el
dominio de CloudFront cambie, y vuelve a subir el frontend al bucket nuevo.

## Self-host Deployment + DynamoDB + S3

El proyecto debe ser compilado y el archivo "app" y el "credentials.json" debe ser subido en el mismo folder.

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

### Configuración de Certbot y Nginx
sudo snap install --classic certbot

sudo ln -s /snap/bin/certbot /usr/bin/certbot

sudo certbot --nginx -d genix-dev-api-1.un.pe

Seguir los pasos para generación de certificado. No se asociará aún pero se crearán los archivos necesario. Saldrá un mensaje como este:

```
Successfully received certificate.
Certificate is saved at: /etc/letsencrypt/live/genix-dev-api-1.un.pe/fullchain.pem
Key is saved at:         /etc/letsencrypt/live/genix-dev-api-1.un.pe/privkey.pem
```

Generar configuracion de nginx

nano /etc/nginx/conf.d/genix-dev-api-1.un.pe.conf

```
# Map block to handle 0-RTT security (prevents replay attacks on POST/PUT)
map $ssl_early_data $is_early_data {
    "~on" 1;
    default 0;
}

# WebSocket upstreams require HTTP/1.1 plus an explicit Upgrade tunnel.
map $http_upgrade $connection_upgrade {
    default upgrade;
    "" close;
}

server {
    # 1. Standard TCP and HTTP/3 UDP listeners
    listen 443 quic reuseport;
    listen 443 ssl;
    listen [::]:443 quic reuseport;
    listen [::]:443 ssl;

    server_name genix-dev-api-1.un.pe;

    # 2. SSL/TLS Settings (Pointed to Certbot paths)
    ssl_certificate /etc/letsencrypt/live/genix-dev-api-1.un.pe/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/genix-dev-api-1.un.pe/privkey.pem;
    
    ssl_protocols TLSv1.3; # 0-RTT requires TLS 1.3
    ssl_early_data on;     # The "Zero Round Trip" magic
    
    # 3. Session Optimization for returning users in Peru
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets on;

    # 4. HTTP/3 Discovery Header
    add_header Alt-Svc 'h3=":443"; ma=86400';

    location / {
        # 5. Security: Reject non-GET 0-RTT requests to prevent replay attacks
        # If your Go backend doesn't handle the "Early-Data" header, 
        # Nginx can block potentially dangerous early requests here.
        if ($request_method != GET) {
            set $early_data_check "${is_early_data}";
        }
        if ($early_data_check = "1") {
            return 425; # "Too Early" - browser will retry automatically
        }

        # 6. Proxy Headers for Go Backend
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Passes 0-RTT status so Go can see it
        proxy_set_header Early-Data $ssl_early_data;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        # 7. Reverse Proxy to Go
        proxy_pass http://127.0.0.1:3589;

        # 8. Timeouts optimized for trans-continental connections
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        # Agent WebSockets can sit idle while the user thinks between messages.
        proxy_read_timeout 3600s;
        
        # Buffer settings for performance
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }
}
```

sudo nginx -t

sudo systemctl restart nginx
