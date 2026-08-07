#!/usr/bin/env python3
"""Configura los dos motores de datos que necesita el backend en un mismo host.

    sudo ./app.sh configure_db            # ambos (equivale a 'all')
    sudo ./app.sh configure_db scylla     # solo ScyllaDB
    sudo ./app.sh configure_db search     # solo GenixSearch

ScyllaDB: ajusta sysconfig/scylla.yaml, abre el puerto CQL, recupera el
superusuario si hace falta y crea el keyspace de DB_NAME.

GenixSearch: descarga el binario musl estatico publicado en los releases de
GitHub (sin toolchain ni glibc en el host), lo verifica contra el SHA256SUMS
del release, escribe /etc/genixsearch/genixsearch.cfg y la unit de systemd, y
guarda GENIXSEARCH_URL / GENIXSEARCH_PASSWORD en credentials.json para que el
backend Go los lea desde core.Env.
"""

import argparse
import hashlib
import os
import shutil
import subprocess
import tarfile
import tempfile
import time
import socket
import re
import secrets
import string
import sys
import json
import threading
import contextlib
import urllib.error
import urllib.request
from pathlib import Path
from shutil import which
import ipaddress
import shlex

PROJECT_ROOT_DIRECTORY = Path(__file__).resolve().parents[1]
PROJECT_CREDENTIALS_FILE = PROJECT_ROOT_DIRECTORY / "credentials.json"
DEFAULT_SCYLLA_PORT = 9042
DEFAULT_CASSANDRA_USERNAME = "cassandra"
DEFAULT_CASSANDRA_PASSWORD = "cassandra"
SCYLLA_YAML_PATH = "/etc/scylla/scylla.yaml"
SCYLLA_DEFAULT_WORKDIR = "/var/lib/scylla"
MAINTENANCE_SOCKET_FILENAME = "cql.m"

# El puerto CQL se abre antes de que el servicio de auth termine de inicializarse,
# asi que damos un margen de reintentos antes de asumir que no hay superusuario.
AUTH_READY_TIMEOUT_SECONDS = 30
AUTH_RETRY_INTERVAL_SECONDS = 5
AUTH_RECOVERY_VERIFY_TIMEOUT_SECONDS = 60

EXECUTION_MODE_ALL = "all"
EXECUTION_MODE_SCYLLA = "scylla"
EXECUTION_MODE_SEARCH = "search"

# Los alias numericos replican la convencion de configure_server.py.
EXECUTION_MODE_BY_ARGUMENT = {
    "1": EXECUTION_MODE_ALL,
    "all": EXECUTION_MODE_ALL,
    "2": EXECUTION_MODE_SCYLLA,
    "scylla": EXECUTION_MODE_SCYLLA,
    "db": EXECUTION_MODE_SCYLLA,
    "3": EXECUTION_MODE_SEARCH,
    "search": EXECUTION_MODE_SEARCH,
    "genixsearch": EXECUTION_MODE_SEARCH,
}

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

def ensure_firewalld_port_open(database_port):
    firewalld_state_result = run_capture_command(["firewall-cmd", "--state"])
    if firewalld_state_result.returncode != 0 or "running" not in firewalld_state_result.stdout:
        print("[*] firewalld is not running.")
        return False

    port_specification = f"{database_port}/tcp"
    query_port_result = run_capture_command(["firewall-cmd", "--query-port", port_specification])
    if query_port_result.returncode == 0 and "yes" in query_port_result.stdout.lower():
        print(f"[*] firewalld already allows {port_specification}.")
        return True

    print(f"[*] Opening {port_specification} with firewalld.")
    add_port_result = run_capture_command(["firewall-cmd", "--permanent", "--add-port", port_specification])
    if add_port_result.returncode != 0:
        print(f"[!] Failed to add {port_specification} with firewalld.")
        return False

    reload_firewalld_result = run_capture_command(["firewall-cmd", "--reload"])
    if reload_firewalld_result.returncode != 0:
        print("[!] Failed to reload firewalld after opening port.")
        return False

    verification_result = run_capture_command(["firewall-cmd", "--query-port", port_specification])
    return verification_result.returncode == 0 and "yes" in verification_result.stdout.lower()

def ensure_ufw_port_open(database_port):
    ufw_status_result = run_capture_command(["ufw", "status"])
    if ufw_status_result.returncode != 0:
        print("[*] ufw is not available or not active.")
        return False

    normalized_status_output = ufw_status_result.stdout.lower()
    if "status: inactive" in normalized_status_output:
        print("[*] ufw is installed but inactive.")
        return False

    port_rule_pattern = rf'^{re.escape(str(database_port))}/tcp\s+allow\b'
    if re.search(port_rule_pattern, ufw_status_result.stdout, flags=re.IGNORECASE | re.MULTILINE):
        print(f"[*] ufw already allows {database_port}/tcp.")
        return True

    print(f"[*] Opening {database_port}/tcp with ufw.")
    allow_port_result = run_capture_command(["ufw", "allow", f"{database_port}/tcp"])
    if allow_port_result.returncode != 0:
        print(f"[!] Failed to add ufw rule for {database_port}/tcp.")
        return False

    verification_result = run_capture_command(["ufw", "status"])
    return re.search(port_rule_pattern, verification_result.stdout, flags=re.IGNORECASE | re.MULTILINE) is not None

def ensure_tcp_port_open(database_port):
    print(f"[*] Ensuring TCP port {database_port} is open in the host firewall...")
    if which("firewall-cmd") is not None and ensure_firewalld_port_open(database_port):
        print(f"[*] Firewall confirmed open for TCP port {database_port} via firewalld.")
        return

    if which("ufw") is not None and ensure_ufw_port_open(database_port):
        print(f"[*] Firewall confirmed open for TCP port {database_port} via ufw.")
        return

    print("[*] No supported active firewall manager detected. Skipping automatic firewall changes.")

