#!/usr/bin/env python3
"""Configura los motores de datos que necesita el backend en un mismo host.

    sudo python3 scripts/configure.py 18                    # todos
    sudo python3 scripts/configure/configure_db.py mariadb  # un solo motor

MariaDB (primary): installs mariadb-server, writes the tuned server config, creates the
DB_NAME database and the [mariadb] user.

Qdrant: descarga el release musl estatico de qdrant/qdrant (sin toolchain ni
glibc en el host), configurado por variables QDRANT__* en un EnvironmentFile. Los puertos salen de [qdrant] y la
api key es internal_apikey, la misma que ya usa el backend proceso-a-proceso.
"""

import argparse
import os
import shutil
import subprocess
import tarfile
import tempfile
import time
import socket
import re
import sys
import json
import tomllib
import contextlib
import urllib.error
import urllib.request
from pathlib import Path
from shutil import which
import ipaddress
import shlex

from toml_config import get_config_value, read_config, set_config_values
# Vive en su propio modulo porque configure_fareward.py abre su puerto TCP con lo mismo, y
# ese script no tiene por que importar el instalador de la base de datos para pedir un socket.
from firewall_ports import ensure_tcp_port_open

PROJECT_ROOT_DIRECTORY = Path(__file__).resolve().parents[2]
PROJECT_CONFIG_FILE = PROJECT_ROOT_DIRECTORY / "config.toml"


EXECUTION_MODE_ALL = "all"
EXECUTION_MODE_QDRANT = "qdrant"
EXECUTION_MODE_MARIADB = "mariadb"

# Los alias numericos replican la convencion de configure_server.py. MariaDB is the database
# ("db" and "all" install it). 2 was ScyllaDB, which is no longer supported: the number stays
# unused rather than shifting the others.
EXECUTION_MODE_BY_ARGUMENT = {
    "1": EXECUTION_MODE_ALL,
    "all": EXECUTION_MODE_ALL,
    "3": EXECUTION_MODE_QDRANT,
    "qdrant": EXECUTION_MODE_QDRANT,
    "vector": EXECUTION_MODE_QDRANT,
    "4": EXECUTION_MODE_MARIADB,
    "mariadb": EXECUTION_MODE_MARIADB,
    "db": EXECUTION_MODE_MARIADB,
}

DEFAULT_MARIADB_PORT = 3306
# Debian-family servers read conf.d snippets from here, RHEL-family from /etc/my.cnf.d.
MARIADB_CONFIG_DIRECTORIES = (Path("/etc/mysql/mariadb.conf.d"), Path("/etc/my.cnf.d"))
MARIADB_CONFIG_FILENAME = "90-genix.cnf"

SYSTEMD_UNIT_DIRECTORY = Path("/etc/systemd/system")

def print_debug_block(block_title, block_content):
    print(f"[*] {block_title}")
    if not block_content:
        print("    (empty)")
        return

    for block_line in block_content.rstrip().splitlines():
        print(f"    {block_line}")

def print_file_debug_lines(file_path, interesting_patterns):
    try:
        with open(file_path, 'r') as debug_file:
            debug_lines = debug_file.readlines()
    except OSError as file_error:
        print(f"[!] Failed to read {file_path}: {file_error}")
        return

    print(f"[*] Debug snapshot from {file_path}:")
    matched_any_line = False
    for line_number, line_text in enumerate(debug_lines, start=1):
        if any(re.search(pattern, line_text) for pattern in interesting_patterns):
            print(f"    {line_number}: {line_text.rstrip()}")
            matched_any_line = True

    if not matched_any_line:
        print("    (no matching lines)")

def print_service_failure_debug(service_name):
    service_status_command = ["systemctl", "status", service_name, "--no-pager", "--full"]
    service_journal_command = ["journalctl", "-u", service_name, "-n", "50", "--no-pager"]

    service_status_result = subprocess.run(service_status_command, text=True, capture_output=True)
    print_debug_block(f"systemctl status {service_name}", service_status_result.stdout or service_status_result.stderr)

    service_journal_result = subprocess.run(service_journal_command, text=True, capture_output=True)
    print_debug_block(f"journalctl -u {service_name} -n 50", service_journal_result.stdout or service_journal_result.stderr)

def run_capture_command(command_arguments, quiet=False):
    if not quiet:
        print(f"[*] Running argv command: {' '.join(command_arguments)}")
    command_result = subprocess.run(command_arguments, text=True, capture_output=True)
    if quiet:
        return command_result

    print(f"[*] Exit code: {command_result.returncode}")
    if command_result.stdout.strip():
        print_debug_block("stdout", command_result.stdout)
    if command_result.stderr.strip():
        print_debug_block("stderr", command_result.stderr)
    return command_result


def run_command(command, ignore_errors=False):
    print(f"[*] Running: {command}")
    result = subprocess.run(command, shell=True, text=True, capture_output=True)
    print(f"[*] Exit code: {result.returncode}")
    if result.stdout.strip():
        print_debug_block("stdout", result.stdout)
    if result.stderr.strip():
        print_debug_block("stderr", result.stderr)
    if result.returncode != 0 and not ignore_errors:
        failed_command_output = result.stderr.strip() or result.stdout.strip() or "(no output)"
        print(f"[!] Command failed: {failed_command_output}")
        sys.exit(1)
    return result.stdout

