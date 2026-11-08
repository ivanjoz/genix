#!/usr/bin/env python3

"""Install fareward/ on this host: systemd units plus the HAProxy site for its SSE bridge.

One Rust binary, fareward, over two transports:

  - A raw TCP port where the frame's opcode routes to the credit rate limiter or the lock
    service. It is authenticated by HMAC but not encrypted, so it never gets a public site.
    [fareward] public decides whether the daemon binds loopback or every interface: false
    when the backend runs on this same host, true when it runs elsewhere (Lambda, another VPS).
    When it is true this script also opens the port in the host firewall — a daemon bound to
    0.0.0.0 behind a REJECT rule looks healthy from here and times out from everywhere else.
  - The SSE bridge, on an HTTP port that must be reachable by browsers. HAProxy terminates TLS
    for it on this very machine: what it proxies is a permanent stream, and a second hop buys
    nothing.
  - The SPOE agent HAProxy asks before forwarding an API request ([spoe].socket). Its HAProxy
    side belongs to the API site, which configure_server.py writes.

Deliberately simpler than configure_server.py, which covers a backend host and an edge host
that may be two different machines. Here there is no upstream to configure — the bridge site
always forwards to 127.0.0.1. Both scripts share the HAProxy edge helpers in configure_server.py,
and each writes only its own site file.

It asks for nothing. Values come from config.toml, and a missing one either fails with the key
name or gets a documented default written into the file — never a prompt. Secrets fail (they must
match what the backend uses); the credit ceilings are filled in, since
those are policy and a default beats a daemon that exits at startup on every restart.

HAProxy specifics: it never buffers a response (a buffered text/event-stream is a stalled
request), the bridge site adds no CORS headers (the bridge answers preflights itself — a
duplicated Access-Control-Allow-Origin makes browsers reject the response), and it serves HTTP/3
once the hostname's certificate exists, which this script issues.
"""

import argparse
import os
import shutil
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from configure_server import (  # noqa: E402  (the sys.path line above must run first)
    HAPROXY_SITES_DIRECTORY,
    SERVICE_INSTALL_DIRECTORY,
    SYSTEMD_DIRECTORY,
    build_unprivileged_command,
    configure_haproxy_site,
    detect_repository_config_path,
    detect_runtime_username,
    detect_unprivileged_username,
    ensure_binary_directory,
    fail_with_error,
    is_usable_executable,
    load_project_credentials,
    print_debug,
    require_root_execution,
    resolve_go_architecture,
    resolve_runtime_user,
    run_command,
    validate_domain,
    validate_server_port,
    write_unit_file,
)
from toml_config import get_config_value, set_config_values  # noqa: E402
from firewall_ports import ensure_tcp_port_open  # noqa: E402

SOURCE_DIRECTORY_NAME = "fareward"
# Cargo's package name, which is also the file name it produces.
BINARY_NAME = "fareward"
BINARY_PATH = SERVICE_INSTALL_DIRECTORY / BINARY_NAME
BINARY_SOURCE_AUTO = "auto"
BINARY_SOURCE_SOURCE = "source"
BINARY_SOURCE_PRECOMPILED = "precompiled"
SERVICE_NAME = "fareward.service"
RESTART_SERVICE_NAME = "fareward-restart.service"
RESTART_PATH_NAME = "fareward-restart.path"

# The unit names this daemon shipped under before the repository was renamed to fareward. They are
# not history: a host configured before the rename still has them enabled and running, and the old
# process keeps the listen port, so the new unit cannot bind and restarts forever while the old
# binary — which speaks an older wire domain and knows nothing of the newer opcodes — answers the
# backend and closes every frame it cannot parse. Removing them is part of installing this service.
LEGACY_UNIT_NAMES = (
    "genix-server-utils.service",
    "genix-server-utils-restart.service",
    "genix-server-utils-restart.path",
)
LEGACY_BINARY_PATH = SERVICE_INSTALL_DIRECTORY / "genix-server-utils"