def run_cqlsh_query(cql_query, database_port, database_password, ignore_errors=False, quiet=False):
    cqlsh_command_arguments = [
        "cqlsh",
        "127.0.0.1",
        str(database_port),
        "-u",
        DEFAULT_CASSANDRA_USERNAME,
        "-p",
        database_password,
        "-e",
        cql_query,
    ]
    cqlsh_result = run_capture_command(cqlsh_command_arguments, quiet=quiet)
    if cqlsh_result.returncode != 0 and not ignore_errors:
        print(f"[!] cqlsh query failed: {cql_query}")
        if quiet:
            print_debug_block("cqlsh stderr", cqlsh_result.stderr or cqlsh_result.stdout)
        sys.exit(1)
    return cqlsh_result

def build_candidate_password_list(configured_database_password):
    candidate_passwords = []
    for candidate_password in [configured_database_password, DEFAULT_CASSANDRA_PASSWORD]:
        if candidate_password not in candidate_passwords:
            candidate_passwords.append(candidate_password)
    return candidate_passwords

def describe_candidate_password(candidate_password, configured_database_password):
    return "***configured***" if candidate_password == configured_database_password else "***default***"

def detect_working_cassandra_password(configured_database_password, database_port, timeout_seconds=AUTH_READY_TIMEOUT_SECONDS):
    print("[*] Detecting which Cassandra password is currently active...")
    candidate_passwords = build_candidate_password_list(configured_database_password)
    deadline_timestamp = time.monotonic() + timeout_seconds
    attempt_number = 0
    last_failure_output = ""

    while True:
        attempt_number += 1
        for candidate_password in candidate_passwords:
            test_query_result = run_cqlsh_query(
                "DESCRIBE KEYSPACES;",
                database_port,
                candidate_password,
                ignore_errors=True,
                quiet=True,
            )
            if test_query_result.returncode == 0:
                masked_password_hint = describe_candidate_password(candidate_password, configured_database_password)
                print(f"[*] Cassandra authentication succeeded with {masked_password_hint} password (attempt {attempt_number}).")
                return candidate_password
            last_failure_output = (test_query_result.stderr or test_query_result.stdout).strip()

        if time.monotonic() >= deadline_timestamp:
            break

        # Scylla acepta conexiones antes de terminar de cargar los roles, asi que un
        # 'Bad credentials' temprano no siempre significa que el rol no exista.
        print(f"    ... auth not ready yet, retrying in {AUTH_RETRY_INTERVAL_SECONDS}s (attempt {attempt_number}) ...")
        time.sleep(AUTH_RETRY_INTERVAL_SECONDS)

    print(f"[!] Could not authenticate with the configured or default Cassandra password after {timeout_seconds}s.")
    print_debug_block("last cqlsh failure", last_failure_output)
    return None

def ensure_database_keyspace_exists(database_name, database_port, active_database_password):
    print(f"[*] Ensuring keyspace '{database_name}' exists...")
    check_keyspace_query = (
        "SELECT keyspace_name FROM system_schema.keyspaces "
        f"WHERE keyspace_name = '{database_name}';"
    )
    check_keyspace_result = run_cqlsh_query(check_keyspace_query, database_port, active_database_password, ignore_errors=True)
    if check_keyspace_result.returncode == 0 and database_name in check_keyspace_result.stdout:
        print(f"[*] Keyspace '{database_name}' already exists.")
        return

    # NetworkTopologyStrategy es obligatorio desde Scylla 2026.x: los keyspaces nuevos
    # usan tablets por defecto y tablets no soporta SimpleStrategy.
    create_keyspace_query = (
        f"CREATE KEYSPACE IF NOT EXISTS {database_name} "
        "WITH replication = {'class': 'NetworkTopologyStrategy', 'replication_factor': 1};"
    )
    run_cqlsh_query(create_keyspace_query, database_port, active_database_password)
    print(f"[*] Keyspace '{database_name}' is ready.")

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
        if "systemctl restart scylla-server.service" in command:
            print_service_failure_debug("scylla-server")
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
    """Reads credentials.json once; both configuration modes take their values from it."""
    print(f"[*] Loading credentials from: {PROJECT_CREDENTIALS_FILE}")

    if not PROJECT_CREDENTIALS_FILE.exists():
        print("[!] credentials.json not found in project root.")
        print("    Create it from credentials.example.json and set DB_PASSWORD/DB_PORT.")
        sys.exit(1)

    try:
        with open(PROJECT_CREDENTIALS_FILE, 'r') as credentials_file:
            credentials_data = json.load(credentials_file)
    except json.JSONDecodeError as json_parse_error:
        print(f"[!] Invalid JSON in credentials.json: {json_parse_error}")
        sys.exit(1)

    if not isinstance(credentials_data, dict):
        print("[!] credentials.json must contain a JSON object at the top level.")
        sys.exit(1)

    return credentials_data

def save_project_credentials(credentials_updates):
    """Merges keys back into credentials.json keeping the file owned by the invoking user."""
    credentials_data = load_project_credentials()
    credentials_data.update(credentials_updates)

    PROJECT_CREDENTIALS_FILE.write_text(json.dumps(credentials_data, indent=4) + "\n", encoding="utf-8")

    # El script corre bajo sudo: sin este chown el archivo del repo queda de root.
    sudo_user_id = os.environ.get("SUDO_UID")
    sudo_group_id = os.environ.get("SUDO_GID")
    if sudo_user_id and sudo_group_id:
        with contextlib.suppress(OSError, ValueError):
            os.chown(PROJECT_CREDENTIALS_FILE, int(sudo_user_id), int(sudo_group_id))

    print(f"[*] Saved {', '.join(credentials_updates)} into {PROJECT_CREDENTIALS_FILE}")

def resolve_scylla_credentials(credentials_data):
    """Validates the Scylla password, keyspace name and port coming from credentials.json."""
    configured_database_password = credentials_data.get("DB_PASSWORD")
    configured_database_name = credentials_data.get("DB_NAME")
    configured_database_port = credentials_data.get("DB_PORT", DEFAULT_SCYLLA_PORT)

    # Keep validation explicit because the script directly changes live DB auth.
    if not isinstance(configured_database_password, str) or not configured_database_password.strip():
        print("[!] DB_PASSWORD must exist in credentials.json and be a non-empty string.")
        sys.exit(1)

    # Keep validation explicit because this value is interpolated into a CQL identifier.
    if not isinstance(configured_database_name, str) or not re.fullmatch(r'[A-Za-z][A-Za-z0-9_]*', configured_database_name):
        print("[!] DB_NAME must exist in credentials.json and contain a valid keyspace identifier.")
        sys.exit(1)

    try:
        normalized_database_port = int(configured_database_port)
    except (TypeError, ValueError):
        print("[!] DB_PORT in credentials.json must be a valid integer.")
        sys.exit(1)

    if not (1 <= normalized_database_port <= 65535):
        print("[!] DB_PORT must be between 1 and 65535.")
        sys.exit(1)

    return configured_database_password, configured_database_name, normalized_database_port