def get_internal_ip():
    """Gets the internal IP address of the machine in the VPC."""
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        # Doesn't have to be reachable, just forces the socket to route
        s.connect(('10.255.255.255', 1))
        ip = s.getsockname()[0]
    except Exception:
        ip = '127.0.0.1'
    finally:
        s.close()
    return ip

def run_json_command(command_arguments):
    """Runs a command expected to return JSON and parses it safely."""
    try:
        command_result = subprocess.run(
            command_arguments,
            text=True,
            capture_output=True,
            check=True
        )
    except FileNotFoundError:
        print(f"[*] JSON command not found: {' '.join(command_arguments)}")
        return None
    except subprocess.CalledProcessError as command_error:
        print(f"[*] JSON command failed: {' '.join(command_arguments)} (exit {command_error.returncode})")
        if command_error.stdout:
            print_debug_block("json stdout", command_error.stdout)
        if command_error.stderr:
            print_debug_block("json stderr", command_error.stderr)
        return None

    try:
        return json.loads(command_result.stdout)
    except json.JSONDecodeError:
        print_debug_block(f"invalid JSON from {' '.join(command_arguments)}", command_result.stdout)
        return None

def is_tailnet_ip(candidate_ip_address):
    """Matches the standard Tailscale/Headscale address ranges."""
    try:
        parsed_ip_address = ipaddress.ip_address(candidate_ip_address)
    except ValueError:
        return False

    if parsed_ip_address.version == 4:
        return parsed_ip_address in ipaddress.ip_network("100.64.0.0/10")

    return parsed_ip_address in ipaddress.ip_network("fd7a:115c:a1e0::/48")

def detect_tailnet_ip_from_interfaces():
    """Detects a tailnet IP from local interfaces when the CLI is unavailable."""
    interface_address_data = run_json_command(["ip", "-j", "addr", "show"])
    if not interface_address_data:
        return None

    for interface_details in interface_address_data:
        interface_name = interface_details.get("ifname", "")
        interface_addresses = interface_details.get("addr_info") or []
        for interface_address in interface_addresses:
            candidate_ip_address = interface_address.get("local")
            if candidate_ip_address and is_tailnet_ip(candidate_ip_address):
                print(f"[*] Detected tailnet IP from interface {interface_name}: {candidate_ip_address}")
                return candidate_ip_address

    return None

def detect_tailnet_network():
    """Detects Tailscale or Headscale connectivity and returns its IP when available."""
    if which("tailscale") is None:
        detected_tailnet_ip = detect_tailnet_ip_from_interfaces()
        if detected_tailnet_ip:
            detected_network_provider = "headscale" if which("headscale") else "tailnet"
            return detected_network_provider, detected_tailnet_ip

        print("[*] tailscale CLI not found. Falling back to local network IP detection.")
        return None, None

    tailscale_status_data = run_json_command(["tailscale", "status", "--json"])
    if not tailscale_status_data:
        print("[*] tailscale status unavailable. Falling back to local network IP detection.")
        return None, None

    current_node_state = tailscale_status_data.get("Self") or {}
    tailscale_ip_addresses = current_node_state.get("TailscaleIPs") or []
    if not tailscale_ip_addresses:
        print("[*] tailscale is installed but no tunnel IP is active. Falling back to local network IP detection.")
        return None, None

    tailscale_preferences_data = run_json_command(["tailscale", "debug", "prefs"]) or {}
    configured_control_url = str(tailscale_preferences_data.get("ControlURL") or "").strip().lower()

    # Headscale typically uses a self-hosted control URL instead of Tailscale's SaaS control plane.
    if configured_control_url and "login.tailscale.com" not in configured_control_url:
        detected_network_provider = "headscale"
    else:
        detected_network_provider = "tailscale"

    selected_tunnel_ip = tailscale_ip_addresses[0]
    print(f"[*] Detected {detected_network_provider} tunnel IP: {selected_tunnel_ip}")
    return detected_network_provider, selected_tunnel_ip

def get_preferred_broadcast_ip():
    detected_network_provider, detected_network_ip = detect_tailnet_network()
    if detected_network_ip:
        return detected_network_provider, detected_network_ip

    fallback_internal_ip = get_internal_ip()
    print(f"[*] Using local internal IP: {fallback_internal_ip}")
    return "local", fallback_internal_ip

def load_project_credentials():
    """Reads config.toml once; both configuration modes take their values from it."""
    print(f"[*] Loading config from: {PROJECT_CONFIG_FILE}")

    if not PROJECT_CONFIG_FILE.exists():
        print("[!] config.toml not found in project root.")
        print("    Create it from config.example.toml and set db.password/db.port.")
        sys.exit(1)

    try:
        credentials_data = read_config(PROJECT_CONFIG_FILE)
    except tomllib.TOMLDecodeError as toml_parse_error:
        print(f"[!] Invalid TOML in config.toml: {toml_parse_error}")
        sys.exit(1)

    return credentials_data

