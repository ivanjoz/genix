#!/usr/bin/env python3

import os
import subprocess
import time
import socket
import re
import sys
import json
from pathlib import Path
from shutil import which
import ipaddress
import shlex

PROJECT_ROOT_DIRECTORY = Path(__file__).resolve().parents[1]
PROJECT_CREDENTIALS_FILE = PROJECT_ROOT_DIRECTORY / "credentials.json"
DEFAULT_SCYLLA_PORT = 9042
DEFAULT_CASSANDRA_USERNAME = "cassandra"
DEFAULT_CASSANDRA_PASSWORD = "cassandra"

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

def run_capture_command(command_arguments):
    print(f"[*] Running argv command: {' '.join(command_arguments)}")
    command_result = subprocess.run(command_arguments, text=True, capture_output=True)
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

def ensure_database_port_open(database_port):
    print(f"[*] Ensuring TCP port {database_port} is open in the host firewall...")
    if which("firewall-cmd") is not None and ensure_firewalld_port_open(database_port):
        print(f"[*] Firewall confirmed open for TCP port {database_port} via firewalld.")
        return

    if which("ufw") is not None and ensure_ufw_port_open(database_port):
        print(f"[*] Firewall confirmed open for TCP port {database_port} via ufw.")
        return

    print("[*] No supported active firewall manager detected. Skipping automatic firewall changes.")

def run_cqlsh_query(cql_query, database_port, database_password, ignore_errors=False):
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
    cqlsh_result = run_capture_command(cqlsh_command_arguments)
    if cqlsh_result.returncode != 0 and not ignore_errors:
        print(f"[!] cqlsh query failed: {cql_query}")
        sys.exit(1)
    return cqlsh_result

def detect_working_cassandra_password(configured_database_password, database_port):
    print("[*] Detecting which Cassandra password is currently active...")
    candidate_passwords = []

    for candidate_password in [configured_database_password, DEFAULT_CASSANDRA_PASSWORD]:
        if candidate_password not in candidate_passwords:
            candidate_passwords.append(candidate_password)

    for candidate_password in candidate_passwords:
        test_query_result = run_cqlsh_query("DESCRIBE KEYSPACES;", database_port, candidate_password, ignore_errors=True)
        if test_query_result.returncode == 0:
            masked_password_hint = "***configured***" if candidate_password == configured_database_password else "***default***"
            print(f"[*] Cassandra authentication succeeded with {masked_password_hint} password.")
            return candidate_password

    print("[!] Could not authenticate with either the configured or default Cassandra password.")
    sys.exit(1)

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

    create_keyspace_query = (
        f"CREATE KEYSPACE IF NOT EXISTS {database_name} "
        "WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};"
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

def load_scylla_credentials():
    """Loads Scylla credentials, keyspace name, and port from project credentials.json."""
    print(f"[*] Loading credentials from: {PROJECT_CREDENTIALS_FILE}")

    if not PROJECT_CREDENTIALS_FILE.exists():
        print("[!] credentials.json not found in project root.")
        print("    Create it from credentials_example.json and set DB_PASSWORD/DB_PORT.")
        sys.exit(1)

    try:
        with open(PROJECT_CREDENTIALS_FILE, 'r') as credentials_file:
            credentials_data = json.load(credentials_file)
    except json.JSONDecodeError as json_parse_error:
        print(f"[!] Invalid JSON in credentials.json: {json_parse_error}")
        sys.exit(1)

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

def configure_sysconfig():
    print("[*] Configuring /etc/sysconfig/scylla-server for 8GB memory & 2 cores...")
    sysconfig_path = "/etc/sysconfig/scylla-server"

    desired_scylla_arguments = ["--smp", "2", "-m", "4G", "--overprovisioned"]
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

def configure_yaml(ip_address, database_port):
    print(f"[*] Configuring /etc/scylla/scylla.yaml (IP: {ip_address}, Port: {database_port})...")
    yaml_path = "/etc/scylla/scylla.yaml"
    
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

def change_cassandra_password(new_database_password, database_port, active_database_password):
    print("[*] Changing default cassandra password...")
    cql_query = f"ALTER ROLE cassandra WITH PASSWORD = '{new_database_password}';"

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

def main():
    if os.geteuid() != 0:
        print("[!] Please run this script with sudo.")
        sys.exit(1)

    configured_database_password, configured_database_name, configured_database_port = load_scylla_credentials()
    detected_network_provider, ip_address = get_preferred_broadcast_ip()
    
    configure_sysconfig()
    run_command("scylla_dev_mode_setup --developer-mode 1")
    configure_yaml(ip_address, configured_database_port)
    ensure_database_port_open(configured_database_port)
    
    print("[*] Restarting Scylla Server daemon...")
    run_command("systemctl daemon-reload")
    run_command("systemctl restart scylla-server.service")
    
    wait_for_scylla(configured_database_port)
    active_database_password = detect_working_cassandra_password(configured_database_password, configured_database_port)
    change_cassandra_password(configured_database_password, configured_database_port, active_database_password)
    ensure_database_keyspace_exists(configured_database_name, configured_database_port, configured_database_password)
    
    print("\n[+] ScyllaDB Dev Configuration Complete!")
    print(f"    - Access internally via: cqlsh 127.0.0.1 {configured_database_port} -u cassandra -p '{configured_database_password}'")
    print(f"    - Access via {detected_network_provider}: cqlsh {ip_address} {configured_database_port} -u cassandra -p '{configured_database_password}'")

if __name__ == "__main__":
    main()
