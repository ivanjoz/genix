#!/usr/bin/env python3
"""Configure a VPS from this machine: build the binaries here, upload them, run configure.py there.

    python3 scripts/configure_remote.py 123                       # the only [[servers]] entry
    python3 scripts/configure_remote.py 23 --server 150.136.42.240 --backend-mode 1

The selection takes the component digits of configure.py (1 database, 2 backend, 3 fareward); there
is no binary-source digit because the binaries always come from this checkout. That is the point of
running from here: what is on disk compiles, unpushed submodule commits included, and the VPS needs
no Go or Rust toolchain.

The server is a [[servers]] entry of config.toml. Its `config` names the server's own config file,
uploaded as the VPS's config.toml: the server needs values this machine does not (its database on a
Unix socket, providers.backend, ...). `repository` is the checkout the scripts run from, cloned on
first use, default /home/<user>/genix.

The remote user needs passwordless sudo (Oracle and AWS images give it to their default user):
configure.py installs packages and writes systemd units, nginx and firewall rules as root.
"""

import argparse
import os
import shlex
import subprocess
import sys
import tomllib
from datetime import datetime
from pathlib import Path

PROJECT_ROOT_DIRECTORY = Path(__file__).resolve().parent.parent
LOCAL_CONFIG_PATH = PROJECT_ROOT_DIRECTORY / "config.toml"
BUILD_OUTPUT_DIRECTORY = PROJECT_ROOT_DIRECTORY / "tmp" / "remote"
REPOSITORY_REMOTE = "https://github.com/ivanjoz/genix.git"

COMPONENT_DIGITS = {"1", "2", "3"}
# Same tags and CPU generation as scripts/deploy_vps.go (backendBuildTags, arm64Generation): LSE
# atomics on Ampere A1 / Graviton2 and later.
BACKEND_BUILD_TAGS = "lambda.norpc"
GOARM64_BY_ARCHITECTURE = {"arm64": "v8.2"}
# fareward links glibc dynamically: target the oldest glibc of a supported server (Ubuntu 24.04).
FAREWARD_TARGET_BY_ARCHITECTURE = {
    "arm64": "aarch64-unknown-linux-gnu",
    "amd64": "x86_64-unknown-linux-gnu",
}
FAREWARD_GLIBC_VERSION = "2.39"
UNAME_MACHINE_BY_ARCHITECTURE = {"arm64": {"aarch64", "arm64"}, "amd64": {"x86_64", "amd64"}}


def run_local(command_arguments, working_directory=None, environment=None):
    print(f"[*] Running: {' '.join(command_arguments)}")
    try:
        subprocess.run(command_arguments, cwd=working_directory, env=environment, check=True)
    except FileNotFoundError:
        raise SystemExit(f"[!] {command_arguments[0]} is not installed on this machine.") from None
    except subprocess.CalledProcessError as command_error:
        raise SystemExit(f"[!] {command_arguments[0]} stopped with exit code {command_error.returncode}.") from None


def select_server(server_host):
    """The [[servers]] entry to configure: the one named by --server, or the only one there is."""
    servers = tomllib.loads(LOCAL_CONFIG_PATH.read_text(encoding="utf-8")).get("servers", [])
    if server_host:
        matching_servers = [server for server in servers if server.get("host") == server_host]
    else:
        matching_servers = servers if len(servers) == 1 else []
    if len(matching_servers) != 1:
        known_hosts = ", ".join(server.get("host", "?") for server in servers) or "none"
        raise SystemExit(f"[!] Name the server with --server. [[servers]] in config.toml: {known_hosts}.")

    server = matching_servers[0]
    for required_key in ("host", "user", "key", "arch", "config"):
        if not server.get(required_key):
            raise SystemExit(f"[!] The [[servers]] entry for {server.get('host')} has no `{required_key}`.")
    if server["arch"] not in FAREWARD_TARGET_BY_ARCHITECTURE:
        raise SystemExit(f"[!] Unsupported arch {server['arch']!r}: use arm64 or amd64.")
    server_config_path = PROJECT_ROOT_DIRECTORY / server["config"]
    if not server_config_path.is_file():
        raise SystemExit(f"[!] The server's config file {server_config_path} does not exist.")
    return server, server_config_path


def build_backend(architecture):
    output_path = BUILD_OUTPUT_DIRECTORY / f"genix_app_linux_{architecture}"
    build_environment = {**os.environ, "GOOS": "linux", "GOARCH": architecture, "CGO_ENABLED": "0"}
    if architecture in GOARM64_BY_ARCHITECTURE:
        build_environment["GOARM64"] = GOARM64_BY_ARCHITECTURE[architecture]
    build_date = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    run_local(
        ["go", "build", "-tags", BACKEND_BUILD_TAGS, "-ldflags", f"-s -w -X 'app/core.BuildDate={build_date}'",
         "-o", str(output_path), "."],
        working_directory=PROJECT_ROOT_DIRECTORY / "backend",
        environment=build_environment,
    )
    return output_path