def save_project_credentials(credentials_updates):
    """Merges dotted keys back into config.toml keeping the file owned by the invoking user."""
    set_config_values(PROJECT_CONFIG_FILE, credentials_updates)

    # El script corre bajo sudo: sin este chown el archivo del repo queda de root.
    sudo_user_id = os.environ.get("SUDO_UID")
    sudo_group_id = os.environ.get("SUDO_GID")
    if sudo_user_id and sudo_group_id:
        with contextlib.suppress(OSError, ValueError):
            os.chown(PROJECT_CONFIG_FILE, int(sudo_user_id), int(sudo_group_id))

    print(f"[*] Saved {', '.join(credentials_updates)} into {PROJECT_CONFIG_FILE}")


def get_available_processing_units():
    # Prefer the scheduler affinity mask because it reflects CPU restrictions better than os.cpu_count().
    try:
        available_processing_units = len(os.sched_getaffinity(0))
        if available_processing_units > 0:
            return available_processing_units
    except AttributeError:
        pass

    detected_processing_units = os.cpu_count() or 1
    return max(1, detected_processing_units)




# ---------------------------------------------------------------------------
# Service install helpers (release download, system user, hardened unit)
# ---------------------------------------------------------------------------

NOLOGIN_SHELL_CANDIDATES = ("/usr/sbin/nologin", "/sbin/nologin", "/usr/bin/false", "/bin/false")

def http_get_bytes(request_url, timeout_seconds=60.0):
    # GitHub rechaza las peticiones sin User-Agent.
    http_request = urllib.request.Request(request_url, headers={"User-Agent": "genix-configure-db"})
    try:
        with urllib.request.urlopen(http_request, timeout=timeout_seconds) as http_response:
            return http_response.read()
    except urllib.error.HTTPError as http_error:
        print(f"[!] HTTP {http_error.code} fetching {request_url}")
        sys.exit(1)
    except urllib.error.URLError as url_error:
        print(f"[!] Could not fetch {request_url}: {url_error.reason}")
        sys.exit(1)

def resolve_latest_release_tag(release_repository):
    print(f"[*] Resolving the latest release of {release_repository}...")
    release_payload = http_get_bytes(f"https://api.github.com/repos/{release_repository}/releases/latest")
    try:
        release_tag = json.loads(release_payload)["tag_name"]
    except (ValueError, KeyError):
        print(f"[!] Could not read the latest release tag for {release_repository}.")
        sys.exit(1)

    print(f"[*] Latest published release: {release_tag}")
    return release_tag

def ensure_service_system_user(service_name, service_home_directory):
    """Crea el usuario/grupo de sistema sin login bajo el que corre el daemon."""
    if subprocess.run(["id", service_name], capture_output=True).returncode == 0:
        print(f"[*] System user '{service_name}' already exists.")
        return

    nologin_shell = next((shell_path for shell_path in NOLOGIN_SHELL_CANDIDATES if Path(shell_path).exists()), "/bin/false")
    print(f"[*] Creating system user '{service_name}'...")
    run_command(
        f"useradd --system --user-group --home-dir {service_home_directory} "
        f"--no-create-home --shell {nologin_shell} {service_name}"
    )

def prepare_service_directories(service_name, directory_paths):
    """Crea cada directorio 0750 y se lo entrega al usuario del servicio.

    El orden de la lista importa: los padres van primero para que el chmod del padre no
    pise el del hijo creado con parents=True.
    """
    for directory_path in directory_paths:
        print(f"[*] Preparing state directory {directory_path} (0750 {service_name}:{service_name})")
        directory_path.mkdir(parents=True, exist_ok=True)
        directory_path.chmod(0o750)
        shutil.chown(directory_path, user=service_name, group=service_name)

def install_service_binary(source_binary_path, target_binary_path):
    print(f"[*] Installing {source_binary_path} -> {target_binary_path}")
    target_binary_path.parent.mkdir(parents=True, exist_ok=True)
    # install(1) en vez de cp: reemplaza el inodo, asi un servicio en ejecucion sigue
    # usando su binario actual hasta que se reinicia.
    run_command(f"install -m 0755 -o root -g root {shlex.quote(str(source_binary_path))} {target_binary_path}")

def write_hardened_service_unit(
    service_name,
    unit_description,
    working_directory,
    exec_start_command,
    state_directory,
    needs_privileged_port=False,
    extra_service_lines=(),
):
    """Escribe la unit de systemd de un daemon que solo escucha un socket y escribe su estado.

    El bloque de hardening es identico para todos los servicios que instala este script,
    asi que vive en un solo sitio: lo que cambia entre uno y otro entra por parametro.
    """
    # Un puerto privilegiado necesita la capability; con los puertos altos no hace falta.
    service_capabilities = "CAP_NET_BIND_SERVICE" if needs_privileged_port else ""
    extra_service_block = "".join(f"{extra_service_line}\n" for extra_service_line in extra_service_lines)
    unit_file_path = SYSTEMD_UNIT_DIRECTORY / f"{service_name}.service"

    unit_content = f"""\
# {service_name} systemd unit — written by scripts/configure/configure_db.py.
[Unit]
Description={unit_description}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={service_name}
Group={service_name}
WorkingDirectory={working_directory}
ExecStart={exec_start_command}
Restart=on-failure
RestartSec=2s
LimitNOFILE=65536
SyslogIdentifier={service_name}
{extra_service_block}
# Hardening: el proceso solo escucha en un socket y escribe en su directorio de estado.
AmbientCapabilities={service_capabilities}
CapabilityBoundingSet={service_capabilities}
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelModules=true
ProtectKernelTunables=true
ProtectControlGroups=true
ReadWritePaths={state_directory}
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallArchitectures=native
SystemCallFilter=@system-service

[Install]
WantedBy=multi-user.target
"""
    print(f"[*] Writing {unit_file_path}")
    unit_file_path.write_text(unit_content, encoding="utf-8")
    unit_file_path.chmod(0o644)