def upsert_yaml_setting(yaml_content, setting_name, setting_value):
    # This handles both commented and uncommented keys and appends if missing.
    setting_pattern = rf'^#?\s*{re.escape(setting_name)}:.*$'
    replacement_line = f"{setting_name}: {setting_value}"
    updated_content, replacement_count = re.subn(
        setting_pattern,
        replacement_line,
        yaml_content,
        flags=re.MULTILINE
    )
    if replacement_count == 0:
        if not updated_content.endswith('\n'):
            updated_content += '\n'
        updated_content += replacement_line + '\n'
    return updated_content

def resolve_scylla_environment_file():
    # Scylla packages use different env-file locations across distro families.
    candidate_environment_paths = [
        "/etc/sysconfig/scylla-server",
        "/etc/default/scylla-server",
    ]

    for candidate_environment_path in candidate_environment_paths:
        if os.path.exists(candidate_environment_path):
            print(f"[*] Using Scylla environment file: {candidate_environment_path}")
            return candidate_environment_path

    print("[!] Could not find a Scylla environment file.")
    print("    Checked: /etc/sysconfig/scylla-server, /etc/default/scylla-server")
    print("    Verify that the Scylla package is installed correctly on this host.")
    sys.exit(1)

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

def configure_sysconfig():
    sysconfig_path = resolve_scylla_environment_file()
    available_processing_units = get_available_processing_units()
    desired_smp_units = max(1, min(2, available_processing_units))
    print(
        f"[*] Configuring {sysconfig_path} for 8GB memory and "
        f"{desired_smp_units} Scylla core(s) based on {available_processing_units} available CPU(s)..."
    )

    # Clamp the core count so small cloud instances stay bootable while preserving the lightweight profile.
    desired_scylla_arguments = ["--smp", str(desired_smp_units), "-m", "4G", "--overprovisioned"]
    managed_option_names = {
        "--smp",
        "-m",
        "--memory",
        "--overprovisioned",
        "--developer-mode",
        "--log-to-syslog",
        "--log-to-stdout",
    }

    def build_safe_scylla_args(existing_scylla_args):
        parsed_existing_arguments = shlex.split(existing_scylla_args)
        sanitized_arguments = []
        argument_index = 0

        while argument_index < len(parsed_existing_arguments):
            current_argument = parsed_existing_arguments[argument_index]
            next_argument = parsed_existing_arguments[argument_index + 1] if argument_index + 1 < len(parsed_existing_arguments) else None

            if current_argument in managed_option_names:
                # Skip managed flags and any boolean/value remnants from previous script runs.
                if next_argument and not next_argument.startswith("-"):
                    argument_index += 2
                else:
                    argument_index += 1
                continue

            sanitized_arguments.append(current_argument)
            argument_index += 1

        return shlex.join(sanitized_arguments + desired_scylla_arguments)

    with open(sysconfig_path, 'r') as file:
        lines = file.readlines()

    with open(sysconfig_path, 'w') as file:
        did_update_scylla_args = False
        for line in lines:
            if line.startswith("SCYLLA_ARGS="):
                existing_scylla_args = line.split("=", 1)[1].strip().strip('"')
                print(f"[*] Existing SCYLLA_ARGS: {existing_scylla_args}")
                safe_scylla_args = build_safe_scylla_args(existing_scylla_args)
                print(f"[*] Using SCYLLA_ARGS: {safe_scylla_args}")
                file.write(f'SCYLLA_ARGS="{safe_scylla_args}"\n')
                did_update_scylla_args = True
            else:
                file.write(line)

        if not did_update_scylla_args:
            safe_scylla_args = shlex.join(desired_scylla_arguments)
            print(f"[*] Appending SCYLLA_ARGS: {safe_scylla_args}")
            file.write(f'SCYLLA_ARGS="{safe_scylla_args}"\n')

    print_file_debug_lines(sysconfig_path, [r'^SCYLLA_ARGS=', r'^SEASTAR_IO=', r'^CPUSET=', r'^MEM_CONF=', r'^DEV_MODE='])

def read_yaml_setting(setting_name, default_value=None):
    try:
        with open(SCYLLA_YAML_PATH, 'r') as yaml_file:
            yaml_content = yaml_file.read()
    except OSError:
        return default_value

    setting_match = re.search(rf'^\s*{re.escape(setting_name)}:\s*(.+?)\s*$', yaml_content, flags=re.MULTILINE)
    return setting_match.group(1) if setting_match else default_value

def apply_yaml_settings(settings_to_apply):
    with open(SCYLLA_YAML_PATH, 'r') as yaml_file:
        yaml_content = yaml_file.read()

    for setting_name, setting_value in settings_to_apply.items():
        yaml_content = upsert_yaml_setting(yaml_content, setting_name, setting_value)

    with open(SCYLLA_YAML_PATH, 'w') as yaml_file:
        yaml_file.write(yaml_content)

def restart_scylla_server(database_port):
    run_command("systemctl restart scylla-server.service")
    wait_for_scylla(database_port)

def configure_yaml(ip_address, database_port):
    print(f"[*] Configuring /etc/scylla/scylla.yaml (IP: {ip_address}, Port: {database_port})...")
    yaml_path = SCYLLA_YAML_PATH

    with open(yaml_path, 'r') as file:
        content = file.read()

    # Apply networking and single-node configs
    content = upsert_yaml_setting(content, "listen_address", "127.0.0.1")
    content = upsert_yaml_setting(content, "rpc_address", "0.0.0.0")
    content = upsert_yaml_setting(content, "broadcast_rpc_address", ip_address)
    content = upsert_yaml_setting(content, "native_transport_port", database_port)

    # Enforce password authentication for non-default secured setup.
    content = upsert_yaml_setting(content, "authenticator", "PasswordAuthenticator")

    with open(yaml_path, 'w') as file:
        file.write(content)

    print_file_debug_lines(
        yaml_path,
        [
            r'^\s*listen_address:',
            r'^\s*rpc_address:',
            r'^\s*broadcast_rpc_address:',
            r'^\s*native_transport_port:',
            r'^\s*authenticator:',
        ]
    )

