#!/usr/bin/env python3

"""Install the SSE bridge (sse_bridge/) on this host: systemd units plus its Nginx vhost.

Deliberately simpler than configure_server.py, which has to cover a backend host and an Nginx
edge host that are usually two different machines. The bridge is the opposite case: Nginx must
terminate TLS on the very machine the bridge runs on, because what it proxies is a permanent
stream and there is nothing to gain from a second hop. So there are no install modes and no
upstream to configure — the vhost always forwards to 127.0.0.1.

It also asks for exactly one thing, and only when it is missing: sse_bridge.apikey, the shared
secret the bridge needs to authenticate anyone (the browser's session token and the backend's
X-Bridge-Auth header are both HMACs keyed with it). Everything else is read from config.toml
or defaulted, and a missing value fails with the key name instead of opening a prompt.

Nginx specifics: the vhost never buffers (a buffered text/event-stream is a stalled request), it
adds no CORS headers (the bridge answers preflights itself in handlers.go — a duplicated
Access-Control-Allow-Origin makes browsers reject the response), and it serves HTTP/3 whenever a
certificate for the hostname exists.
"""

import getpass
import os
import re
import shutil
import sys
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from configure_server import (  # noqa: E402  (the sys.path line above must run first)
    LETSENCRYPT_DIRECTORY,
    NGINX_CONFIGURATION_DIRECTORY,
    SERVICE_INSTALL_DIRECTORY,
    SYSTEMD_DIRECTORY,
    build_unprivileged_command,
    detect_go_binary,
    detect_repository_config_path,
    detect_runtime_username,
    detect_unprivileged_username,
    ensure_binary_directory,
    ensure_nginx_is_installed,
    extract_existing_certbot_tls_lines,
    fail_with_error,
    is_usable_executable,
    load_project_credentials,
    print_debug,
    require_root_execution,
    resolve_go_architecture,
    resolve_runtime_user,
    run_command,
    validate_nginx_domain,
    validate_server_port,
    write_unit_file,
)
from toml_config import get_config_value, set_config_values  # noqa: E402

BRIDGE_SOURCE_DIRECTORY_NAME = "sse_bridge"
BRIDGE_BINARY_NAME = "sse_bridge"
BRIDGE_BINARY_PATH = SERVICE_INSTALL_DIRECTORY / BRIDGE_BINARY_NAME
BRIDGE_SERVICE_NAME = "genix-sse-bridge.service"
BRIDGE_RESTART_SERVICE_NAME = "genix-sse-bridge-restart.service"
BRIDGE_RESTART_PATH_NAME = "genix-sse-bridge-restart.path"

# Must match defaultListenPort in sse_bridge/config.go.
DEFAULT_BRIDGE_PORT = 14012

# The bridge reads sse_bridge.apikey; a developer machine has the backend's full config.toml
# instead, where the same value is secret_phrase. config.go accepts both, and so does this script.
BRIDGE_API_KEY_NAME = "sse_bridge.apikey"
BACKEND_SECRET_NAME = "secret_phrase"
MINIMUM_API_KEY_LENGTH = 8

# An AWS function URL in sse_bridge.url means "no bridge" (the backend serves its own
# /agent/stream), so it cannot be the hostname this vhost is built for.
LAMBDA_URL_HOST_SUFFIXES = (".on.aws", ".amazonaws.com")

# reuseport may be set by exactly one server block per address:port. A second one anywhere in the
# Nginx configuration fails `nginx -t` with "duplicate listen options", which is what would happen
# when the bridge shares its host with the backend vhost written by configure_server.py.
NGINX_LISTEN_REUSEPORT_PATTERN = re.compile(r"^\s*listen\s+[^;]*\breuseport\b", re.MULTILINE)