def restore_selinux_context(target_paths):
    """Relabels the installed files when SELinux is enforcing (Fedora, RHEL, ...)."""
    if which("selinuxenabled") is None or which("restorecon") is None:
        return
    if subprocess.run(["selinuxenabled"], capture_output=True).returncode != 0:
        return

    print("[*] Restoring SELinux contexts...")
    run_command(f"restorecon -R -F {' '.join(str(target_path) for target_path in target_paths)}", ignore_errors=True)

def wait_for_service_active(service_name, timeout_seconds=15):
    """Polls is-active so a crash-on-startup surfaces here and not hours later."""
    deadline_timestamp = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline_timestamp:
        service_state = subprocess.run(
            ["systemctl", "is-active", f"{service_name}.service"], text=True, capture_output=True
        ).stdout.strip()
        if service_state == "active":
            # Un segundo de gracia para atrapar un proceso que muere apenas arranca.
            time.sleep(1)
            return subprocess.run(
                ["systemctl", "is-active", f"{service_name}.service"], text=True, capture_output=True
            ).stdout.strip() == "active"
        if service_state == "failed":
            return False
        time.sleep(0.5)
    return False

# ---------------------------------------------------------------------------
# Qdrant
# ---------------------------------------------------------------------------

QDRANT_SERVICE_NAME = "qdrant"
QDRANT_BINARY_PATH = Path("/usr/local/bin") / QDRANT_SERVICE_NAME
QDRANT_CONFIG_DIRECTORY = Path("/etc") / QDRANT_SERVICE_NAME
QDRANT_ENVIRONMENT_FILE = QDRANT_CONFIG_DIRECTORY / f"{QDRANT_SERVICE_NAME}.env"
QDRANT_DATA_DIRECTORY = Path("/var/lib") / QDRANT_SERVICE_NAME
QDRANT_STORAGE_DIRECTORY = QDRANT_DATA_DIRECTORY / "storage"
QDRANT_SNAPSHOTS_DIRECTORY = QDRANT_DATA_DIRECTORY / "snapshots"
QDRANT_RELEASE_REPOSITORY = "qdrant/qdrant"

# Defaults del propio Qdrant, usados cuando [qdrant] no declara los puertos.
DEFAULT_QDRANT_HTTP_PORT = 6333
DEFAULT_QDRANT_GRPC_PORT = 6334

# Solo hay builds Linux musl para estas dos arquitecturas en los releases de Qdrant.
QDRANT_ASSET_SUFFIX_BY_MACHINE = {
    "x86_64": "x86_64-unknown-linux-musl",
    "amd64": "x86_64-unknown-linux-musl",
    "aarch64": "aarch64-unknown-linux-musl",
    "arm64": "aarch64-unknown-linux-musl",
}

def detect_qdrant_asset_suffix():
    machine_architecture = os.uname().machine
    asset_suffix = QDRANT_ASSET_SUFFIX_BY_MACHINE.get(machine_architecture)
    if asset_suffix is None:
        print(f"[!] Architecture '{machine_architecture}' has no published Qdrant musl build.")
        print(f"    Published static builds: {', '.join(sorted(set(QDRANT_ASSET_SUFFIX_BY_MACHINE.values())))}")
        sys.exit(1)

    print(f"[*] Detected architecture '{machine_architecture}' -> asset {asset_suffix}")
    return asset_suffix

def download_qdrant_release(release_version, download_directory):
    """Descarga y desempaqueta el release musl. Devuelve la ruta del binario extraido."""
    asset_suffix = detect_qdrant_asset_suffix()
    if release_version in (None, "", "latest"):
        release_version = resolve_latest_release_tag(QDRANT_RELEASE_REPOSITORY)
    if not release_version.startswith("v"):
        release_version = f"v{release_version}"

    archive_name = f"{QDRANT_SERVICE_NAME}-{asset_suffix}.tar.gz"
    archive_url = (
        f"https://github.com/{QDRANT_RELEASE_REPOSITORY}/releases/download/{release_version}/{archive_name}"
    )

    print(f"[*] Downloading {archive_url}")
    # Los releases de Qdrant no publican un SHA256SUMS, asi
    # que aqui no hay checksum contra el que validar: la unica garantia es el TLS de GitHub.
    archive_path = download_directory / archive_name
    archive_path.write_bytes(http_get_bytes(archive_url, timeout_seconds=300.0))

    with tarfile.open(archive_path, "r:gz") as release_archive:
        try:
            # Python 3.12+ rechaza miembros inseguros; las versiones anteriores no
            # conocen el parametro, de ahi el fallback.
            release_archive.extractall(download_directory, filter="data")
        except TypeError:
            release_archive.extractall(download_directory)

    # El tarball trae solo el binario en la raiz, pero el layout ha cambiado entre
    # versiones, asi que se busca en vez de asumir la ruta.
    downloaded_binary_path = next(
        (
            candidate_path
            for candidate_path in sorted(download_directory.rglob(QDRANT_SERVICE_NAME))
            if candidate_path.is_file()
        ),
        None,
    )
    if downloaded_binary_path is None:
        print(f"[!] {archive_name} did not contain a '{QDRANT_SERVICE_NAME}' binary.")
        sys.exit(1)

    downloaded_binary_path.chmod(0o755)
    return downloaded_binary_path