def wait_for_scylla(database_port):
    print(f"[*] Waiting for ScyllaDB to start on port {database_port}...")
    for i in range(30):
        try:
            with socket.create_connection(("127.0.0.1", database_port), timeout=1):
                print("[*] ScyllaDB is up and accepting CQL connections!")
                time.sleep(3) # Give it a brief moment to fully initialize auth
                return
        except OSError:
            time.sleep(2)
            print(f"    ... still waiting ({i+1}/30) ...")
    
    print("[!] ScyllaDB did not start in time. Check 'journalctl -xeu scylla-server'.")
    sys.exit(1)

def resolve_maintenance_socket_path():
    # El socket vive dentro del workdir de Scylla; 'workdir' es el valor tipico de maintenance_socket.
    configured_working_directory = read_yaml_setting("workdir", SCYLLA_DEFAULT_WORKDIR).strip().strip('"').strip("'")
    return os.path.join(configured_working_directory, MAINTENANCE_SOCKET_FILENAME)

def pump_socket_data(source_socket, destination_socket):
    try:
        while True:
            data_chunk = source_socket.recv(65536)
            if not data_chunk:
                break
            destination_socket.sendall(data_chunk)
    except OSError:
        pass
    finally:
        with contextlib.suppress(OSError):
            destination_socket.shutdown(socket.SHUT_WR)

class MaintenanceSocketBridge:
    """Publica el socket unix de mantenimiento como un puerto TCP local efimero.

    El cqlsh empaquetado para Ubuntu no acepta --maintenance-socket, asi que en vez de
    depender de esa opcion puenteamos el socket con un forwarder de stdlib y apuntamos
    el cqlsh existente al puerto local. El puente solo escucha en 127.0.0.1 y vive lo
    que dura la recuperacion.
    """

    def __init__(self, maintenance_socket_path):
        self.maintenance_socket_path = maintenance_socket_path
        self.listener_socket = None
        self.bridge_port = None
        self.should_stop = False

    def __enter__(self):
        self.listener_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.listener_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.listener_socket.bind(("127.0.0.1", 0))
        self.listener_socket.listen(8)
        self.bridge_port = self.listener_socket.getsockname()[1]

        threading.Thread(target=self.accept_connections_loop, daemon=True).start()
        print(f"[*] Bridging {self.maintenance_socket_path} on 127.0.0.1:{self.bridge_port}")
        return self

    def __exit__(self, exception_type, exception_value, exception_traceback):
        self.should_stop = True
        with contextlib.suppress(OSError):
            self.listener_socket.close()
        return False

    def accept_connections_loop(self):
        while not self.should_stop:
            try:
                client_connection, _ = self.listener_socket.accept()
            except OSError:
                return
            threading.Thread(target=self.forward_connection, args=(client_connection,), daemon=True).start()

    def forward_connection(self, client_connection):
        upstream_connection = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        try:
            upstream_connection.connect(self.maintenance_socket_path)
        except OSError as connect_error:
            print(f"[!] Could not connect to the maintenance socket: {connect_error}")
            client_connection.close()
            return

        threading.Thread(target=pump_socket_data, args=(client_connection, upstream_connection), daemon=True).start()
        pump_socket_data(upstream_connection, client_connection)

        with contextlib.suppress(OSError):
            client_connection.close()
        with contextlib.suppress(OSError):
            upstream_connection.close()

def run_maintenance_cqlsh_query(cql_query, bridge_port):
    # Sin -u/-p: el socket de mantenimiento ya autentica como superusuario.
    maintenance_command_arguments = ["cqlsh", "127.0.0.1", str(bridge_port), "-e", cql_query]
    return run_capture_command(maintenance_command_arguments)

def wait_for_maintenance_socket(maintenance_socket_path, timeout_seconds=60):
    deadline_timestamp = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline_timestamp:
        if os.path.exists(maintenance_socket_path):
            return True
        time.sleep(2)
    return os.path.exists(maintenance_socket_path)

def ensure_maintenance_socket_available(database_port):
    """Garantiza que el socket de mantenimiento exista, habilitandolo si hace falta."""
    maintenance_socket_path = resolve_maintenance_socket_path()
    if os.path.exists(maintenance_socket_path):
        print(f"[*] Maintenance socket available at {maintenance_socket_path}.")
        return maintenance_socket_path

    configured_maintenance_socket = read_yaml_setting("maintenance_socket", "ignore").strip().strip('"').strip("'")
    if configured_maintenance_socket == "workdir":
        print(f"[!] maintenance_socket is enabled but {maintenance_socket_path} does not exist.")
        return None

    print("[*] Enabling maintenance_socket in scylla.yaml to recover the superuser...")
    apply_yaml_settings({"maintenance_socket": "workdir"})
    restart_scylla_server(database_port)

    maintenance_socket_path = resolve_maintenance_socket_path()
    if wait_for_maintenance_socket(maintenance_socket_path):
        return maintenance_socket_path

    print(f"[!] Maintenance socket never appeared at {maintenance_socket_path}.")
    return None

