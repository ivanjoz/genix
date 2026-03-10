#!/usr/bin/env python3

import json
import os
import pwd
import shutil
import stat
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlparse

SYSTEMD_DIRECTORY = Path("/etc/systemd/system")
SERVICE_INSTALL_DIRECTORY = Path("/usr/local/bin/genix")
SERVICE_BINARY_PATH = SERVICE_INSTALL_DIRECTORY / "genix_app"
SERVICE_NAME = "genix.service"
RESTART_SERVICE_NAME = "genix-restart.service"
RESTART_PATH_NAME = "genix-restart.path"
NGINX_CONFIGURATION_DIRECTORY = Path("/etc/nginx/conf.d")
LETSENCRYPT_DIRECTORY = Path("/etc/letsencrypt/live")
BACKEND_PROXY_URL = "http://127.0.0.1:3589"


def print_debug(message_text):
    print(f"[*] {message_text}")


def fail_with_error(error_message):
    print(f"[!] {error_message}", file=sys.stderr)
    sys.exit(1)


def run_command(command_arguments):
    print_debug(f"Running command: {' '.join(command_arguments)}")
    command_result = subprocess.run(command_arguments, text=True, capture_output=True)
    print_debug(f"Exit code: {command_result.returncode}")

    if command_result.stdout.strip():
        print_debug("stdout:")
        for stdout_line in command_result.stdout.rstrip().splitlines():
            print(f"    {stdout_line}")

    if command_result.stderr.strip():
        print_debug("stderr:")
        for stderr_line in command_result.stderr.rstrip().splitlines():
            print(f"    {stderr_line}")

    if command_result.returncode != 0:
        fail_with_error(f"Command failed: {' '.join(command_arguments)}")

    return command_result


def require_root_execution():
    if os.geteuid() != 0:
        fail_with_error("This script must be executed as root.")


def detect_repository_credentials_path():
    # Walk upward from this script until the repository root named "genix" is found.
    script_path = Path(__file__).resolve()
    for parent_path in script_path.parents:
        if parent_path.name == "genix":
            repository_credentials_path = parent_path / "credentials.json"
            print_debug(f"Using repository credentials path: {repository_credentials_path}")
            return repository_credentials_path

    fail_with_error("Could not detect the repository root named 'genix' from configure_server.py.")


def load_project_credentials(repository_credentials_path):
    print_debug(f"Loading project credentials from {repository_credentials_path}")
    try:
        credentials_content = repository_credentials_path.read_text(encoding="utf-8")
    except OSError as read_error:
        fail_with_error(f"Could not read credentials.json: {read_error}")

    try:
        parsed_credentials = json.loads(credentials_content)
    except json.JSONDecodeError as parse_error:
        fail_with_error(f"Could not parse credentials.json: {parse_error}")

    return parsed_credentials


def extract_api_endpoint_options(project_credentials):
    raw_endpoint_options = project_credentials.get("ENPOINTS")
    if not isinstance(raw_endpoint_options, list) or not raw_endpoint_options:
        fail_with_error("credentials.json must contain a non-empty ENPOINTS array.")

    normalized_endpoint_options = []
    for endpoint_index, raw_endpoint_option in enumerate(raw_endpoint_options, start=1):
        if not isinstance(raw_endpoint_option, dict):
            fail_with_error(f"ENPOINTS[{endpoint_index}] must be an object.")

        endpoint_name = str(raw_endpoint_option.get("name", "")).strip()
        endpoint_route = str(raw_endpoint_option.get("route", "")).strip()
        endpoint_hostname = urlparse(endpoint_route).hostname or ""

        if not endpoint_name or not endpoint_route or not endpoint_hostname:
            fail_with_error(
                f"ENPOINTS[{endpoint_index}] must include valid 'name' and 'route' values."
            )

        normalized_endpoint_options.append(
            {
                "index": endpoint_index,
                "name": endpoint_name,
                "route": endpoint_route,
                "hostname": endpoint_hostname,
            }
        )

    return normalized_endpoint_options