def read_port_setting(credentials_data, setting_path, default_port):
    configured_port = get_config_value(credentials_data, setting_path, default_port)
    try:
        normalized_port = int(configured_port)
    except (TypeError, ValueError):
        print(f"[!] {setting_path} in config.toml must be a valid integer.")
        sys.exit(1)

    if not (1 <= normalized_port <= 65535):
        print(f"[!] {setting_path} must be between 1 and 65535.")
        sys.exit(1)
    return normalized_port

def resolve_qdrant_settings(credentials_data):
    """Valida los dos puertos, el bind y la api key de Qdrant antes de tocar el host."""
    http_port = read_port_setting(credentials_data, "qdrant.http_port", DEFAULT_QDRANT_HTTP_PORT)
    grpc_port = read_port_setting(credentials_data, "qdrant.grpc_port", DEFAULT_QDRANT_GRPC_PORT)
    if http_port == grpc_port:
        print("[!] qdrant.http_port and qdrant.grpc_port must be different ports.")
        sys.exit(1)

    is_public = bool(get_config_value(credentials_data, "qdrant.public", False))

    # Qdrant sin api key no autentica nada: cualquiera que alcance el puerto lee y escribe
    # todas las colecciones. Se reutiliza internal_apikey, el secreto proceso-a-proceso que
    # el backend ya tiene cargado, en vez de inventar una credencial mas.
    qdrant_api_key = str(get_config_value(credentials_data, "internal_apikey") or "").strip()
    if not qdrant_api_key:
        print("[!] internal_apikey must be set in config.toml: it is the API key Qdrant will require.")
        sys.exit(1)

    # El valor viaja en un EnvironmentFile de systemd, que parte por lineas y quita comillas.
    if any(forbidden_character in qdrant_api_key for forbidden_character in ('"', "'", "\\", "\n")):
        print("[!] internal_apikey cannot contain quotes, backslashes or newlines.")
        sys.exit(1)

    return http_port, grpc_port, is_public, qdrant_api_key

def resolve_qdrant_host(credentials_data, broadcast_ip_address):
    """Devuelve (host, se_genero) para la direccion por la que se alcanza Qdrant.

    Un valor ya presente es una decision del operador y no se toca, porque la direccion
    real (IP publica, dominio o tunel) no se deduce desde el host. No lleva puerto: los
    puertos ya estan en [qdrant], como en [db].
    """
    configured_host = str(get_config_value(credentials_data, "qdrant.host") or "").strip()
    if configured_host:
        print(f"[*] Reusing qdrant.host from config.toml: {configured_host}")
        return configured_host, False

    print(f"[*] No qdrant.host in config.toml. Using the detected address: {broadcast_ip_address}")
    with contextlib.suppress(ValueError):
        if ipaddress.ip_address(broadcast_ip_address).is_private and not is_tailnet_ip(broadcast_ip_address):
            print(f"[!] {broadcast_ip_address} is a private address. A backend outside this network")
            print("    will not reach it: set qdrant.host in config.toml to the public host and re-run.")

    return broadcast_ip_address, True

def write_qdrant_environment_file(bind_address, http_port, grpc_port, qdrant_api_key):
    """Escribe las QDRANT__*, que es toda la configuracion del servicio.

    El tarball del release trae unicamente el binario: no hay config.yaml que editar. Qdrant
    avisa por log de que no lo encuentra, arranca con sus defaults compilados y estas
    variables pisan las que nos importan.

    Van en un EnvironmentFile 0640 y no en 'Environment=' dentro de la unit porque la unit
    es legible por todo el mundo (0644) y la api key no puede estarlo.
    """
    environment_lines = [
        f"QDRANT__SERVICE__HOST={bind_address}",
        f"QDRANT__SERVICE__HTTP_PORT={http_port}",
        f"QDRANT__SERVICE__GRPC_PORT={grpc_port}",
        f"QDRANT__SERVICE__API_KEY={qdrant_api_key}",
        f"QDRANT__STORAGE__STORAGE_PATH={QDRANT_STORAGE_DIRECTORY}",
        f"QDRANT__STORAGE__SNAPSHOTS_PATH={QDRANT_SNAPSHOTS_DIRECTORY}",
        "QDRANT__TELEMETRY_DISABLED=true",
    ]

    print(f"[*] Writing {QDRANT_ENVIRONMENT_FILE} (0640 root:{QDRANT_SERVICE_NAME})")
    QDRANT_ENVIRONMENT_FILE.parent.mkdir(parents=True, exist_ok=True)
    QDRANT_ENVIRONMENT_FILE.write_text(
        "# Written by scripts/configure/configure_db.py. Replaced on every run.\n"
        + "\n".join(environment_lines)
        + "\n",
        encoding="utf-8",
    )
    QDRANT_ENVIRONMENT_FILE.chmod(0o640)
    shutil.chown(QDRANT_ENVIRONMENT_FILE, user="root", group=QDRANT_SERVICE_NAME)