def recover_cassandra_superuser(new_database_password, database_port):
    """Crea el superusuario cuando ningun login funciona.

    Scylla siembra el rol 'cassandra' por defecto solo durante el bootstrap inicial del
    cluster. Si el nodo arranco por primera vez con AllowAllAuthenticator, activar
    PasswordAuthenticator despues deja el nodo sin ningun rol con el que autenticarse.
    El socket de mantenimiento da acceso CQL local con permisos de superusuario y sin
    autenticacion, que es justo lo que se necesita para crearlo.
    """
    print("[*] No usable superuser found. Recovering via the Scylla maintenance socket...")

    maintenance_socket_path = ensure_maintenance_socket_available(database_port)
    if maintenance_socket_path is None:
        print("[!] Superuser recovery is not possible without the maintenance socket.")
        print("    Set 'maintenance_socket: workdir' in /etc/scylla/scylla.yaml, restart scylla-server and re-run.")
        sys.exit(1)

    escaped_database_password = new_database_password.replace("'", "''")

    # CREATE cubre el caso "el rol nunca existio"; ALTER cubre "existe con otra password".
    create_role_query = (
        f"CREATE ROLE IF NOT EXISTS {DEFAULT_CASSANDRA_USERNAME} "
        f"WITH PASSWORD = '{escaped_database_password}' AND LOGIN = true AND SUPERUSER = true;"
    )
    alter_role_query = (
        f"ALTER ROLE {DEFAULT_CASSANDRA_USERNAME} "
        f"WITH PASSWORD = '{escaped_database_password}' AND LOGIN = true AND SUPERUSER = true;"
    )

    with MaintenanceSocketBridge(maintenance_socket_path) as maintenance_bridge:
        create_role_result = run_maintenance_cqlsh_query(create_role_query, maintenance_bridge.bridge_port)
        alter_role_result = run_maintenance_cqlsh_query(alter_role_query, maintenance_bridge.bridge_port)

    if create_role_result.returncode != 0 and alter_role_result.returncode != 0:
        print("[!] Could not create or update the cassandra role through the maintenance socket.")
        sys.exit(1)

    recovered_password = detect_working_cassandra_password(
        new_database_password,
        database_port,
        timeout_seconds=AUTH_RECOVERY_VERIFY_TIMEOUT_SECONDS,
    )
    if recovered_password is None:
        print("[!] Superuser recovery failed. Inspect 'journalctl -xeu scylla-server' for auth errors.")
        sys.exit(1)

    print("[*] Cassandra superuser recovery completed.")
    return recovered_password

def change_cassandra_password(new_database_password, database_port, active_database_password):
    print("[*] Changing default cassandra password...")
    escaped_database_password = new_database_password.replace("'", "''")
    cql_query = f"ALTER ROLE {DEFAULT_CASSANDRA_USERNAME} WITH PASSWORD = '{escaped_database_password}';"

    result = run_cqlsh_query(cql_query, database_port, active_database_password, ignore_errors=True)

    if result.returncode == 0:
        print("[*] Password changed successfully!")
    elif active_database_password == new_database_password:
        print("[*] Cassandra password already matches the configured password.")
    else:
        print(f"[!] Failed to change password. It might have already been changed. Error: {result.stderr}")
        if result.stdout.strip():
            print_debug_block("cqlsh stdout", result.stdout)
        if result.stderr.strip():
            print_debug_block("cqlsh stderr", result.stderr)

def configure_scylladb(credentials_data, broadcast_ip_address, detected_network_provider):
    configured_database_password, configured_database_name, configured_database_port = resolve_scylla_credentials(credentials_data)

    configure_sysconfig()
    run_command("scylla_dev_mode_setup --developer-mode 1")
    configure_yaml(broadcast_ip_address, configured_database_port)
    ensure_tcp_port_open(configured_database_port)

    print("[*] Restarting Scylla Server daemon...")
    run_command("systemctl daemon-reload")
    run_command("systemctl restart scylla-server.service")

    wait_for_scylla(configured_database_port)
    active_database_password = detect_working_cassandra_password(configured_database_password, configured_database_port)
    if active_database_password is None:
        active_database_password = recover_cassandra_superuser(configured_database_password, configured_database_port)

    change_cassandra_password(configured_database_password, configured_database_port, active_database_password)
    ensure_database_keyspace_exists(configured_database_name, configured_database_port, configured_database_password)

    print("\n[+] ScyllaDB Dev Configuration Complete!")
    print(f"    - Access internally via: cqlsh 127.0.0.1 {configured_database_port} -u cassandra -p '{configured_database_password}'")
    print(f"    - Access via {detected_network_provider}: cqlsh {broadcast_ip_address} {configured_database_port} -u cassandra -p '{configured_database_password}'")

# ---------------------------------------------------------------------------
# GenixSearch
# ---------------------------------------------------------------------------

GENIXSEARCH_SERVICE_NAME = "genixsearch"
GENIXSEARCH_BINARY_PATH = Path("/usr/local/bin") / GENIXSEARCH_SERVICE_NAME
GENIXSEARCH_CONFIG_DIRECTORY = Path("/etc") / GENIXSEARCH_SERVICE_NAME
GENIXSEARCH_CONFIG_FILE = GENIXSEARCH_CONFIG_DIRECTORY / f"{GENIXSEARCH_SERVICE_NAME}.cfg"
GENIXSEARCH_DATA_DIRECTORY = Path("/var/lib") / GENIXSEARCH_SERVICE_NAME
GENIXSEARCH_KV_DIRECTORY = GENIXSEARCH_DATA_DIRECTORY / "store" / "kv"
GENIXSEARCH_UNIT_FILE = Path("/etc/systemd/system") / f"{GENIXSEARCH_SERVICE_NAME}.service"
GENIXSEARCH_RELEASE_REPOSITORY = "ivanjoz/genix-search"

# Debe coincidir con el default de core.ParseGenixSearchURL en backend/core/security.go.
DEFAULT_GENIXSEARCH_PORT = 14446
GENIXSEARCH_PASSWORD_LENGTH = 64
GENIXSEARCH_PASSWORD_ALPHABET = string.ascii_lowercase + string.digits
NOLOGIN_SHELL_CANDIDATES = ("/usr/sbin/nologin", "/sbin/nologin", "/usr/bin/false", "/bin/false")

# Sufijos de los assets que publica genix-search/scripts/release_binaries.sh.
# Solo hay builds musl para estas dos arquitecturas.
GENIXSEARCH_ASSET_SUFFIX_BY_MACHINE = {
    "x86_64": "x86_64-linux-musl",
    "amd64": "x86_64-linux-musl",
    "aarch64": "aarch64-linux-musl",
    "arm64": "aarch64-linux-musl",
}
GENIXSEARCH_NEOVERSE_N1_ASSET_SUFFIX = "aarch64-linux-musl-neoverse-n1"
# MIDR_EL1 part 0xd0c = Arm Neoverse N1, el core del Ampere Altra (Oracle "Ampere A1").
NEOVERSE_N1_CPU_PART = "0xd0c"

