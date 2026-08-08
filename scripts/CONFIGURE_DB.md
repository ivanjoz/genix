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
correr como `root` y lee `config.toml` de la raíz del repositorio.

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

El script **solo escribe en `config.toml` los valores que faltaban**. Un valor ya
presente es una decisión del operador y no se toca:

- `search.url` — si ya tiene valor, se reutiliza tal cual. El script **no puede
  deducir** la dirección por la que el backend entra de verdad (IP pública, dominio o
  túnel): la IP que detecta el host suele ser la privada de la VPC, que desde Lambda no
  resuelve a nada. Solo cuando está vacía escribe la IP detectada, y avisa si es privada.
- `search.password` — se reutiliza la existente; si no hay, genera una de 64
  caracteres (minúsculas + dígitos) y la escribe.

El backend Go las lee como `core.Env.GENIXSEARCH_URL` / `core.Env.GENIXSEARCH_PASSWORD`
(`backend/core/security.go`), pobladas desde `search.url` / `search.password` de
`config.toml`. El puerto sale de `search.url` si ya trae uno; si no,
usa `14446`, el mismo default de `core.ParseGenixSearchURL`. Es decir: el puerto en el
que escucha el servicio se controla poniendo el puerto en `search.url`.

El archivo se reescribe con indentación de 4 espacios y se le devuelve la propiedad al
usuario que invocó `sudo`.

### Flags

- `--release-version v0.1.0` — fija el tag en vez de tomar el último release.
- `--binary ./genixsearch` — instala un binario local en vez de descargar. Toma el
  `.cfg` de referencia del repo hermano `../genix-search/genixsearch.cfg`.

---

## Firewall

Ambos modos abren su puerto con el gestor que **realmente** esté administrando netfilter
en el host, en este orden:

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

Ambos modos publican la misma dirección: el script prefiere la IP de la tailnet
(Tailscale o Headscale, detectada por la CLI o por los rangos `100.64.0.0/10` y
`fd7a:115c:a1e0::/48` en las interfaces) y cae a la IP interna de la VPC si no hay
túnel. Esa IP es la que va a `broadcast_rpc_address` de Scylla y a `search.url`.