def resolve_bridge_domain(project_credentials, repository_config_path):
    """Take the vhost hostname from sse_bridge.url. Never prompts: this key is not a secret.

    It is also the key the backend and the frontend read to decide whether to use the bridge at
    all, so getting it wrong here would install a host nobody talks to.
    """
    configured_url = str(get_config_value(project_credentials, "sse_bridge.url", "")).strip()
    if not configured_url:
        fail_with_error(
            f"sse_bridge.url is not set in {repository_config_path}. Add the public URL of "
            'this bridge (e.g. url = "https://genix-sse.un.pe/" under [sse_bridge]) and run again.'
        )

    candidate_value = configured_url
    if "://" in candidate_value:
        url_scheme, _, candidate_value = candidate_value.partition("://")
        if url_scheme not in {"http", "https"}:
            fail_with_error(f"sse_bridge.url has an unsupported scheme '{url_scheme}://': {configured_url}")

    candidate_host = candidate_value.split("/", 1)[0].partition(":")[0]
    bridge_domain, domain_error = validate_nginx_domain(candidate_host)
    if bridge_domain is None:
        fail_with_error(f"sse_bridge.url is unusable ({domain_error}): {configured_url}")

    if bridge_domain.endswith(LAMBDA_URL_HOST_SUFFIXES):
        fail_with_error(
            f"sse_bridge.url points at the AWS function URL ({bridge_domain}), which is how the "
            "project says 'no bridge'. Set it to the public domain of this host and run again."
        )

    print_debug(f"Bridge domain from sse_bridge.url: {bridge_domain}")
    return bridge_domain


def resolve_bridge_port(project_credentials):
    """Resolve the port the bridge binds to. Absent is normal — the Go default covers it."""
    configured_port = get_config_value(project_credentials, "sse_bridge.port")
    if configured_port is None:
        print_debug(f"sse_bridge.port is not set in config.toml. Using {DEFAULT_BRIDGE_PORT}.")
        return DEFAULT_BRIDGE_PORT

    bridge_port, validation_error = validate_server_port(str(configured_port))
    if bridge_port is None:
        # Never fall back silently: the Nginx upstream is built from this number, so a typo would
        # produce a vhost proxying to a port nothing listens on.
        fail_with_error(f"sse_bridge.port in config.toml is unusable: {validation_error}")

    print_debug(f"Bridge listen port: {bridge_port}")
    return bridge_port


def store_bridge_api_key(repository_config_path, bridge_api_key):
    """Merge the typed-in key into config.toml, which is where the bridge reads it from.

    Not offered as a choice: the service cannot start without it, so declining would only install
    a unit that fails on boot.
    """
    config_file_already_existed = repository_config_path.exists()
    if not config_file_already_existed:
        repository_config_path.parent.mkdir(parents=True, exist_ok=True)
        repository_config_path.touch()

    try:
        set_config_values(repository_config_path, {BRIDGE_API_KEY_NAME: bridge_api_key})
    except OSError as write_error:
        fail_with_error(f"Could not write config.toml: {write_error}")

    if not config_file_already_existed:
        # A root-owned file would be unreadable to the non-root service, so hand it to whoever
        # owns the directory holding it.
        parent_directory_stat = repository_config_path.parent.stat()
        os.chown(repository_config_path, parent_directory_stat.st_uid, parent_directory_stat.st_gid)

    os.chmod(repository_config_path, 0o600)
    print_debug(f"Stored {BRIDGE_API_KEY_NAME} in {repository_config_path} (mode 0600).")


