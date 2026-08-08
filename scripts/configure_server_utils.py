#!/usr/bin/env python3

"""Install server_utils/ on this host: systemd units plus the Nginx vhost for its SSE bridge.

One Rust binary, genix-server-utils, hosts two independent services:

  - The credit rate limiter, on a raw TCP port that stays on loopback. It is authenticated by
    HMAC but not encrypted, so it never gets an Nginx vhost.
  - The SSE bridge, on an HTTP port that must be reachable by browsers. Nginx terminates TLS
    for it on this very machine: what it proxies is a permanent stream, and a second hop buys
    nothing.

Deliberately simpler than configure_server.py, which covers a backend host and an Nginx edge
host that are usually two different machines. Here there is no upstream to configure — the
vhost always forwards to 127.0.0.1.

It asks for nothing. Every value comes from config.toml, and a missing one fails with the key
name instead of opening a prompt: both secrets it needs (secret_phrase, internal_apikey) are
root-level keys that initial project setup already writes, exactly like admin_password.

Nginx specifics: the vhost never buffers (a buffered text/event-stream is a stalled request),
it adds no CORS headers (the bridge answers preflights itself — a duplicated
Access-Control-Allow-Origin makes browsers reject the response), and it serves HTTP/3 whenever
a certificate for the hostname exists.
"""

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
from toml_config import get_config_value  # noqa: E402

SOURCE_DIRECTORY_NAME = "server_utils"
# Cargo's package name, which is also the file name it produces.
BINARY_NAME = "genix-server-utils"
BINARY_PATH = SERVICE_INSTALL_DIRECTORY / BINARY_NAME
SERVICE_NAME = "genix-server-utils.service"
RESTART_SERVICE_NAME = "genix-server-utils-restart.service"
RESTART_PATH_NAME = "genix-server-utils-restart.path"

# Must match DEFAULT_BRIDGE_PORT in server_utils/src/config.rs.
DEFAULT_BRIDGE_PORT = 14012

# Both are root-level keys in config.toml. secret_phrase verifies the browser's session token;
# internal_apikey authenticates the backend's calls (and the rate limiter's TCP frames).
REQUIRED_SECRET_NAMES = ("secret_phrase", "internal_apikey")

# An AWS function URL in sse_bridge.url means "no bridge" (the backend serves its own
# /agent/stream), so it cannot be the hostname this vhost is built for.
LAMBDA_URL_HOST_SUFFIXES = (".on.aws", ".amazonaws.com")

# reuseport may be set by exactly one server block per address:port. A second one anywhere in
# the Nginx configuration fails `nginx -t` with "duplicate listen options", which is what would
# happen when this host also serves the backend vhost written by configure_server.py.
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
    """Resolve the port the SSE bridge binds to. Absent is normal — the Rust default covers it."""
    configured_port = get_config_value(project_credentials, "sse_bridge.port")
    if configured_port is None:
        print_debug(f"sse_bridge.port is not set in config.toml. Using {DEFAULT_BRIDGE_PORT}.")
        return DEFAULT_BRIDGE_PORT

    bridge_port, validation_error = validate_server_port(str(configured_port))
    if bridge_port is None:
        # Never fall back silently: the Nginx upstream is built from this number, so a typo
        # would produce a vhost proxying to a port nothing listens on.
        fail_with_error(f"sse_bridge.port in config.toml is unusable: {validation_error}")

    print_debug(f"SSE bridge listen port: {bridge_port}")
    return bridge_port


def verify_required_secrets(project_credentials, repository_config_path):
    """Fail early when a secret the process needs at startup is missing.

    Not prompted for: these are root-level keys written during initial project setup, and the
    values must match the backend byte for byte. Guessing one here would install a service that
    rejects every request at runtime instead of failing visibly now.
    """
    for secret_name in REQUIRED_SECRET_NAMES:
        if not str(get_config_value(project_credentials, secret_name, "")).strip():
            fail_with_error(
                f"{secret_name} is not set in {repository_config_path}. It must hold the same "
                "value the backend uses, otherwise every request is rejected at runtime."
            )

    print_debug(f"Found both required secrets: {', '.join(REQUIRED_SECRET_NAMES)}.")


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

        # No CORS headers here on purpose: the bridge sets them and answers preflights itself,
        # and a duplicated Access-Control-Allow-Origin is rejected by browsers."""

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


def detect_source_directory(repository_root_path):
    source_directory = repository_root_path / SOURCE_DIRECTORY_NAME
    has_cargo_manifest = (source_directory / "Cargo.toml").is_file()
    has_main_source = (source_directory / "src" / "main.rs").is_file()

    if has_cargo_manifest and has_main_source:
        print_debug(f"Rust source found at {source_directory}")
        return source_directory

    print_debug(f"No compilable Rust source at {source_directory}")
    return None


