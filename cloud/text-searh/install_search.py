#!/usr/bin/env python3
"""Install and configure the GenixSearch backend from a precompiled binary.

GenixSearch is a compact, lossy, ranked search backend optimised for
Spanish product names and short commercial text. See
https://gitea.local/genix-search for protocol and configuration docs.

Looks for any file matching ``genixsearch*`` in the binary directory
(defaults to this script's directory), then:

  1. Copies it to /usr/local/bin/genixsearch (chmod 0755)
  2. Creates a dedicated system user 'genix-search'
  3. Creates the data directory /var/lib/genix-search/kv
  4. Reads ``./credentials.json`` (in the invoking shell's working
     directory). If it already contains ``GENIXSEARCH_PASSWORD``, that
     value is reused. Otherwise a new lowercase+digit password is
     generated and written back under ``GENIXSEARCH_PASSWORD``. The Go
     backend reads it as ``core.Env.GENIXSEARCH_PASSWORD`` (alongside
     ``GENIXSEARCH_HOST`` / ``GENIXSEARCH_PORT``).
  5. Writes /etc/genix-search/config.cfg with that password and the
     bind address/port.
  6. Writes a systemd unit named ``genix-search.service``, starts and
     enables the service.

Run as root:

    sudo python3 install_search.py [--binary-dir DIR] [--bind 0.0.0.0] [--port 14446]
"""
from __future__ import annotations

import argparse
import json
import os
import secrets
import shutil
import string
import subprocess
import sys
from pathlib import Path

SERVICE_NAME = "genix-search"
BINARY_NAME = "genixsearch"
INSTALL_PATH = Path("/usr/local/bin") / BINARY_NAME
DATA_DIR = Path("/var/lib") / SERVICE_NAME
KV_DIR = DATA_DIR / "kv"
CONFIG_DIR = Path("/etc") / SERVICE_NAME
CONFIG_FILE = CONFIG_DIR / "config.cfg"
SERVICE_FILE = Path("/etc/systemd/system") / f"{SERVICE_NAME}.service"
SERVICE_USER = SERVICE_NAME

CREDENTIALS_FILENAME = "credentials.json"
# Matches the Go backend's core.Env.GENIXSEARCH_PASSWORD field; the
# struct is populated from credentials.json via json.Unmarshal, which
# uses the field name as the JSON key.
CREDENTIALS_KEY = "GENIXSEARCH_PASSWORD"

_SECRET_ALPHABET = string.ascii_lowercase + string.digits


def generate_password(length: int = 64) -> str:
    """Generate a password using only lowercase letters and digits."""
    return "".join(secrets.choice(_SECRET_ALPHABET) for _ in range(length))


def info(msg: str) -> None:
    print(f"  -> {msg}")


def die(msg: str) -> None:
    sys.exit(f"error: {msg}")


def require_root() -> None:
    if os.geteuid() != 0:
        die("this script must be run as root (use sudo)")


def find_binary(search_dir: Path) -> Path:
    candidates = [
        p for p in sorted(search_dir.glob(f"{BINARY_NAME}*"))
        if p.is_file() and p.suffix not in {".py", ".md", ".txt", ".sha256", ".asc", ".cfg"}
    ]
    if not candidates:
        die(f"no {BINARY_NAME}* binary found in {search_dir}")
    if len(candidates) > 1:
        info(f"multiple binaries found: {[c.name for c in candidates]}")
        info(f"using: {candidates[0].name}")
    return candidates[0]


def ensure_user() -> None:
    if subprocess.run(["id", SERVICE_USER], capture_output=True).returncode == 0:
        return
    subprocess.run(
        [
            "useradd",
            "--system",
            "--home-dir", str(DATA_DIR),
            "--shell", "/usr/sbin/nologin",
            SERVICE_USER,
        ],
        check=True,
    )


def install_binary(binary: Path) -> None:
    INSTALL_PATH.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(binary, INSTALL_PATH)
    INSTALL_PATH.chmod(0o755)
    shutil.chown(INSTALL_PATH, user="root", group="root")


def ensure_data_dir() -> None:
    for d in (DATA_DIR, KV_DIR):
        d.mkdir(parents=True, exist_ok=True)
        shutil.chown(d, user=SERVICE_USER, group=SERVICE_USER)
        d.chmod(0o750)


def credentials_path() -> Path:
    """Locate credentials.json in the invoking user's working directory."""
    return Path.cwd() / CREDENTIALS_FILENAME


def load_credentials(path: Path) -> dict:
    if not path.exists():
        return {}
    try:
        data = json.loads(path.read_text())
    except json.JSONDecodeError as exc:
        die(f"{path} is not valid JSON: {exc}")
    if not isinstance(data, dict):
        die(f"{path} must contain a JSON object at the top level")
    return data


