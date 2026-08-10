#!/usr/bin/env python3

"""Install server_utils/ on this host: systemd units plus the Nginx vhost for its SSE bridge.

One Rust binary, genix-server-utils, over two transports:

  - A raw TCP port that stays on loopback, where the frame's opcode routes to the credit rate
    limiter or the lock service. It is authenticated by HMAC but not encrypted, so it never gets
    an Nginx vhost.
  - The SSE bridge, on an HTTP port that must be reachable by browsers. Nginx terminates TLS
    for it on this very machine: what it proxies is a permanent stream, and a second hop buys
    nothing.

Deliberately simpler than configure_server.py, which covers a backend host and an Nginx edge
host that are usually two different machines. Here there is no upstream to configure — the
vhost always forwards to 127.0.0.1.

It asks for nothing. Values come from config.toml, and a missing one either fails with the key
name or gets a documented default written into the file — never a prompt. Secrets and database
settings fail (they must match what the backend uses); the credit ceilings are filled in, since
those are policy and a default beats a daemon that exits at startup on every restart.

Nginx specifics: the vhost never buffers (a buffered text/event-stream is a stalled request),
it adds no CORS headers (the bridge answers preflights itself — a duplicated
Access-Control-Allow-Origin makes browsers reject the response), and it serves HTTP/3 whenever
a certificate for the hostname exists.
"""

import os
import re
import shutil
import sys
import time
import urllib.error
import urllib.request
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
from toml_config import get_config_value, set_config_values  # noqa: E402

SOURCE_DIRECTORY_NAME = "server_utils"
# Cargo's package name, which is also the file name it produces.
BINARY_NAME = "genix-server-utils"
BINARY_PATH = SERVICE_INSTALL_DIRECTORY / BINARY_NAME
SERVICE_NAME = "genix-server-utils.service"
RESTART_SERVICE_NAME = "genix-server-utils-restart.service"
RESTART_PATH_NAME = "genix-server-utils-restart.path"

# Must match DEFAULT_BRIDGE_PORT in server_utils/src/config.rs.
DEFAULT_BRIDGE_PORT = 14012
# The daemon loads existing usage from ScyllaDB before it serves anything, so give it a moment
# before deciding whether it stayed up.
SERVICE_SETTLE_SECONDS = 3
HEALTH_PROBE_TIMEOUT_SECONDS = 5

# Both are root-level keys in config.toml. secret_phrase verifies the browser's session token;
# internal_apikey authenticates the backend's calls (and the rate limiter's TCP frames).
REQUIRED_SECRET_NAMES = ("secret_phrase", "internal_apikey")

# The daemon opens a ScyllaDB session before it serves anything, so these have no default either
# and cannot be invented here.
REQUIRED_DATABASE_SETTINGS = ("db.host", "db.port", "db.name", "db.user", "db.password")

# Every way this daemon is known to refuse to start, matched against its journal. Ordered from the
# most specific fragment to the most generic, because a line can contain more than one.
KNOWN_SERVICE_FAILURE_CAUSES = (
    (
        "Cannot assign requested address",
        "the listen address is not on any interface of this host. The bind is derived from "
        "[server_utils] public (true = 0.0.0.0, false = 127.0.0.1) precisely so a NAT'd public IP "
        "is never bound: on a cloud VM that address lives in the provider's NAT, not on the NIC.",
    ),
    (
        "Address already in use",
        "another process already holds that port. Find it with 'ss -lntp | grep <port>' — most "
        "often a copy of this daemon started outside systemd.",
    ),
    (
        "credit_usage",
        "the backend tables are not deployed, so the rate limiter exits rather than admit traffic "
        "it cannot account for. Run 'cd scripts && go run . check_tables' and rerun this script.",
    ),
    (
        "missing required setting",
        "config.toml lacks a key the daemon has no default for. The log line names it.",
    ),
    (
        "Connection refused",
        "ScyllaDB did not answer at db.host/db.port. Check that it is up and reachable from here.",
    ),
)

# The twelve credit ceilings the daemon requires. Unlike every other [rate_limit] key they have
# no fallback in server_utils/src/config.rs — a guessed quota is worse than none — so an absent
# one makes the process exit at startup and systemd restart it every RestartSec forever. The
# numbers mirror config.example.toml: (10s, 1h, 24h) per scope and credit kind.
RATE_LIMIT_CREDIT_DEFAULTS = {
    "company_cpu": (2000, 40000, 200000),
    "company_inference": (1000, 10000, 20000),
    "user_cpu": (1000, 20000, 100000),
    "user_inference": (500, 5000, 10000),
}
RATE_LIMIT_WINDOW_SUFFIXES = ("10s", "1h", "24h")