def resolve_selected_endpoint_option(endpoint_options):
    if len(sys.argv) >= 3:
        selected_value = sys.argv[2].strip()
        print_debug(f"Selecting endpoint from CLI argument: {selected_value}")

        for endpoint_option in endpoint_options:
            if selected_value in {
                str(endpoint_option["index"]),
                endpoint_option["name"],
                endpoint_option["route"],
                endpoint_option["hostname"],
            }:
                print_debug(
                    f"Selected endpoint '{endpoint_option['name']}' -> {endpoint_option['hostname']}"
                )
                return endpoint_option

        fail_with_error(
            "The provided endpoint option does not match any ENPOINTS entry by index, name, route, or hostname."
        )

    if not sys.stdin.isatty():
        fail_with_error("No endpoint option provided and no interactive terminal is available.")

    print_debug("Available endpoint options:")
    for endpoint_option in endpoint_options:
        print(
            f"    [{endpoint_option['index']}] {endpoint_option['name']} -> "
            f"{endpoint_option['route']}"
        )

    selected_value = input("Select the endpoint number to configure in Nginx: ").strip()
    for endpoint_option in endpoint_options:
        if selected_value == str(endpoint_option["index"]):
            print_debug(
                f"Selected endpoint '{endpoint_option['name']}' -> {endpoint_option['hostname']}"
            )
            return endpoint_option

    fail_with_error("Invalid endpoint selection.")


def detect_runtime_username():
    ubuntu_account_exists = shutil.which("id") is not None and subprocess.run(
        ["id", "ubuntu"],
        text=True,
        capture_output=True,
    ).returncode == 0
    if ubuntu_account_exists:
        print_debug("Using existing 'ubuntu' account as the service runtime user.")
        return "ubuntu"

    sudo_username = os.environ.get("SUDO_USER", "").strip()
    if sudo_username and sudo_username != "root":
        print_debug(f"Using SUDO_USER '{sudo_username}' as the service runtime user.")
        return sudo_username

    fail_with_error("Could not detect a non-root runtime user. Create 'ubuntu' or run via sudo from a non-root user.")


def resolve_runtime_user(runtime_username):
    try:
        runtime_user_entry = pwd.getpwnam(runtime_username)
    except KeyError as user_error:
        fail_with_error(f"Runtime user '{runtime_username}' does not exist: {user_error}")

    return runtime_user_entry


def ensure_binary_directory(runtime_user_entry):
    print_debug(f"Ensuring install directory exists: {SERVICE_INSTALL_DIRECTORY}")
    SERVICE_INSTALL_DIRECTORY.mkdir(parents=True, exist_ok=True)

    print_debug(
        f"Assigning ownership to {runtime_user_entry.pw_name}:{runtime_user_entry.pw_name} "
        f"for {SERVICE_INSTALL_DIRECTORY}"
    )
    os.chown(SERVICE_INSTALL_DIRECTORY, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)

    # Group write plus setgid keeps uploaded binaries owned by the runtime account and its primary group.
    os.chmod(SERVICE_INSTALL_DIRECTORY, 0o2775)


def ensure_binary_placeholder(runtime_user_entry):
    if SERVICE_BINARY_PATH.exists():
        print_debug(f"Binary already exists at {SERVICE_BINARY_PATH}. Preserving it.")
    else:
        print_debug(f"Creating placeholder binary at {SERVICE_BINARY_PATH}.")
        SERVICE_BINARY_PATH.touch()

    os.chown(SERVICE_BINARY_PATH, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)

    current_mode = stat.S_IMODE(SERVICE_BINARY_PATH.stat().st_mode)
    executable_mode = current_mode | 0o750
    os.chmod(SERVICE_BINARY_PATH, executable_mode)
    print_debug(f"Binary permissions set to {oct(executable_mode)}.")


def ensure_nginx_is_installed():
    nginx_binary_path = shutil.which("nginx")
    if not nginx_binary_path:
        fail_with_error("Nginx is not installed or not available in PATH.")

    if not NGINX_CONFIGURATION_DIRECTORY.exists():
        fail_with_error(f"Nginx configuration directory not found: {NGINX_CONFIGURATION_DIRECTORY}")

    print_debug(f"Detected Nginx binary at {nginx_binary_path}")