def resolve_bridge_api_key(project_credentials, repository_config_path):
    """The only value this script ever asks for, and only when the file does not already have it.

    It must be byte-identical to the backend's secret_phrase: it is the HMAC key of the session
    tokens the backend issues and of the X-Bridge-Auth header it signs. A mismatch is not a
    startup error, it is every request being rejected at runtime.
    """
    for credential_name in (BRIDGE_API_KEY_NAME, BACKEND_SECRET_NAME):
        stored_api_key = str(get_config_value(project_credentials, credential_name, "")).strip()
        if stored_api_key:
            print_debug(f"Using the shared secret already stored as {credential_name}.")
            return stored_api_key, False

    if not sys.stdin.isatty():
        fail_with_error(
            f"{BRIDGE_API_KEY_NAME} is missing from {repository_config_path} and there is no "
            f"interactive terminal to ask for it. Add it (the backend's {BACKEND_SECRET_NAME} "
            "value) and run the script again."
        )

    print_debug(
        f"{BRIDGE_API_KEY_NAME} is not stored yet. It must be the same value as the backend's "
        f"{BACKEND_SECRET_NAME}, otherwise the bridge rejects every browser and every backend call."
    )
    while True:
        # getpass: this is the key that authenticates the whole bridge, so it is not echoed and
        # never printed back in the summary.
        typed_api_key = getpass.getpass(f"Enter {BRIDGE_API_KEY_NAME} (input hidden): ").strip()
        if len(typed_api_key) >= MINIMUM_API_KEY_LENGTH:
            return typed_api_key, True

        print(f"[!] The key must have at least {MINIMUM_API_KEY_LENGTH} characters.", file=sys.stderr)


def build_tls_directive_lines(bridge_domain, existing_nginx_configuration_contents):
    """Return the ssl_* lines for this hostname, or an empty list when there is no certificate.

    Certbot rewrites the vhost it manages, so its own directives win over the default paths: a
    certificate stored somewhere else keeps working across reruns of this script.
    """
    preserved_tls_lines = extract_existing_certbot_tls_lines(existing_nginx_configuration_contents)
    has_certificate = any(line.strip().startswith("ssl_certificate ") for line in preserved_tls_lines)
    has_certificate_key = any(line.strip().startswith("ssl_certificate_key ") for line in preserved_tls_lines)
    if has_certificate and has_certificate_key:
        return preserved_tls_lines

    certificate_directory = LETSENCRYPT_DIRECTORY / bridge_domain
    certificate_fullchain_path = certificate_directory / "fullchain.pem"
    certificate_private_key_path = certificate_directory / "privkey.pem"
    if certificate_fullchain_path.exists() and certificate_private_key_path.exists():
        print_debug(f"Detected TLS certificates for {bridge_domain} at {certificate_directory}")
        return [
            f"    ssl_certificate {certificate_fullchain_path};",
            f"    ssl_certificate_key {certificate_private_key_path};",
        ]

    return []


def detect_reuseport_listener_owner(own_configuration_path):
    """Find another Nginx file that already claims reuseport, so this vhost can drop it.

    reuseport asks Nginx for a dedicated socket per worker on that address:port, and only the
    first server block may request it. The backend vhost written by configure_server.py does.
    """
    candidate_configuration_paths = []
    nginx_root_directory = NGINX_CONFIGURATION_DIRECTORY.parent
    for configuration_glob in ("nginx.conf", "conf.d/*.conf", "sites-enabled/*"):
        candidate_configuration_paths.extend(sorted(nginx_root_directory.glob(configuration_glob)))

    for candidate_configuration_path in candidate_configuration_paths:
        if not candidate_configuration_path.is_file():
            continue
        try:
            if candidate_configuration_path.resolve() == own_configuration_path.resolve():
                continue
            configuration_contents = candidate_configuration_path.read_text(encoding="utf-8")
        except OSError:
            continue

        if NGINX_LISTEN_REUSEPORT_PATTERN.search(configuration_contents):
            return candidate_configuration_path

    return None