def is_neoverse_n1_cpu():
    try:
        cpuinfo_content = Path("/proc/cpuinfo").read_text(encoding="utf-8", errors="replace")
    except OSError:
        return False

    return any(
        cpuinfo_line.split(":", 1)[1].strip().lower() == NEOVERSE_N1_CPU_PART
        for cpuinfo_line in cpuinfo_content.splitlines()
        if cpuinfo_line.lower().startswith("cpu part") and ":" in cpuinfo_line
    )

def detect_genixsearch_asset_suffix():
    """Resolves the published asset for this machine, or aborts if there is none."""
    machine_architecture = os.uname().machine
    asset_suffix = GENIXSEARCH_ASSET_SUFFIX_BY_MACHINE.get(machine_architecture)
    if asset_suffix is None:
        print(f"[!] Architecture '{machine_architecture}' is not compatible with GenixSearch.")
        print(f"    Published static builds: {', '.join(sorted(set(GENIXSEARCH_ASSET_SUFFIX_BY_MACHINE.values())))}")
        sys.exit(1)

    # El build para Neoverse N1 usa el mismo target musl pero con -Ctarget-cpu, asi que
    # solo aplica cuando el host es realmente un Ampere Altra.
    if asset_suffix == "aarch64-linux-musl" and is_neoverse_n1_cpu():
        print("[*] Detected an Arm Neoverse N1 (Ampere Altra) CPU.")
        return GENIXSEARCH_NEOVERSE_N1_ASSET_SUFFIX

    print(f"[*] Detected architecture '{machine_architecture}' -> asset {asset_suffix}")
    return asset_suffix

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

def resolve_latest_genixsearch_release_tag():
    print(f"[*] Resolving the latest release of {GENIXSEARCH_RELEASE_REPOSITORY}...")
    release_payload = http_get_bytes(f"https://api.github.com/repos/{GENIXSEARCH_RELEASE_REPOSITORY}/releases/latest")
    try:
        release_tag = json.loads(release_payload)["tag_name"]
    except (ValueError, KeyError):
        print(f"[!] Could not read the latest release tag for {GENIXSEARCH_RELEASE_REPOSITORY}.")
        sys.exit(1)

    print(f"[*] Latest published release: {release_tag}")
    return release_tag

def verify_release_checksum(release_base_url, archive_name, archive_digest):
    """Compares the downloaded asset against the release manifest.

    Un asset corrupto o sustituido de otro modo solo se nota cuando el servicio no
    arranca, asi que se valida antes de instalarlo.
    """
    checksums_content = http_get_bytes(f"{release_base_url}/SHA256SUMS").decode("utf-8", errors="replace")
    expected_digest = None
    for checksum_line in checksums_content.splitlines():
        checksum_parts = checksum_line.split()
        if len(checksum_parts) == 2 and checksum_parts[1] == archive_name:
            expected_digest = checksum_parts[0]
            break

    if expected_digest is None:
        print(f"[!] SHA256SUMS has no entry for {archive_name}.")
        sys.exit(1)

    if expected_digest != archive_digest:
        print(f"[!] Checksum mismatch for {archive_name}: expected {expected_digest}, got {archive_digest}")
        sys.exit(1)

    print("[*] Checksum verified against the release SHA256SUMS.")

def download_genixsearch_release(release_version, download_directory):
    """Downloads and unpacks the static musl release. Returns (binary, reference config)."""
    asset_suffix = detect_genixsearch_asset_suffix()
    if release_version in (None, "", "latest"):
        release_version = resolve_latest_genixsearch_release_tag()
    if not release_version.startswith("v"):
        release_version = f"v{release_version}"

    archive_stem = f"{GENIXSEARCH_SERVICE_NAME}-{release_version}-{asset_suffix}"
    archive_name = f"{archive_stem}.tar.gz"
    release_base_url = f"https://github.com/{GENIXSEARCH_RELEASE_REPOSITORY}/releases/download/{release_version}"

    print(f"[*] Downloading {release_base_url}/{archive_name}")
    archive_payload = http_get_bytes(f"{release_base_url}/{archive_name}")
    archive_path = download_directory / archive_name
    archive_path.write_bytes(archive_payload)

    verify_release_checksum(release_base_url, archive_name, hashlib.sha256(archive_payload).hexdigest())

    with tarfile.open(archive_path, "r:gz") as release_archive:
        try:
            # Python 3.12+ rechaza miembros inseguros; las versiones anteriores no
            # conocen el parametro, de ahi el fallback.
            release_archive.extractall(download_directory, filter="data")
        except TypeError:
            release_archive.extractall(download_directory)

    downloaded_binary_path = download_directory / archive_stem / GENIXSEARCH_SERVICE_NAME
    if not downloaded_binary_path.is_file():
        print(f"[!] {archive_name} did not contain {archive_stem}/{GENIXSEARCH_SERVICE_NAME}")
        sys.exit(1)
    downloaded_binary_path.chmod(0o755)

    return downloaded_binary_path, download_directory / archive_stem / f"{GENIXSEARCH_SERVICE_NAME}.cfg"

def ensure_genixsearch_system_user():
    if subprocess.run(["id", GENIXSEARCH_SERVICE_NAME], capture_output=True).returncode == 0:
        print(f"[*] System user '{GENIXSEARCH_SERVICE_NAME}' already exists.")
        return

    nologin_shell = next((shell_path for shell_path in NOLOGIN_SHELL_CANDIDATES if Path(shell_path).exists()), "/bin/false")
    print(f"[*] Creating system user '{GENIXSEARCH_SERVICE_NAME}'...")
    run_command(
        f"useradd --system --user-group --home-dir {GENIXSEARCH_DATA_DIRECTORY} "
        f"--no-create-home --shell {nologin_shell} {GENIXSEARCH_SERVICE_NAME}"
    )

