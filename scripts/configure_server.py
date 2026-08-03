#!/usr/bin/env python3

import json
import os
import platform
import pwd
import re
import shutil
import stat
import subprocess
import sys
from datetime import datetime
from pathlib import Path

SYSTEMD_DIRECTORY = Path("/etc/systemd/system")
SERVICE_INSTALL_DIRECTORY = Path("/usr/local/bin/genix")
SERVICE_BINARY_PATH = SERVICE_INSTALL_DIRECTORY / "genix_app"
SERVICE_NAME = "genix.service"
RESTART_SERVICE_NAME = "genix-restart.service"
RESTART_PATH_NAME = "genix-restart.path"
NGINX_CONFIGURATION_DIRECTORY = Path("/etc/nginx/conf.d")
LETSENCRYPT_DIRECTORY = Path("/etc/letsencrypt/live")

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

# The Nginx edge host and the backend host are usually two different machines: Nginx terminates
# TLS for NGINX_DOMAIN and forwards to NGINX_PROCESS, while the backend host only runs the
# systemd units and listens on SERVER_PORT. Each install mode configures one side of that pair.
EXECUTION_MODE_FULL = "full"
EXECUTION_MODE_SYSTEMD = "systemd"
EXECUTION_MODE_NGINX = "nginx"

EXECUTION_MODE_OPTIONS = [
    {
        "index": 1,
        "mode": EXECUTION_MODE_FULL,
        "label": "Full (systemd backend service + Nginx reverse proxy)",
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
        "mode": EXECUTION_MODE_NGINX,
        "label": "Only Nginx Proxy (edge host)",
        "aliases": {"nginx", "proxy", "edge"},
    },
]

CERTBOT_TLS_DIRECTIVES = {
    "ssl_certificate",
    "ssl_certificate_key",
    "ssl_trusted_certificate",
    "ssl_dhparam",
}


def print_debug(message_text):
    print(f"[*] {message_text}")


def fail_with_error(error_message):
    print(f"[!] {error_message}", file=sys.stderr)
    sys.exit(1)


def run_command(command_arguments, working_directory=None, stream_output=False, allow_failure=False):
    print_debug(f"Running command: {' '.join(command_arguments)}")

    if stream_output:
        # A compile can take minutes; let its progress and errors reach the terminal live.
        command_result = subprocess.run(command_arguments, text=True, cwd=working_directory)
        print_debug(f"Exit code: {command_result.returncode}")
        if command_result.returncode != 0 and not allow_failure:
            fail_with_error(f"Command failed: {' '.join(command_arguments)}")
        return command_result

    command_result = subprocess.run(
        command_arguments, text=True, capture_output=True, cwd=working_directory
    )
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


def detect_repository_credentials_path():
    # Prefer a repository root literally named "genix"; otherwise fall back to the directory that
    # holds this script's parent ("<root>/scripts/configure_server.py"), so a clone with a
    # different directory name still resolves. The file itself may not exist yet.
    script_path = Path(__file__).resolve()
    for parent_path in script_path.parents:
        if parent_path.name == "genix":
            repository_credentials_path = parent_path / "credentials.json"
            print_debug(f"Using repository credentials path: {repository_credentials_path}")
            return repository_credentials_path

    fallback_credentials_path = script_path.parents[1] / "credentials.json"
    print_debug(
        f"No repository root named 'genix' found. Using {fallback_credentials_path} instead."
    )
    return fallback_credentials_path


def load_project_credentials(repository_credentials_path):
    # A missing file is not fatal: every value this script needs can be typed in instead, which is
    # what an Nginx-only host without a full clone of the repository normally does.
    if not repository_credentials_path.exists():
        print_debug(f"No credentials.json at {repository_credentials_path}. Values will be requested.")
        return {}

    print_debug(f"Loading project credentials from {repository_credentials_path}")
    try:
        credentials_content = repository_credentials_path.read_text(encoding="utf-8")
    except OSError as read_error:
        fail_with_error(f"Could not read credentials.json: {read_error}")

    try:
        parsed_credentials = json.loads(credentials_content)
    except json.JSONDecodeError as parse_error:
        # Refuse to guess here: prompting would later overwrite a real but broken file.
        fail_with_error(f"Could not parse credentials.json: {parse_error}")

    if not isinstance(parsed_credentials, dict):
        fail_with_error("credentials.json must contain a JSON object.")

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


