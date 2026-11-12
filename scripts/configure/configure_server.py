#!/usr/bin/env python3

import argparse
import os
import platform
import pwd
import re
import shutil
import stat
import subprocess
import sys
import tomllib
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from firewall_ports import ensure_tcp_port_open, ensure_udp_port_open  # noqa: E402
from toml_config import (  # noqa: E402
    LOCAL_CONFIG_FILE_NAME,
    apply_local_config_override,
    get_config_value,
    set_config_values,
)

SYSTEMD_DIRECTORY = Path("/etc/systemd/system")
SERVICE_INSTALL_DIRECTORY = Path("/usr/local/bin/genix")
SERVICE_BINARY_PATH = SERVICE_INSTALL_DIRECTORY / "genix_app"
SERVICE_NAME = "genix.service"
RESTART_SERVICE_NAME = "genix-restart.service"
RESTART_PATH_NAME = "genix-restart.path"
# In full mode HAProxy reaches the backend through this socket instead of loopback TCP. systemd
# creates /run/genix (RuntimeDirectory=genix) and the backend binds the file inside it.
SERVICE_RUNTIME_DIRECTORY_NAME = "genix"
SERVICE_SOCKET_PATH = Path("/run") / SERVICE_RUNTIME_DIRECTORY_NAME / "api.sock"
# StateDirectory=genix: the backend writes [fareward].routes_file here at startup, and under
# ProtectSystem=strict it is the only place outside the install directory it may write.
SERVICE_STATE_DIRECTORY = Path("/var/lib") / SERVICE_RUNTIME_DIRECTORY_NAME
LETSENCRYPT_DIRECTORY = Path("/etc/letsencrypt/live")
LETSENCRYPT_RENEWAL_DIRECTORY = Path("/etc/letsencrypt/renewal")

# HAProxy is the edge of every public domain on the host: the API (this script) and the SSE bridge
# (configure_fareward.py). haproxy.cfg is regenerated as a shared head (global, defaults, the one
# :80/:443 frontend) followed by every file in HAPROXY_SITES_DIRECTORY, one per domain, holding that
# domain's backends. Each script writes only its own site file, so neither overwrites the other.
HAPROXY_DIRECTORY = Path("/etc/haproxy")
HAPROXY_CONFIGURATION_PATH = HAPROXY_DIRECTORY / "haproxy.cfg"
HAPROXY_SITES_DIRECTORY = HAPROXY_DIRECTORY / "sites"
HAPROXY_SPOE_DIRECTORY = HAPROXY_DIRECTORY / "spoe"
# certbot's deploy hook writes <domain>.pem here (chain and key in one file, as HAProxy wants);
# the :443 binds load the whole directory and pick the certificate by SNI.
HAPROXY_CERTIFICATES_DIRECTORY = HAPROXY_DIRECTORY / "certs"
CERTIFICATE_DEPLOY_HOOK_PATH = Path("/usr/local/sbin/genix-haproxy-certificate")
# certbot --standalone answers HTTP-01 on this loopback port and HAProxy forwards
# /.well-known/acme-challenge/ to it, so issuing and renewing never stop the edge.
CERTBOT_HTTP01_PORT = 8888
# The fareward SPOE agent needs `mode spop` (SPOP 2.0), which arrived in HAProxy 3.1. 3.2 is the LTS
# branch the official Debian/Ubuntu repositories publish.
HAPROXY_MINIMUM_VERSION = (3, 1)
HAPROXY_REPOSITORY_BRANCH = "3.2"
# Defense independent of the fareward agent: no API request body above 50 MiB, whatever the route.
API_MAX_BODY_BYTES = 52_428_800
# Host-wide TCP settings for the clients HAProxy serves (see write_network_tuning).
NETWORK_TUNING_SYSCTL_PATH = Path("/etc/sysctl.d/90-genix-network.conf")
NETWORK_TUNING_MODULES_PATH = Path("/etc/modules-load.d/genix-network.conf")

# Same tags as scripts/deploy_vps.go and the release workflow: lambda.norpc drops net/rpc (~12 MB of
# retained methods). The untagged database driver is MariaDB.
BACKEND_BUILD_TAGS = "lambda.norpc"

# Repositories go.mod expects beside the genix checkout rather than inside it as submodules.
SIBLING_REPOSITORY_REMOTES = {"genix-search": "https://github.com/ivanjoz/genix-search.git"}

# sudo resets PATH to secure_path, which on most distributions excludes /usr/local/go/bin, so the
# Go toolchain has to be looked for explicitly. GO_BINARY in the environment overrides all of this.
GO_BINARY_SEARCH_PATHS = [
    Path("/usr/local/go/bin/go"),
    Path("/usr/lib/golang/bin/go"),
    Path("/usr/lib/go/bin/go"),
    Path("/opt/go/bin/go"),
    Path("/snap/bin/go"),
]
ELF_MAGIC_BYTES = b"\x7fELF"

# Full mode is the normal install: HAProxy terminates TLS/HTTP3 for server.domain and forwards to
# the backend on the same host through a Unix socket (SERVICE_SOCKET_PATH), asking the fareward
# SPOE agent first when [spoe].socket is set. The systemd-only and edge-only modes split the pair
# across two machines: the backend listens on server.port, HAProxy forwards to server.upstream over
# TCP, and there is no SPOE (the backend only trusts the agent's headers from a local peer).
EXECUTION_MODE_FULL = "full"
EXECUTION_MODE_SYSTEMD = "systemd"
EXECUTION_MODE_EDGE = "edge"
BINARY_SOURCE_AUTO = "auto"
BINARY_SOURCE_SOURCE = "source"
BINARY_SOURCE_PRECOMPILED = "precompiled"

EXECUTION_MODE_OPTIONS = [
    {
        "index": 1,
        "mode": EXECUTION_MODE_FULL,
        "label": "Full (systemd backend service + HAProxy edge)",
        "aliases": {"full", "all", "both"},
    },
    {
        "index": 2,
        "mode": EXECUTION_MODE_SYSTEMD,
        "label": "Only Systemd Service (backend host)",
        "aliases": {"systemd", "service", "backend"},
    },
    {
        "index": 3,
        "mode": EXECUTION_MODE_EDGE,
        "label": "Only HAProxy edge (edge host)",
        "aliases": {"edge", "proxy", "haproxy"},
    },
]


def print_debug(message_text):
    print(f"[*] {message_text}")


def fail_with_error(error_message):
    print(f"[!] {error_message}", file=sys.stderr)
    sys.exit(1)


def run_command(command_arguments, working_directory=None, stream_output=False, allow_failure=False):
    print_debug(f"Running command: {' '.join(command_arguments)}")

    # An absent binary raises FileNotFoundError instead of returning an exit code, so allow_failure
    # alone did not cover it: probing for an optional tool (firewall-cmd on a host without
    # firewalld) crashed the whole script with a traceback after all the real work was done. It is
    # reported as 127, the same code a shell uses for "command not found".
    def run_or_report_missing_binary(**subprocess_options):
        try:
            return subprocess.run(command_arguments, cwd=working_directory, **subprocess_options)
        except FileNotFoundError:
            if not allow_failure:
                fail_with_error(f"Command not found: {command_arguments[0]}")
            print_debug(f"Command not found, skipping: {command_arguments[0]}")
            return subprocess.CompletedProcess(command_arguments, 127, stdout="", stderr="")

    if stream_output:
        # A compile can take minutes; let its progress and errors reach the terminal live.
        command_result = run_or_report_missing_binary(text=True)
        print_debug(f"Exit code: {command_result.returncode}")
        if command_result.returncode != 0 and not allow_failure:
            fail_with_error(f"Command failed: {' '.join(command_arguments)}")
        return command_result

    command_result = run_or_report_missing_binary(text=True, capture_output=True)
    print_debug(f"Exit code: {command_result.returncode}")

    if command_result.stdout.strip():
        print_debug("stdout:")
        for stdout_line in command_result.stdout.rstrip().splitlines():
            print(f"    {stdout_line}")

    if command_result.stderr.strip():
        print_debug("stderr:")
        for stderr_line in command_result.stderr.rstrip().splitlines():
            print(f"    {stderr_line}")

    if command_result.returncode != 0 and not allow_failure:
        fail_with_error(f"Command failed: {' '.join(command_arguments)}")

    return command_result


def require_root_execution():
    if os.geteuid() != 0:
        fail_with_error("This script must be executed as root.")


def detect_repository_config_path():
    # Prefer a repository root literally named "genix"; otherwise fall back to the directory that
    # contains this script ("<root>/scripts/configure/configure_server.py"), so a clone with a
    # different directory name still resolves. The file itself may not exist yet.
    script_path = Path(__file__).resolve()
    for parent_path in script_path.parents:
        if parent_path.name == "genix":
            repository_config_path = parent_path / "config.toml"
            print_debug(f"Using repository config path: {repository_config_path}")
            return repository_config_path

    fallback_config_path = script_path.parents[2] / "config.toml"
    print_debug(
        f"No repository root named 'genix' found. Using {fallback_config_path} instead."
    )
    return fallback_config_path