def build_http3_nginx_configuration(endpoint_hostname):
    certificate_directory = LETSENCRYPT_DIRECTORY / endpoint_hostname
    certificate_fullchain_path = certificate_directory / "fullchain.pem"
    certificate_private_key_path = certificate_directory / "privkey.pem"

    if certificate_fullchain_path.exists() and certificate_private_key_path.exists():
        print_debug(f"Detected TLS certificates for {endpoint_hostname} at {certificate_directory}")
        return f"""# Map block to handle 0-RTT security (prevents replay attacks on POST/PUT)
map $ssl_early_data $is_early_data {{
    "~on" 1;
    default 0;
}}

server {{
    # Standard TCP and HTTP/3 UDP listeners for TLS-enabled deployments.
    listen 443 quic reuseport;
    listen 443 ssl;
    listen [::]:443 quic reuseport;
    listen [::]:443 ssl;

    server_name {endpoint_hostname};

    ssl_certificate {certificate_fullchain_path};
    ssl_certificate_key {certificate_private_key_path};

    ssl_protocols TLSv1.3;
    ssl_early_data on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets on;

    add_header Alt-Svc 'h3=":443"; ma=86400';

    location / {{
        # Handle browser preflight requests directly at the edge to reduce backend load.
        if ($request_method = 'OPTIONS') {{
            add_header 'Access-Control-Allow-Origin' $http_origin always;
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
            add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type' always;
            add_header 'Access-Control-Max-Age' 86400 always;
            add_header 'Content-Length' 0;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            return 204;
        }}

        if ($request_method != GET) {{
            set $early_data_check "${{is_early_data}}";
        }}
        #if ($early_data_check = "1") {{
        #    return 425;
        #}}

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Early-Data $ssl_early_data;

        proxy_pass {BACKEND_PROXY_URL};

        proxy_pass_header Server;
        server_tokens off;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }}
}}
"""

    print_debug(
        f"No TLS certificates found for {endpoint_hostname}. Writing HTTP-only reverse proxy config."
    )
    return f"""server {{
    # Fallback HTTP config used until TLS certificates are provisioned for this hostname.
    listen 80;
    listen [::]:80;

    server_name {endpoint_hostname};

    location / {{
        # Handle browser preflight requests directly at the edge to reduce backend load.
        if ($request_method = 'OPTIONS') {{
            add_header 'Access-Control-Allow-Origin' $http_origin always;
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
            add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type' always;
            add_header 'Access-Control-Max-Age' 86400 always;
            add_header 'Content-Length' 0;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            return 204;
        }}

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_pass {BACKEND_PROXY_URL};

        proxy_pass_header Server;
        server_tokens off;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }}
}}
"""


def configure_nginx_reverse_proxy(selected_endpoint_option):
    ensure_nginx_is_installed()

    endpoint_hostname = selected_endpoint_option["hostname"]
    nginx_configuration_path = NGINX_CONFIGURATION_DIRECTORY / f"{endpoint_hostname}.conf"
    nginx_configuration_contents = build_http3_nginx_configuration(endpoint_hostname)

    existing_nginx_configuration_contents = None
    if nginx_configuration_path.exists():
        existing_nginx_configuration_contents = nginx_configuration_path.read_text(encoding="utf-8")

    if existing_nginx_configuration_contents == nginx_configuration_contents:
        print_debug(f"Nginx configuration unchanged: {nginx_configuration_path}")
        return False

    print_debug(f"Writing Nginx reverse proxy config: {nginx_configuration_path}")
    nginx_configuration_path.write_text(nginx_configuration_contents, encoding="utf-8")
    os.chmod(nginx_configuration_path, 0o644)

    run_command(["nginx", "-t"])
    run_command(["systemctl", "enable", "nginx"])
    run_command(["systemctl", "restart", "nginx"])
    return True