# Must match DEFAULT_BRIDGE_PORT and DEFAULT_LISTEN_PORT in fareward/src/config.rs, and
# defaultFarewardPort in backend/core/security.go, which is the client's half of the same
# default. The three drifting apart puts the daemon on one port and every caller on another.
DEFAULT_BRIDGE_PORT = 14012
DEFAULT_LISTEN_PORT = 14013
# The daemon loads existing usage from its own store before it serves anything, so give it a
# moment before deciding whether it stayed up.
SERVICE_SETTLE_SECONDS = 3
HEALTH_PROBE_TIMEOUT_SECONDS = 5

# Both are root-level keys in config.toml. secret_phrase verifies the browser's session token;
# internal_apikey authenticates the backend's calls (and the rate limiter's TCP frames).
REQUIRED_SECRET_NAMES = ("secret_phrase", "internal_apikey")

# Every way this daemon is known to refuse to start, matched against its journal. Ordered from the
# most specific fragment to the most generic, because a line can contain more than one.
KNOWN_SERVICE_FAILURE_CAUSES = (
    (
        "Cannot assign requested address",
        "the listen address is not on any interface of this host. The bind is derived from "
        "[fareward] public (true = 0.0.0.0, false = 127.0.0.1) precisely so a NAT'd public IP "
        "is never bound: on a cloud VM that address lives in the provider's NAT, not on the NIC.",
    ),
    # Two fragments for one failure: the daemon reports the OS error verbatim as "Address in use
    # (os error 98)", which does not contain the phrase systemd and most tools print. Matching only
    # the long form is why a bind conflict used to be diagnosed as "no known cause".
    (
        "Address in use",
        "another process already holds that port. Find it with 'ss -lntp | grep <port>' — most "
        "often the pre-rename genix-server-utils service, or a copy started outside systemd.",
    ),
    (
        "Address already in use",
        "another process already holds that port. Find it with 'ss -lntp | grep <port>' — most "
        "often the pre-rename genix-server-utils service, or a copy started outside systemd.",
    ),
    (
        "the store directory",
        "the daemon could not create its store directory ([store].path, default "
        "/var/lib/fareward/store). The unit's StateDirectory=fareward creates it; check the path "
        "and that the service user owns it.",
    ),
    (
        "fareward store",
        "the daemon could not open its store directory ([store].path, default "
        "/var/lib/fareward/store). The unit's StateDirectory=fareward creates it; check the path "
        "and that the service user owns it.",
    ),
    (
        "missing required setting",
        "config.toml lacks a key the daemon has no default for. The log line names it.",
    ),
)

# The eight credit ceilings the daemon requires. Unlike every other [rate_limit] key they have
# no fallback in fareward/src/config.rs — a guessed quota is worse than none — so an absent
# one makes the process exit at startup and systemd restart it every RestartSec forever. The
# numbers mirror config.example.toml: (short window, long window) per scope and credit kind.
# There is no daily ceiling here: the daily and monthly entitlements are per company, in the
# daemon's budget store, administered from the SaaS panel.
# The window lengths themselves (short_window_seconds, long_window_minutes) do have a default in
# the daemon (10 seconds and 60 minutes), so they are never written here.
RATE_LIMIT_CREDIT_DEFAULTS = {
    "company_cpu": (2000, 40000),
    "company_inference": (1000, 10000),
    "user_cpu": (400, 20000),
    "user_inference": (500, 5000),
}
RATE_LIMIT_WINDOW_SUFFIXES = ("short", "long")

# An AWS function URL in sse_bridge.url means "no bridge" (the backend serves its own
# /agent/stream), so it cannot be the hostname this vhost is built for.
LAMBDA_URL_HOST_SUFFIXES = (".on.aws", ".amazonaws.com")

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
    bridge_domain, domain_error = validate_domain(candidate_host)
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
        # Never fall back silently: the HAProxy backend is built from this number, so a typo
        # would produce a vhost proxying to a port nothing listens on.
        fail_with_error(f"sse_bridge.port in config.toml is unusable: {validation_error}")

    print_debug(f"SSE bridge listen port: {bridge_port}")
    return bridge_port