def detect_cargo_binary(unprivileged_username):
    """Locate cargo. sudo strips ~/.cargo/bin from PATH, so the owner's rustup install is
    checked explicitly before giving up."""
    configured_cargo_binary = os.environ.get("CARGO_BINARY", "").strip()
    if configured_cargo_binary:
        configured_cargo_path = Path(configured_cargo_binary)
        if not os.access(configured_cargo_path, os.X_OK):
            fail_with_error(f"CARGO_BINARY is set to '{configured_cargo_binary}' but it is not executable.")
        return configured_cargo_path

    cargo_binary_in_path = shutil.which("cargo")
    if cargo_binary_in_path:
        return Path(cargo_binary_in_path)

    candidate_cargo_paths = [Path("/usr/local/cargo/bin/cargo")]
    if unprivileged_username:
        candidate_cargo_paths.insert(0, Path(f"~{unprivileged_username}/.cargo/bin/cargo").expanduser())
    for candidate_cargo_path in candidate_cargo_paths:
        if os.access(candidate_cargo_path, os.X_OK):
            return candidate_cargo_path

    return None


def compile_binary(source_directory, repository_root_path):
    """Build server_utils/ in release mode, as the repository owner so the Cargo registry and
    target/ cache are reused instead of being recreated root-owned inside the clone."""
    unprivileged_username = detect_unprivileged_username(repository_root_path)
    cargo_binary_path = detect_cargo_binary(unprivileged_username)
    if not cargo_binary_path:
        fail_with_error(
            "Rust source is present but cargo was not found. Install Rust (https://rustup.rs) or "
            "run the script with CARGO_BINARY=/path/to/cargo (sudo strips ~/.cargo/bin from PATH)."
        )

    print_debug(f"Using cargo at {cargo_binary_path}")
    if os.geteuid() == 0 and unprivileged_username:
        print_debug(f"Compiling as '{unprivileged_username}' to reuse that account's Cargo cache.")

    build_command = build_unprivileged_command(
        [str(cargo_binary_path), "build", "--release"], unprivileged_username
    )
    build_started_at = datetime.now().strftime("%H:%M:%S")
    print_debug(f"Compiling {source_directory} in release mode (started {build_started_at})...")
    run_command(build_command, working_directory=source_directory, stream_output=True)

    build_output_path = source_directory / "target" / "release" / BINARY_NAME
    if not is_usable_executable(build_output_path):
        fail_with_error(f"cargo reported success but {build_output_path} is not a valid executable.")

    print_debug(f"Compilation successful: {build_output_path}")
    return build_output_path


def find_prebuilt_binary(repository_root_path):
    candidate_binary_paths = [
        BINARY_PATH,
        repository_root_path / "tmp" / f"{BINARY_NAME}_linux_{resolve_go_architecture()}",
        repository_root_path / SOURCE_DIRECTORY_NAME / "target" / "release" / BINARY_NAME,
    ]

    for candidate_binary_path in candidate_binary_paths:
        if is_usable_executable(candidate_binary_path):
            print_debug(f"Found a prebuilt binary at {candidate_binary_path}")
            return candidate_binary_path

        print_debug(f"No usable binary at {candidate_binary_path}")

    return None