def load_project_credentials(repository_config_path):
    # A missing file is not fatal: every value this script needs can be typed in instead, which is
    # what an edge-only host without a full clone of the repository normally does.
    if not repository_config_path.exists():
        print_debug(f"No config.toml at {repository_config_path}. Values will be requested.")
        return {}

    print_debug(f"Loading project config from {repository_config_path}")
    try:
        config_content = repository_config_path.read_bytes()
    except OSError as read_error:
        fail_with_error(f"Could not read config.toml: {read_error}")

    try:
        parsed_credentials = tomllib.loads(config_content.decode("utf-8"))
        parsed_credentials = apply_local_config_override(parsed_credentials, repository_config_path)
    except tomllib.TOMLDecodeError as parse_error:
        # Refuse to guess here: prompting would later overwrite a real but broken file.
        fail_with_error(f"Could not parse config.toml or {LOCAL_CONFIG_FILE_NAME}: {parse_error}")

    if not isinstance(parsed_credentials, dict):
        fail_with_error("config.toml must contain a table at the top level.")

    return parsed_credentials


def match_execution_mode(selected_value):
    normalized_value = selected_value.strip().lower()
    for mode_option in EXECUTION_MODE_OPTIONS:
        if normalized_value == str(mode_option["index"]) or normalized_value in mode_option["aliases"]:
            return mode_option

    return None


def print_execution_mode_options():
    print_debug("Available install modes:")
    for mode_option in EXECUTION_MODE_OPTIONS:
        print(f"    [{mode_option['index']}] {mode_option['label']}")


def resolve_execution_mode(selected_value=None):
    if selected_value is not None:
        selected_value = selected_value.strip()
        print_debug(f"Selecting install mode from CLI argument: {selected_value}")

        mode_option = match_execution_mode(selected_value)
        if not mode_option:
            print_execution_mode_options()
            fail_with_error(f"Unknown install mode: {selected_value}")

        print_debug(f"Selected install mode: {mode_option['label']}")
        return mode_option["mode"]

    if not sys.stdin.isatty():
        fail_with_error(
            "No install mode provided and no interactive terminal is available. "
            "Pass 1 (full), 2 (systemd) or 3 (edge)."
        )

    print_execution_mode_options()
    mode_option = match_execution_mode(input("Select the install mode: "))
    if not mode_option:
        fail_with_error("Invalid install mode selection.")

    print_debug(f"Selected install mode: {mode_option['label']}")
    return mode_option["mode"]


def parse_command_arguments():
    """Keep the host mode separate from the requested binary provenance."""
    argument_parser = argparse.ArgumentParser(
        description="Install the Genix backend systemd service and/or its HAProxy edge."
    )
    argument_parser.add_argument("mode", nargs="?", help="1/full, 2/systemd or 3/edge")
    argument_parser.add_argument(
        "--binary-source",
        choices=(BINARY_SOURCE_AUTO, BINARY_SOURCE_SOURCE, BINARY_SOURCE_PRECOMPILED),
        default=BINARY_SOURCE_AUTO,
        help="compile source, require a precompiled artifact, or retain automatic selection",
    )
    return argument_parser.parse_args()


def validate_domain(raw_value):
    domain_value = raw_value.strip().rstrip(".").lower()
    if not domain_value:
        return None, "The domain cannot be empty."
    if not re.fullmatch(r"[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+", domain_value):
        return None, f"'{raw_value.strip()}' is not a valid domain name (expected e.g. genix-api-4.un.pe)."

    return domain_value, None


def validate_upstream(raw_value):
    # server.upstream is normally a bare "host:port" pointing at the backend host, but a full
    # http:// or https:// upstream is accepted so an already-TLS backend can be reused as-is.
    upstream_value = raw_value.strip()
    if not upstream_value:
        return None, "The upstream cannot be empty."

    if "://" in upstream_value:
        upstream_scheme, _, host_and_port = upstream_value.partition("://")
        if upstream_scheme not in {"http", "https"}:
            return None, (
                f"Unsupported scheme '{upstream_scheme}://' in '{upstream_value}'. "
                "Use http://, https:// or a bare host:port."
            )
    else:
        host_and_port = upstream_value

    host_and_port = host_and_port.rstrip("/")
    if ":" not in host_and_port:
        return None, f"'{upstream_value}' must include the backend port (expected e.g. 100.64.0.2:14010)."

    upstream_host, _, upstream_port = host_and_port.rpartition(":")
    if not upstream_host:
        return None, f"'{upstream_value}' is missing the backend host."
    if not upstream_port.isdigit() or not 1 <= int(upstream_port) <= 65535:
        return None, f"'{upstream_port}' is not a valid port in '{upstream_value}'."

    return upstream_value, None


def validate_server_port(raw_value):
    port_value = raw_value.strip()
    if not port_value:
        return None, "The port cannot be empty."
    if not port_value.isdigit() or not 1 <= int(port_value) <= 65535:
        return None, f"'{port_value}' is not a valid TCP port (expected 1-65535)."

    return int(port_value), None


def resolve_credential_value(project_credentials, dotted_key, prompt_text, validate_value):
    """Read one credential, asking for it on the terminal when it is absent or invalid.

    Returns (value, was_prompted) so main() can offer to persist whatever had to be typed in.
    """
    raw_value = get_config_value(project_credentials, dotted_key)
    if raw_value is not None:
        validated_value, validation_error = validate_value(str(raw_value))
        if validated_value is not None:
            return validated_value, False
        print_debug(f"{dotted_key} in config.toml is unusable: {validation_error}")
    else:
        print_debug(f"{dotted_key} is not set in config.toml.")

    if not sys.stdin.isatty():
        fail_with_error(
            f"{dotted_key} is missing or invalid and there is no interactive terminal to ask for it. "
            f"Add {dotted_key} to config.toml and run the script again."
        )

    while True:
        validated_value, validation_error = validate_value(input(f"{prompt_text}: "))
        if validated_value is not None:
            print_debug(f"Using {dotted_key}={validated_value}")
            return validated_value, True

        print(f"[!] {validation_error}", file=sys.stderr)


def extract_edge_settings(project_credentials, backend_is_local=False):
    """Resolve the HAProxy side of the install.

    backend_is_local is set when this same host also runs the systemd service (full mode). The
    upstream is then always the backend's Unix socket, and server.upstream is not used.
    """
    edge_domain, domain_was_prompted = resolve_credential_value(
        project_credentials,
        "server.domain",
        "Enter server.domain, the public domain HAProxy serves (e.g. genix-api-4.un.pe)",
        validate_domain,
    )

    upstream_was_prompted = False
    if backend_is_local:
        upstream_scheme = "http"
        upstream_server = f"unix@{SERVICE_SOCKET_PATH}"
        print_debug("Full mode: forwarding to the local backend through its Unix socket.")
    else:
        configured_upstream, upstream_was_prompted = resolve_credential_value(
            project_credentials,
            "server.upstream",
            "Enter server.upstream, the backend host:port HAProxy forwards to (e.g. 10.0.0.7:14010)",
            validate_upstream,
        )
        # The `server` line takes a bare address; https becomes its ssl option.
        upstream_scheme, _, upstream_server = configured_upstream.rpartition("://")
        upstream_scheme = upstream_scheme or "http"
        upstream_server = upstream_server.rstrip("/")

    print_debug(f"Edge domain: {edge_domain}")
    print_debug(f"Edge upstream: {upstream_scheme}://{upstream_server}")
    return {
        "domain": edge_domain,
        "upstream_scheme": upstream_scheme,
        "upstream_server": upstream_server,
        "prompted_values": {
            **({"server.domain": edge_domain} if domain_was_prompted else {}),
            **({"server.upstream": configured_upstream} if upstream_was_prompted else {}),
        },
    }


def resolve_spoe_settings(project_credentials):
    """Read whether the API edge asks the fareward SPOE agent, and in which mode.

    Returns None when [spoe].socket is unset: HAProxy then forwards without the filter and the
    backend gates every request itself, as it does on Lambda. Only full mode calls this, because
    the backend trusts the agent's x-fareward-* headers only from a local peer.

    The agent and HAProxy must agree on the mode, and both follow [spoe].mode in config.toml:
    "shadow" (the default) only logs the verdict, "enforce" refuses on it.
    """
    spoe_socket = str(get_config_value(project_credentials, "spoe.socket", "")).strip()
    if not spoe_socket:
        print_debug("[spoe].socket is not set: the API edge forwards without the fareward agent.")
        return None

    spoe_mode = str(get_config_value(project_credentials, "spoe.mode", "shadow")).strip()
    if spoe_mode not in {"shadow", "enforce"}:
        fail_with_error(f'[spoe].mode must be "shadow" or "enforce", not "{spoe_mode}".')

    # The backend writes the route table at startup and the agent refuses to start without it.
    # Under ProtectSystem=strict the backend can only write inside its StateDirectory.
    routes_file = str(get_config_value(project_credentials, "fareward.routes_file", "")).strip()
    if not routes_file:
        fail_with_error(
            f"[spoe].socket is set, so [fareward].routes_file must name the route table "
            f'(e.g. "{SERVICE_STATE_DIRECTORY}/fareward-routes.toml").'
        )
    if Path(routes_file).parent != SERVICE_STATE_DIRECTORY:
        fail_with_error(
            f"[fareward].routes_file ({routes_file}) must be inside {SERVICE_STATE_DIRECTORY}: it is "
            "the only directory the backend's systemd unit lets it write."
        )

    print_debug(f"fareward SPOE agent at {spoe_socket}, mode {spoe_mode}.")
    return {"socket": spoe_socket, "enforce": spoe_mode == "enforce"}