def resolve_execution_mode():
    if len(sys.argv) >= 2:
        selected_value = sys.argv[1].strip()
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
            "Pass 1 (full), 2 (systemd) or 3 (nginx)."
        )

    print_execution_mode_options()
    mode_option = match_execution_mode(input("Select the install mode: "))
    if not mode_option:
        fail_with_error("Invalid install mode selection.")

    print_debug(f"Selected install mode: {mode_option['label']}")
    return mode_option["mode"]


def validate_nginx_domain(raw_value):
    domain_value = raw_value.strip().rstrip(".").lower()
    if not domain_value:
        return None, "The domain cannot be empty."
    if not re.fullmatch(r"[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+", domain_value):
        return None, f"'{raw_value.strip()}' is not a valid domain name (expected e.g. genix-api-4.un.pe)."

    return domain_value, None


def validate_nginx_process(raw_value):
    # NGINX_PROCESS is normally a bare "host:port" pointing at the backend host, but a full
    # http:// or https:// upstream is accepted so an already-TLS backend can be reused as-is.
    process_value = raw_value.strip()
    if not process_value:
        return None, "The upstream cannot be empty."

    if "://" in process_value:
        upstream_scheme, _, host_and_port = process_value.partition("://")
        if upstream_scheme not in {"http", "https"}:
            return None, (
                f"Unsupported scheme '{upstream_scheme}://' in '{process_value}'. "
                "Use http://, https:// or a bare host:port."
            )
    else:
        host_and_port = process_value

    host_and_port = host_and_port.rstrip("/")
    if ":" not in host_and_port:
        return None, f"'{process_value}' must include the backend port (expected e.g. 100.64.0.2:14010)."

    upstream_host, _, upstream_port = host_and_port.rpartition(":")
    if not upstream_host:
        return None, f"'{process_value}' is missing the backend host."
    if not upstream_port.isdigit() or not 1 <= int(upstream_port) <= 65535:
        return None, f"'{upstream_port}' is not a valid port in '{process_value}'."

    return process_value, None


def validate_server_port(raw_value):
    port_value = raw_value.strip()
    if not port_value:
        return None, "The port cannot be empty."
    if not port_value.isdigit() or not 1 <= int(port_value) <= 65535:
        return None, f"'{port_value}' is not a valid TCP port (expected 1-65535)."

    return int(port_value), None


def resolve_credential_value(project_credentials, variable_name, prompt_text, validate_value):
    """Read one credential, asking for it on the terminal when it is absent or invalid.

    Returns (value, was_prompted) so main() can offer to persist whatever had to be typed in.
    """
    raw_value = project_credentials.get(variable_name)
    if raw_value is not None:
        validated_value, validation_error = validate_value(str(raw_value))
        if validated_value is not None:
            return validated_value, False
        print_debug(f"{variable_name} in credentials.json is unusable: {validation_error}")
    else:
        print_debug(f"{variable_name} is not set in credentials.json.")

    if not sys.stdin.isatty():
        fail_with_error(
            f"{variable_name} is missing or invalid and there is no interactive terminal to ask for it. "
            f"Add {variable_name} to credentials.json and run the script again."
        )

    while True:
        validated_value, validation_error = validate_value(input(f"{prompt_text}: "))
        if validated_value is not None:
            print_debug(f"Using {variable_name}={validated_value}")
            return validated_value, True

        print(f"[!] {validation_error}", file=sys.stderr)