def build_bridge_nginx_configuration(
    bridge_domain,
    bridge_port,
    existing_nginx_configuration_contents="",
    reuseport_is_available=True,
):
    """Render the vhost: HTTP/3 over TLS when a certificate exists, plain HTTP otherwise."""
    tls_directive_lines = build_tls_directive_lines(bridge_domain, existing_nginx_configuration_contents)

    # The settings that make the difference between a stream and a stalled request. Identical in
    # both variants below, so they live in one string.
    streaming_location_body = f"""        # The bridge always runs on this host: nothing is gained by proxying a permanent
        # stream across a second machine.
        proxy_pass http://127.0.0.1:{bridge_port};

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # HTTP/1.1 with an empty Connection header: the bridge speaks plain HTTP and there is
        # no Upgrade handshake to tunnel, unlike the WebSocket-era backend vhost.
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        # Without these three the events pile up in Nginx instead of reaching the browser.
        proxy_buffering off;
        proxy_cache off;
        gzip off;

        proxy_connect_timeout 10s;
        proxy_send_timeout 3600s;
        # An idle stream must survive between agent turns; the bridge keepalive is every 20s.
        proxy_read_timeout 3600s;

        proxy_pass_header Server;
        server_tokens off;

        # No CORS headers here on purpose: the bridge sets them and answers preflights itself
        # (withClientCORS), and a duplicated Access-Control-Allow-Origin is rejected by browsers."""

    if tls_directive_lines:
        tls_directives = "\n".join(tls_directive_lines)
        quic_listen_options = " reuseport" if reuseport_is_available else ""
        if not reuseport_is_available:
            print_debug("Another Nginx vhost already owns reuseport on :443. Listening without it.")

        return f"""server {{
    # HTTP/3 (QUIC) plus the TCP listener browsers use before they see Alt-Svc.
    listen 443 quic{quic_listen_options};
    listen 443 ssl;
    listen [::]:443 quic{quic_listen_options};
    listen [::]:443 ssl;
    http2 on;

    server_name {bridge_domain};

{tls_directives}

    ssl_protocols TLSv1.3;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets on;
    # ssl_early_data stays off: POST /in is not replay-safe and 0-RTT buys nothing for a
    # connection that stays open for the whole session.

    add_header Alt-Svc 'h3=":443"; ma=86400' always;

    location / {{
{streaming_location_body}
    }}
}}
"""

    print_debug(
        f"No TLS certificates found for {bridge_domain}. Writing HTTP-only config "
        "(run certbot, then this script again, to get HTTP/3)."
    )
    return f"""server {{
    # Fallback used until TLS certificates are provisioned for this hostname.
    listen 80;
    listen [::]:80;

    server_name {bridge_domain};

    location / {{
{streaming_location_body}
    }}
}}
"""


def configure_bridge_nginx_vhost(bridge_domain, bridge_port):
    ensure_nginx_is_installed()

    nginx_configuration_path = NGINX_CONFIGURATION_DIRECTORY / f"{bridge_domain}.conf"

    existing_nginx_configuration_contents = None
    if nginx_configuration_path.exists():
        existing_nginx_configuration_contents = nginx_configuration_path.read_text(encoding="utf-8")

    reuseport_owner_path = detect_reuseport_listener_owner(nginx_configuration_path)
    if reuseport_owner_path:
        print_debug(f"reuseport is already claimed by {reuseport_owner_path}")

    nginx_configuration_contents = build_bridge_nginx_configuration(
        bridge_domain,
        bridge_port,
        existing_nginx_configuration_contents or "",
        reuseport_is_available=reuseport_owner_path is None,
    )

    if existing_nginx_configuration_contents == nginx_configuration_contents:
        print_debug(f"Nginx configuration unchanged: {nginx_configuration_path}")
        return

    print_debug(f"Writing Nginx vhost for the bridge: {nginx_configuration_path}")
    nginx_configuration_path.write_text(nginx_configuration_contents, encoding="utf-8")
    os.chmod(nginx_configuration_path, 0o644)

    run_command(["nginx", "-t"])
    run_command(["systemctl", "enable", "nginx"])
    run_command(["systemctl", "restart", "nginx"])


def detect_bridge_source_directory(repository_root_path):
    bridge_source_directory = repository_root_path / BRIDGE_SOURCE_DIRECTORY_NAME
    has_go_module = (bridge_source_directory / "go.mod").is_file()
    has_main_package = (bridge_source_directory / "main.go").is_file()

    if has_go_module and has_main_package:
        print_debug(f"Bridge source found at {bridge_source_directory}")
        return bridge_source_directory

    print_debug(f"No compilable bridge source at {bridge_source_directory}")
    return None