def sync_edge_trust_setting(project_credentials, repository_config_path, spoe_settings):
    """Keep [fareward].trust_edge_headers equal to "HAProxy enforces the agent's verdict".

    In enforce mode the agent has already charged an admitted request, so a backend that ignores
    the edge headers charges it a second time. In shadow mode, or with no agent, the edge sends no
    such headers and the backend must gate every request itself. It is not a preference, so it is
    written without asking, before the service (re)starts and reads it.
    """
    trust_edge_headers = bool(spoe_settings and spoe_settings["enforce"])
    if get_config_value(project_credentials, "fareward.trust_edge_headers", False) == trust_edge_headers:
        return
    print_debug(f"Setting [fareward].trust_edge_headers = {str(trust_edge_headers).lower()} to match the edge.")
    set_config_values(repository_config_path, {"fareward.trust_edge_headers": trust_edge_headers})


def extract_server_port(project_credentials):
    server_port, port_was_prompted = resolve_credential_value(
        project_credentials,
        "server.port",
        "Enter server.port, the port the backend listens on (e.g. 14010)",
        validate_server_port,
    )

    print_debug(f"Backend listen port: {server_port}")
    return server_port, {"server.port": server_port} if port_was_prompted else {}


def persist_prompted_credentials(repository_config_path, prompted_values):
    if not prompted_values:
        return

    described_values = ", ".join(f"{key}={value}" for key, value in prompted_values.items())
    print_debug(f"These values were typed in and are not stored yet: {described_values}")

    if not sys.stdin.isatty():
        print_debug("No interactive terminal available. Skipping config.toml update.")
        return

    answer = input(f"Save them to {repository_config_path}? [Y/n]: ").strip().lower()
    if answer not in {"", "y", "yes"}:
        print_debug("Leaving config.toml untouched.")
        return

    config_file_already_existed = repository_config_path.exists()
    if not config_file_already_existed:
        repository_config_path.parent.mkdir(parents=True, exist_ok=True)
        repository_config_path.touch()

    try:
        set_config_values(repository_config_path, prompted_values)
    except OSError as write_error:
        fail_with_error(f"Could not write config.toml: {write_error}")

    if not config_file_already_existed:
        # A file created by root would be unreadable to the non-root service, so hand it to
        # whoever owns the repository directory — the account that cloned it.
        repository_directory_stat = repository_config_path.parent.stat()
        os.chown(
            repository_config_path,
            repository_directory_stat.st_uid,
            repository_directory_stat.st_gid,
        )
        os.chmod(repository_config_path, 0o600)

    print_debug(f"Saved {len(prompted_values)} value(s) to {repository_config_path}")


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


def remove_empty_placeholder_binary():
    """Delete a zero-byte genix_app left behind by an older version of this script.

    No placeholder is created any more. An empty file would be enabled as ExecStart and fail
    with 203/EXEC on the next boot, and it was never needed by genix-restart.path: a systemd
    path unit watches the closest existing parent directory, so PathChanged= fires when the
    binary is created for the first time.
    """
    if SERVICE_BINARY_PATH.is_file() and SERVICE_BINARY_PATH.stat().st_size == 0:
        print_debug(f"Removing empty placeholder binary at {SERVICE_BINARY_PATH}.")
        SERVICE_BINARY_PATH.unlink()


def resolve_go_architecture():
    machine_architecture = platform.machine().lower()
    if machine_architecture in {"aarch64", "arm64"}:
        return "arm64"
    if machine_architecture in {"x86_64", "amd64"}:
        return "amd64"

    return machine_architecture


def is_usable_executable(candidate_path):
    # An ELF header is what separates a real build from the old empty placeholder, a stray
    # shell script, or a half-written download.
    if not candidate_path.is_file() or candidate_path.stat().st_size == 0:
        return False

    try:
        with candidate_path.open("rb") as candidate_file:
            return candidate_file.read(len(ELF_MAGIC_BYTES)) == ELF_MAGIC_BYTES
    except OSError:
        return False


def detect_backend_source_directory(repository_root_path):
    backend_source_directory = repository_root_path / "backend"
    has_go_module = (backend_source_directory / "go.mod").is_file()
    has_main_package = (backend_source_directory / "main.go").is_file()

    if has_go_module and has_main_package:
        print_debug(f"Backend source found at {backend_source_directory}")
        return backend_source_directory

    print_debug(f"No compilable backend source at {backend_source_directory}")
    return None


def detect_go_binary():
    configured_go_binary = os.environ.get("GO_BINARY", "").strip()
    if configured_go_binary:
        configured_go_path = Path(configured_go_binary)
        if not os.access(configured_go_path, os.X_OK):
            fail_with_error(f"GO_BINARY is set to '{configured_go_binary}' but it is not executable.")
        return configured_go_path

    go_binary_in_path = shutil.which("go")
    if go_binary_in_path:
        return Path(go_binary_in_path)

    for candidate_go_path in GO_BINARY_SEARCH_PATHS:
        if os.access(candidate_go_path, os.X_OK):
            return candidate_go_path

    return None


def detect_unprivileged_username(repository_root_path):
    """Pick the account that should run go and git.

    Prefer the repository's owner: its module cache is already warm, git will not refuse the
    directory over "dubious ownership", and nothing root-owned is left inside the clone.
    """
    try:
        repository_owner = pwd.getpwuid(repository_root_path.stat().st_uid)
        if repository_owner.pw_name != "root":
            return repository_owner.pw_name
    except (KeyError, OSError):
        pass

    sudo_username = os.environ.get("SUDO_USER", "").strip()
    if sudo_username and sudo_username != "root":
        return sudo_username

    return None


def build_unprivileged_command(command_arguments, unprivileged_username, environment_overrides=None):
    prefixed_command = command_arguments
    if environment_overrides:
        prefixed_command = [
            "env",
            *[f"{name}={value}" for name, value in environment_overrides.items()],
            *command_arguments,
        ]

    if os.geteuid() == 0 and unprivileged_username:
        # -H sets HOME, which is what gives git the user's SSH keys and go its module cache.
        return ["sudo", "-u", unprivileged_username, "-H", *prefixed_command]

    return prefixed_command


def parse_local_replace_directories(backend_source_directory):
    """Return the directories that go.mod redirects modules into with a filesystem path.

    genix-orm is wired in as `replace github.com/ivanjoz/genix-orm => ./genix-orm`, so the build
    cannot resolve anything until that directory holds a real go.mod.
    """
    go_module_path = backend_source_directory / "go.mod"
    if not go_module_path.is_file():
        return []

    local_replace_directories = []
    inside_replace_block = False

    for raw_line in go_module_path.read_text(encoding="utf-8").splitlines():
        stripped_line = raw_line.split("//", 1)[0].strip()
        if not stripped_line:
            continue

        if inside_replace_block:
            if stripped_line == ")":
                inside_replace_block = False
                continue
            replace_body = stripped_line
        elif stripped_line == "replace (":
            inside_replace_block = True
            continue
        elif stripped_line.startswith("replace "):
            replace_body = stripped_line[len("replace "):]
        else:
            continue

        _, separator, replacement_target = replace_body.partition("=>")
        if not separator:
            continue

        # A replacement is a filesystem path only when it looks like one; module paths are ignored.
        replacement_fields = replacement_target.strip().split()
        replacement_path = replacement_fields[0] if replacement_fields else ""
        if not replacement_path.startswith(("./", "../", "/")):
            continue

        resolved_directory = (backend_source_directory / replacement_path).resolve()
        if resolved_directory not in local_replace_directories:
            local_replace_directories.append(resolved_directory)

    return local_replace_directories


def read_git_modules_setting(git_modules_path, setting_key, unprivileged_username):
    setting_result = run_command(
        build_unprivileged_command(
            ["git", "config", "--file", str(git_modules_path), "--get-regexp", setting_key],
            unprivileged_username,
        ),
        allow_failure=True,
    )
    if setting_result.returncode != 0:
        return {}

    settings_by_submodule_name = {}
    for listing_line in setting_result.stdout.splitlines():
        listing_fields = listing_line.split(maxsplit=1)
        if len(listing_fields) != 2:
            continue

        # Keys look like "submodule.backend/genix-orm.path"; the middle part is the submodule name.
        submodule_name = listing_fields[0].strip()[len("submodule."):].rsplit(".", 1)[0]
        settings_by_submodule_name[submodule_name] = listing_fields[1].strip()

    return settings_by_submodule_name


def list_git_submodules(repository_root_path, unprivileged_username):
    """Map each submodule path declared in .gitmodules to its name and remote URL."""
    git_modules_path = repository_root_path / ".gitmodules"
    if not git_modules_path.is_file() or not shutil.which("git"):
        return {}

    paths_by_name = read_git_modules_setting(
        git_modules_path, r"^submodule\..*\.path$", unprivileged_username
    )
    urls_by_name = read_git_modules_setting(
        git_modules_path, r"^submodule\..*\.url$", unprivileged_username
    )

    return {
        submodule_path: {"name": submodule_name, "url": urls_by_name.get(submodule_name, "")}
        for submodule_name, submodule_path in paths_by_name.items()
    }


