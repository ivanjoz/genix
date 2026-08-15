#!/usr/bin/env python3
"""Configure one or more Genix services from source or the latest release."""

import argparse
import hashlib
import os
import platform
import re
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

SCRIPT_DIRECTORY = Path(__file__).resolve().parent
PROJECT_ROOT_DIRECTORY = SCRIPT_DIRECTORY.parent
CONFIGURE_DIRECTORY = SCRIPT_DIRECTORY / "configure"
DOWNLOAD_DIRECTORY = PROJECT_ROOT_DIRECTORY / "tmp"
LATEST_RELEASE_URL = "https://github.com/ivanjoz/genix/releases/latest/download"

COMPONENT_DATABASE = "1"
COMPONENT_BACKEND = "2"
COMPONENT_SERVER_UTILS = "3"
BINARY_SOURCE = "7"
BINARY_PRECOMPILED = "8"
COMPONENT_DIGITS = {COMPONENT_DATABASE, COMPONENT_BACKEND, COMPONENT_SERVER_UTILS}
BINARY_MODE_DIGITS = {BINARY_SOURCE, BINARY_PRECOMPILED}


def print_menu():
    """Show the digits that may be combined in one selection."""
    print("Select components:")
    print("    [1] Database")
    print("    [2] Backend Service")
    print("    [3] Server Utils")
    print("Select one binary source:")
    print("    [7] Build from source")
    print("    [8] Download precompiled binaries from the latest release")
    print("Example: 238 configures Backend Service and Server Utils from precompiled binaries.")


def parse_selection(raw_selection):
    """Return selected components and one binary mode from a compact value such as 238."""
    normalized_selection = raw_selection.strip()
    if not normalized_selection or not normalized_selection.isdigit():
        raise ValueError("The selection must contain only the menu digits, for example 238.")

    repeated_digits = sorted({digit for digit in normalized_selection if normalized_selection.count(digit) > 1})
    if repeated_digits:
        raise ValueError(f"Do not repeat choices: {', '.join(repeated_digits)}.")

    unknown_digits = sorted(set(normalized_selection) - COMPONENT_DIGITS - BINARY_MODE_DIGITS)
    if unknown_digits:
        raise ValueError(f"Unknown choice(s): {', '.join(unknown_digits)}.")

    selected_components = COMPONENT_DIGITS.intersection(normalized_selection)
    selected_binary_modes = BINARY_MODE_DIGITS.intersection(normalized_selection)
    if not selected_components:
        raise ValueError("Select at least one component: 1, 2 or 3.")
    if len(selected_binary_modes) != 1:
        raise ValueError("Select exactly one binary source: 7 or 8.")

    selected_binary_mode = selected_binary_modes.pop()
    return selected_components, "source" if selected_binary_mode == BINARY_SOURCE else "precompiled"


def resolve_release_architecture():
    """Map Linux machine names to the suffix used by Genix release assets."""
    machine_architecture = platform.machine().lower()
    if machine_architecture in {"x86_64", "amd64"}:
        return "amd64"
    if machine_architecture in {"aarch64", "arm64"}:
        return "arm64"
    raise RuntimeError(f"Unsupported architecture: {machine_architecture}.")


def download_latest_release_file(asset_name):
    """Download one latest-release file atomically into tmp/."""
    DOWNLOAD_DIRECTORY.mkdir(parents=True, exist_ok=True)
    destination_path = DOWNLOAD_DIRECTORY / asset_name
    staged_path = destination_path.with_name(f".{destination_path.name}.download")
    release_url = f"{LATEST_RELEASE_URL}/{asset_name}"
    print(f"[*] Downloading {release_url}")

    request = urllib.request.Request(release_url, headers={"User-Agent": "genix-configure"})
    try:
        with urllib.request.urlopen(request, timeout=120) as response, staged_path.open("wb") as output_file:
            while response_chunk := response.read(1024 * 1024):
                output_file.write(response_chunk)
        os.replace(staged_path, destination_path)
    except (OSError, urllib.error.URLError) as download_error:
        staged_path.unlink(missing_ok=True)
        raise RuntimeError(f"Could not download {asset_name}: {download_error}") from download_error

    print(f"[*] Downloaded {destination_path} ({destination_path.stat().st_size} bytes)")
    return destination_path