def compile_bridge_binary(bridge_source_directory, repository_root_path):
    """Build sse_bridge/ into tmp/. Simpler than the backend: no submodules, no local replaces."""
    go_binary_path = detect_go_binary()
    if not go_binary_path:
        fail_with_error(
            "Bridge source is present but the Go toolchain was not found. Install Go or run the "
            "script with GO_BINARY=/path/to/go (sudo strips /usr/local/go/bin from PATH)."
        )

    print_debug(f"Using Go toolchain at {go_binary_path}")
    unprivileged_username = detect_unprivileged_username(repository_root_path)

    build_output_directory = repository_root_path / "tmp"
    build_output_directory.mkdir(parents=True, exist_ok=True)

    # The compile runs as a non-root account, so tmp/ must belong to the repository owner rather
    # than to root — which is what it would be after mkdir here, or after an older root-run build.
    repository_directory_stat = repository_root_path.stat()
    if os.geteuid() == 0 and build_output_directory.stat().st_uid != repository_directory_stat.st_uid:
        print_debug(f"Handing {build_output_directory} to uid {repository_directory_stat.st_uid}")
        os.chown(build_output_directory, repository_directory_stat.st_uid, repository_directory_stat.st_gid)

    build_output_path = build_output_directory / f"{BRIDGE_BINARY_NAME}_linux_{resolve_go_architecture()}"
    build_command = build_unprivileged_command(
        [str(go_binary_path), "build", "-ldflags", "-s -w", "-o", str(build_output_path), "."],
        unprivileged_username,
    )
    if os.geteuid() == 0 and unprivileged_username:
        print_debug(f"Compiling as '{unprivileged_username}' to reuse that account's Go module cache.")

    build_started_at = datetime.now().strftime("%H:%M:%S")
    print_debug(f"Compiling the bridge from {bridge_source_directory} (started {build_started_at})...")
    run_command(build_command, working_directory=bridge_source_directory, stream_output=True)

    if not is_usable_executable(build_output_path):
        fail_with_error(f"The compiler reported success but {build_output_path} is not a valid executable.")

    print_debug(f"Compilation successful: {build_output_path}")
    return build_output_path


def find_prebuilt_bridge_binary(repository_root_path):
    go_architecture = resolve_go_architecture()
    candidate_binary_paths = [
        BRIDGE_BINARY_PATH,
        repository_root_path / "tmp" / f"{BRIDGE_BINARY_NAME}_linux_{go_architecture}",
        repository_root_path / BRIDGE_SOURCE_DIRECTORY_NAME / BRIDGE_BINARY_NAME,
    ]

    for candidate_binary_path in candidate_binary_paths:
        if is_usable_executable(candidate_binary_path):
            print_debug(f"Found a prebuilt bridge binary at {candidate_binary_path}")
            return candidate_binary_path

        print_debug(f"No usable binary at {candidate_binary_path}")

    return None