def wait_for_qdrant_readiness(http_port, timeout_seconds=60):
    """Sondea /readyz, que Qdrant deja fuera del api key justo para esto.

    'systemctl is-active' solo dice que el proceso vive; esto confirma que ademas esta
    sirviendo en el puerto configurado, que es lo que el backend necesita.
    """
    readiness_url = f"http://127.0.0.1:{http_port}/readyz"
    print(f"[*] Waiting for {readiness_url} ...")
    deadline_timestamp = time.monotonic() + timeout_seconds
    last_probe_error = "(no attempt completed)"

    while time.monotonic() < deadline_timestamp:
        try:
            with urllib.request.urlopen(readiness_url, timeout=5) as probe_response:
                if probe_response.status == 200:
                    print("[*] Qdrant reports ready.")
                    return True
                last_probe_error = f"HTTP {probe_response.status}"
        except (urllib.error.URLError, OSError) as probe_error:
            last_probe_error = str(probe_error)
        time.sleep(2)

    print(f"[!] {readiness_url} never answered 200: {last_probe_error}")
    return False

def configure_qdrant(credentials_data, broadcast_ip_address, release_version):
    http_port, grpc_port, is_public, qdrant_api_key = resolve_qdrant_settings(credentials_data)
    qdrant_host, was_host_generated = resolve_qdrant_host(credentials_data, broadcast_ip_address)
    # public decide el bind, igual que en [fareward]: expuesto o solo loopback.
    bind_address = "0.0.0.0" if is_public else "127.0.0.1"

    with tempfile.TemporaryDirectory(prefix="qdrant-release-") as download_directory_name:
        source_binary_path = download_qdrant_release(release_version, Path(download_directory_name))
        ensure_service_system_user(QDRANT_SERVICE_NAME, QDRANT_DATA_DIRECTORY)
        prepare_service_directories(
            QDRANT_SERVICE_NAME,
            [QDRANT_DATA_DIRECTORY, QDRANT_STORAGE_DIRECTORY, QDRANT_SNAPSHOTS_DIRECTORY],
        )
        install_service_binary(source_binary_path, QDRANT_BINARY_PATH)

    write_qdrant_environment_file(bind_address, http_port, grpc_port, qdrant_api_key)
    write_hardened_service_unit(
        QDRANT_SERVICE_NAME,
        "Qdrant — vector database",
        QDRANT_DATA_DIRECTORY,
        str(QDRANT_BINARY_PATH),
        QDRANT_DATA_DIRECTORY,
        needs_privileged_port=min(http_port, grpc_port) < 1024,
        extra_service_lines=(f"EnvironmentFile={QDRANT_ENVIRONMENT_FILE}",),
    )
    restore_selinux_context([QDRANT_BINARY_PATH, QDRANT_CONFIG_DIRECTORY, QDRANT_DATA_DIRECTORY])

    if is_public:
        ensure_tcp_port_open(http_port, "Qdrant HTTP")
        ensure_tcp_port_open(grpc_port, "Qdrant gRPC")
    else:
        print("[*] qdrant.public = false: bound to 127.0.0.1, so no firewall port is opened.")

    installed_version_result = run_capture_command([str(QDRANT_BINARY_PATH), "--version"], quiet=True)
    print(f"[*] Installed binary reports: {(installed_version_result.stdout or installed_version_result.stderr).strip()}")

    run_command("systemctl daemon-reload")
    run_command(f"systemctl enable {QDRANT_SERVICE_NAME}.service")
    run_command(f"systemctl restart {QDRANT_SERVICE_NAME}.service")

    if not wait_for_service_active(QDRANT_SERVICE_NAME) or not wait_for_qdrant_readiness(http_port):
        print(f"[!] {QDRANT_SERVICE_NAME}.service did not come up.")
        print_service_failure_debug(QDRANT_SERVICE_NAME)
        sys.exit(1)

    # Solo se escribe lo que el script dedujo; un qdrant.host puesto a mano no se pisa.
    if was_host_generated:
        save_project_credentials({"qdrant.host": qdrant_host})

    print("\n[+] Qdrant Installation Complete!")
    print(f"    - binary:  {QDRANT_BINARY_PATH}")
    print(f"    - env:     {QDRANT_ENVIRONMENT_FILE}")
    print(f"    - state:   {QDRANT_DATA_DIRECTORY}")
    print(f"    - listen:  {bind_address} (HTTP {http_port}, gRPC {grpc_port})")
    print(f"    - reach:   {qdrant_host}:{http_port}")
    print("    - api key: internal_apikey from config.toml (header 'api-key')")
    print(f"    - systemctl status {QDRANT_SERVICE_NAME}")

def read_total_memory_megabytes():
    with open("/proc/meminfo") as meminfo_file:
        for meminfo_line in meminfo_file:
            if meminfo_line.startswith("MemTotal:"):
                return int(meminfo_line.split()[1]) // 1024
    print("[!] Could not read MemTotal from /proc/meminfo.")
    sys.exit(1)

