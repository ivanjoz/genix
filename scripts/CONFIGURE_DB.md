# Configuración de los motores de datos (`configure_db.py`)

Un solo script deja listo el host de datos: **ScyllaDB** (base de datos principal) y
**GenixSearch** (búsqueda léxica). Ambos suelen vivir en la misma VPS, separados del
backend, que corre en Lambda o en otro servidor.

```bash
sudo ./app.sh configure_db            # ambos (default)
sudo ./app.sh configure_db scylla     # solo ScyllaDB
sudo ./app.sh configure_db search     # solo GenixSearch
```

Los alias numéricos `1`/`2`/`3` equivalen a `all`/`scylla`/`search`. El script debe
correr como `root` y lee `credentials.json` de la raíz del repositorio.

---

## ScyllaDB (`scylla`)

| Paso | Detalle |
| ---- | ------- |
| `SCYLLA_ARGS` | Reescribe el env-file (`/etc/sysconfig/scylla-server` o `/etc/default/scylla-server`) con `--smp` acotado a los CPU disponibles, `-m 4G` y `--overprovisioned`. Limpia los flags que gestionó una corrida anterior. |
| `scylla.yaml` | `listen_address=127.0.0.1`, `rpc_address=0.0.0.0`, `broadcast_rpc_address=<IP alcanzable>`, `native_transport_port=DB_PORT` y `authenticator=PasswordAuthenticator`. |
| Firewall | Abre `DB_PORT/tcp` con firewalld o ufw, el que esté activo. |
| Superusuario | Prueba la password de `credentials.json` y la default `cassandra`. Si ninguna entra, recupera el rol por el **maintenance socket** de Scylla (ver abajo) y luego la cambia a `DB_PASSWORD`. |
| Keyspace | Crea `DB_NAME` con `NetworkTopologyStrategy` y `replication_factor: 1` si no existe. |

Claves usadas: `DB_PASSWORD`, `DB_NAME`, `DB_PORT`.

### Recuperación del superusuario

Scylla siembra el rol `cassandra` solo en el bootstrap inicial del cluster. Si el nodo
arrancó por primera vez con `AllowAllAuthenticator`, activar `PasswordAuthenticator`
después lo deja sin ningún rol con el que autenticarse. El script habilita entonces
`maintenance_socket: workdir`, reinicia el servicio y publica ese socket unix como un
puerto TCP efímero en `127.0.0.1` (el `cqlsh` empaquetado para Ubuntu no acepta
`--maintenance-socket`), por donde crea o corrige el rol sin autenticación.

---

## GenixSearch (`search`)

Instala el daemon desde los **paquetes estáticos** que publica
[`ivanjoz/genix-search`](https://github.com/ivanjoz/genix-search/releases): son builds
musl, así que el host no necesita toolchain ni una glibc concreta.

| Paso | Detalle |
| ---- | ------- |
| Arquitectura | Elige el asset según `uname -m`: `x86_64-linux-musl` o `aarch64-linux-musl`, y la variante `-neoverse-n1` cuando el CPU es un Ampere Altra (MIDR part `0xd0c`). **Cualquier otra arquitectura aborta**: no hay build publicado. |
| Descarga | Baja el `.tar.gz` del release (`--release-version`, default el último) y lo verifica contra el `SHA256SUMS` del propio release. Un mismatch aborta. |
| Binario | `/usr/local/bin/genixsearch`, instalado con `install(1)` para reemplazar el inodo: un servicio en ejecución sigue con su binario viejo hasta reiniciarse. |
| Usuario | Usuario y grupo de sistema `genixsearch`, sin login. |
| Estado | `/var/lib/genixsearch/store/kv` (0750), del usuario del servicio. |
| Config | `/etc/genixsearch/genixsearch.cfg` (0640 `root:genixsearch`). Se parte del `.cfg` de referencia que trae el release y solo se sobrescriben `server.log_level`, `channel.inet`, `channel.auth_password` y `store.kv.path`; así se heredan las claves nuevas que agregue GenixSearch. |
| Bind | `0.0.0.0:<puerto>` y el puerto abierto en el firewall, porque el backend en Lambda entra desde fuera del host. |
| systemd | `genixsearch.service`, endurecida (`ProtectSystem=strict`, `NoNewPrivileges`, `SystemCallFilter=@system-service`, …), habilitada al boot. El script espera a que la unit quede `active` y vuelca el journal si se cae. |

### Credenciales que escribe

Al terminar, el script **actualiza `credentials.json`**:

- `GENIXSEARCH_URL` = `<IP alcanzable>:<puerto>`.
- `GENIXSEARCH_PASSWORD` — se reutiliza la existente; si no hay, genera una de 64
  caracteres (minúsculas + dígitos) y la escribe.

El backend Go las lee como `core.Env.GENIXSEARCH_URL` / `core.Env.GENIXSEARCH_PASSWORD`
(`backend/core/security.go`). El puerto sale de `GENIXSEARCH_URL` si ya trae uno; si no,
usa `14446`, el mismo default de `core.ParseGenixSearchURL`.

El archivo se reescribe con indentación de 4 espacios y se le devuelve la propiedad al
usuario que invocó `sudo`.

### Flags

- `--release-version v0.1.0` — fija el tag en vez de tomar el último release.
- `--binary ./genixsearch` — instala un binario local en vez de descargar. Toma el
  `.cfg` de referencia del repo hermano `../genix-search/genixsearch.cfg`.

---

## IP de broadcast

Ambos modos publican la misma dirección: el script prefiere la IP de la tailnet
(Tailscale o Headscale, detectada por la CLI o por los rangos `100.64.0.0/10` y
`fd7a:115c:a1e0::/48` en las interfaces) y cae a la IP interna de la VPC si no hay
túnel. Esa IP es la que va a `broadcast_rpc_address` de Scylla y a `GENIXSEARCH_URL`.