def install_binary(source_binary_path, runtime_user_entry):
    if source_binary_path == BINARY_PATH:
        print_debug(f"Binary already in place at {BINARY_PATH}. Fixing ownership only.")
        os.chown(BINARY_PATH, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
        os.chmod(BINARY_PATH, 0o750)
        return

    # Write next to the destination and rename, so the running service never reads a half-copied
    # file. PathChanged= watches IN_MOVED_TO as well, so the restart watcher still fires.
    staged_binary_path = SERVICE_INSTALL_DIRECTORY / f".{BINARY_NAME}.staged"
    print_debug(f"Installing {source_binary_path} to {BINARY_PATH}")
    try:
        shutil.copyfile(source_binary_path, staged_binary_path)
        os.chown(staged_binary_path, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
        os.chmod(staged_binary_path, 0o750)
        os.replace(staged_binary_path, BINARY_PATH)
    except OSError as install_error:
        staged_binary_path.unlink(missing_ok=True)
        fail_with_error(f"Could not install the binary: {install_error}")


def provide_binary(repository_root_path, runtime_user_entry):
    """Put a runnable binary at BINARY_PATH: compile it, or find one, or fail."""
    source_directory = detect_source_directory(repository_root_path)
    if source_directory:
        install_binary(compile_binary(source_directory, repository_root_path), runtime_user_entry)
        return

    print_debug("Falling back to a prebuilt binary because there is no Rust source to compile.")
    prebuilt_binary_path = find_prebuilt_binary(repository_root_path)
    if not prebuilt_binary_path:
        fail_with_error(
            f"No source under {repository_root_path / SOURCE_DIRECTORY_NAME} and no prebuilt "
            f"binary found. Clone the repository with its {SOURCE_DIRECTORY_NAME} folder, or "
            f"place a compiled binary at {BINARY_PATH}, and run the script again."
        )

    install_binary(prebuilt_binary_path, runtime_user_entry)


def build_service_contents(runtime_username, repository_config_path, bridge_port):
    return f"""[Unit]
Description=Genix Server Utilities (credit rate limiter + SSE bridge)
# The rate limiter loads existing usage from ScyllaDB before admitting anything and exits when
# it cannot, which also stops the bridge: one process, shared fate. Deploy the backend tables
# (so credit_usage exists) before enabling this unit.
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={runtime_username}
Group={runtime_username}
WorkingDirectory={SERVICE_INSTALL_DIRECTORY}
# secret_phrase and internal_apikey are read from this file rather than exported here: a unit
# under /etc/systemd/system is world-readable, and config.toml is not.
Environment=GENIX_CONFIG_FILE={repository_config_path}
Environment=SSE_BRIDGE_PORT={bridge_port}
ExecStart={BINARY_PATH}
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


def build_restart_path_contents():
    return f"""[Unit]
Description=Watch for changes to the genix server-utils binary

[Path]
PathChanged={BINARY_PATH}

[Install]
WantedBy=multi-user.target
"""


def build_restart_service_contents():
    return f"""[Unit]
Description=Restart Genix Server Utilities

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart {SERVICE_NAME}
"""


def configure_systemd_units(runtime_username, repository_config_path, bridge_port):
    unit_files_changed = [
        write_unit_file(
            SYSTEMD_DIRECTORY / SERVICE_NAME,
            build_service_contents(runtime_username, repository_config_path, bridge_port),
        ),
        write_unit_file(SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME, build_restart_service_contents()),
        write_unit_file(SYSTEMD_DIRECTORY / RESTART_PATH_NAME, build_restart_path_contents()),
    ]
    return any(unit_files_changed)


def start_service(systemd_configuration_changed):
    if systemd_configuration_changed:
        run_command(["systemctl", "daemon-reload"])

    run_command(["systemctl", "enable", SERVICE_NAME])
    run_command(["systemctl", "enable", RESTART_PATH_NAME])
    if systemd_configuration_changed:
        run_command(["systemctl", "restart", RESTART_PATH_NAME])

    # The binary is installed before the watcher is (re)started, so that first change is never
    # seen by it. A service that cannot start must not fail the whole run: the units are already
    # in place and journalctl has the reason (most often: credit_usage is not deployed yet).
    service_start_result = run_command(["systemctl", "restart", SERVICE_NAME], allow_failure=True)
    if service_start_result.returncode != 0:
        print_debug(
            f"WARNING: {SERVICE_NAME} did not start. The units are installed; "
            f"check 'journalctl -u {SERVICE_NAME} -n 50' for the reason."
        )
        return

    print_debug(f"{SERVICE_NAME} is running.")


def warn_if_credentials_are_unreadable(runtime_username, repository_config_path):
    """The unit points at this file and the service reads it as a non-root user.

    A config.toml copied onto the host as root with mode 0600 is invisible to that account, and
    the only symptom would be the service exiting at boot over a missing setting. Say it here
    instead, where the fix is one command away.
    """
    readability_result = run_command(
        ["sudo", "-u", runtime_username, "test", "-r", str(repository_config_path)],
        allow_failure=True,
    )
    if readability_result.returncode == 0:
        return

    print_debug(
        f"WARNING: '{runtime_username}' cannot read {repository_config_path}, so the service "
        f"will not find its secrets. Fix it with: chown {runtime_username} {repository_config_path}"
    )


def install_systemd_service(repository_config_path, bridge_port):
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    warn_if_credentials_are_unreadable(runtime_username, repository_config_path)
    ensure_binary_directory(runtime_user_entry)
    provide_binary(repository_config_path.parent, runtime_user_entry)
    systemd_configuration_changed = configure_systemd_units(
        runtime_username, repository_config_path, bridge_port
    )
    start_service(systemd_configuration_changed)
    return runtime_username


def print_summary(
    repository_config_path, runtime_username, bridge_domain, bridge_port, rate_limit_address
):
    print_debug("Server utilities configuration completed.")
    print_debug(f"Config file: {repository_config_path}")
    print_debug(f"Runtime user: {runtime_username}")

    binary_size_in_bytes = BINARY_PATH.stat().st_size if BINARY_PATH.is_file() else 0
    print_debug(f"Binary path: {BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
    print_debug(f"SSE bridge port (SSE_BRIDGE_PORT): {bridge_port}")
    print_debug(f"Rate limiter address (loopback, no vhost): {rate_limit_address}")
    print_debug(f"Service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
    print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
    print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
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
    verify_required_secrets(project_credentials, repository_config_path)
    bridge_domain = resolve_bridge_domain(project_credentials, repository_config_path)
    bridge_port = resolve_bridge_port(project_credentials)
    rate_limit_address = str(
        get_config_value(project_credentials, "rate_limit.address", "127.0.0.1:14013")
    ).strip()

    runtime_username = install_systemd_service(repository_config_path, bridge_port)
    configure_bridge_nginx_vhost(bridge_domain, bridge_port)
    print_summary(
        repository_config_path, runtime_username, bridge_domain, bridge_port, rate_limit_address
    )


if __name__ == "__main__":
    main()