def build_main_service_contents(runtime_username, repository_credentials_path):
    return f"""[Unit]
Description=Genix Backend Service
After=network.target

[Service]
Type=simple
User={runtime_username}
Group={runtime_username}
WorkingDirectory={SERVICE_INSTALL_DIRECTORY}
Environment=GENIX_CREDENTIALS_FILE={repository_credentials_path}
ExecStart={SERVICE_BINARY_PATH}
Restart=always
RestartSec=5

# Security hardening keeps the process non-root and limits write access to the binary directory only.
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=read-only
ProtectControlGroups=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes
RestrictRealtime=yes
CapabilityBoundingSet=
ReadWritePaths={SERVICE_INSTALL_DIRECTORY}

[Install]
WantedBy=multi-user.target
"""


def build_restart_path_contents():
    return f"""[Unit]
Description=Watch for changes to genix backend binary

[Path]
PathChanged={SERVICE_BINARY_PATH}

[Install]
WantedBy=multi-user.target
"""


def build_restart_service_contents():
    return f"""[Unit]
Description=Restart Genix Service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart {SERVICE_NAME}
"""


def write_unit_file(unit_file_path, unit_contents):
    existing_unit_contents = None
    if unit_file_path.exists():
        existing_unit_contents = unit_file_path.read_text(encoding="utf-8")

    if existing_unit_contents == unit_contents:
        print_debug(f"Configuration unchanged: {unit_file_path}")
        return False

    print_debug(f"Writing systemd unit: {unit_file_path}")
    unit_file_path.write_text(unit_contents, encoding="utf-8")
    os.chmod(unit_file_path, 0o644)
    return True


def configure_systemd_units(runtime_username, repository_credentials_path):
    main_service_configuration_changed = write_unit_file(
        SYSTEMD_DIRECTORY / SERVICE_NAME,
        build_main_service_contents(runtime_username, repository_credentials_path),
    )
    restart_service_configuration_changed = write_unit_file(
        SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME,
        build_restart_service_contents(),
    )
    restart_path_configuration_changed = write_unit_file(
        SYSTEMD_DIRECTORY / RESTART_PATH_NAME,
        build_restart_path_contents(),
    )
    return (
        main_service_configuration_changed
        or restart_service_configuration_changed
        or restart_path_configuration_changed
    )


def enable_units():
    run_command(["systemctl", "enable", SERVICE_NAME])
    run_command(["systemctl", "enable", RESTART_PATH_NAME])


def reload_systemd_if_configuration_changed(systemd_configuration_changed):
    if not systemd_configuration_changed:
        print_debug("Systemd configuration unchanged. Skipping daemon-reload and watcher restart.")
        return

    run_command(["systemctl", "daemon-reload"])
    run_command(["systemctl", "restart", RESTART_PATH_NAME])


def print_summary(runtime_username, repository_credentials_path, selected_endpoint_option):
    print_debug("Configuration completed.")
    print_debug(f"Runtime user: {runtime_username}")
    print_debug(f"Binary path: {SERVICE_BINARY_PATH}")
    print_debug(f"Repository credentials path: {repository_credentials_path}")
    print_debug(
        f"Nginx endpoint: {selected_endpoint_option['name']} -> {selected_endpoint_option['hostname']}"
    )
    print_debug(f"Main service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
    print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
    print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
    print_debug("Upload or replace the executable at the binary path to trigger an automatic restart.")
    print_debug("The service will also keep checking the install directory via its working directory.")


def main():
    require_root_execution()
    repository_credentials_path = detect_repository_credentials_path()
    project_credentials = load_project_credentials(repository_credentials_path)
    endpoint_options = extract_api_endpoint_options(project_credentials)
    selected_endpoint_option = resolve_selected_endpoint_option(endpoint_options)
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    ensure_binary_directory(runtime_user_entry)
    ensure_binary_placeholder(runtime_user_entry)
    systemd_configuration_changed = configure_systemd_units(
        runtime_username, repository_credentials_path
    )
    configure_nginx_reverse_proxy(selected_endpoint_option)
    enable_units()
    reload_systemd_if_configuration_changed(systemd_configuration_changed)
    print_summary(runtime_username, repository_credentials_path, selected_endpoint_option)


if __name__ == "__main__":
    main()