def parse_release_checksums(checksum_manifest_path):
    """Read the release checksum format without accepting malformed or duplicate entries."""
    release_checksums = {}
    for manifest_line in checksum_manifest_path.read_text(encoding="utf-8").splitlines():
        checksum_match = re.fullmatch(r"([0-9a-fA-F]{64})\s+\*?(.+)", manifest_line.strip())
        if not checksum_match:
            continue
        checksum_value, asset_name = checksum_match.groups()
        if asset_name in release_checksums:
            raise RuntimeError(f"SHA256SUMS contains a duplicate entry for {asset_name}.")
        release_checksums[asset_name] = checksum_value.lower()
    return release_checksums


def verify_release_asset(asset_path, release_checksums):
    """Reject a release binary unless its bytes match the published manifest."""
    expected_checksum = release_checksums.get(asset_path.name)
    if not expected_checksum:
        asset_path.unlink(missing_ok=True)
        raise RuntimeError(f"SHA256SUMS has no entry for {asset_path.name}.")

    calculated_checksum = hashlib.sha256(asset_path.read_bytes()).hexdigest()
    if calculated_checksum != expected_checksum:
        asset_path.unlink(missing_ok=True)
        raise RuntimeError(
            f"Checksum mismatch for {asset_path.name}: expected {expected_checksum}, "
            f"calculated {calculated_checksum}."
        )
    print(f"[*] Verified SHA-256 for {asset_path.name}: {calculated_checksum}")


def download_selected_binaries(selected_components, backend_mode):
    """Download only the latest binaries needed on this host."""
    release_architecture = resolve_release_architecture()
    selected_assets = []
    if COMPONENT_BACKEND in selected_components and backend_mode != "3":
        selected_assets.append(f"genix_app_linux_{release_architecture}")
    if COMPONENT_SERVER_UTILS in selected_components:
        selected_assets.append(f"genix-server-utils_linux_{release_architecture}")
    if not selected_assets:
        print("[*] No Genix service binary is needed for the selected components.")
        return

    checksum_manifest_path = download_latest_release_file("SHA256SUMS")
    release_checksums = parse_release_checksums(checksum_manifest_path)
    for selected_asset in selected_assets:
        verify_release_asset(download_latest_release_file(selected_asset), release_checksums)


def resolve_backend_mode():
    """Ask which side of the split backend deployment this host owns."""
    print("Backend Service mode:")
    print("    [1] Full (systemd service + Nginx proxy)")
    print("    [2] Only Systemd Service")
    print("    [3] Only Nginx Proxy")
    selected_mode = input("Select the Backend Service mode: ").strip()
    if selected_mode not in {"1", "2", "3"}:
        raise ValueError("Backend Service mode must be 1, 2 or 3.")
    return selected_mode


def run_configurer(script_name, *arguments):
    """Run one installer and stop immediately when it cannot complete."""
    command_arguments = [sys.executable, str(CONFIGURE_DIRECTORY / script_name), *arguments]
    print(f"[*] Running: {' '.join(command_arguments)}")
    subprocess.run(command_arguments, check=True)


def parse_command_arguments():
    argument_parser = argparse.ArgumentParser(
        description="Configure Database, Backend Service and Server Utils with one compact selection."
    )
    argument_parser.add_argument(
        "selection",
        nargs="?",
        help="combined menu digits, for example 238",
    )
    return argument_parser.parse_args()


def main():
    command_arguments = parse_command_arguments()
    if os.geteuid() != 0:
        raise SystemExit("[!] Run this script with sudo.")

    print_menu()
    raw_selection = command_arguments.selection or input("Selection: ")
    try:
        selected_components, binary_source = parse_selection(raw_selection)
        backend_mode = resolve_backend_mode() if COMPONENT_BACKEND in selected_components else None
    except ValueError as selection_error:
        raise SystemExit(f"[!] {selection_error}") from selection_error

    print(f"[*] Selected components: {', '.join(sorted(selected_components))}")
    print(f"[*] Binary source for Genix services: {binary_source}")

    if binary_source == "precompiled":
        try:
            download_selected_binaries(selected_components, backend_mode)
        except RuntimeError as download_error:
            raise SystemExit(f"[!] {download_error}") from download_error

    # Database first because Server Utils needs its schema and connectivity at startup.
    if COMPONENT_DATABASE in selected_components:
        run_configurer("configure_db.py")
    if COMPONENT_BACKEND in selected_components:
        run_configurer("configure_server.py", backend_mode, "--binary-source", binary_source)
    if COMPONENT_SERVER_UTILS in selected_components:
        run_configurer("configure_server_utils.py", "--binary-source", binary_source)


if __name__ == "__main__":
    main()
