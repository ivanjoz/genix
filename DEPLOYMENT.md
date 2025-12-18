# Deployment Options

Este proyecto se puede desplegar en AWS Lambda + una base de datos ScyllaDB en un VPS/EC2

## Instalación de Scylla


## Parámetros de configuración


## Lambda Deployment + DynamoDB + S3


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

        # 7. Reverse Proxy to Go
        proxy_pass http://127.0.0.1:3589;

        # 8. Timeouts optimized for trans-continental connections
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Buffer settings for performance
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }
}
```

sudo nginx -t

sudo systemctl restart nginx