def extract_nginx_settings(project_credentials):
    nginx_domain, domain_was_prompted = resolve_credential_value(
        project_credentials,
        "NGINX_DOMAIN",
        "Enter NGINX_DOMAIN, the public domain Nginx serves (e.g. genix-api-4.un.pe)",
        validate_nginx_domain,
    )
    nginx_process, process_was_prompted = resolve_credential_value(
        project_credentials,
        "NGINX_PROCESS",
        "Enter NGINX_PROCESS, the backend host:port Nginx forwards to (e.g. 100.64.0.2:14010)",
        validate_nginx_process,
    )

    backend_proxy_url = nginx_process if "://" in nginx_process else f"http://{nginx_process}"

    print_debug(f"Nginx domain: {nginx_domain}")
    print_debug(f"Nginx upstream: {backend_proxy_url}")
    return {
        "domain": nginx_domain,
        "process": nginx_process,
        "backend_proxy_url": backend_proxy_url,
        "prompted_values": {
            **({"NGINX_DOMAIN": nginx_domain} if domain_was_prompted else {}),
            **({"NGINX_PROCESS": nginx_process} if process_was_prompted else {}),
        },
    }


def extract_server_port(project_credentials):
    server_port, port_was_prompted = resolve_credential_value(
        project_credentials,
        "SERVER_PORT",
        "Enter SERVER_PORT, the port the backend listens on (e.g. 14010)",
        validate_server_port,
    )

    print_debug(f"Backend listen port: {server_port}")
    return server_port, {"SERVER_PORT": server_port} if port_was_prompted else {}


def warn_on_port_mismatch(server_port, nginx_settings):
    if server_port is None or not nginx_settings:
        return

    upstream_port = nginx_settings["process"].rstrip("/").rpartition(":")[2]
    if upstream_port.isdigit() and int(upstream_port) != server_port:
        print_debug(
            f"WARNING: SERVER_PORT ({server_port}) does not match the port in NGINX_PROCESS "
            f"({upstream_port}). Nginx will proxy to a port the backend is not listening on."
        )


def persist_prompted_credentials(repository_credentials_path, prompted_values):
    if not prompted_values:
        return

    described_values = ", ".join(f"{key}={value}" for key, value in prompted_values.items())
    print_debug(f"These values were typed in and are not stored yet: {described_values}")

    if not sys.stdin.isatty():
        print_debug("No interactive terminal available. Skipping credentials.json update.")
        return

    answer = input(f"Save them to {repository_credentials_path}? [Y/n]: ").strip().lower()
    if answer not in {"", "y", "yes"}:
        print_debug("Leaving credentials.json untouched.")
        return

    # Re-read instead of reusing the parsed dict so unrelated keys written by another process
    # between the load and this point survive the rewrite.
    credentials_file_already_existed = repository_credentials_path.exists()
    stored_credentials = load_project_credentials(repository_credentials_path)
    stored_credentials.update(prompted_values)

    try:
        repository_credentials_path.parent.mkdir(parents=True, exist_ok=True)
        repository_credentials_path.write_text(
            json.dumps(stored_credentials, indent=4, ensure_ascii=False) + "\n",
            encoding="utf-8",
        )
    except OSError as write_error:
        fail_with_error(f"Could not write credentials.json: {write_error}")

    if not credentials_file_already_existed:
        # A file created by root would be unreadable to the non-root service, so hand it to
        # whoever owns the repository directory — the account that cloned it.
        repository_directory_stat = repository_credentials_path.parent.stat()
        os.chown(
            repository_credentials_path,
            repository_directory_stat.st_uid,
            repository_directory_stat.st_gid,
        )
        os.chmod(repository_credentials_path, 0o600)

    print_debug(f"Saved {len(prompted_values)} value(s) to {repository_credentials_path}")


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


def find_prebuilt_binary(repository_root_path):
    go_architecture = resolve_go_architecture()
    candidate_binary_paths = [
        SERVICE_BINARY_PATH,
        repository_root_path / "tmp" / f"genix_app_linux_{go_architecture}",
        repository_root_path / "backend" / "genix_app",
        repository_root_path / "genix_app",
    ]

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