def convert_ssh_remote_to_https(remote_url):
    """Turn git@host:owner/repo.git (or ssh://git@host/owner/repo.git) into an https:// URL.

    A host with no deploy key cannot use the SSH form at all, but the submodule is public, so
    plain HTTPS needs no credentials.
    """
    ssh_remote_match = re.match(r"^(?:ssh://)?[^@/]+@([^:/]+)[:/](.+)$", remote_url.strip())
    if not ssh_remote_match:
        return None

    return f"https://{ssh_remote_match.group(1)}/{ssh_remote_match.group(2)}"


def run_git_submodule_update(repository_root_path, relative_submodule_path, unprivileged_username, initialise):
    submodule_command = ["git", "submodule", "update", "--recursive"]
    if initialise:
        submodule_command.append("--init")
    submodule_command += ["--", relative_submodule_path]

    return run_command(
        build_unprivileged_command(
            submodule_command,
            unprivileged_username,
            # Never block on a credential or host-key prompt: a failure here has a fallback, a
            # hung script does not.
            environment_overrides={
                "GIT_TERMINAL_PROMPT": "0",
                "GIT_SSH_COMMAND": "ssh -o BatchMode=yes",
            },
        ),
        working_directory=repository_root_path,
        stream_output=True,
        allow_failure=True,
    )


def initialise_git_submodule(
    repository_root_path, relative_submodule_path, submodule_details, unprivileged_username
):
    """Clone a submodule, falling back from its declared SSH remote to HTTPS.

    .gitmodules declares git@github.com:… because that is what a developer with a key uses. A
    server usually has no key, and these submodules are public, so retry over HTTPS instead of
    giving up. The rewritten URL is stored in .git/config only — .gitmodules stays untouched, so
    nothing is committed and developers keep pushing over SSH.
    """
    print_debug(f"Initialising git submodule {relative_submodule_path}...")
    submodule_result = run_git_submodule_update(
        repository_root_path, relative_submodule_path, unprivileged_username, initialise=True
    )
    if submodule_result.returncode == 0:
        return

    https_remote_url = convert_ssh_remote_to_https(submodule_details["url"])
    if not https_remote_url:
        return

    print_debug(
        f"Cloning {submodule_details['url']} failed. Retrying over HTTPS: {https_remote_url}"
    )
    override_result = run_command(
        build_unprivileged_command(
            ["git", "config", f"submodule.{submodule_details['name']}.url", https_remote_url],
            unprivileged_username,
        ),
        working_directory=repository_root_path,
        allow_failure=True,
    )
    if override_result.returncode != 0:
        return

    # The first attempt already registered the submodule, so --init is not needed again; the URL
    # override in .git/config is what the retry picks up.
    run_git_submodule_update(
        repository_root_path, relative_submodule_path, unprivileged_username, initialise=False
    )


def clone_sibling_repository(missing_replace_directory, repository_root_path, unprivileged_username):
    """Clone a repository that go.mod expects beside the genix checkout (not a submodule).

    genix-s3 compiles from ../../genix-search/genix-s3 until it is published as a module, so a server
    building from source needs that checkout next to the repository, cloned as its owner.
    """
    try:
        sibling_name = missing_replace_directory.relative_to(repository_root_path.parent).parts[0]
    except ValueError:
        return
    sibling_remote = SIBLING_REPOSITORY_REMOTES.get(sibling_name)
    sibling_directory = repository_root_path.parent / sibling_name
    if not sibling_remote or sibling_directory.exists():
        return
    print_debug(f"Cloning {sibling_remote} into {sibling_directory}, which go.mod expects there.")
    run_command(build_unprivileged_command(
        ["git", "clone", "--depth", "1", sibling_remote, str(sibling_directory)], unprivileged_username
    ))


def ensure_local_replace_directories(
    backend_source_directory, repository_root_path, unprivileged_username
):
    """Populate the go.mod replace targets, initialising git submodules when that is what they are."""
    missing_replace_directories = [
        replace_directory
        for replace_directory in parse_local_replace_directories(backend_source_directory)
        if not (replace_directory / "go.mod").is_file()
    ]
    if not missing_replace_directories:
        return

    for missing_replace_directory in missing_replace_directories:
        print_debug(f"go.mod redirects a module to {missing_replace_directory}, which has no go.mod.")

    submodules_by_path = list_git_submodules(repository_root_path, unprivileged_username)
    for missing_replace_directory in missing_replace_directories:
        try:
            relative_replace_path = missing_replace_directory.relative_to(repository_root_path).as_posix()
        except ValueError:
            clone_sibling_repository(missing_replace_directory, repository_root_path, unprivileged_username)
            continue

        # Initialise only the submodule the backend actually needs: a backend host has no use for
        # the frontend ones, and pulling them can fail or waste minutes.
        submodule_details = submodules_by_path.get(relative_replace_path)
        if not submodule_details:
            continue

        initialise_git_submodule(
            repository_root_path, relative_replace_path, submodule_details, unprivileged_username
        )

    still_missing_directories = [
        replace_directory
        for replace_directory in missing_replace_directories
        if not (replace_directory / "go.mod").is_file()
    ]
    if still_missing_directories:
        described_directories = ", ".join(str(directory) for directory in still_missing_directories)
        fail_with_error(
            f"go.mod redirects modules to {described_directories}, but they are still empty. "
            "Both the declared remote and its https:// equivalent failed. Populate them manually "
            "with 'git submodule update --init --recursive' in the repository as its owner, then "
            "run this script again."
        )


def compile_backend_binary(backend_source_directory, repository_root_path):
    go_binary_path = detect_go_binary()
    if not go_binary_path:
        fail_with_error(
            "Backend source is present but the Go toolchain was not found. Install Go or run the "
            "script with GO_BINARY=/path/to/go (sudo strips /usr/local/go/bin from PATH)."
        )

    print_debug(f"Using Go toolchain at {go_binary_path}")

    unprivileged_username = detect_unprivileged_username(repository_root_path)
    ensure_local_replace_directories(
        backend_source_directory, repository_root_path, unprivileged_username
    )

    build_output_directory = repository_root_path / "tmp"
    build_output_directory.mkdir(parents=True, exist_ok=True)

    # The compile runs as a non-root account, so tmp/ must belong to the repository owner rather
    # than to root — which is what it would be after mkdir here, or after an older root-run build.
    repository_directory_stat = repository_root_path.stat()
    if os.geteuid() == 0 and build_output_directory.stat().st_uid != repository_directory_stat.st_uid:
        print_debug(f"Handing {build_output_directory} to uid {repository_directory_stat.st_uid}")
        os.chown(build_output_directory, repository_directory_stat.st_uid, repository_directory_stat.st_gid)

    # Same artifact name and ldflags as scripts/deploy_vps.go, so a locally compiled binary and a
    # deployed one report the same build metadata.
    build_output_path = build_output_directory / f"genix_app_linux_{resolve_go_architecture()}"
    build_date = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    build_flags = f"-s -w -X 'app/core.BuildDate={build_date}'"

    build_command = [
        str(go_binary_path),
        "build",
        "-tags",
        BACKEND_BUILD_TAGS,
        "-ldflags",
        build_flags,
        "-o",
        str(build_output_path),
        ".",
    ]

    if os.geteuid() == 0 and unprivileged_username:
        print_debug(f"Compiling as '{unprivileged_username}' to reuse that account's Go module cache.")
    build_command = build_unprivileged_command(build_command, unprivileged_username)

    print_debug(f"Compiling backend from {backend_source_directory} (this can take a while)...")
    run_command(build_command, working_directory=backend_source_directory, stream_output=True)

    if not is_usable_executable(build_output_path):
        fail_with_error(f"The compiler reported success but {build_output_path} is not a valid executable.")

    print_debug(f"Compilation successful: {build_output_path}")
    return build_output_path