def build_fareward(architecture):
    """cargo zigbuild cross-links against an old glibc without a cross toolchain on this machine."""
    rust_target = FAREWARD_TARGET_BY_ARCHITECTURE[architecture]
    fareward_directory = PROJECT_ROOT_DIRECTORY / "fareward"
    # The toolchain fareward pins (rust-toolchain.toml) needs the target's standard library.
    run_local(["rustup", "target", "add", rust_target], working_directory=fareward_directory)
    run_local(
        ["cargo", "zigbuild", "--release", "--target", f"{rust_target}.{FAREWARD_GLIBC_VERSION}"],
        working_directory=fareward_directory,
    )
    built_path = fareward_directory / "target" / rust_target / "release" / "fareward"
    output_path = BUILD_OUTPUT_DIRECTORY / f"fareward_linux_{architecture}"
    output_path.write_bytes(built_path.read_bytes())
    return output_path


class RemoteHost:
    def __init__(self, server):
        self.destination = f"{server['user']}@{server['host']}"
        self.ssh_options = ["-i", str(Path(server["key"]).expanduser()), "-o", "ServerAliveInterval=30"]

    def run(self, remote_command, interactive=False):
        """Run a shell command on the server. interactive gives it a terminal for configure.py's prompts."""
        ssh_command = ["ssh", *(["-t"] if interactive else []), *self.ssh_options, self.destination, remote_command]
        print(f"[*] {self.destination}$ {remote_command}")
        command_result = subprocess.run(ssh_command)
        if command_result.returncode != 0:
            raise SystemExit(f"[!] The remote command stopped with exit code {command_result.returncode}.")

    def read(self, remote_command):
        ssh_command = ["ssh", *self.ssh_options, "-o", "BatchMode=yes", self.destination, remote_command]
        return subprocess.run(ssh_command, text=True, capture_output=True, check=True).stdout.strip()

    def upload(self, local_paths, remote_target):
        """rsync compressed, so the ~30 MB backend goes over the wire as ~8 MB. A remote_target
        ending in / is a directory; otherwise it is the name the single file gets there."""
        rsync_shell = " ".join(["ssh", *map(shlex.quote, self.ssh_options)])
        uploads_a_single_file = not remote_target.endswith("/")
        # A single file is the config: 0600, since it holds every secret.
        file_mode_options = ["--chmod=F600"] if uploads_a_single_file else []
        run_local([
            "rsync", "-az", "--exclude=__pycache__", *file_mode_options,
            "-e", rsync_shell, *map(str, local_paths), f"{self.destination}:{remote_target}",
        ])


def main():
    argument_parser = argparse.ArgumentParser(description="Configure a [[servers]] VPS from this machine.")
    argument_parser.add_argument("selection", help="component digits: 1 database, 2 backend, 3 fareward")
    argument_parser.add_argument("--server", help="the [[servers]] host to configure (default: the only one)")
    argument_parser.add_argument(
        "--backend-mode", choices=("1", "2", "3"), default="1",
        help="1 full (default), 2 only the systemd service, 3 only the nginx proxy",
    )
    command_arguments = argument_parser.parse_args()

    selected_components = set(command_arguments.selection)
    if not selected_components or not selected_components <= COMPONENT_DIGITS:
        raise SystemExit("[!] The selection takes only the digits 1, 2 and 3, for example 123.")

    server, server_config_path = select_server(command_arguments.server)
    architecture = server["arch"]
    remote_host = RemoteHost(server)
    repository_path = server.get("repository") or f"/home/{server['user']}/genix"

    # Checked before compiling anything: a wrong arch here is a binary that cannot execute there.
    remote_machine = remote_host.read("uname -m")
    if remote_machine not in UNAME_MACHINE_BY_ARCHITECTURE[architecture]:
        raise SystemExit(f"[!] [[servers]] says arch = {architecture}, but the server is {remote_machine}.")

    BUILD_OUTPUT_DIRECTORY.mkdir(parents=True, exist_ok=True)
    built_binaries = []
    if "2" in selected_components and command_arguments.backend_mode != "3":
        built_binaries.append(build_backend(architecture))
    if "3" in selected_components:
        built_binaries.append(build_fareward(architecture))

    # The checkout supplies what the backend reads at runtime (GENIX_REPOSITORY_ROOT); this
    # machine's scripts replace its own, so the configuration that runs is the one on disk here.
    quoted_repository = shlex.quote(repository_path)
    remote_host.run(
        # rsync must be on both ends; cloud images often ship git without it.
        f"for tool in git rsync; do command -v $tool >/dev/null || "
        f"sudo DEBIAN_FRONTEND=noninteractive apt-get install -y $tool || sudo dnf install -y $tool; done; "
        f"[ -d {quoted_repository}/.git ] || git clone --depth 1 {REPOSITORY_REMOTE} {quoted_repository}; "
        f"mkdir -p {quoted_repository}/tmp"
    )
    scripts_directory = PROJECT_ROOT_DIRECTORY / "scripts"
    remote_host.upload([scripts_directory / "configure.py", scripts_directory / "configure"], f"{repository_path}/scripts/")
    # 0600: it holds every secret, and the services read it as the remote user.
    remote_host.upload([server_config_path], f"{repository_path}/config.toml")
    if built_binaries:
        remote_host.upload(built_binaries, f"{repository_path}/tmp/")

    # 8 + --staged-binaries: install the binaries just uploaded to tmp/, download nothing.
    remote_selection = "".join(sorted(selected_components)) + "8"
    remote_host.run(
        f"cd {quoted_repository} && sudo python3 scripts/configure.py {remote_selection} "
        f"--staged-binaries --backend-mode {command_arguments.backend_mode}",
        interactive=True,
    )
    print(f"[+] {server['host']} configured with {remote_selection} from this checkout.")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit("\n[!] Configuration cancelled.")