def resolve_listen_port(project_credentials):
    """Resolve the raw-TCP port. Absent is normal — the Rust default covers it."""
    configured_port = get_config_value(project_credentials, "fareward.port")
    if configured_port is None:
        print_debug(f"fareward.port is not set in config.toml. Using {DEFAULT_LISTEN_PORT}.")
        return DEFAULT_LISTEN_PORT

    listen_port, validation_error = validate_server_port(str(configured_port))
    if listen_port is None:
        # Same reason resolve_bridge_port refuses to guess: this number decides which port gets
        # a firewall rule, and opening the wrong one is worse than not opening any.
        fail_with_error(f"fareward.port in config.toml is unusable: {validation_error}")

    print_debug(f"Raw TCP listen port: {listen_port}")
    return listen_port


def resolve_listen_is_public(project_credentials):
    """Read [fareward] public, the flag the daemon turns into 0.0.0.0 vs 127.0.0.1.

    Only a real TOML boolean counts, and absent means false. The daemon reads this same key with
    the same rule (optional_bool in fareward/src/config.rs), so a value this script would
    interpret differently is a value that opens a firewall port for a listener that never binds
    to it — or worse, leaves one closed for a listener that does.
    """
    configured_flag = get_config_value(project_credentials, "fareward.public")
    if isinstance(configured_flag, bool):
        return configured_flag
    if configured_flag is None:
        return False

    # The daemon accepts a quoted boolean too; anything else is a typo worth naming.
    normalized_flag = str(configured_flag).strip().lower()
    if normalized_flag in {"true", "false"}:
        return normalized_flag == "true"
    fail_with_error(
        f"fareward.public in config.toml is not a boolean: {configured_flag!r}. Use "
        "true (bind 0.0.0.0, backend runs off this host) or false (bind 127.0.0.1)."
    )


def resolve_client_address(project_credentials, listen_port, listen_is_public):
    """The address a backend dials, mirroring makeFarewardAddress in backend/core/security.go.

    Only for the summary, but it is the number the reader has to put in their own config.toml,
    so it is derived the same way the backend derives it rather than echoed back from a key.
    """
    if not listen_is_public:
        return f"127.0.0.1:{listen_port}"

    configured_host = str(get_config_value(project_credentials, "fareward.host", "")).strip()
    return f"{configured_host}:{listen_port}" if configured_host else ""


def open_listen_port_in_firewall(listen_port, listen_is_public):
    """Open the raw-TCP port, but only when the daemon is actually binding every interface.

    A private daemon must stay unreachable, so a stale `public = true` left in config.toml is the
    only thing that opens this port — the flag governs the bind and the firewall together and
    they cannot disagree. When it is false there is nothing to do: loopback never crosses INPUT.

    Not fatal when it fails. The units are installed and the daemon serves this host either way,
    and the operator can add the rule by hand; what must not happen is failing silently, because
    the symptom on the far side is a two-second timeout that names neither this port nor this
    host.
    """
    if not listen_is_public:
        print_debug(
            f"[fareward] public is false: the daemon binds 127.0.0.1:{listen_port} and needs "
            "no firewall rule. Set it to true only if the backend runs on another host."
        )
        return

    if ensure_tcp_port_open(listen_port, "fareward raw TCP"):
        # The host firewall is one of two, and the other one is not on this machine.
        print_debug(
            f"TCP port {listen_port} is open on this host. On a cloud VM the provider's own "
            "ingress rules are separate and are not visible from here: Oracle Cloud security "
            "lists / NSGs, AWS security groups. Confirm from outside with "
            f"'nc -vz <public-ip> {listen_port}'."
        )
        return

    print_debug(
        f"WARNING: TCP port {listen_port} could not be opened, but [fareward] public is "
        "true, so the daemon is listening on every interface for callers that cannot reach it. "
        "An off-host backend will fail every lock and credit call with a timeout, which it "
        "reports as 'El servicio no está disponible.' (HTTP 503)."
    )