def provide_service_binary(repository_root_path, runtime_user_entry):
    """Put a runnable binary at SERVICE_BINARY_PATH: compile it, or find one, or fail."""
    remove_empty_placeholder_binary()

    backend_source_directory = detect_backend_source_directory(repository_root_path)
    if backend_source_directory:
        built_binary_path = compile_backend_binary(backend_source_directory, repository_root_path)
        install_service_binary(built_binary_path, runtime_user_entry)
        return

    print_debug("Falling back to a prebuilt binary because there is no backend source to compile.")
    prebuilt_binary_path = find_prebuilt_binary(repository_root_path)
    if not prebuilt_binary_path:
        fail_with_error(
            f"No backend source under {repository_root_path / 'backend'} and no prebuilt binary "
            f"found. Clone the repository with its backend folder, or place a compiled binary at "
            f"{SERVICE_BINARY_PATH}, and run the script again."
        )

    install_service_binary(prebuilt_binary_path, runtime_user_entry)


def ensure_nginx_is_installed():
    nginx_binary_path = shutil.which("nginx")
    if not nginx_binary_path:
        fail_with_error("Nginx is not installed or not available in PATH.")

    if not NGINX_CONFIGURATION_DIRECTORY.exists():
        fail_with_error(f"Nginx configuration directory not found: {NGINX_CONFIGURATION_DIRECTORY}")

    print_debug(f"Detected Nginx binary at {nginx_binary_path}")


def extract_existing_certbot_tls_lines(existing_nginx_configuration_contents):
    preserved_tls_lines = []
    seen_directives = set()

    if not existing_nginx_configuration_contents:
        return preserved_tls_lines

    for raw_line in existing_nginx_configuration_contents.splitlines():
        stripped_line = raw_line.strip()
        if not stripped_line or stripped_line.startswith("#"):
            continue

        directive_match = re.match(r"^([A-Za-z0-9_]+)\s+(.+?);(?:\s*#.*)?$", stripped_line)
        if not directive_match:
            continue

        directive_name = directive_match.group(1)
        directive_value = directive_match.group(2)
        is_certbot_include = (
            directive_name == "include"
            and (
                "letsencrypt" in directive_value
                or "certbot" in directive_value.lower()
            )
        )
        should_preserve_directive = directive_name in CERTBOT_TLS_DIRECTIVES or is_certbot_include
        if not should_preserve_directive:
            continue

        # Keep the first value per directive so repeated stale lines do not multiply.
        directive_key = f"{directive_name}:{directive_value}"
        if directive_key in seen_directives:
            continue
        seen_directives.add(directive_key)
        preserved_tls_lines.append(f"    {stripped_line}")

    if preserved_tls_lines:
        print_debug("Preserving TLS directives from existing Nginx config:")
        for preserved_tls_line in preserved_tls_lines:
            print_debug(preserved_tls_line.strip())

    return preserved_tls_lines


def build_tls_directive_lines(endpoint_hostname, existing_nginx_configuration_contents):
    certificate_directory = LETSENCRYPT_DIRECTORY / endpoint_hostname
    certificate_fullchain_path = certificate_directory / "fullchain.pem"
    certificate_private_key_path = certificate_directory / "privkey.pem"

    preserved_tls_lines = extract_existing_certbot_tls_lines(existing_nginx_configuration_contents)
    has_preserved_certificate = any(line.strip().startswith("ssl_certificate ") for line in preserved_tls_lines)
    has_preserved_certificate_key = any(line.strip().startswith("ssl_certificate_key ") for line in preserved_tls_lines)
    if has_preserved_certificate and has_preserved_certificate_key:
        return preserved_tls_lines

    if certificate_fullchain_path.exists() and certificate_private_key_path.exists():
        print_debug(f"Detected TLS certificates for {endpoint_hostname} at {certificate_directory}")
        return [
            f"    ssl_certificate {certificate_fullchain_path};",
            f"    ssl_certificate_key {certificate_private_key_path};",
        ]

    return []


