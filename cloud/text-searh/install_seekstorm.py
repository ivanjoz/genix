#!/usr/bin/env python3
"""Install and configure SeekStorm server from a precompiled binary.

Looks for any file matching ``seekstorm_server*`` in the binary directory
(defaults to this script's directory), then:

  1. Copies it to /usr/local/bin/seekstorm_server (chmod 0755)
  2. Creates a dedicated system user 'seekstorm'
  3. Creates the data directory /var/lib/seekstorm
  4. Reads ``./credentials.json`` (in the invoking shell's working directory).
     If it already contains ``SEEKSTORM_APIKEY``, that value is used.
     Otherwise a new lowercase+digit key is generated and written to
     ``credentials.json`` under ``SEEKSTORM_APIKEY``.
  5. Writes that value as ``MASTER_KEY_SECRET`` in the systemd EnvironmentFile
  6. Writes a systemd unit, starts and enables the service.

Run as root:

    sudo python3 install_seekstorm.py [--binary-dir DIR] [--bind 127.0.0.1] [--port 8080]
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

BINARY_NAME = "seekstorm_server"
INSTALL_PATH = Path("/usr/local/bin/seekstorm_server")
DATA_DIR = Path("/var/lib/seekstorm")
ENV_DIR = Path("/etc/seekstorm")
ENV_FILE = ENV_DIR / "seekstorm.env"
SERVICE_FILE = Path("/etc/systemd/system/seekstorm.service")
SERVICE_USER = "seekstorm"

CREDENTIALS_FILENAME = "credentials.json"
CREDENTIALS_KEY = "SEEKSTORM_APIKEY"

_SECRET_ALPHABET = string.ascii_lowercase + string.digits


def generate_apikey(length: int = 64) -> str:
    """Generate a key using only lowercase letters and digits."""
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
        p for p in sorted(search_dir.glob("seekstorm_server*"))
        if p.is_file() and p.suffix not in {".py", ".md", ".txt", ".sha256", ".asc"}
    ]
    if not candidates:
        die(f"no seekstorm_server* binary found in {search_dir}")
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
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    shutil.chown(DATA_DIR, user=SERVICE_USER, group=SERVICE_USER)
    DATA_DIR.chmod(0o750)


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
    # When run via sudo, hand the file back to the invoking user so they can
    # read/edit it without elevated privileges.
    sudo_uid = os.environ.get("SUDO_UID")
    sudo_gid = os.environ.get("SUDO_GID")
    if sudo_uid and sudo_gid:
        try:
            os.chown(path, int(sudo_uid), int(sudo_gid))
        except (OSError, ValueError):
            pass


def resolve_apikey() -> tuple[str, Path, str]:
    """Return (apikey, credentials_path, source-label).

    Uses an existing SEEKSTORM_APIKEY from credentials.json if present,
    otherwise generates one and writes it back.
    """
    path = credentials_path()
    creds = load_credentials(path)

    existing = creds.get(CREDENTIALS_KEY)
    if isinstance(existing, str) and existing.strip():
        return existing.strip(), path, f"reused from {path}"

    apikey = generate_apikey()
    creds[CREDENTIALS_KEY] = apikey
    save_credentials(path, creds)
    return apikey, path, f"generated and written to {path}"


def write_env_file(apikey: str, bind_addr: str, port: int) -> None:
    ENV_DIR.mkdir(parents=True, exist_ok=True)
    content = (
        f'MASTER_KEY_SECRET="{apikey}"\n'
        f'LOCAL_IP="{bind_addr}"\n'
        f'LOCAL_PORT="{port}"\n'
        f'INDEX_PATH="{DATA_DIR}"\n'
    )
    ENV_FILE.write_text(content)
    ENV_FILE.chmod(0o640)
    shutil.chown(ENV_FILE, user="root", group=SERVICE_USER)


def write_service() -> None:
    unit = f"""[Unit]
Description=SeekStorm search server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={SERVICE_USER}
Group={SERVICE_USER}
EnvironmentFile={ENV_FILE}
ExecStart={INSTALL_PATH} local_ip=${{LOCAL_IP}} local_port=${{LOCAL_PORT}} index_path=${{INDEX_PATH}}
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
    systemctl("enable", "seekstorm.service")
    systemctl("restart", "seekstorm.service")


def main() -> None:
    parser = argparse.ArgumentParser(description="Install SeekStorm server from a precompiled binary.")
    parser.add_argument(
        "--binary-dir",
        default=str(Path(__file__).resolve().parent),
        help="directory containing seekstorm_server* (default: script directory)",
    )
    parser.add_argument("--bind", default="127.0.0.1", help="bind address (default 127.0.0.1)")
    parser.add_argument("--port", type=int, default=14445, help="bind port (default 14445)")
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

    apikey, creds_path, apikey_source = resolve_apikey()
    info(f"{CREDENTIALS_KEY}: {apikey_source}")

    write_env_file(apikey, args.bind, args.port)
    info(f"env file:  {ENV_FILE}")

    if args.no_service:
        print()
        print("Skipped systemd unit. Run manually with:")
        print(
            f"  MASTER_KEY_SECRET='{apikey}' {INSTALL_PATH} "
            f"local_ip={args.bind} local_port={args.port} index_path={DATA_DIR}"
        )
        return

    write_service()
    info(f"unit:      {SERVICE_FILE}")

    reload_and_start()
    info("service started and enabled at boot")

    print()
    print("==================== SeekStorm ====================")
    print(f"  URL:                http://{args.bind}:{args.port}")
    print(f"  {CREDENTIALS_KEY}:  {apikey}")
    print(f"  credentials file:   {creds_path}")
    print("===================================================")


if __name__ == "__main__":
    main()