def ensure_genixsearch_data_directories():
    print(f"[*] Preparing state directory {GENIXSEARCH_DATA_DIRECTORY} (0750)")
    GENIXSEARCH_KV_DIRECTORY.mkdir(parents=True, exist_ok=True)
    for directory_path in (GENIXSEARCH_DATA_DIRECTORY, GENIXSEARCH_DATA_DIRECTORY / "store", GENIXSEARCH_KV_DIRECTORY):
        directory_path.chmod(0o750)
        shutil.chown(directory_path, user=GENIXSEARCH_SERVICE_NAME, group=GENIXSEARCH_SERVICE_NAME)

def install_genixsearch_binary(source_binary_path):
    print(f"[*] Installing {source_binary_path} -> {GENIXSEARCH_BINARY_PATH}")
    GENIXSEARCH_BINARY_PATH.parent.mkdir(parents=True, exist_ok=True)
    # install(1) en vez de cp: reemplaza el inodo, asi un servicio en ejecucion sigue
    # usando su binario actual hasta que se reinicia.
    run_command(f"install -m 0755 -o root -g root {shlex.quote(str(source_binary_path))} {GENIXSEARCH_BINARY_PATH}")

CONFIG_SECTION_PATTERN = re.compile(r"^\s*\[([^\]]+)\]")
CONFIG_KEY_PATTERN = re.compile(r"^(\s*)([A-Za-z0-9_.\-]+)(\s*=\s*)(.*)$")

def apply_config_overrides(reference_config_content, config_overrides):
    """Replaces `(section, key) -> TOML value` in the reference cfg shipped by the release.

    Se parte del cfg del release en vez de una plantilla propia para heredar cualquier
    clave nueva que agregue GenixSearch; solo se sobrescriben las cuatro que dependen
    del host. Si una clave esperada no aparece, se aborta: seguir dejaria el servicio
    escuchando o guardando datos donde no corresponde.
    """
    rewritten_lines = []
    current_section_name = ""
    applied_override_keys = set()

    for config_line in reference_config_content.splitlines():
        section_match = CONFIG_SECTION_PATTERN.match(config_line)
        if section_match:
            current_section_name = section_match.group(1).strip()
            rewritten_lines.append(config_line)
            continue

        key_match = CONFIG_KEY_PATTERN.match(config_line)
        override_key = (current_section_name, key_match.group(2)) if key_match else None
        if override_key in config_overrides:
            line_indent, config_key, key_separator, _ = key_match.groups()
            rewritten_lines.append(f"{line_indent}{config_key}{key_separator}{config_overrides[override_key]}")
            applied_override_keys.add(override_key)
            continue

        rewritten_lines.append(config_line)

    missing_override_keys = set(config_overrides) - applied_override_keys
    if missing_override_keys:
        print(f"[!] The reference genixsearch.cfg has no entry for: {sorted(missing_override_keys)}")
        sys.exit(1)

    return "\n".join(rewritten_lines) + "\n"

def write_genixsearch_config(reference_config_path, genixsearch_password, genixsearch_port):
    if not reference_config_path.is_file():
        print(f"[!] Reference genixsearch.cfg not found at {reference_config_path}")
        sys.exit(1)

    # json.dumps produce un string entre comillas con escapes compatibles con TOML.
    config_overrides = {
        ("server", "log_level"): json.dumps("info"),
        ("channel", "inet"): json.dumps(f"0.0.0.0:{genixsearch_port}"),
        ("channel", "auth_password"): json.dumps(genixsearch_password),
        ("store.kv", "path"): json.dumps(f"{GENIXSEARCH_KV_DIRECTORY}/"),
    }
    config_content = apply_config_overrides(
        reference_config_path.read_text(encoding="utf-8"), config_overrides
    )

    print(f"[*] Writing {GENIXSEARCH_CONFIG_FILE} (0640 root:{GENIXSEARCH_SERVICE_NAME})")
    GENIXSEARCH_CONFIG_DIRECTORY.mkdir(parents=True, exist_ok=True)
    GENIXSEARCH_CONFIG_FILE.write_text(
        "# Escrito por scripts/configure_db.py. Se sobrescribe en cada ejecucion.\n" + config_content,
        encoding="utf-8",
    )
    GENIXSEARCH_CONFIG_FILE.chmod(0o640)
    shutil.chown(GENIXSEARCH_CONFIG_FILE, user="root", group=GENIXSEARCH_SERVICE_NAME)