def build_http3_nginx_configuration(
    endpoint_hostname,
    backend_proxy_url,
    existing_nginx_configuration_contents="",
):
    tls_directive_lines = build_tls_directive_lines(
        endpoint_hostname,
        existing_nginx_configuration_contents,
    )

    if tls_directive_lines:
        tls_directives = "\n".join(tls_directive_lines)
        return f"""# Map block to handle 0-RTT security (prevents replay attacks on POST/PUT)
map $ssl_early_data $is_early_data {{
    "~on" 1;
    default 0;
}}

# WebSocket upstreams require HTTP/1.1 plus an explicit Upgrade tunnel.
map $http_upgrade $connection_upgrade {{
    default upgrade;
    "" close;
}}

server {{
    # Standard TCP and HTTP/3 UDP listeners for TLS-enabled deployments.
    listen 443 quic reuseport;
    listen 443 ssl;
    listen [::]:443 quic reuseport;
    listen [::]:443 ssl;

    server_name {endpoint_hostname};

{tls_directives}

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
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_pass {backend_proxy_url};

        proxy_pass_header Server;
        server_tokens off;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        # Agent WebSockets can sit idle while the user thinks between messages.
        proxy_read_timeout 3600s;
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }}
}}
"""

    print_debug(
        f"No TLS certificates found for {endpoint_hostname}. Writing HTTP-only reverse proxy config."
    )
    return f"""# WebSocket upstreams require HTTP/1.1 plus an explicit Upgrade tunnel.
map $http_upgrade $connection_upgrade {{
    default upgrade;
    "" close;
}}

server {{
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
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_pass {backend_proxy_url};

        proxy_pass_header Server;
        server_tokens off;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        # Agent WebSockets can sit idle while the user thinks between messages.
        proxy_read_timeout 3600s;
        proxy_buffering on;
        proxy_buffer_size 16k;
        proxy_buffers 4 16k;
    }}
}}
"""


def configure_nginx_reverse_proxy(nginx_settings):
    ensure_nginx_is_installed()

    endpoint_hostname = nginx_settings["domain"]
    nginx_configuration_path = NGINX_CONFIGURATION_DIRECTORY / f"{endpoint_hostname}.conf"

    existing_nginx_configuration_contents = None
    if nginx_configuration_path.exists():
        existing_nginx_configuration_contents = nginx_configuration_path.read_text(encoding="utf-8")

    nginx_configuration_contents = build_http3_nginx_configuration(
        endpoint_hostname,
        nginx_settings["backend_proxy_url"],
        existing_nginx_configuration_contents or "",
    )

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