def save_credentials(path: Path, data: dict) -> None:
    path.write_text(json.dumps(data, indent=2) + "\n")
    path.chmod(0o600)
    sudo_uid = os.environ.get("SUDO_UID")
    sudo_gid = os.environ.get("SUDO_GID")
    if sudo_uid and sudo_gid:
        try:
            os.chown(path, int(sudo_uid), int(sudo_gid))
        except (OSError, ValueError):
            pass


def resolve_password() -> tuple[str, Path, str]:
    """Return (password, credentials_path, source-label).

    Uses an existing GENIXSEARCH_PASSWORD from credentials.json if present,
    otherwise generates one and writes it back.
    """
    path = credentials_path()
    creds = load_credentials(path)

    existing = creds.get(CREDENTIALS_KEY)
    if isinstance(existing, str) and existing.strip():
        return existing.strip(), path, f"reused from {path}"

    password = generate_password()
    creds[CREDENTIALS_KEY] = password
    save_credentials(path, creds)
    return password, path, f"generated and written to {path}"


def write_config(password: str, bind_addr: str, port: int) -> None:
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    content = f"""# GenixSearch configuration file
# Reference: CONFIGURATION.md in the genix-search repository.

[server]
log_level = "info"

[channel]
inet = "{bind_addr}:{port}"
tcp_timeout = 300
auth_password = "{password}"

[channel.search]
query_limit_default = 10
query_limit_maximum = 100

[store.kv]
path = "{KV_DIR}/"

pool.inactive_after = 1800

database.flush_after = 900
database.compress = true
database.parallelism = 2
database.max_files = 100
database.max_compactions = 1
database.max_flushes = 1
database.write_buffer = 16384
database.write_ahead_log = true
"""
    CONFIG_FILE.write_text(content)
    CONFIG_FILE.chmod(0o640)
    shutil.chown(CONFIG_FILE, user="root", group=SERVICE_USER)


def write_service() -> None:
    unit = f"""[Unit]
Description=GenixSearch backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={SERVICE_USER}
Group={SERVICE_USER}
ExecStart={INSTALL_PATH} -c {CONFIG_FILE}
Restart=on-failure
RestartSec=3
LimitNOFILE=65536

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths={DATA_DIR}
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
"""
    SERVICE_FILE.write_text(unit)
    SERVICE_FILE.chmod(0o644)


def systemctl(*args: str) -> None:
    subprocess.run(["systemctl", *args], check=True)


def reload_and_start() -> None:
    systemctl("daemon-reload")
    systemctl("enable", f"{SERVICE_NAME}.service")
    systemctl("restart", f"{SERVICE_NAME}.service")


def main() -> None:
    parser = argparse.ArgumentParser(description="Install GenixSearch backend from a precompiled binary.")
    parser.add_argument(
        "--binary-dir",
        default=str(Path(__file__).resolve().parent),
        help=f"directory containing {BINARY_NAME}* (default: script directory)",
    )
    parser.add_argument("--bind", default="0.0.0.0", help="bind address (default 0.0.0.0 — listens on all interfaces)")
    parser.add_argument("--port", type=int, default=14446, help="bind port (default 14446)")
    parser.add_argument("--no-service", action="store_true", help="skip systemd unit installation")
    args = parser.parse_args()

    require_root()

    binary = find_binary(Path(args.binary_dir))
    info(f"binary: {binary}")

    ensure_user()
    info(f"service user: {SERVICE_USER}")

    install_binary(binary)
    info(f"installed: {INSTALL_PATH}")

    ensure_data_dir()
    info(f"data dir:  {DATA_DIR}")

    password, creds_path, password_source = resolve_password()
    info(f"{CREDENTIALS_KEY}: {password_source}")

    write_config(password, args.bind, args.port)
    info(f"config:    {CONFIG_FILE}")

    if args.no_service:
        print()
        print("Skipped systemd unit. Run manually with:")
        print(f"  {INSTALL_PATH} -c {CONFIG_FILE}")
        return

    write_service()
    info(f"unit:      {SERVICE_FILE}")

    reload_and_start()
    info(f"{SERVICE_NAME}.service started and enabled at boot")

    print()
    print("==================== GenixSearch ====================")
    print(f"  Channel:            {args.bind}:{args.port}")
    print(f"  {CREDENTIALS_KEY}:     {password}")
    print(f"  config file:        {CONFIG_FILE}")
    print(f"  credentials file:   {creds_path}")
    print(f"  service:            systemctl status {SERVICE_NAME}")
    print("=====================================================")


if __name__ == "__main__":
    main()