def verify_required_settings(project_credentials, repository_config_path):
    """Fail early when a setting the process needs at startup is missing and cannot be defaulted.

    Not prompted for: the secrets are root-level keys written during initial project setup and
    their values must match the backend byte for byte. Guessing them would install a service that fails at
    runtime instead of failing visibly now — and because the unit is Restart=always, "fails at
    runtime" means a three-second crash loop nobody is watching.
    """
    for secret_name in REQUIRED_SECRET_NAMES:
        if not str(get_config_value(project_credentials, secret_name, "")).strip():
            fail_with_error(
                f"{secret_name} is not set in {repository_config_path}. It must hold the same "
                "value the backend uses, otherwise every request is rejected at runtime."
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
    exiting with `missing required setting rate_limit.company_cpu_short` on every restart. They go
    into the file rather than the unit's Environment= so the operator can see and tune them.

    Values already present are never touched. A missing window is filled with the example default
    raised to the previous window's value, so completing a hand-tuned set cannot produce the decreasing
    sequence the daemon rejects (short <= long).
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


def build_bridge_site_configuration(bridge_domain, bridge_port):
    """The bridge's HAProxy site file: one backend forwarding the domain to the local bridge.

    HAProxy forwards response data as it arrives, so a text/event-stream is never held back and
    there is nothing to turn off. No CORS here: the bridge answers preflights itself, and a
    duplicated Access-Control-Allow-Origin is rejected by browsers. Early data on POST /in waits
    for the handshake (the shared frontend's rule): it is not replay-safe.
    """
    return f"""# Written by scripts/configure/configure_fareward.py (build_bridge_site_configuration).
# The fareward SSE bridge at {bridge_domain}.

backend site_{bridge_domain}
    # An idle stream must survive between agent turns; the bridge keepalive is every 20s.
    timeout server 3600s
    option forwardfor
    http-request set-header x-real-ip %[src]
    # The bridge always runs on this host: nothing is gained by proxying a permanent stream
    # across a second machine.
    server bridge 127.0.0.1:{bridge_port}
"""


def configure_bridge_edge(bridge_domain, bridge_port, runtime_username):
    site_file_path = HAPROXY_SITES_DIRECTORY / f"{bridge_domain}.cfg"
    configure_haproxy_site(
        bridge_domain,
        runtime_username,
        {site_file_path: build_bridge_site_configuration(bridge_domain, bridge_port)},
    )


def detect_source_directory(repository_root_path):
    """Find the crate, which is a submodule (the fareward repository) and therefore an empty
    directory on a checkout that never initialised it. That is why the probe is for the files and
    not for the directory: an uninitialised submodule must read as "no source", not as a broken
    tree, so the prebuilt binary path stays available."""
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
    """Build fareward/ in release mode, as the repository owner so the Cargo registry and
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


def find_prebuilt_binary(repository_root_path, prefer_downloaded=False):
    downloaded_binary_path = (
        repository_root_path / "tmp" / f"{BINARY_NAME}_linux_{resolve_go_architecture()}"
    )
    candidate_binary_paths = [
        downloaded_binary_path,
        BINARY_PATH,
        repository_root_path / SOURCE_DIRECTORY_NAME / "target" / "release" / BINARY_NAME,
    ]
    if not prefer_downloaded:
        candidate_binary_paths[0], candidate_binary_paths[1] = (
            candidate_binary_paths[1], candidate_binary_paths[0]
        )

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


def provide_binary(repository_root_path, runtime_user_entry, binary_source=BINARY_SOURCE_AUTO):
    """Put a runnable binary at BINARY_PATH: compile it, or find one, or fail."""
    source_directory = detect_source_directory(repository_root_path)
    if binary_source == BINARY_SOURCE_SOURCE:
        if not source_directory:
            fail_with_error(
                f"Building from source was selected, but no compilable Rust source exists under "
                f"{repository_root_path / SOURCE_DIRECTORY_NAME}. It is a git submodule: run "
                f"`git submodule update --init {SOURCE_DIRECTORY_NAME}` and try again."
            )
        install_binary(compile_binary(source_directory, repository_root_path), runtime_user_entry)
        return

    if binary_source == BINARY_SOURCE_AUTO and source_directory:
        install_binary(compile_binary(source_directory, repository_root_path), runtime_user_entry)
        return

    print_debug(f"Selecting a prebuilt Fareward binary (binary source: {binary_source}).")
    prebuilt_binary_path = find_prebuilt_binary(
        repository_root_path,
        prefer_downloaded=binary_source == BINARY_SOURCE_PRECOMPILED,
    )
    if not prebuilt_binary_path:
        fail_with_error(
            f"No source under {repository_root_path / SOURCE_DIRECTORY_NAME} and no prebuilt "
            f"binary found. Initialise the submodule with "
            f"`git submodule update --init {SOURCE_DIRECTORY_NAME}`, or place a compiled binary "
            f"at {BINARY_PATH}, and run the script again."
        )

    install_binary(prebuilt_binary_path, runtime_user_entry)


def build_service_contents(runtime_username, repository_config_path, bridge_port):
    return f"""[Unit]
Description=Genix Server Utilities (credit rate limiter + SSE bridge)
# The rate limiter loads existing usage from its own store (StateDirectory below) before admitting
# anything and exits when it cannot, which also stops the bridge: one process, shared fate. It
# needs no database: the backend pushes the grants and writes its own request log.
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
# Home of the optional fareward.unix_socket (e.g. /run/fareward/fareward.sock). systemd creates it
# owned by the service user, writable under ProtectSystem=strict, and removes it on stop. 0755 so
# the backend can traverse it; the socket itself is 0660.
RuntimeDirectory=fareward
RuntimeDirectoryMode=0755
# /var/lib/fareward: the default [store].path (/var/lib/fareward/store) lives here. systemd creates
# it owned by the service user and keeps it writable under ProtectSystem=strict. It is the whole
# state of the credit system, so it is what a backup copies.
StateDirectory=fareward
StateDirectoryMode=0700
ExecStart={BINARY_PATH}
Restart=always
RestartSec=3

# Without this the unit inherits systemd's default soft limit of 1024 open files, which is below
# the daemon's own rate_limit.max_connections of 1024 — every SSE stream, the store's files
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
Description=Watch for changes to the genix fareward binary

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


def remove_legacy_units():
    """Retire the pre-rename units so the port they hold is free for this one.

    Each step tolerates failure because most hosts have none of them: stop and disable are no-ops
    on a name systemd never knew, and the unit file may already be gone. The old binary is left on
    disk — inert once nothing starts it, and evidence of what was running here.
    """
    removed_any_unit = False
    for legacy_unit_name in LEGACY_UNIT_NAMES:
        unit_path = SYSTEMD_DIRECTORY / legacy_unit_name
        unit_is_known = (
            run_command(
                ["systemctl", "list-unit-files", legacy_unit_name], allow_failure=True
            ).returncode
            == 0
            and unit_path.exists()
        )
        if not unit_is_known:
            continue
        print_debug(f"Removing the pre-rename unit {legacy_unit_name}.")
        run_command(["systemctl", "stop", legacy_unit_name], allow_failure=True)
        run_command(["systemctl", "disable", legacy_unit_name], allow_failure=True)
        unit_path.unlink(missing_ok=True)
        removed_any_unit = True

    if removed_any_unit and LEGACY_BINARY_PATH.exists():
        print_debug(f"The superseded binary is left in place: {LEGACY_BINARY_PATH}")
    return removed_any_unit


def configure_systemd_units(runtime_username, repository_config_path, bridge_port):
    unit_files_changed = [
        remove_legacy_units(),
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
    # in place and journalctl has the reason (most often: the store directory cannot be opened).
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
    Restart=always: a daemon that exits immediately — which is exactly what it does when its store
    cannot be opened — looks identical to a healthy one at that point. So wait for it to settle,
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
    # answering, which also means the rate limiter got past loading its store.
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


def install_systemd_service(
    repository_config_path, bridge_port, binary_source=BINARY_SOURCE_AUTO
):
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    warn_if_credentials_are_unreadable(runtime_username, repository_config_path)
    ensure_binary_directory(runtime_user_entry)
    provide_binary(repository_config_path.parent, runtime_user_entry, binary_source)
    systemd_configuration_changed = configure_systemd_units(
        runtime_username, repository_config_path, bridge_port
    )
    start_service(systemd_configuration_changed, bridge_port)
    return runtime_username


def print_summary(
    repository_config_path, runtime_username, bridge_domain, bridge_port,
    listen_port, listen_is_public, client_address,
):
    print_debug("Server utilities configuration completed.")
    print_debug(f"Config file: {repository_config_path}")
    print_debug(f"Runtime user: {runtime_username}")

    binary_size_in_bytes = BINARY_PATH.stat().st_size if BINARY_PATH.is_file() else 0
    print_debug(f"Binary path: {BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
    print_debug(f"SSE bridge port (SSE_BRIDGE_PORT): {bridge_port}")

    bind_address = f"0.0.0.0:{listen_port}" if listen_is_public else f"127.0.0.1:{listen_port}"
    print_debug(f"Raw TCP bind, opcode-routed (no vhost): {bind_address}")
    if client_address:
        print_debug(f"Raw TCP address the backend must dial: {client_address}")
    else:
        # public = true with no host is the one combination that installs cleanly and then
        # refuses to start on the backend side, where the message names neither key.
        print_debug(
            "WARNING: [fareward] public is true but host is empty. The backend builds its "
            "dial address from that key and refuses an empty one at startup. Set host to this "
            "machine's public address in every config.toml that talks to this daemon."
        )
    print_debug(f"Service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
    print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
    print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
    print_debug(f"Health check: curl -s http://127.0.0.1:{bridge_port}/health")
    if bridge_domain is None:
        print_debug(
            "Public SSE bridge: skipped because the self-hosted backend serves /agent/stream."
        )
        return

    print_debug(f"HAProxy site file: {HAPROXY_SITES_DIRECTORY / (bridge_domain + '.cfg')}")
    print_debug(f"Through HAProxy: curl -s https://{bridge_domain}/health")
    print_debug(
        f"Reminder: sse_bridge.url (https://{bridge_domain}/) configures the backend, and the "
        "matching [[endpoints]].bridge configures the frontend selector."
    )


def parse_command_arguments():
    argument_parser = argparse.ArgumentParser(
        description="Install the Genix Fareward service and SSE bridge proxy."
    )
    argument_parser.add_argument(
        "--binary-source",
        choices=(BINARY_SOURCE_AUTO, BINARY_SOURCE_SOURCE, BINARY_SOURCE_PRECOMPILED),
        default=BINARY_SOURCE_AUTO,
        help="compile source, require a precompiled artifact, or retain automatic selection",
    )
    argument_parser.add_argument(
        "--service-only",
        action="store_true",
        help="install the daemon without requiring sse_bridge.url or configuring its HAProxy site",
    )
    return argument_parser.parse_args()


def main():
    command_arguments = parse_command_arguments()
    require_root_execution()

    repository_config_path = detect_repository_config_path()
    project_credentials = load_project_credentials(repository_config_path)

    # Resolve everything before touching the system, so a missing value stops the run instead of
    # leaving the host half configured.
    verify_required_settings(project_credentials, repository_config_path)
    ensure_rate_limit_credit_limits(project_credentials, repository_config_path)
    bridge_domain = None
    if not command_arguments.service_only:
        bridge_domain = resolve_bridge_domain(project_credentials, repository_config_path)
    bridge_port = resolve_bridge_port(project_credentials)
    listen_port = resolve_listen_port(project_credentials)
    listen_is_public = resolve_listen_is_public(project_credentials)
    client_address = resolve_client_address(project_credentials, listen_port, listen_is_public)

    runtime_username = install_systemd_service(
        repository_config_path, bridge_port, command_arguments.binary_source
    )
    if bridge_domain is not None:
        configure_bridge_edge(bridge_domain, bridge_port, runtime_username)
    # After the service, so the rule is added for a port something is actually listening on.
    open_listen_port_in_firewall(listen_port, listen_is_public)
    print_summary(
        repository_config_path, runtime_username, bridge_domain, bridge_port,
        listen_port, listen_is_public, client_address,
    )


if __name__ == "__main__":
    main()