def build_main_service_contents(runtime_username, repository_credentials_path, server_port):
    repository_root_path = repository_credentials_path.parent
    return f"""[Unit]
Description=Genix Backend Service
After=network.target

[Service]
Type=simple
User={runtime_username}
Group={runtime_username}
WorkingDirectory={SERVICE_INSTALL_DIRECTORY}
Environment=GENIX_CREDENTIALS_FILE={repository_credentials_path}
Environment=GENIX_REPOSITORY_ROOT={repository_root_path}
# SERVER_PORT comes from credentials.json and must match the port half of NGINX_PROCESS,
# otherwise the Nginx host proxies to a port nothing is listening on.
Environment=SERVER_PORT={server_port}
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


def configure_systemd_units(runtime_username, repository_credentials_path, server_port):
    main_service_configuration_changed = write_unit_file(
        SYSTEMD_DIRECTORY / SERVICE_NAME,
        build_main_service_contents(runtime_username, repository_credentials_path, server_port),
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


def install_systemd_service(repository_credentials_path, server_port):
    runtime_username = detect_runtime_username()
    runtime_user_entry = resolve_runtime_user(runtime_username)
    ensure_binary_directory(runtime_user_entry)
    provide_service_binary(repository_credentials_path.parent, runtime_user_entry)
    systemd_configuration_changed = configure_systemd_units(
        runtime_username, repository_credentials_path, server_port
    )
    enable_units()
    reload_systemd_if_configuration_changed(systemd_configuration_changed)
    start_service()
    return runtime_username


def print_summary(
    execution_mode,
    repository_credentials_path,
    runtime_username,
    server_port,
    nginx_settings,
):
    print_debug("Configuration completed.")
    print_debug(f"Install mode: {execution_mode}")
    print_debug(f"Repository credentials path: {repository_credentials_path}")

    if runtime_username and not repository_credentials_path.exists():
        # The backend reads credentials.json through GENIX_CREDENTIALS_FILE and panics without it.
        print_debug(
            f"WARNING: {repository_credentials_path} does not exist. The service is installed but "
            "will fail to start until that file is in place."
        )

    if runtime_username:
        print_debug(f"Runtime user: {runtime_username}")
        binary_size_in_bytes = SERVICE_BINARY_PATH.stat().st_size if SERVICE_BINARY_PATH.is_file() else 0
        print_debug(f"Binary path: {SERVICE_BINARY_PATH} ({binary_size_in_bytes / 1_048_576:.1f} MiB)")
        print_debug(f"Backend listen port (SERVER_PORT): {server_port}")
        print_debug(f"Main service unit: {SYSTEMD_DIRECTORY / SERVICE_NAME}")
        print_debug(f"Path watcher unit: {SYSTEMD_DIRECTORY / RESTART_PATH_NAME}")
        print_debug(f"Restart helper unit: {SYSTEMD_DIRECTORY / RESTART_SERVICE_NAME}")
        print_debug("Upload or replace the executable at the binary path to trigger an automatic restart.")
        print_debug("The service will also keep checking the install directory via its working directory.")

    if nginx_settings:
        print_debug(f"Nginx domain (NGINX_DOMAIN): {nginx_settings['domain']}")
        print_debug(f"Nginx upstream (NGINX_PROCESS): {nginx_settings['backend_proxy_url']}")
        print_debug(
            f"Nginx config file: {NGINX_CONFIGURATION_DIRECTORY / (nginx_settings['domain'] + '.conf')}"
        )

    if execution_mode == EXECUTION_MODE_NGINX:
        print_debug("Run this script with mode 2 on the backend host so it listens on SERVER_PORT.")
    if execution_mode == EXECUTION_MODE_SYSTEMD:
        print_debug("Run this script with mode 3 on the Nginx host so NGINX_DOMAIN forwards here.")


def main():
    require_root_execution()
    execution_mode = resolve_execution_mode()
    repository_credentials_path = detect_repository_credentials_path()
    project_credentials = load_project_credentials(repository_credentials_path)

    configure_systemd = execution_mode in {EXECUTION_MODE_FULL, EXECUTION_MODE_SYSTEMD}
    configure_nginx = execution_mode in {EXECUTION_MODE_FULL, EXECUTION_MODE_NGINX}

    # Resolve every credential the selected mode needs before touching the system, so anything
    # missing is asked for up front instead of leaving half of the host configured.
    prompted_values = {}
    server_port = None
    nginx_settings = None

    if configure_systemd:
        server_port, prompted_server_port = extract_server_port(project_credentials)
        prompted_values.update(prompted_server_port)

    if configure_nginx:
        nginx_settings = extract_nginx_settings(project_credentials)
        prompted_values.update(nginx_settings["prompted_values"])

    warn_on_port_mismatch(server_port, nginx_settings)
    persist_prompted_credentials(repository_credentials_path, prompted_values)

    runtime_username = None
    if configure_systemd:
        runtime_username = install_systemd_service(repository_credentials_path, server_port)

    if configure_nginx:
        configure_nginx_reverse_proxy(nginx_settings)

    print_summary(
        execution_mode,
        repository_credentials_path,
        runtime_username,
        server_port,
        nginx_settings,
    )


if __name__ == "__main__":
    main()