def build_mariadb_server_configuration(database_port, bind_address, total_memory_megabytes, processing_units):
    """The server the load tests measured (backend/tests/loadtest/run_postgres.sh, MARIADB_TUNING.md),
    scaled to this host.

    The buffer pool takes half the RAM: the backend (GOMEMLIMIT), HAProxy, fareward and the OS share the
    rest. innodb_flush_log_at_trx_commit=2 writes the redo log at every commit but fsyncs it once a
    second: a MariaDB crash loses nothing, an OS or VM crash loses up to ~1 s of commits (decided for
    the VPS on 2026-11-07). There is no binary log, so no point-in-time recovery: backups only. The
    thread pool runs statements on one thread per core instead of one per connection, which measured
    15% less CPU per sale on 2 vCPUs.
    """
    buffer_pool_megabytes = max(256, total_memory_megabytes // 2)
    return f"""# Written by scripts/configure/configure_db.py (mariadb mode). Edits are overwritten on the next run.
[mariadb]
port = {database_port}
bind-address = {bind_address}
skip-name-resolve = ON
max-connections = 150
# The ORM sends write statements of up to 8 MB, half of this.
max-allowed-packet = 16M
transaction-isolation = READ-COMMITTED

innodb-buffer-pool-size = {buffer_pool_megabytes}M
innodb-log-file-size = 512M
innodb-flush-log-at-trx-commit = 2
skip-log-bin

thread-handling = pool-of-threads
thread-pool-size = {processing_units}
performance-schema = OFF

# MyRocks keeps only the log tables (the ORM's TableSchema.Log: user_logs), compressed (Snappy), their
# rows expired by TTL and written without its WAL. Write-mostly, so a small block cache; loose- so the
# server still starts before the plugin is installed.
loose-rocksdb-block-cache-size = 64M
"""

def resolve_mariadb_settings(credentials_data):
    """[mariadb] host/port/user/password and [db] name, the same keys the backend reads."""
    database_name = get_config_value(credentials_data, "db.name", "genix")
    database_port = int(get_config_value(credentials_data, "mariadb.port", DEFAULT_MARIADB_PORT))
    database_host = str(get_config_value(credentials_data, "mariadb.host", "127.0.0.1"))
    database_user = str(get_config_value(credentials_data, "mariadb.user", "genix"))
    database_password = str(get_config_value(credentials_data, "mariadb.password", ""))
    if not database_password:
        print("[!] mariadb.password is empty in config.toml. Set it before running this mode.")
        sys.exit(1)
    if not re.fullmatch(r"[A-Za-z0-9_]+", database_name) or not re.fullmatch(r"[A-Za-z0-9_]+", database_user):
        print("[!] db.name and mariadb.user may only hold letters, digits and underscores.")
        sys.exit(1)
    # A backend on this host reaches it on loopback; any other host means it must listen outside.
    # public = true (as in [fareward] and [qdrant]) also listens outside while this host's backend
    # keeps dialing loopback, which is how a dev machine reaches the VPS database.
    is_public = bool(get_config_value(credentials_data, "mariadb.public", False))
    is_local = database_host in ("127.0.0.1", "localhost", "::1") and not is_public
    return database_name, database_port, database_user, database_password, is_local, is_public

def install_mariadb_server_package():
    if which("mariadbd") or which("mysqld"):
        print("[*] MariaDB server is already installed.")
        return
    if which("dnf"):
        run_command("dnf install -y mariadb-server")
    elif which("apt-get"):
        run_command("apt-get update")
        run_command("DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-server")
    else:
        print("[!] Neither dnf nor apt-get is available: install mariadb-server by hand, then rerun.")
        sys.exit(1)

def install_mariadb_rocksdb_plugin():
    """MyRocks, which the log tables need: the backend's deploy refuses to create them without it."""
    if which("dnf"):
        run_command("dnf install -y mariadb-rocksdb-engine")
    elif which("apt-get"):
        run_command("DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-plugin-rocksdb")
    else:
        print("[!] Neither dnf nor apt-get is available: install MariaDB's RocksDB plugin by hand, then rerun.")
        sys.exit(1)

def ensure_rocksdb_engine_loaded():
    """The packages load the plugin from their own config snippet; INSTALL SONAME covers one that does not."""
    engine_query = subprocess.run(
        ["mariadb", "--protocol=socket", "-uroot", "-N", "-B", "-e",
         "SELECT support FROM information_schema.engines WHERE engine = 'ROCKSDB'"],
        capture_output=True, text=True)
    if engine_query.stdout.strip() in ("YES", "DEFAULT"):
        return
    run_mariadb_root_statement("INSTALL SONAME 'ha_rocksdb'")

def resolve_mariadb_service_name():
    for service_name in ("mariadb", "mysql", "mysqld"):
        if run_capture_command(["systemctl", "cat", f"{service_name}.service"], quiet=True).returncode == 0:
            return service_name
    print("[!] No mariadb/mysql systemd service found after installing the package.")
    sys.exit(1)

def run_mariadb_root_statement(sql_statement):
    """Runs a statement as root over the local socket (unix_socket auth, the packages' default)."""
    statement_result = subprocess.run(["mariadb", "--protocol=socket", "-uroot", "-e", sql_statement],
                                      capture_output=True, text=True)
    if statement_result.returncode != 0:
        print(f"[!] MariaDB statement failed: {statement_result.stderr.strip()}")
        sys.exit(1)

def configure_mariadb(credentials_data, broadcast_ip_address):
    database_name, database_port, database_user, database_password, is_local, is_public = (
        resolve_mariadb_settings(credentials_data)
    )
    # Public listens on every interface: loopback for this host's backend, the VPC address the
    # cloud NAT forwards the public IP to.
    bind_address = "127.0.0.1" if is_local else "0.0.0.0" if is_public else broadcast_ip_address
    install_mariadb_server_package()
    install_mariadb_rocksdb_plugin()

    configuration_directory = next((directory for directory in MARIADB_CONFIG_DIRECTORIES if directory.is_dir()), None)
    if configuration_directory is None:
        print(f"[!] None of {[str(directory) for directory in MARIADB_CONFIG_DIRECTORIES]} exists.")
        sys.exit(1)
    configuration_path = configuration_directory / MARIADB_CONFIG_FILENAME
    total_memory_megabytes = read_total_memory_megabytes()
    configuration_path.write_text(build_mariadb_server_configuration(
        database_port, bind_address, total_memory_megabytes, get_available_processing_units()))
    os.chmod(configuration_path, 0o644)
    print_file_debug_lines(str(configuration_path), [r".*"])

    service_name = resolve_mariadb_service_name()
    run_command("systemctl daemon-reload")
    run_command(f"systemctl enable {service_name}.service")
    run_command(f"systemctl restart {service_name}.service")
    if not wait_for_service_active(service_name, timeout_seconds=60):
        print_service_failure_debug(service_name)
        sys.exit(1)

    ensure_rocksdb_engine_loaded()

    # The backend creates its own tables (and the ORM's side tables) in this database.
    # With skip-name-resolve, 'localhost' only matches the Unix socket ([mariadb].socket) and TCP
    # from loopback needs an account for the literal 127.0.0.1 or ::1. A public or remote server
    # adds '%' for the other machines; the local accounts stay for this host's backend.
    user_hosts = ("localhost", "127.0.0.1", "::1") + (() if is_local else ("%",))
    escaped_password = database_password.replace("\\", "\\\\").replace("'", "\\'")
    user_statements = "".join(
        f"CREATE USER IF NOT EXISTS '{database_user}'@'{user_host}' IDENTIFIED BY '{escaped_password}'; "
        f"ALTER USER '{database_user}'@'{user_host}' IDENTIFIED BY '{escaped_password}'; "
        f"GRANT ALL PRIVILEGES ON `{database_name}`.* TO '{database_user}'@'{user_host}'; "
        for user_host in user_hosts
    )
    run_mariadb_root_statement(
        f"CREATE DATABASE IF NOT EXISTS `{database_name}` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin; "
        + user_statements
    )
    if not is_local:
        ensure_tcp_port_open(database_port, "MariaDB")

    print("\n[+] MariaDB configured!")
    print(f"    - config:   {configuration_path}")
    print(f"    - listen:   {bind_address}:{database_port}")
    print(f"    - database: {database_name}, user '{database_user}'@{'/'.join(user_hosts)}")
    print(f"    - buffer pool: {max(256, total_memory_megabytes // 2)} MB of {total_memory_megabytes} MB RAM")
    print(f"    - systemctl status {service_name}")

def parse_command_arguments():
    argument_parser = argparse.ArgumentParser(
        prog="configure_db.py",
        description="Configures MariaDB and installs Qdrant on this host.",
    )
    argument_parser.add_argument(
        "mode",
        nargs="?",
        default=EXECUTION_MODE_ALL,
        help="all|1 (default: mariadb + qdrant), qdrant|3, mariadb|db|4",
    )
    argument_parser.add_argument(
        "--qdrant-version",
        default="latest",
        help="Qdrant release tag to install, e.g. v1.19.0 (default: latest)",
    )
    parsed_arguments = argument_parser.parse_args()

    execution_mode = EXECUTION_MODE_BY_ARGUMENT.get(parsed_arguments.mode.strip().lower())
    if execution_mode is None:
        argument_parser.error(f"Unknown mode: {parsed_arguments.mode}. Use all, mariadb or qdrant.")

    parsed_arguments.mode = execution_mode
    return parsed_arguments

def main():
    # Se parsea antes del chequeo de root para que --help funcione sin sudo.
    command_arguments = parse_command_arguments()

    if os.geteuid() != 0:
        print("[!] Please run this script with sudo.")
        sys.exit(1)

    print(f"[*] Execution mode: {command_arguments.mode}")

    credentials_data = load_project_credentials()
    _, broadcast_ip_address = get_preferred_broadcast_ip()

    if command_arguments.mode in (EXECUTION_MODE_ALL, EXECUTION_MODE_MARIADB):
        configure_mariadb(credentials_data, broadcast_ip_address)

    if command_arguments.mode in (EXECUTION_MODE_ALL, EXECUTION_MODE_QDRANT):
        configure_qdrant(credentials_data, broadcast_ip_address, command_arguments.qdrant_version)

if __name__ == "__main__":
    main()
