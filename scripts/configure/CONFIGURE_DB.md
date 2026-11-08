# Configuración de los motores de datos (`configure_db.py`)

One script prepares the data host: **MariaDB** (the primary database) and **Qdrant** (the
vector database). Text search needs no host: genix-s3 keeps its index in the `[file_store]`
bucket (see `backend/core/text_search.go`).

The public entrypoint configures both (`all`). The `7`/`8` digit is required by the unified
grammar but does not change how this installer gets its packages:

```bash
sudo python3 scripts/configure.py 18
```

One engine at a time, through the nested installer:

```bash
sudo python3 scripts/configure/configure_db.py mariadb
sudo python3 scripts/configure/configure_db.py qdrant
```

`all` installs MariaDB and Qdrant. ScyllaDB is no longer supported and only runs when named
(`scylla`). The script must run as `root` and reads `config.toml` from the repository root.

---

## MariaDB (`mariadb`)

| Step | Detail |
| ---- | ------ |
| Package | `mariadb-server` with apt or dnf, when no `mariadbd` is installed. |
| Server config | `/etc/mysql/mariadb.conf.d/90-genix.cnf` (or the dnf equivalent), rewritten each run: `port = mariadb.port`, buffer pool at half the RAM, `innodb_flush_log_at_trx_commit = 2` (an OS crash loses up to ~1 s of commits), no binary log, a thread pool of one thread per core, `READ-COMMITTED`, `skip-name-resolve`, no performance schema. |
| Bind | `127.0.0.1` when `mariadb.host` is loopback; `0.0.0.0` with `mariadb.public = true`; otherwise the tailnet/VPC address. |
| Accounts | `mariadb.user` on `localhost` (the Unix socket), `127.0.0.1` and `::1`, plus `%` when public or remote: with `skip-name-resolve`, `localhost` does not match TCP from 127.0.0.1. All get every privilege on `db.name`, created `utf8mb4_bin`. |
| Firewall | Opens `mariadb.port/tcp` only when the server listens outside loopback. |

```toml
[mariadb]
host     = "127.0.0.1"                 # what this host's backend dials
port     = 14016
user     = "genix"
password = "..."
public   = true                        # also reachable from other machines
socket   = "/run/mysqld/mysqld.sock"   # backend and fareward skip loopback TCP
```

The backend and fareward dial `socket` when it is set. On a 2-core Ampere A1, `SELECT 1` at 16
connections ran ~35k queries/s over the socket against ~24.6k over loopback TCP.

The tables are not created here: `configure.py` runs `fn-homologate` after installing the backend.

---

## ScyllaDB (`scylla`)