def find_prebuilt_binary(repository_root_path, prefer_downloaded=False):
    go_architecture = resolve_go_architecture()
    downloaded_binary_path = repository_root_path / "tmp" / f"genix_app_linux_{go_architecture}"
    candidate_binary_paths = [
        downloaded_binary_path,
        SERVICE_BINARY_PATH,
        repository_root_path / "backend" / "genix_app",
        repository_root_path / "genix_app",
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


def install_service_binary(source_binary_path, runtime_user_entry):
    if source_binary_path != SERVICE_BINARY_PATH:
        # Write next to the destination and rename, so the service never sees a half-copied file.
        # PathChanged= watches IN_MOVED_TO as well as IN_CLOSE_WRITE, so the watcher still fires.
        staged_binary_path = SERVICE_INSTALL_DIRECTORY / ".genix_app.staged"
        print_debug(f"Installing {source_binary_path} to {SERVICE_BINARY_PATH}")
        try:
            shutil.copyfile(source_binary_path, staged_binary_path)
            os.chown(staged_binary_path, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
            os.chmod(staged_binary_path, 0o750)
            os.replace(staged_binary_path, SERVICE_BINARY_PATH)
        except OSError as install_error:
            staged_binary_path.unlink(missing_ok=True)
            fail_with_error(f"Could not install the binary: {install_error}")
        return

    print_debug(f"Binary already in place at {SERVICE_BINARY_PATH}. Fixing ownership only.")
    os.chown(SERVICE_BINARY_PATH, runtime_user_entry.pw_uid, runtime_user_entry.pw_gid)
    executable_mode = stat.S_IMODE(SERVICE_BINARY_PATH.stat().st_mode) | 0o750
    os.chmod(SERVICE_BINARY_PATH, executable_mode)
    print_debug(f"Binary permissions set to {oct(executable_mode)}.")


def provide_service_binary(
    repository_root_path, runtime_user_entry, binary_source=BINARY_SOURCE_AUTO
):
    """Put a runnable binary at SERVICE_BINARY_PATH: compile it, or find one, or fail."""
    remove_empty_placeholder_binary()

    backend_source_directory = detect_backend_source_directory(repository_root_path)
    if binary_source == BINARY_SOURCE_SOURCE:
        if not backend_source_directory:
            fail_with_error(
                f"Building from source was selected, but no compilable backend exists under "
                f"{repository_root_path / 'backend'}."
            )
        built_binary_path = compile_backend_binary(backend_source_directory, repository_root_path)
        install_service_binary(built_binary_path, runtime_user_entry)
        return

    if binary_source == BINARY_SOURCE_AUTO and backend_source_directory:
        built_binary_path = compile_backend_binary(backend_source_directory, repository_root_path)
        install_service_binary(built_binary_path, runtime_user_entry)
        return

    print_debug(f"Selecting a prebuilt backend binary (binary source: {binary_source}).")
    prebuilt_binary_path = find_prebuilt_binary(
        repository_root_path,
        prefer_downloaded=binary_source == BINARY_SOURCE_PRECOMPILED,
    )
    if not prebuilt_binary_path:
        fail_with_error(
            f"No backend source under {repository_root_path / 'backend'} and no prebuilt binary "
            f"found. Clone the repository with its backend folder, or place a compiled binary at "
            f"{SERVICE_BINARY_PATH}, and run the script again."
        )

    install_service_binary(prebuilt_binary_path, runtime_user_entry)


def read_haproxy_version():
    """(major, minor) of the installed haproxy, or None when it is missing or unreadable."""
    if not shutil.which("haproxy"):
        return None
    version_output = run_command(["haproxy", "-v"], allow_failure=True).stdout
    version_match = re.search(r"version (\d+)\.(\d+)", version_output)
    if not version_match:
        return None
    return int(version_match.group(1)), int(version_match.group(2))


def add_official_haproxy_repository():
    """Point apt at the official HAProxy LTS packages (haproxy.debian.net / the vbernat PPA).

    The distributions' own packages are older than HAPROXY_MINIMUM_VERSION (Ubuntu 24.04 ships 2.8,
    Debian 13 ships 3.0), and 3.1 is the first with the SPOP 2.0 the fareward agent speaks.
    """
    os_release = platform.freedesktop_os_release()
    distribution_id = os_release.get("ID", "")
    distribution_codename = os_release.get("VERSION_CODENAME", "")
    if distribution_id == "ubuntu":
        run_command(["env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y",
                     "--no-install-recommends", "software-properties-common"])
        run_command(["add-apt-repository", "-y", f"ppa:vbernat/haproxy-{HAPROXY_REPOSITORY_BRANCH}"])
        return
    if distribution_id == "debian" and distribution_codename:
        keyring_path = Path("/etc/apt/keyrings/haproxy-archive-keyring.gpg")
        keyring_path.parent.mkdir(parents=True, exist_ok=True)
        run_command(["curl", "-fsSL", "--output", str(keyring_path),
                     "https://haproxy.debian.net/haproxy-archive-keyring.gpg"])
        Path("/etc/apt/sources.list.d/haproxy.list").write_text(
            f"deb [signed-by={keyring_path}] https://haproxy.debian.net "
            f"{distribution_codename}-backports-{HAPROXY_REPOSITORY_BRANCH} main\n",
            encoding="utf-8",
        )
        return
    fail_with_error(
        f"No official HAProxy {HAPROXY_REPOSITORY_BRANCH} repository is known for "
        f"'{distribution_id} {distribution_codename}'. Install HAProxy {HAPROXY_REPOSITORY_BRANCH} by hand and rerun."
    )


def read_apt_candidate_haproxy_version():
    """(major, minor) apt would install from the configured sources, (0, 0) when it has none."""
    policy_output = run_command(["apt-cache", "policy", "haproxy"], allow_failure=True).stdout
    candidate_match = re.search(r"Candidate:\s*(?:\d+:)?(\d+)\.(\d+)", policy_output)
    if not candidate_match:
        return (0, 0)
    return int(candidate_match.group(1)), int(candidate_match.group(2))


def ensure_haproxy_is_installed():
    """Install HAProxy (>= HAPROXY_MINIMUM_VERSION) and certbot when they are missing."""
    if shutil.which("apt-get"):
        installed_version = read_haproxy_version()
        if installed_version is None or installed_version < HAPROXY_MINIMUM_VERSION:
            run_command(["apt-get", "update"])
            if read_apt_candidate_haproxy_version() >= HAPROXY_MINIMUM_VERSION:
                # Ubuntu 26.04 already ships 3.2: the distribution's package is enough.
                run_command(["env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "haproxy"])
            else:
                add_official_haproxy_repository()
                run_command(["apt-get", "update"])
                run_command(["env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y",
                             f"haproxy={HAPROXY_REPOSITORY_BRANCH}.*"])
        if not shutil.which("certbot"):
            run_command(["env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "certbot"])
    elif shutil.which("dnf"):
        # No official repository for dnf: the distribution's package is used when it is new enough.
        missing_packages = [package for package in ("haproxy", "certbot") if not shutil.which(package)]
        if missing_packages:
            run_command(["dnf", "install", "-y", *missing_packages])
    else:
        fail_with_error("Neither apt-get nor dnf is available: install HAProxy and certbot by hand.")

    installed_version = read_haproxy_version()
    minimum_version_text = ".".join(map(str, HAPROXY_MINIMUM_VERSION))
    if installed_version is None or installed_version < HAPROXY_MINIMUM_VERSION:
        installed_version_text = ".".join(map(str, installed_version)) if installed_version else "none"
        fail_with_error(
            f"HAProxy {minimum_version_text}+ is required (SPOP 2.0 for the fareward agent); found "
            f"{installed_version_text}. Install HAProxy {HAPROXY_REPOSITORY_BRANCH} and rerun."
        )
    if not shutil.which("certbot"):
        fail_with_error("certbot is not installed or not available in PATH.")
    print_debug(f"Detected HAProxy {'.'.join(map(str, installed_version))}")


def detect_haproxy_quic_support():
    """'native', 'compat' or None, from the build's feature list.

    'compat' is HAProxy linked against an OpenSSL without the QUIC API: QUIC then needs the
    `limited-quic` global and cannot take 0-RTT. None means no QUIC at all, so no UDP listener and no
    Alt-Svc: advertising h3 with nothing answering on UDP 443 would only slow browsers down.
    """
    build_report = run_command(["haproxy", "-vv"], allow_failure=True).stdout
    feature_line = next((line for line in build_report.splitlines() if line.startswith("Feature list")), "")
    build_features = set(feature_line.split())
    if "+QUIC" not in build_features:
        print_debug("WARNING: this HAProxy was built without QUIC. The edge serves TLS over TCP only (no HTTP/3).")
        return None
    return "compat" if "+QUIC_OPENSSL_COMPAT" in build_features else "native"


def stop_nginx_if_running():
    """HAProxy replaces Nginx as the edge: an Nginx still holding :80/:443 keeps it from binding.

    It is disabled, not uninstalled, and its configuration is left where it was.
    """
    if run_command(["systemctl", "is-active", "--quiet", "nginx"], allow_failure=True).returncode != 0:
        return
    print_debug("Nginx is running and holds the web ports HAProxy needs. Stopping and disabling it.")
    run_command(["systemctl", "disable", "--now", "nginx"])


def open_web_ports():
    """TCP 80 (HTTP-01 renewals), TCP 443 (TLS and the HTTP/2 fallback) and UDP 443 (QUIC)."""
    ensure_tcp_port_open(80, "HTTP, certificate renewals")
    ensure_tcp_port_open(443, "HTTPS")
    ensure_udp_port_open(443, "HTTP/3 QUIC")


def write_network_tuning():
    """BBR congestion control with the fq qdisc it paces through.

    It only changes TCP: the HTTP/2 fallback and TCP 80, which on lossy mobile links keep more
    throughput than with cubic. QUIC runs its own congestion control inside HAProxy.
    """
    sysctl_text = (
        "# Written by scripts/configure/configure_server.py (write_network_tuning).\n"
        "net.core.default_qdisc = fq\n"
        "net.ipv4.tcp_congestion_control = bbr\n"
    )
    existing_text = NETWORK_TUNING_SYSCTL_PATH.read_text(encoding="utf-8") if NETWORK_TUNING_SYSCTL_PATH.exists() else None
    if existing_text == sysctl_text:
        print_debug(f"{NETWORK_TUNING_SYSCTL_PATH} unchanged.")
        return
    # The module loads on boot before sysctl.d applies, and now for this run.
    NETWORK_TUNING_MODULES_PATH.write_text("tcp_bbr\n", encoding="utf-8")
    run_command(["modprobe", "tcp_bbr"], allow_failure=True)
    NETWORK_TUNING_SYSCTL_PATH.write_text(sysctl_text, encoding="utf-8")
    run_command(["sysctl", "--load", str(NETWORK_TUNING_SYSCTL_PATH)], allow_failure=True)


def build_certificate_deploy_hook():
    """The script certbot runs after issuing or renewing a certificate (--deploy-hook).

    HAProxy reads the chain and the key from one file, so the hook joins them into
    HAPROXY_CERTIFICATES_DIRECTORY/<domain>.pem and reloads. The rename keeps a reload that races
    the hook from ever reading half a file.
    """
    return f"""#!/bin/sh
# Written by scripts/configure/configure_server.py (build_certificate_deploy_hook).
# certbot sets RENEWED_LINEAGE=/etc/letsencrypt/live/<domain>.
set -eu
domain=$(basename "$RENEWED_LINEAGE")
umask 077
mkdir -p {HAPROXY_CERTIFICATES_DIRECTORY}
cat "$RENEWED_LINEAGE/fullchain.pem" "$RENEWED_LINEAGE/privkey.pem" > "{HAPROXY_CERTIFICATES_DIRECTORY}/$domain.pem.tmp"
mv "{HAPROXY_CERTIFICATES_DIRECTORY}/$domain.pem.tmp" "{HAPROXY_CERTIFICATES_DIRECTORY}/$domain.pem"
systemctl reload haproxy 2>/dev/null || true
"""


def certificate_is_renewed_through_haproxy(endpoint_hostname):
    """Whether certbot's renewal config for the domain uses the standalone port HAProxy forwards to.

    A certificate issued through `certbot --nginx` renews through Nginx, which is gone, so it has
    to be issued again once through the standalone authenticator.
    """
    renewal_configuration_path = LETSENCRYPT_RENEWAL_DIRECTORY / f"{endpoint_hostname}.conf"
    if not renewal_configuration_path.exists():
        return False
    renewal_configuration = renewal_configuration_path.read_text(encoding="utf-8")
    return (
        re.search(r"^authenticator\s*=\s*standalone\s*$", renewal_configuration, re.M) is not None
        and re.search(rf"^http01_port\s*=\s*{CERTBOT_HTTP01_PORT}\s*$", renewal_configuration, re.M) is not None
        and str(CERTIFICATE_DEPLOY_HOOK_PATH) in renewal_configuration
    )


def install_certificate_deploy_hook(endpoint_hostname):
    """Write certbot's deploy hook and, when the domain already has a certificate (an Nginx-era
    one included), give HAProxy its .pem now, so the edge starts with TLS instead of serving plain
    HTTP while the certificate is reissued."""
    CERTIFICATE_DEPLOY_HOOK_PATH.write_text(build_certificate_deploy_hook(), encoding="utf-8")
    os.chmod(CERTIFICATE_DEPLOY_HOOK_PATH, 0o755)
    lineage_directory = LETSENCRYPT_DIRECTORY / endpoint_hostname
    haproxy_certificate_path = HAPROXY_CERTIFICATES_DIRECTORY / f"{endpoint_hostname}.pem"
    if (lineage_directory / "fullchain.pem").exists() and not haproxy_certificate_path.exists():
        run_command(["env", f"RENEWED_LINEAGE={lineage_directory}", str(CERTIFICATE_DEPLOY_HOOK_PATH)])


def ensure_certificate(endpoint_hostname):
    """Issue the domain's Let's Encrypt certificate and keep HAProxy's copy of it in place.

    HTTP-01 through HAProxy's :80, which forwards the challenge to certbot on CERTBOT_HTTP01_PORT,
    so the domain's DNS must already point at this host. certbot's timer renews it the same way and
    the deploy hook refreshes the .pem. ECDSA P-256: each TLS and QUIC handshake is several times
    cheaper than with RSA 2048. No email is registered: Let's Encrypt no longer sends expiry mail.
    """
    lineage_directory = LETSENCRYPT_DIRECTORY / endpoint_hostname
    if not certificate_is_renewed_through_haproxy(endpoint_hostname):
        certbot_arguments = [
            "certbot", "certonly", "--standalone",
            "--http-01-address", "127.0.0.1", "--http-01-port", str(CERTBOT_HTTP01_PORT),
            "--non-interactive", "--agree-tos", "--register-unsafely-without-email",
            "--key-type", "ecdsa", "--elliptic-curve", "secp256r1",
            "--cert-name", endpoint_hostname, "-d", endpoint_hostname,
            "--deploy-hook", str(CERTIFICATE_DEPLOY_HOOK_PATH),
        ]
        if (lineage_directory / "fullchain.pem").exists():
            # Issued through Nginx (or as RSA): reissue now so the renewal config switches over.
            print_debug(f"Reissuing the certificate of {endpoint_hostname} so it renews through HAProxy.")
            certbot_arguments.append("--force-renewal")
        run_command(certbot_arguments)

    # The hook runs on issuance; this covers a host whose .pem was deleted since.
    if not (HAPROXY_CERTIFICATES_DIRECTORY / f"{endpoint_hostname}.pem").exists():
        run_command(["env", f"RENEWED_LINEAGE={lineage_directory}", str(CERTIFICATE_DEPLOY_HOOK_PATH)])


def list_haproxy_certificates():
    if not HAPROXY_CERTIFICATES_DIRECTORY.is_dir():
        return []
    return sorted(HAPROXY_CERTIFICATES_DIRECTORY.glob("*.pem"))


def build_haproxy_main_configuration(process_group, tls_is_available, quic_support):
    """The shared head of haproxy.cfg: global, defaults and the single frontend for every domain.

    Routing is by Host: `site_<domain>` is the backend each site file declares, and a Host no site
    claims gets 404. The :443 binds exist only once a certificate does (`crt <directory>` refuses an
    empty one); until then every site is plain HTTP, which is also how the first HTTP-01 challenge
    gets answered.

    process_group: the service account's group (the units run as Group=<runtime user>). HAProxy
    drops every supplementary group when it drops root, so adding its user to that group does
    nothing; running with it as the primary group is the only way to reach the fareward agent's
    socket, which is 0660 and owned by the service account.
    """
    tls_binds = ""
    https_redirect = ""
    alt_svc_header = ""
    limited_quic = ""
    if tls_is_available:
        certificate_options = f"ssl crt {HAPROXY_CERTIFICATES_DIRECTORY}/"
        tls_binds = (
            f"    bind :443 {certificate_options} alpn h2,http/1.1 allow-0rtt\n"
            f"    bind [::]:443 v6only {certificate_options} alpn h2,http/1.1 allow-0rtt\n"
        )
        if quic_support:
            # The OpenSSL compatibility layer cannot do 0-RTT over QUIC.
            quic_early_data = " allow-0rtt" if quic_support == "native" else ""
            tls_binds += (
                f"    bind quic4@:443 {certificate_options} alpn h3{quic_early_data}\n"
                f"    bind quic6@[::]:443 v6only {certificate_options} alpn h3{quic_early_data}\n"
            )
            # Also on errors: a client that only ever got errors would never switch to HTTP/3.
            alt_svc_header = '    http-response set-header alt-svc "h3=\\":443\\"; ma=86400" if { ssl_fc }\n'
            if quic_support == "compat":
                limited_quic = "    limited-quic\n"
        https_redirect = "    http-request redirect scheme https code 301 if !{ ssl_fc } !acme_challenge\n"

    return f"""# Written by scripts/configure/configure_server.py (build_haproxy_main_configuration). Do not edit:
# rerun configure_server.py or configure_fareward.py. Every file in {HAPROXY_SITES_DIRECTORY}/ is
# appended below this head, one per public domain.

global
    # stdout under systemd lands in the journal: journalctl -u haproxy.
    log stdout format raw local0
    user haproxy
    group {process_group}
    ssl-default-bind-options ssl-min-ver TLSv1.3
{limited_quic}
defaults
    mode http
    log global
    option httplog
    timeout connect 5s
    timeout client 60s
    timeout server 60s
    timeout http-request 10s
    timeout http-keep-alive 10s
    # A WebSocket after its Upgrade.
    timeout tunnel 3600s

frontend edge
    bind :80
    bind [::]:80 v6only
{tls_binds}
    # Slots 0-2 hold the fareward verdict (status, reason, error) for the access log. Only a
    # frontend can declare them; the API site's backend fills them with `capture ... id N`.
    declare capture request len 4
    declare capture request len 16
    declare capture request len 8

    acl acme_challenge path_beg /.well-known/acme-challenge/
{https_redirect}{alt_svc_header}    # TLS 1.3 early data can be replayed by anyone who captured it: GET and HEAD are safe to
    # replay, anything else waits for the handshake to complete.
    http-request wait-for-handshake if {{ ssl_fc_has_early }} !{{ method GET HEAD }}

    use_backend acme_certbot if acme_challenge
    use_backend site_%[req.hdr(host),host_only,lower]
    default_backend unknown_site

backend acme_certbot
    # certbot --standalone, only listening while it issues or renews.
    server certbot 127.0.0.1:{CERTBOT_HTTP01_PORT}

backend unknown_site
    # A Host no site file claims.
    http-request deny deny_status 404
"""


def build_api_spoe_configuration(endpoint_hostname):
    """The SPOE engine file of the API site: what HAProxy sends the fareward agent, and when.

    Asked once per request on the headers, before the body is read (nothing on this edge sets
    `option http-buffer-request`). Preflights are answered by the edge and never reach the agent.
    """
    return f"""# Written by scripts/configure/configure_server.py (build_api_spoe_configuration).
# The fareward SPOE agent for {endpoint_hostname}; see fareward/plans/plan.md.

[fareward]
spoe-agent fareward-agent
    messages     fareward-check
    # The agent's variables arrive as txn.fw.<name>: status, reason, company, user, route, grant,
    # and, in enforce mode, usage (plus window, spent, refused and retry on a credit refusal).
    option       var-prefix      fw
    # Any agent failure (down, timeout, protocol error) sets txn.fw.error.
    option       set-on-error    error
    # One local round trip. Phase 0 of the plan measures p99; set this just above it.
    timeout      processing      50ms
    use-backend  spoe_{endpoint_hostname}

spoe-message fareward-check
    # A missing header is sent as NULL: no Content-Length on a charged write is answered 411.
    args method=method path=path auth=req.hdr(authorization) length=req.hdr(content-length)
    event on-backend-http-request if !METH_OPTIONS
"""


def build_api_site_configuration(endpoint_hostname, upstream_scheme, upstream_server, spoe_settings):
    """The API's site file: its backend and, with the fareward agent, the agent's SPOP backend.

    spoe_settings is None (no agent: the backend gates every request itself) or
    {"socket": path, "enforce": bool}. In shadow mode the verdict is only logged; in enforce mode a
    refusal is answered here and an admitted request carries the agent's identity headers.
    """
    if upstream_server.startswith("unix@"):
        server_line = f"    server genix {upstream_server}"
    elif upstream_scheme == "https":
        # Not verified, as with the Nginx proxy before it: the upstream is a private address.
        server_line = f"    server genix {upstream_server} ssl verify none alpn http/1.1"
    else:
        server_line = f"    server genix {upstream_server}"

    spoe_rules = ""
    spoe_backend = ""
    if spoe_settings:
        spoe_rules = f"""
    # The fareward SPOE agent (fareward/plans/plan.md). Its verdict goes to the access log in both
    # modes: shadow mode is compared against the backend's own decisions from these lines.
    filter spoe engine fareward config {HAPROXY_SPOE_DIRECTORY}/{endpoint_hostname}.conf
    http-request capture var(txn.fw.status) id 0
    http-request capture var(txn.fw.reason) id 1
    http-request capture var(txn.fw.error) id 2
"""
        if spoe_settings["enforce"]:
            spoe_rules += """
    # Enforce mode. Fail closed: an agent error, or no verdict at all, is a 503.
    http-request deny deny_status 503 if { var(txn.fw.error) -m found }
    http-request deny deny_status 503 unless { var(txn.fw.status) -m found }
    # The backend's own statuses (fareward src/routes/table.rs and spoe/agent.rs). Every refusal
    # from here on carries the client's usage header (fareward plan D10): the agent always sets
    # txn.fw.usage in enforce mode, so the two 503s above, where it never answered, go without it.
    http-request deny deny_status 400 hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 400 }
    http-request deny deny_status 401 hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 401 }
    http-request deny deny_status 403 hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 403 }
    # 411 has no built-in error page, so it is answered with an explicit body.
    http-request return status 411 content-type "application/json" string "{\\"error\\":\\"Content-Length required\\"}" hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 411 }
    http-request deny deny_status 413 hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 413 }
    # 429 says what the client did and when to come back, and carries Retry-After (fareward plan
    # D10, D11). The agent sets window, spent, refused and retry on every credit refusal; window is
    # already worded from the configured window length.
    http-request return status 429 content-type "text/html; charset=utf-8" lf-string "<!doctype html><html lang=\\"es\\"><head><meta charset=\\"utf-8\\"><title>429</title></head><body><h1>Demasiadas solicitudes</h1><p>Has consumido %[var(txn.fw.spent)] créditos %[var(txn.fw.window)], además de %[var(txn.fw.refused)] solicitud(es) rechazada(s).</p><p>¿Eres humano? Intenta nuevamente en %[var(txn.fw.retry)] segundos.</p></body></html>" hdr x-fareward-usage %[var(txn.fw.usage)] hdr retry-after %[var(txn.fw.retry)] if { var(txn.fw.status) -m int eq 429 }
    http-request deny deny_status 500 hdr x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.status) -m int eq 500 }
    # Anything else that is not "go on" fails closed.
    http-request deny deny_status 503 hdr x-fareward-usage %[var(txn.fw.usage)] unless { var(txn.fw.status) -m int eq 0 }
    # Admitted: who it is and that the gate already ran. The backend believes these only with
    # [fareward].trust_edge_headers = true and from a local peer, and only because the client's
    # copies were deleted above.
    http-request set-header x-fareward-admitted 1
    http-request set-header x-fareward-company %[var(txn.fw.company)]
    http-request set-header x-fareward-user %[var(txn.fw.user)]
    http-request set-header x-fareward-route %[var(txn.fw.route)]
    http-request set-header x-fareward-grant %[var(txn.fw.grant),hex] if { var(txn.fw.grant) -m found }
    # The client's usage header on what the backend answers; set-header replaces any the backend
    # wrote. The figures are from before the GET top-up, charged after the response leaves.
    http-response set-header x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.usage) -m found }
"""
        spoe_backend = f"""
backend spoe_{endpoint_hostname}
    # SPOP 2.0 to the fareward agent ([spoe].socket in config.toml).
    mode spop
    timeout connect 1s
    timeout server 30s
    server fareward unix@{spoe_settings["socket"]}
"""

    return f"""# Written by scripts/configure/configure_server.py (build_api_site_configuration).
# The Genix API at {endpoint_hostname}.

backend site_{endpoint_hostname}
    # Agent WebSockets and /agent/stream stay open while the user thinks between messages.
    timeout server 3600s
    option forwardfor
    http-request deny deny_status 413 if {{ req.hdr_val(content-length) gt {API_MAX_BODY_BYTES} }}
    # Browser preflights are answered here, before the backend or the fareward agent sees them.
    http-request return status 204 hdr Access-Control-Allow-Origin "%[req.hdr(origin)]" hdr Access-Control-Allow-Methods "GET, POST, OPTIONS" hdr Access-Control-Allow-Headers "Authorization, Content-Type, X-Customer-Token" hdr Access-Control-Max-Age 86400 if METH_OPTIONS
    # The identity headers belong to the edge: a client sending them is lying.
    http-request del-header x-fareward- -m beg
    # The backend reads the caller's address from X-Real-IP (core.ClientIPFromRequest).
    http-request set-header x-real-ip %[src]
    http-request set-header x-forwarded-proto https if {{ ssl_fc }}
{spoe_rules}
{server_line}
{spoe_backend}"""


def write_file_if_changed(file_path, file_contents, file_mode=0o644):
    """Write the file only when its contents differ. Returns the previous contents (None if absent)
    and whether it changed, so a rejected configuration can be put back."""
    previous_contents = file_path.read_text(encoding="utf-8") if file_path.exists() else None
    if previous_contents == file_contents:
        return previous_contents, False
    file_path.parent.mkdir(parents=True, exist_ok=True)
    file_path.write_text(file_contents, encoding="utf-8")
    os.chmod(file_path, file_mode)
    return previous_contents, True


def restore_file(file_path, previous_contents):
    if previous_contents is None:
        file_path.unlink(missing_ok=True)
    else:
        file_path.write_text(previous_contents, encoding="utf-8")


def apply_haproxy_configuration(process_group, site_files):
    """Write this run's site files, regenerate haproxy.cfg around every site, validate and reload.

    site_files maps each path to its contents. `haproxy -c` covers every site on the host, so a
    rejected file is put back before failing: the other sites keep reloading. Returns whether
    anything changed.
    """
    previous_site_contents = {}
    for site_file_path, site_file_contents in site_files.items():
        previous_contents, site_file_changed = write_file_if_changed(site_file_path, site_file_contents)
        if site_file_changed:
            previous_site_contents[site_file_path] = previous_contents

    main_configuration = build_haproxy_main_configuration(
        process_group, bool(list_haproxy_certificates()), detect_haproxy_quic_support()
    )
    site_configurations = [
        site_file_path.read_text(encoding="utf-8") for site_file_path in sorted(HAPROXY_SITES_DIRECTORY.glob("*.cfg"))
    ]
    full_configuration = "\n".join([main_configuration, *site_configurations])
    previous_full_configuration, configuration_changed = write_file_if_changed(
        HAPROXY_CONFIGURATION_PATH, full_configuration
    )
    if not configuration_changed and not previous_site_contents:
        print_debug(f"HAProxy configuration unchanged: {HAPROXY_CONFIGURATION_PATH}")
        return False

    validation_result = run_command(["haproxy", "-c", "-f", str(HAPROXY_CONFIGURATION_PATH)], allow_failure=True)
    if validation_result.returncode != 0:
        for site_file_path, previous_contents in previous_site_contents.items():
            restore_file(site_file_path, previous_contents)
        restore_file(HAPROXY_CONFIGURATION_PATH, previous_full_configuration)
        fail_with_error(
            "HAProxy rejected the generated configuration (see the output above). The previous "
            "files were put back, so the other sites on this host keep working."
        )

    print_debug(f"Wrote {HAPROXY_CONFIGURATION_PATH}")
    run_command(["systemctl", "enable", "haproxy"])
    # reload keeps established connections; a stopped HAProxy has nothing to reload.
    if run_command(["systemctl", "is-active", "--quiet", "haproxy"], allow_failure=True).returncode == 0:
        run_command(["systemctl", "reload", "haproxy"])
    else:
        run_command(["systemctl", "restart", "haproxy"])
    return True


def configure_haproxy_site(endpoint_hostname, process_group, site_files):
    """Put a domain behind the HAProxy edge: install, ports, its files, its certificate.

    Written twice: plain HTTP first so :80 answers the HTTP-01 challenge, then again once the
    certificate exists, which adds the :443 binds.
    """
    ensure_haproxy_is_installed()
    # Ports first: the certificate's HTTP-01 challenge arrives on TCP 80.
    open_web_ports()
    stop_nginx_if_running()
    HAPROXY_CERTIFICATES_DIRECTORY.mkdir(mode=0o700, parents=True, exist_ok=True)
    install_certificate_deploy_hook(endpoint_hostname)
    apply_haproxy_configuration(process_group, site_files)
    ensure_certificate(endpoint_hostname)
    apply_haproxy_configuration(process_group, site_files)


def configure_api_edge(edge_settings, spoe_settings, process_group):
    endpoint_hostname = edge_settings["domain"]
    write_network_tuning()
    site_files = {
        HAPROXY_SITES_DIRECTORY / f"{endpoint_hostname}.cfg": build_api_site_configuration(
            endpoint_hostname, edge_settings["upstream_scheme"], edge_settings["upstream_server"], spoe_settings
        ),
    }
    spoe_configuration_path = HAPROXY_SPOE_DIRECTORY / f"{endpoint_hostname}.conf"
    if spoe_settings:
        site_files[spoe_configuration_path] = build_api_spoe_configuration(endpoint_hostname)
    else:
        spoe_configuration_path.unlink(missing_ok=True)
    configure_haproxy_site(endpoint_hostname, process_group, site_files)


def build_main_service_contents(runtime_username, repository_config_path, server_port):
    """server_port None means full mode: the backend binds SERVICE_SOCKET_PATH instead of a port."""
    repository_root_path = repository_config_path.parent
    if server_port is None:
        # systemd creates /run/genix owned by the service user (also writable under
        # ProtectSystem=strict) and removes it on stop, so no stale socket survives a restart.
        listen_lines = f"""# HAProxy on this host proxies to this socket (see SERVICE_SOCKET_PATH in configure_server.py).
RuntimeDirectory={SERVICE_RUNTIME_DIRECTORY_NAME}
RuntimeDirectoryMode=0755
Environment=SERVER_SOCKET={SERVICE_SOCKET_PATH}"""
    else:
        listen_lines = f"""# SERVER_PORT comes from server.port in config.toml and must match the port half of
# server.upstream, otherwise the edge host proxies to a port nothing is listening on.
Environment=SERVER_PORT={server_port}"""
    return f"""[Unit]
Description=Genix Backend Service
After=network.target

[Service]
Type=simple
User={runtime_username}
Group={runtime_username}
WorkingDirectory={SERVICE_INSTALL_DIRECTORY}
Environment=GENIX_CONFIG_FILE={repository_config_path}
Environment=GENIX_REPOSITORY_ROOT={repository_root_path}
{listen_lines}
# /var/lib/genix, owned by the service user and writable under ProtectSystem=strict: the backend
# writes [fareward].routes_file here at startup for the fareward SPOE agent.
StateDirectory={SERVICE_RUNTIME_DIRECTORY_NAME}
StateDirectoryMode=0755
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


def configure_systemd_units(runtime_username, repository_config_path, server_port):
    main_service_configuration_changed = write_unit_file(
        SYSTEMD_DIRECTORY / SERVICE_NAME,
        build_main_service_contents(runtime_username, repository_config_path, server_port),
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


def start_service():
    # The binary is installed before genix-restart.path is (re)started, so the watcher never sees
    # that first change. Start the service here instead of waiting for the next reboot. A backend
    # that cannot reach its database must not fail the whole configuration run, so this tolerates
    # a non-zero exit and points at journalctl.
    service_start_result = run_command(["systemctl", "restart", SERVICE_NAME], allow_failure=True)
    if service_start_result.returncode != 0:
        print_debug(
            f"WARNING: {SERVICE_NAME} did not start. The units are installed; "
            f"check 'journalctl -u {SERVICE_NAME} -n 50' for the reason."
        )
        return

    print_debug(f"{SERVICE_NAME} is running.")


def install_systemd_service(
    repository_config_path, server_port, binary_source=BINARY_SOURCE_AUTO
):
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    ensure_binary_directory(runtime_user_entry)
    provide_service_binary(repository_config_path.parent, runtime_user_entry, binary_source)
    systemd_configuration_changed = configure_systemd_units(
        runtime_username, repository_config_path, server_port
    )
    enable_units()
    reload_systemd_if_configuration_changed(systemd_configuration_changed)
    start_service()
    return runtime_username


def print_summary(
    execution_mode,
    repository_config_path,
    runtime_username,
    server_port,
    edge_settings,
    spoe_settings,
):
    print_debug("Configuration completed.")
    print_debug(f"Install mode: {execution_mode}")
    print_debug(f"Repository config path: {repository_config_path}")

    if runtime_username and not repository_config_path.exists():
        # The backend reads config.toml through GENIX_CONFIG_FILE and panics without it.
        print_debug(
            f"WARNING: {repository_config_path} does not exist. The service is installed but "
            "will fail to start until that file is in place."
        )

    if runtime_username:
        print_debug(f"Runtime user: {runtime_username}")
        binary_size_in_bytes = SERVICE_BINARY_PATH.stat().st_size if SERVICE_BINARY_PATH.is_file() else 0
        print_debug(f"Binary path: {SERVICE_BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
        if server_port is None:
            print_debug(f"Backend listen socket: {SERVICE_SOCKET_PATH}")
        else:
            print_debug(f"Backend listen port (server.port): {server_port}")
        print_debug(f"Main service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
        print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
        print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
        print_debug("Upload or replace the executable at the binary path to trigger an automatic restart.")
        print_debug("The service will also keep checking the install directory via its working directory.")

    if edge_settings:
        print_debug(f"Edge domain (server.domain): {edge_settings['domain']}")
        print_debug(f"Edge upstream: {edge_settings['upstream_scheme']}://{edge_settings['upstream_server']}")
        print_debug(f"HAProxy site file: {HAPROXY_SITES_DIRECTORY / (edge_settings['domain'] + '.cfg')}")
        print_debug(f"HAProxy configuration: {HAPROXY_CONFIGURATION_PATH} (access log: journalctl -u haproxy)")
        if spoe_settings:
            spoe_mode = "enforce" if spoe_settings["enforce"] else "shadow"
            print_debug(f"fareward SPOE agent: {spoe_settings['socket']} ({spoe_mode} mode)")
            print_debug("Run configure_fareward.py on this host too: the agent lives in the fareward daemon.")

    if execution_mode == EXECUTION_MODE_EDGE:
        print_debug("Run this script with mode 2 on the backend host so it listens on server.port.")
    if execution_mode == EXECUTION_MODE_SYSTEMD:
        print_debug("Run this script with mode 3 on the edge host so server.domain forwards here.")


def main():
    command_arguments = parse_command_arguments()
    require_root_execution()
    execution_mode = resolve_execution_mode(command_arguments.mode)
    repository_config_path = detect_repository_config_path()
    project_credentials = load_project_credentials(repository_config_path)

    configure_systemd = execution_mode in {EXECUTION_MODE_FULL, EXECUTION_MODE_SYSTEMD}
    configure_edge = execution_mode in {EXECUTION_MODE_FULL, EXECUTION_MODE_EDGE}

    # Resolve every credential the selected mode needs before touching the system, so anything
    # missing is asked for up front instead of leaving half of the host configured.
    prompted_values = {}
    server_port = None
    edge_settings = None
    spoe_settings = None

    # Only a split-host backend listens on a TCP port; in full mode it binds the Unix socket.
    if execution_mode == EXECUTION_MODE_SYSTEMD:
        server_port, prompted_server_port = extract_server_port(project_credentials)
        prompted_values.update(prompted_server_port)

    if configure_edge:
        edge_settings = extract_edge_settings(
            project_credentials, backend_is_local=execution_mode == EXECUTION_MODE_FULL
        )
        prompted_values.update(edge_settings["prompted_values"])

    # The agent's headers are only trusted from a local peer, so only full mode gets SPOE.
    if execution_mode == EXECUTION_MODE_FULL:
        spoe_settings = resolve_spoe_settings(project_credentials)

    persist_prompted_credentials(repository_config_path, prompted_values)

    runtime_username = detect_runtime_username()
    if configure_systemd:
        if repository_config_path.exists():
            sync_edge_trust_setting(project_credentials, repository_config_path, spoe_settings)
        runtime_username = install_systemd_service(
            repository_config_path, server_port, command_arguments.binary_source
        )

    if configure_edge:
        configure_api_edge(edge_settings, spoe_settings, process_group=runtime_username)

    print_summary(
        execution_mode,
        repository_config_path,
        runtime_username if configure_systemd else None,
        server_port,
        edge_settings,
        spoe_settings,
    )


if __name__ == "__main__":
    main()