def write_genixsearch_unit(genixsearch_port):
    # Un puerto privilegiado necesita la capability; con el default (14446) no hace falta.
    service_capabilities = "CAP_NET_BIND_SERVICE" if genixsearch_port < 1024 else ""
    unit_content = f"""\
# GenixSearch systemd unit — escrita por scripts/configure_db.py.
[Unit]
Description=GenixSearch — compact, lossy, ranked search backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={GENIXSEARCH_SERVICE_NAME}
Group={GENIXSEARCH_SERVICE_NAME}
WorkingDirectory={GENIXSEARCH_DATA_DIRECTORY}
ExecStart={GENIXSEARCH_BINARY_PATH} --config {GENIXSEARCH_CONFIG_FILE}
Restart=on-failure
RestartSec=2s
# El KV store se vacia de forma sincrona con SIGTERM; hay que darle margen.
KillSignal=SIGTERM
TimeoutStopSec=120s
LimitNOFILE=65536
SyslogIdentifier={GENIXSEARCH_SERVICE_NAME}

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
ReadWritePaths={GENIXSEARCH_DATA_DIRECTORY}
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
    print(f"[*] Writing {GENIXSEARCH_UNIT_FILE}")
    GENIXSEARCH_UNIT_FILE.write_text(unit_content, encoding="utf-8")
    GENIXSEARCH_UNIT_FILE.chmod(0o644)

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

def resolve_genixsearch_port(credentials_data):
    """Takes the port from GENIXSEARCH_URL when it carries one, else the backend default.

    Acepta las mismas formas que core.ParseGenixSearchURL: 'host:port',
    'scheme://host:port' y con path/query al final. Un host sin puerto (incluido
    un IPv6 sin corchetes) cae al default.
    """
    configured_search_url = str(credentials_data.get("GENIXSEARCH_URL") or "").strip()
    if "://" in configured_search_url:
        configured_search_url = configured_search_url.split("://", 1)[1]
    configured_search_url = re.split(r"[/?]", configured_search_url, maxsplit=1)[0]

    # Solo el ultimo ':' separa el puerto, y un IPv6 sin corchetes tiene varios.
    host_part, separator, port_part = configured_search_url.rpartition(":")
    if not separator or not port_part.isdigit() or (":" in host_part and not host_part.endswith("]")):
        return DEFAULT_GENIXSEARCH_PORT

    parsed_port = int(port_part)
    if not (1 <= parsed_port <= 65535):
        print(f"[!] GENIXSEARCH_URL has an invalid port: {configured_search_url}")
        sys.exit(1)
    return parsed_port

def resolve_genixsearch_password(credentials_data):
    """Reuses GENIXSEARCH_PASSWORD from credentials.json, or generates one."""
    existing_password = credentials_data.get("GENIXSEARCH_PASSWORD")
    if isinstance(existing_password, str) and existing_password.strip():
        print("[*] Reusing GENIXSEARCH_PASSWORD from credentials.json.")
        return existing_password.strip(), False

    print("[*] No GENIXSEARCH_PASSWORD in credentials.json. Generating one.")
    generated_password = "".join(
        secrets.choice(GENIXSEARCH_PASSWORD_ALPHABET) for _ in range(GENIXSEARCH_PASSWORD_LENGTH)
    )
    return generated_password, True

def configure_genixsearch(credentials_data, broadcast_ip_address, release_version, local_binary_path):
    genixsearch_port = resolve_genixsearch_port(credentials_data)
    genixsearch_password, was_password_generated = resolve_genixsearch_password(credentials_data)

    with tempfile.TemporaryDirectory(prefix="genixsearch-release-") as download_directory_name:
        download_directory = Path(download_directory_name)
        if local_binary_path:
            source_binary_path = Path(local_binary_path).resolve()
            if not source_binary_path.is_file():
                print(f"[!] --binary does not exist: {source_binary_path}")
                sys.exit(1)
            # Sin release descargado no hay cfg de referencia; se usa el del repo hermano si existe.
            reference_config_path = PROJECT_ROOT_DIRECTORY.parent / "genix-search" / f"{GENIXSEARCH_SERVICE_NAME}.cfg"
        else:
            source_binary_path, reference_config_path = download_genixsearch_release(release_version, download_directory)

        ensure_genixsearch_system_user()
        ensure_genixsearch_data_directories()
        install_genixsearch_binary(source_binary_path)
        write_genixsearch_config(reference_config_path, genixsearch_password, genixsearch_port)

    write_genixsearch_unit(genixsearch_port)
    restore_selinux_context([GENIXSEARCH_BINARY_PATH, GENIXSEARCH_CONFIG_DIRECTORY, GENIXSEARCH_DATA_DIRECTORY])
    ensure_tcp_port_open(genixsearch_port)

    installed_version_result = run_capture_command([str(GENIXSEARCH_BINARY_PATH), "--version"], quiet=True)
    print(f"[*] Installed binary reports: {(installed_version_result.stdout or installed_version_result.stderr).strip()}")

    run_command("systemctl daemon-reload")
    run_command(f"systemctl enable {GENIXSEARCH_SERVICE_NAME}.service")
    run_command(f"systemctl restart {GENIXSEARCH_SERVICE_NAME}.service")

    if not wait_for_service_active(GENIXSEARCH_SERVICE_NAME):
        print(f"[!] {GENIXSEARCH_SERVICE_NAME}.service did not stay active.")
        print_service_failure_debug(GENIXSEARCH_SERVICE_NAME)
        sys.exit(1)

    # El backend en Lambda alcanza este host por su IP de broadcast, no por loopback.
    genixsearch_url = f"{broadcast_ip_address}:{genixsearch_port}"
    credentials_updates = {"GENIXSEARCH_URL": genixsearch_url}
    if was_password_generated:
        credentials_updates["GENIXSEARCH_PASSWORD"] = genixsearch_password
    save_project_credentials(credentials_updates)

    print("\n[+] GenixSearch Installation Complete!")
    print(f"    - binary:  {GENIXSEARCH_BINARY_PATH}")
    print(f"    - config:  {GENIXSEARCH_CONFIG_FILE}")
    print(f"    - state:   {GENIXSEARCH_DATA_DIRECTORY}")
    print(f"    - listen:  0.0.0.0:{genixsearch_port}")
    print(f"    - GENIXSEARCH_URL:      {genixsearch_url}")
    print(f"    - GENIXSEARCH_PASSWORD: {genixsearch_password}")
    print(f"    - systemctl status {GENIXSEARCH_SERVICE_NAME}")

def parse_command_arguments():
    argument_parser = argparse.ArgumentParser(
        prog="configure_db.py",
        description="Configures ScyllaDB and installs GenixSearch on this host.",
    )
    argument_parser.add_argument(
        "mode",
        nargs="?",
        default=EXECUTION_MODE_ALL,
        help="all|1 (default), scylla|2, search|3",
    )
    argument_parser.add_argument(
        "--release-version",
        default="latest",
        help="GenixSearch release tag to install, e.g. v0.1.0 (default: latest)",
    )
    argument_parser.add_argument(
        "--binary",
        default=None,
        help="install this local genixsearch binary instead of downloading the release",
    )
    parsed_arguments = argument_parser.parse_args()

    execution_mode = EXECUTION_MODE_BY_ARGUMENT.get(parsed_arguments.mode.strip().lower())
    if execution_mode is None:
        argument_parser.error(f"Unknown mode: {parsed_arguments.mode}. Use all, scylla or search.")

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
    detected_network_provider, broadcast_ip_address = get_preferred_broadcast_ip()

    if command_arguments.mode in (EXECUTION_MODE_ALL, EXECUTION_MODE_SCYLLA):
        configure_scylladb(credentials_data, broadcast_ip_address, detected_network_provider)

    if command_arguments.mode in (EXECUTION_MODE_ALL, EXECUTION_MODE_SEARCH):
        configure_genixsearch(
            credentials_data,
            broadcast_ip_address,
            command_arguments.release_version,
            command_arguments.binary,
        )

if __name__ == "__main__":
    main()