| Paso | Detalle |
| ---- | ------- |
| `SCYLLA_ARGS` | Reescribe el env-file (`/etc/sysconfig/scylla-server` o `/etc/default/scylla-server`) con `--smp` acotado a los CPU disponibles, `-m 4G` y `--overprovisioned`. Limpia los flags que gestionó una corrida anterior. |
| `scylla.yaml` | `listen_address=127.0.0.1`, `rpc_address=0.0.0.0`, `broadcast_rpc_address=<IP alcanzable>`, `native_transport_port=db.port` y `authenticator=PasswordAuthenticator`. |
| Firewall | Abre `db.port/tcp` (ver [Firewall](#firewall) abajo). |
| Superusuario | Prueba la password de `config.toml` y la default `cassandra`. Si ninguna entra, recupera el rol por el **maintenance socket** de Scylla (ver abajo) y luego la cambia a `db.password`. |
| Keyspace | Crea `db.name` con `NetworkTopologyStrategy` y `replication_factor: 1` si no existe. |

Claves usadas: `db.password`, `db.name`, `db.port`.

### Recuperación del superusuario

Scylla siembra el rol `cassandra` solo en el bootstrap inicial del cluster. Si el nodo
arrancó por primera vez con `AllowAllAuthenticator`, activar `PasswordAuthenticator`
después lo deja sin ningún rol con el que autenticarse. El script habilita entonces
`maintenance_socket: workdir`, reinicia el servicio y publica ese socket unix como un
puerto TCP efímero en `127.0.0.1` (el `cqlsh` empaquetado para Ubuntu no acepta
`--maintenance-socket`), por donde crea o corrige el rol sin autenticación.

---

## Qdrant (`qdrant`)

Instala la base vectorial desde los **releases musl** de
[`qdrant/qdrant`](https://github.com/qdrant/qdrant/releases): builds estáticos, así que
el host no necesita toolchain ni una glibc concreta.

| Paso | Detalle |
| ---- | ------- |
| Arquitectura | `x86_64-unknown-linux-musl` o `aarch64-unknown-linux-musl` según `uname -m`. Cualquier otra aborta: no hay build publicado. |
| Descarga | El `.tar.gz` del release (`--qdrant-version`, default el último resuelto por la API de GitHub). **Sin checksum**: los releases de Qdrant no publican `SHA256SUMS`; la única garantía es el TLS de github.com. |
| Binario | `/usr/local/bin/qdrant`, instalado con `install(1)`. El tarball contiene sólo el binario. |
| Usuario | Usuario y grupo de sistema `qdrant`, sin login. |
| Estado | `/var/lib/qdrant/storage` y `/var/lib/qdrant/snapshots` (0750), del usuario del servicio. |
| Config | `/etc/qdrant/qdrant.env` (0640 `root:qdrant`), leído por `EnvironmentFile=`. **No hay `config.yaml`**: el release no trae ninguno, Qdrant arranca con sus defaults compilados (lo avisa por log) y las `QDRANT__*` del env file pisan lo que importa. Va en un archivo aparte y no en `Environment=` dentro de la unit porque la unit es 0644 y la api key no puede ser legible por todos. |
| Bind | `0.0.0.0` cuando `qdrant.public = true` (y abre los dos puertos en el firewall); `127.0.0.1` y sin tocar el firewall cuando es `false`. |
| systemd | `qdrant.service`, endurecida (`ProtectSystem=strict`, `NoNewPrivileges`, `SystemCallFilter=@system-service`, …), habilitada al boot. |
| Verificación | Espera a que la unit quede `active` y además sondea `GET /readyz`, que Qdrant deja fuera del api key justo para esto. `is-active` sólo dice que el proceso vive; el sondeo confirma que sirve en el puerto configurado. |

### Claves de `[qdrant]`

```toml
[qdrant]
host      = ""       # lo escribe el script si está vacío
http_port = 14014
grpc_port = 14015
public    = true
```

- `http_port` / `grpc_port` — Qdrant expone REST y gRPC en **dos puertos distintos**
  (defaults propios: 6333 y 6334). El SDK Go oficial habla gRPC. Si son iguales, aborta.
- `public` — decide el bind y si se abren los puertos, igual que en `[fareward]`.
- `host` — si ya tiene valor no se toca, porque la
  dirección por la que el backend entra de verdad (IP pública, dominio o túnel) no se
  deduce desde el host. Sólo cuando está vacío escribe la IP detectada, y avisa si es
  privada. **No lleva puerto** — los puertos ya están arriba, como en `[db]`.

### Api key

Qdrant sin api key no autentica nada: cualquiera que alcance el puerto lee y escribe
todas las colecciones. El script usa **`internal_apikey`** de la raíz de `config.toml`,
el mismo secreto proceso-a-proceso que ya usa el backend, en vez de generar una
credencial más. Está vacío → el script aborta. Los clientes la mandan en la cabecera
`api-key`.

Los endpoints `/healthz`, `/livez` y `/readyz` quedan fuera de la autenticación.

---

## Firewall

Cada modo abre su puerto (Qdrant, los dos suyos, y sólo si `public = true`) con el gestor
que **realmente** esté administrando netfilter en el host, en este orden:

1. **firewalld** — si está corriendo: `--permanent --add-port <puerto>/tcp` + `--reload`.
2. **ufw** — si está instalado y activo: `ufw allow <puerto>/tcp`.
3. **iptables plano** — solo si ninguno de los dos está gestionando. firewalld y ufw
   escriben en las mismas cadenas de netfilter, así que un `-I INPUT` a mano quedaría
   pisado en su próximo reload.

El caso 3 es el de las imágenes de **Oracle Cloud**, que traen una cadena `INPUT`
terminada en `-j REJECT --reject-with icmp-host-prohibited` y ni firewalld ni ufw. Ese
REJECT es lo que el cliente ve como `no route to host` (un DROP, en cambio, se manifiesta
como timeout).

Con iptables el script **solo actúa si el puerto está efectivamente cerrado**:

- Si ya hay un `ACCEPT` que cubre el puerto **antes** del REJECT/DROP → no hace nada.
  Reconoce `--dport`, `--dports` de multiport y rangos `low:high`.
- Si no hay ninguna regla que lo bloquee y la policy de `INPUT` es `ACCEPT` → no hace nada.
- Si está bloqueado, inserta el `ACCEPT` **justo antes** de la regla que lo tapaba
  (o al final de la cadena cuando lo que bloquea es la policy):

  ```bash
  iptables -I INPUT <n> -p tcp --dport <puerto> -m state --state NEW -j ACCEPT
  ```

  Después relee la cadena para confirmar que la regla quedó por delante del bloqueo, y
  la persiste con `netfilter-persistent save`, o escribiendo `/etc/iptables/rules.v4`
  (Debian/Ubuntu) o `/etc/sysconfig/iptables` (RHEL/Oracle Linux).

Una regla de bloqueo acotada a otros puertos no cuenta como bloqueo. Si ningún gestor
puede confirmar el puerto abierto, el script lo avisa fuerte en vez de seguir en silencio:
un servicio instalado pero inalcanzable es el síntoma más caro de diagnosticar.

> Esto cubre el firewall **del host**. Si el proveedor tiene además un filtro de red
> (Security List / NSG en OCI, Security Group en AWS), hay que abrir el puerto ahí
> también; ese descarte es silencioso y se ve como timeout, no como `no route to host`.

## IP de broadcast

Todos los modos publican la misma dirección: el script prefiere la IP de la tailnet
(Tailscale o Headscale, detectada por la CLI o por los rangos `100.64.0.0/10` y
`fd7a:115c:a1e0::/48` en las interfaces) y cae a la IP interna de la VPC si no hay
túnel. Esa IP es la que va a `broadcast_rpc_address` de Scylla y a `qdrant.host`.