# An AWS function URL in sse_bridge.url means "no bridge" (the backend serves its own
# /agent/stream), so it cannot be the hostname this vhost is built for.
LAMBDA_URL_HOST_SUFFIXES = (".on.aws", ".amazonaws.com")

# reuseport may be set by exactly one server block per address:port. A second one anywhere in
# the Nginx configuration fails `nginx -t` with "duplicate listen options", which is what would
# happen when this host also serves the backend vhost written by configure_server.py.
NGINX_LISTEN_REUSEPORT_PATTERN = re.compile(r"^\s*listen\s+[^;]*\breuseport\b", re.MULTILINE)


def resolve_bridge_domain(project_credentials, repository_config_path):
    """Take the vhost hostname from sse_bridge.url. Never prompts: this key is not a secret.

    The backend also reads this key for publishing. The frontend receives the same public URL
    from the selected [[endpoints]].bridge entry.
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


def verify_required_settings(project_credentials, repository_config_path):
    """Fail early when a setting the process needs at startup is missing and cannot be defaulted.

    Not prompted for: the secrets are root-level keys written during initial project setup and
    their values must match the backend byte for byte, and the database is already configured by
    the time this host runs anything. Guessing either would install a service that fails at
    runtime instead of failing visibly now — and because the unit is Restart=always, "fails at
    runtime" means a three-second crash loop nobody is watching.
    """
    for secret_name in REQUIRED_SECRET_NAMES:
        if not str(get_config_value(project_credentials, secret_name, "")).strip():
            fail_with_error(
                f"{secret_name} is not set in {repository_config_path}. It must hold the same "
                "value the backend uses, otherwise every request is rejected at runtime."
            )

    for database_setting_name in REQUIRED_DATABASE_SETTINGS:
        if not str(get_config_value(project_credentials, database_setting_name, "")).strip():
            fail_with_error(
                f"{database_setting_name} is not set in {repository_config_path}. The daemon "
                "loads existing credit usage from ScyllaDB before it serves anything and exits "
                "when it cannot connect."
            )

    print_debug(f"Found both required secrets: {', '.join(REQUIRED_SECRET_NAMES)}.")


def read_positive_integer_setting(project_credentials, setting_name):
    """Return the setting as a positive int, or None when absent or unusable.

    Quoted numbers count: the daemon accepts them too, so a value written as a string by some
    templating step must not be treated as missing and silently rewritten.
    """
    configured_value = get_config_value(project_credentials, setting_name)
    if isinstance(configured_value, bool) or not isinstance(configured_value, (int, str)):
        return None
    try:
        parsed_value = int(str(configured_value).strip())
    except ValueError:
        return None
    return parsed_value if parsed_value > 0 else None


def ensure_rate_limit_credit_limits(project_credentials, repository_config_path):
    """Write the credit ceilings that have no default in the daemon, when config.toml lacks them.

    This is the one class of missing setting worth fixing instead of reporting: the values are
    policy, not secrets, so a sane starting point exists — and the alternative is the daemon
    exiting with `missing required setting rate_limit.company_cpu_10s` on every restart. They go
    into the file rather than the unit's Environment= so the operator can see and tune them.

    Values already present are never touched. A missing window is filled with the example default
    raised to the previous window's value, so completing a hand-tuned set cannot produce the
    decreasing sequence the daemon rejects (10s <= 1h <= 24h).
    """
    missing_credit_limits = {}
    for credit_scope_name, window_defaults in RATE_LIMIT_CREDIT_DEFAULTS.items():
        previous_window_value = 0
        for window_suffix, default_value in zip(RATE_LIMIT_WINDOW_SUFFIXES, window_defaults):
            setting_name = f"rate_limit.{credit_scope_name}_{window_suffix}"
            configured_value = read_positive_integer_setting(project_credentials, setting_name)
            if configured_value is not None:
                previous_window_value = configured_value
                continue
            previous_window_value = max(default_value, previous_window_value)
            missing_credit_limits[setting_name] = previous_window_value

    if not missing_credit_limits:
        print_debug("Every rate-limit credit ceiling is already set in config.toml.")
        return

    print_debug(
        f"Writing {len(missing_credit_limits)} default credit ceiling(s) to "
        f"{repository_config_path}, which would otherwise stop the daemon at startup:"
    )
    for setting_name, default_value in missing_credit_limits.items():
        print_debug(f"  {setting_name} = {default_value}")
    set_config_values(repository_config_path, missing_credit_limits)


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


def nginx_supports_http3():
    """Report whether this Nginx was built with QUIC.

    `listen 443 quic` is a syntax error to a binary without http_v3_module, and because
    `nginx -t` checks the whole configuration that one line would break every other site on the
    host. HTTP/3 arrived in mainline 1.25, so distributions still shipping 1.24 or an older
    stable build do not have it.
    """
    version_result = run_command(["nginx", "-V"], allow_failure=True)
    if version_result.returncode != 0:
        print_debug("Could not read 'nginx -V'; assuming no HTTP/3 support.")
        return False

    # -V writes its build flags to stderr.
    build_flags = f"{version_result.stdout} {version_result.stderr}"
    if "http_v3_module" in build_flags:
        return True

    print_debug(
        "This Nginx was built without http_v3_module, so the vhost gets TLS over TCP only. "
        "Install Nginx 1.25+ with HTTP/3 and rerun to enable QUIC."
    )
    return False


def warn_if_http3_is_unreachable(nginx_configuration_contents):
    """HTTP/3 fails silently when UDP is closed, so say so instead of letting it look fine.

    Browsers reach the site over TCP and never report that the QUIC handshake was dropped, so a
    firewall that only ever opened 443/tcp leaves HTTP/3 permanently unused with no symptom.
    """
    if "quic" not in nginx_configuration_contents:
        return

    firewall_state = run_command(["firewall-cmd", "--state"], allow_failure=True)
    if firewall_state.returncode == 0 and firewall_state.stdout.strip() == "running":
        open_ports = run_command(["firewall-cmd", "--list-ports"], allow_failure=True)
        if "443/udp" not in open_ports.stdout:
            print_debug(
                "WARNING: HTTP/3 is configured but firewalld does not list 443/udp. Open it with: "
                "firewall-cmd --permanent --add-port=443/udp && firewall-cmd --reload"
            )
        return

    ufw_status = run_command(["ufw", "status"], allow_failure=True)
    if ufw_status.returncode == 0 and "Status: active" in ufw_status.stdout:
        if "443" not in ufw_status.stdout:
            print_debug(
                "WARNING: HTTP/3 is configured but ufw does not allow 443. Open it with: "
                "ufw allow 443"
            )
        return

    print_debug(
        "HTTP/3 is configured. No local firewall is managing this host, so make sure UDP 443 is "
        "open upstream too — a cloud security group that only allows TCP silently disables it."
    )


def build_bridge_nginx_configuration(
    bridge_domain,
    bridge_port,
    existing_nginx_configuration_contents="",
    reuseport_is_available=True,
    http3_is_supported=True,
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

        # Emitting quic on an Nginx without http_v3_module is not a degraded vhost, it is a
        # configuration the binary cannot parse — and `nginx -t` covers every site on the host.
        if http3_is_supported:
            listen_directives = (
                f"    # HTTP/3 (QUIC) plus the TCP listener browsers use before they see Alt-Svc.\n"
                f"    listen 443 quic{quic_listen_options};\n"
                f"    listen 443 ssl;\n"
                f"    listen [::]:443 quic{quic_listen_options};\n"
                f"    listen [::]:443 ssl;"
            )
            alt_svc_directive = "\n    add_header Alt-Svc 'h3=\":443\"; ma=86400' always;\n"
        else:
            listen_directives = (
                "    # TLS over TCP only: this Nginx was built without http_v3_module.\n"
                "    listen 443 ssl;\n"
                "    listen [::]:443 ssl;"
            )
            # Advertising h3 without a QUIC listener sends browsers to a port nothing answers on.
            alt_svc_directive = ""

        return f"""server {{
{listen_directives}
    http2 on;

    server_name {bridge_domain};

{tls_directives}

    ssl_protocols TLSv1.3;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets on;
    # ssl_early_data stays off: POST /in is not replay-safe and 0-RTT buys nothing for a
    # connection that stays open for the whole session.
{alt_svc_directive}
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
        http3_is_supported=nginx_supports_http3(),
    )

    if existing_nginx_configuration_contents == nginx_configuration_contents:
        print_debug(f"Nginx configuration unchanged: {nginx_configuration_path}")
        return

    print_debug(f"Writing Nginx vhost for the bridge: {nginx_configuration_path}")
    nginx_configuration_path.write_text(nginx_configuration_contents, encoding="utf-8")
    os.chmod(nginx_configuration_path, 0o644)

    # `nginx -t` validates the whole configuration, not just this file, so a vhost it rejects
    # would keep every other site on the host from reloading too. Put back what was there before
    # failing, rather than leaving a landmine for the next unrelated reload.
    validation_result = run_command(["nginx", "-t"], allow_failure=True)
    if validation_result.returncode != 0:
        if existing_nginx_configuration_contents is None:
            nginx_configuration_path.unlink(missing_ok=True)
            print_debug(f"Removed the rejected vhost: {nginx_configuration_path}")
        else:
            nginx_configuration_path.write_text(
                existing_nginx_configuration_contents, encoding="utf-8"
            )
            print_debug(f"Restored the previous vhost: {nginx_configuration_path}")
        fail_with_error(
            "Nginx rejected the generated vhost (see the output above). The previous "
            "configuration was put back, so the other sites on this host keep working."
        )

    run_command(["systemctl", "enable", "nginx"])
    run_command(["systemctl", "restart", "nginx"])
    warn_if_http3_is_unreachable(nginx_configuration_contents)


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


def ensure_c_linker():
    """Install a C compiler when the host has none, because rustc cannot link without one.

    rustc emits object files and then shells out to `cc` to turn them into an executable: `cc` is
    what knows where this distribution keeps the C runtime startup files and libc. So a linker is
    needed even though nothing in this tree is written in C — the only crate that would compile
    any, iana-time-zone-haiku, never builds on Linux.

    It is also why the failure lands on crates like proc-macro2, quote or getrandom before any of
    our own code: a build.rs is compiled into a real executable that cargo then runs, so those
    reach the link step first. That is the same reason `--target ...-musl` does not avoid this —
    build scripts are always built for the host, whatever --target says.

    Only the compiler and the libc headers are installed, not a full build-essential: this
    project has no C++ and no make-driven build scripts, and the crt objects that linking really
    needs come from the libc development package.
    """
    if shutil.which("cc"):
        print_debug(f"C linker found at {shutil.which('cc')}")
        return

    package_manager_commands = [
        # apt needs its lists refreshed first: a fresh cloud image usually has none.
        ("apt-get", [["apt-get", "update"], ["apt-get", "install", "-y", "gcc", "libc6-dev"]]),
        ("dnf", [["dnf", "install", "-y", "gcc", "glibc-devel"]]),
        ("yum", [["yum", "install", "-y", "gcc", "glibc-devel"]]),
        ("zypper", [["zypper", "--non-interactive", "install", "gcc", "glibc-devel"]]),
        ("pacman", [["pacman", "-Sy", "--noconfirm", "gcc"]]),
        ("apk", [["apk", "add", "--no-cache", "gcc", "musl-dev"]]),
    ]

    for package_manager_name, install_command_list in package_manager_commands:
        if not shutil.which(package_manager_name):
            continue

        print_debug(f"No C linker on this host. Installing one with {package_manager_name}...")
        for install_command in install_command_list:
            # Noninteractive, so a package manager prompt cannot hang an unattended install.
            run_command(
                ["env", "DEBIAN_FRONTEND=noninteractive", *install_command],
                stream_output=True,
                allow_failure=True,
            )

        if shutil.which("cc"):
            print_debug(f"C linker installed at {shutil.which('cc')}")
            return

        fail_with_error(
            f"{package_manager_name} ran but 'cc' is still not on PATH. Install a C compiler by "
            "hand (Debian/Ubuntu: gcc libc6-dev, RHEL/Fedora: gcc glibc-devel) and run again."
        )

    fail_with_error(
        "Rust needs a C compiler to link and no supported package manager was found here. "
        "Install one (Debian/Ubuntu: gcc libc6-dev, RHEL/Fedora: gcc glibc-devel, Alpine: gcc "
        "musl-dev) and run this script again."
    )


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

    ensure_c_linker()
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

# Without this the unit inherits systemd's default soft limit of 1024 open files, which is below
# the daemon's own rate_limit.max_connections of 1024 — every SSE stream, the ScyllaDB session
# and both listeners come out of the same budget, so it would hit EMFILE before reaching its
# configured ceiling and start refusing connections it was told to accept.
LimitNOFILE=65536

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


def start_service(systemd_configuration_changed, bridge_port):
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
        report_service_failure("systemctl restart returned an error")
        return

    verify_service_is_serving(bridge_port)


def diagnose_service_failure(journal_text):
    """Match the journal against the failures this daemon actually has, newest line first.

    A fixed hypothesis is worse than none: it used to blame the undeployed tables for every
    failure, so a bind error two lines above in the very same output was read past.
    """
    for journal_line in reversed(journal_text.splitlines()):
        for log_fragment, explanation in KNOWN_SERVICE_FAILURE_CAUSES:
            if log_fragment in journal_line:
                return explanation
    return None


def report_service_failure(failure_summary):
    """Print why the daemon is not up, instead of a line telling the reader to go find out."""
    print_debug(f"WARNING: {SERVICE_NAME} is not running ({failure_summary}).")
    print_debug("The units are installed. Last log lines:")
    journal_result = run_command(
        ["journalctl", "-u", SERVICE_NAME, "-n", "20", "--no-pager"], allow_failure=True
    )

    explanation = diagnose_service_failure(journal_result.stdout or "")
    if explanation:
        print_debug(f"Cause: {explanation}")
        return
    print_debug(
        "No known failure matched the log above. Read it top to bottom: the daemon prints its "
        "reason and exits, so the last 'Error:' line is the cause."
    )


def verify_service_is_serving(bridge_port):
    """Confirm the daemon is actually up, not merely exec'd.

    `systemctl restart` succeeds as soon as the process starts, and this unit is Type=simple with
    Restart=always: a daemon that exits immediately — which is exactly what it does when ScyllaDB
    is unreachable — looks identical to a healthy one at that point. So wait for it to settle,
    then ask it something only a working process can answer.
    """
    time.sleep(SERVICE_SETTLE_SECONDS)

    active_state_result = run_command(
        ["systemctl", "is-active", SERVICE_NAME], allow_failure=True
    )
    if active_state_result.stdout.strip() != "active":
        report_service_failure(f"systemctl reports '{active_state_result.stdout.strip()}'")
        return

    # is-active only proves the process exists. /health proves the bridge half is listening and
    # answering, which also means the rate limiter got past its ScyllaDB load.
    health_url = f"http://127.0.0.1:{bridge_port}/health"
    try:
        with urllib.request.urlopen(health_url, timeout=HEALTH_PROBE_TIMEOUT_SECONDS) as response:
            health_payload = response.read().decode("utf-8", errors="replace").strip()
    except (urllib.error.URLError, OSError, ValueError) as health_error:
        report_service_failure(f"{health_url} did not answer: {health_error}")
        return

    print_debug(f"{SERVICE_NAME} is running and answering {health_url}: {health_payload}")


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
    start_service(systemd_configuration_changed, bridge_port)
    return runtime_username


def print_summary(
    repository_config_path, runtime_username, bridge_domain, bridge_port, server_utils_address
):
    print_debug("Server utilities configuration completed.")
    print_debug(f"Config file: {repository_config_path}")
    print_debug(f"Runtime user: {runtime_username}")

    binary_size_in_bytes = BINARY_PATH.stat().st_size if BINARY_PATH.is_file() else 0
    print_debug(f"Binary path: {BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
    print_debug(f"SSE bridge port (SSE_BRIDGE_PORT): {bridge_port}")
    print_debug(f"Raw TCP address, opcode-routed (loopback, no vhost): {server_utils_address}")
    print_debug(f"Service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
    print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
    print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
    print_debug(f"Nginx vhost: {NGINX_CONFIGURATION_DIRECTORY / (bridge_domain + '.conf')}")
    print_debug(f"Health check: curl -s http://127.0.0.1:{bridge_port}/health")
    print_debug(f"Through Nginx: curl -s https://{bridge_domain}/health")
    print_debug(
        f"Reminder: sse_bridge.url (https://{bridge_domain}/) configures the backend, and the "
        "matching [[endpoints]].bridge configures the frontend selector."
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
    verify_required_settings(project_credentials, repository_config_path)
    ensure_rate_limit_credit_limits(project_credentials, repository_config_path)
    bridge_domain = resolve_bridge_domain(project_credentials, repository_config_path)
    bridge_port = resolve_bridge_port(project_credentials)
    server_utils_address = str(
        get_config_value(project_credentials, "server_utils", "127.0.0.1:14013")
    ).strip()

    runtime_username = install_systemd_service(repository_config_path, bridge_port)
    configure_bridge_nginx_vhost(bridge_domain, bridge_port)
    print_summary(
        repository_config_path, runtime_username, bridge_domain, bridge_port, server_utils_address
    )


if __name__ == "__main__":
    main()