def install_bridge_binary(source_binary_path, runtime_user_entry):
    if source_binary_path == BRIDGE_BINARY_PATH:
        print_debug(f"Binary already in place at {BRIDGE_BINARY_PATH}. Fixing ownership only.")
        os.chown(BRIDGE_BINARY_PATH, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
        os.chmod(BRIDGE_BINARY_PATH, 0o750)
        return

    # Write next to the destination and rename, so the running service never sees a half-copied
    # file. PathChanged= watches IN_MOVED_TO as well, so the restart watcher still fires.
    staged_binary_path = SERVICE_INSTALL_DIRECTORY / f".{BRIDGE_BINARY_NAME}.staged"
    print_debug(f"Installing {source_binary_path} to {BRIDGE_BINARY_PATH}")
    try:
        shutil.copyfile(source_binary_path, staged_binary_path)
        os.chown(staged_binary_path, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
        os.chmod(staged_binary_path, 0o750)
        os.replace(staged_binary_path, BRIDGE_BINARY_PATH)
    except OSError as install_error:
        staged_binary_path.unlink(missing_ok=True)
        fail_with_error(f"Could not install the bridge binary: {install_error}")


def provide_bridge_binary(repository_root_path, runtime_user_entry):
    """Put a runnable bridge at BRIDGE_BINARY_PATH: compile it, or find one, or fail."""
    bridge_source_directory = detect_bridge_source_directory(repository_root_path)
    if bridge_source_directory:
        install_bridge_binary(
            compile_bridge_binary(bridge_source_directory, repository_root_path), runtime_user_entry
        )
        return

    print_debug("Falling back to a prebuilt binary because there is no bridge source to compile.")
    prebuilt_binary_path = find_prebuilt_bridge_binary(repository_root_path)
    if not prebuilt_binary_path:
        fail_with_error(
            f"No bridge source under {repository_root_path / BRIDGE_SOURCE_DIRECTORY_NAME} and no "
            f"prebuilt binary found. Clone the repository with its sse_bridge folder, or place a "
            f"compiled binary at {BRIDGE_BINARY_PATH}, and run the script again."
        )

    install_bridge_binary(prebuilt_binary_path, runtime_user_entry)


def build_bridge_service_contents(runtime_username, repository_config_path, bridge_port):
    return f"""[Unit]
Description=Genix SSE Bridge
After=network.target

[Service]
Type=simple
User={runtime_username}
Group={runtime_username}
WorkingDirectory={SERVICE_INSTALL_DIRECTORY}
# sse_bridge.apikey is read from this file rather than exported here: a unit under
# /etc/systemd/system is world-readable, and config.toml is not.
Environment=GENIX_CONFIG_FILE={repository_config_path}
Environment=SSE_BRIDGE_PORT={bridge_port}
ExecStart={BRIDGE_BINARY_PATH}
Restart=always
RestartSec=3

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


def build_bridge_restart_path_contents():
    return f"""[Unit]
Description=Watch for changes to the genix SSE bridge binary

[Path]
PathChanged={BRIDGE_BINARY_PATH}

[Install]
WantedBy=multi-user.target
"""


def build_bridge_restart_service_contents():
    return f"""[Unit]
Description=Restart the Genix SSE Bridge

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart {BRIDGE_SERVICE_NAME}
"""


def configure_bridge_systemd_units(runtime_username, repository_config_path, bridge_port):
    unit_files_changed = [
        write_unit_file(
            SYSTEMD_DIRECTORY / BRIDGE_SERVICE_NAME,
            build_bridge_service_contents(runtime_username, repository_config_path, bridge_port),
        ),
        write_unit_file(
            SYSTEMD_DIRECTORY / BRIDGE_RESTART_SERVICE_NAME,
            build_bridge_restart_service_contents(),
        ),
        write_unit_file(
            SYSTEMD_DIRECTORY / BRIDGE_RESTART_PATH_NAME,
            build_bridge_restart_path_contents(),
        ),
    ]
    return any(unit_files_changed)


def start_bridge_service(systemd_configuration_changed):
    if systemd_configuration_changed:
        run_command(["systemctl", "daemon-reload"])

    run_command(["systemctl", "enable", BRIDGE_SERVICE_NAME])
    run_command(["systemctl", "enable", BRIDGE_RESTART_PATH_NAME])
    if systemd_configuration_changed:
        run_command(["systemctl", "restart", BRIDGE_RESTART_PATH_NAME])

    # The binary is installed before the watcher is (re)started, so that first change is never
    # seen by it. A bridge that cannot start must not fail the whole run: the units are already in
    # place and journalctl has the reason.
    service_start_result = run_command(["systemctl", "restart", BRIDGE_SERVICE_NAME], allow_failure=True)
    if service_start_result.returncode != 0:
        print_debug(
            f"WARNING: {BRIDGE_SERVICE_NAME} did not start. The units are installed; "
            f"check 'journalctl -u {BRIDGE_SERVICE_NAME} -n 50' for the reason."
        )
        return

    print_debug(f"{BRIDGE_SERVICE_NAME} is running.")


def warn_if_credentials_are_unreadable(runtime_username, repository_config_path):
    """The unit points at this file and the service reads it as a non-root user.

    A config.toml copied onto the host as root with mode 0600 is invisible to that account,
    and the only symptom would be the bridge exiting at boot with "no se encontró
    apikey". Say it here instead, where the fix is one command away.
    """
    readability_result = run_command(
        ["sudo", "-u", runtime_username, "test", "-r", str(repository_config_path)],
        allow_failure=True,
    )
    if readability_result.returncode == 0:
        return

    print_debug(
        f"WARNING: '{runtime_username}' cannot read {repository_config_path}, so the bridge "
        f"will not find its API key. Fix it with: chown {runtime_username} {repository_config_path}"
    )


def install_bridge_systemd_service(repository_config_path, bridge_port):
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    warn_if_credentials_are_unreadable(runtime_username, repository_config_path)
    ensure_binary_directory(runtime_user_entry)
    provide_bridge_binary(repository_config_path.parent, runtime_user_entry)
    systemd_configuration_changed = configure_bridge_systemd_units(
        runtime_username, repository_config_path, bridge_port
    )
    start_bridge_service(systemd_configuration_changed)
    return runtime_username


def print_summary(repository_config_path, runtime_username, bridge_domain, bridge_port):
    print_debug("SSE bridge configuration completed.")
    print_debug(f"Config file: {repository_config_path}")
    print_debug(f"Runtime user: {runtime_username}")

    binary_size_in_bytes = BRIDGE_BINARY_PATH.stat().st_size if BRIDGE_BINARY_PATH.is_file() else 0
    print_debug(f"Binary path: {BRIDGE_BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
    print_debug(f"Listen port (SSE_BRIDGE_PORT): {bridge_port}")
    print_debug(f"Service unit: {SYSTEMD_DIRECTORY / BRIDGE_SERVICE_NAME}")
    print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / BRIDGE_RESTART_PATH_NAME}")
    print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / BRIDGE_RESTART_SERVICE_NAME}")
    print_debug(f"Nginx vhost: {NGINX_CONFIGURATION_DIRECTORY / (bridge_domain + '.conf')}")
    print_debug(f"Health check: curl -s http://127.0.0.1:{bridge_port}/health")
    print_debug(f"Through Nginx: curl -s https://{bridge_domain}/health")
    print_debug(
        f"Reminder: the same sse_bridge.url (https://{bridge_domain}/) must be in the "
        "config.toml used to build the backend and the frontend, or neither uses the bridge."
    )


def main():
    require_root_execution()
    if len(sys.argv) > 1:
        # configure_server.py takes an install mode here; this script has only one.
        print_debug(f"Ignoring extra argument(s): {' '.join(sys.argv[1:])}. This script has no modes.")

    repository_config_path = detect_repository_config_path()
    project_credentials = load_project_credentials(repository_config_path)

    # Resolve everything before touching the system, so a missing value stops the run instead of
    # leaving the host half configured.
    bridge_domain = resolve_bridge_domain(project_credentials, repository_config_path)
    bridge_port = resolve_bridge_port(project_credentials)
    bridge_api_key, api_key_was_prompted = resolve_bridge_api_key(
        project_credentials, repository_config_path
    )
    if api_key_was_prompted:
        store_bridge_api_key(repository_config_path, bridge_api_key)

    runtime_username = install_bridge_systemd_service(repository_config_path, bridge_port)
    configure_bridge_nginx_vhost(bridge_domain, bridge_port)
    print_summary(repository_config_path, runtime_username, bridge_domain, bridge_port)


if __name__ == "__main__":
    main()
