#!/usr/bin/env python3

"""Open an inbound TCP port on whatever firewall this host actually runs.

Three managers, one entry point. `ensure_tcp_port_open` probes them in the order that respects
ownership — firewalld and ufw are both frontends over the same netfilter chains, so a rule
inserted with plain iptables behind either one is silently discarded at their next reload. Only
when neither is managing does this fall through to iptables itself, which is what the Oracle
Cloud images ship: an INPUT chain ending in REJECT --reject-with icmp-host-prohibited and no
frontend installed.

Extracted from configure_db.py, which needed it for the CQL port and is still its other caller.
It lives on its own because the second caller (configure_server_utils.py) must not import a
ScyllaDB installer to open a socket, and because the iptables half is subtle enough that two
copies would drift: a rule only helps if it lands *before* whatever was dropping the packet, so
the position is computed rather than appended, and the chain is re-read afterwards to prove it.

Deliberately self-contained: only the standard library, no project imports, no sys.exit. Every
function reports through its return value, so a caller can decide whether an unopened port is
fatal. It never closes a port and never touches a chain other than INPUT.
"""

import contextlib
import os
import re
import subprocess
from shutil import which


def print_debug_block(block_title, block_content):
    print(f"[*] {block_title}")
    if not block_content:
        print("    (empty)")
        return

    for block_line in block_content.rstrip().splitlines():
        print(f"    {block_line}")


def run_capture_command(command_arguments, quiet=False):
    """Run argv and hand back the CompletedProcess. A missing binary is an exit code, not a crash.

    Probing is the normal case here — `firewall-cmd` does not exist on Ubuntu and `ufw` does not
    exist on Oracle Linux — so FileNotFoundError is reported as 127, the shell's code for
    "command not found", and the caller treats it like any other failed probe.
    """
    if not quiet:
        print(f"[*] Running argv command: {' '.join(command_arguments)}")

    try:
        command_result = subprocess.run(command_arguments, text=True, capture_output=True)
    except FileNotFoundError:
        command_result = subprocess.CompletedProcess(command_arguments, 127, stdout="", stderr="")

    if quiet:
        return command_result

    print(f"[*] Exit code: {command_result.returncode}")
    if command_result.stdout.strip():
        print_debug_block("stdout", command_result.stdout)
    if command_result.stderr.strip():
        print_debug_block("stderr", command_result.stderr)
    return command_result


def ensure_firewalld_port_open(tcp_port):
    firewalld_state_result = run_capture_command(["firewall-cmd", "--state"])
    if firewalld_state_result.returncode != 0 or "running" not in firewalld_state_result.stdout:
        print("[*] firewalld is not running.")
        return False

    port_specification = f"{tcp_port}/tcp"
    query_port_result = run_capture_command(["firewall-cmd", "--query-port", port_specification])
    if query_port_result.returncode == 0 and "yes" in query_port_result.stdout.lower():
        print(f"[*] firewalld already allows {port_specification}.")
        return True

    print(f"[*] Opening {port_specification} with firewalld.")
    add_port_result = run_capture_command(["firewall-cmd", "--permanent", "--add-port", port_specification])
    if add_port_result.returncode != 0:
        print(f"[!] Failed to add {port_specification} with firewalld.")
        return False

    reload_firewalld_result = run_capture_command(["firewall-cmd", "--reload"])
    if reload_firewalld_result.returncode != 0:
        print("[!] Failed to reload firewalld after opening port.")
        return False

    verification_result = run_capture_command(["firewall-cmd", "--query-port", port_specification])
    return verification_result.returncode == 0 and "yes" in verification_result.stdout.lower()


def ensure_ufw_port_open(tcp_port):
    ufw_status_result = run_capture_command(["ufw", "status"])
    if ufw_status_result.returncode != 0:
        print("[*] ufw is not available or not active.")
        return False

    normalized_status_output = ufw_status_result.stdout.lower()
    if "status: inactive" in normalized_status_output:
        print("[*] ufw is installed but inactive.")
        return False

    port_rule_pattern = rf'^{re.escape(str(tcp_port))}/tcp\s+allow\b'
    if re.search(port_rule_pattern, ufw_status_result.stdout, flags=re.IGNORECASE | re.MULTILINE):
        print(f"[*] ufw already allows {tcp_port}/tcp.")
        return True

    print(f"[*] Opening {tcp_port}/tcp with ufw.")
    allow_port_result = run_capture_command(["ufw", "allow", f"{tcp_port}/tcp"])
    if allow_port_result.returncode != 0:
        print(f"[!] Failed to add ufw rule for {tcp_port}/tcp.")
        return False

    verification_result = run_capture_command(["ufw", "status"])
    return re.search(port_rule_pattern, verification_result.stdout, flags=re.IGNORECASE | re.MULTILINE) is not None


def is_firewall_frontend_active():
    """Whether firewalld or ufw is the one managing netfilter on this host.

    Los dos son frontends de las mismas cadenas de netfilter: si alguno esta activo,
    una regla insertada a mano con iptables queda pisada en el proximo reload del
    frontend. Solo cuando ninguno gestiona tiene sentido tocar iptables directamente.
    """
    if which("firewall-cmd") is not None:
        firewalld_state_result = run_capture_command(["firewall-cmd", "--state"], quiet=True)
        if firewalld_state_result.returncode == 0 and "running" in firewalld_state_result.stdout:
            print("[*] firewalld is managing netfilter; not touching iptables directly.")
            return True

    if which("ufw") is not None:
        ufw_status_result = run_capture_command(["ufw", "status"], quiet=True)
        if ufw_status_result.returncode == 0 and "status: inactive" not in ufw_status_result.stdout.lower():
            print("[*] ufw is managing netfilter; not touching iptables directly.")
            return True

    return False


# Targets que descartan el paquete. Un REJECT devuelve icmp-host-prohibited, que es el
# 'no route to host' que ve el cliente; un DROP se manifiesta como timeout.
IPTABLES_BLOCKING_TARGETS = {"REJECT", "DROP"}


def iptables_ports_cover(port_specification, tcp_port):
    """Whether an iptables --dport/--dports value matches this port."""
    for port_token in port_specification.split(","):
        with contextlib.suppress(ValueError):
            if ":" in port_token:
                # Rango 'low:high', con cualquiera de los dos extremos opcional.
                range_low, _, range_high = port_token.partition(":")
                if int(range_low or 1) <= tcp_port <= int(range_high or 65535):
                    return True
            elif int(port_token) == tcp_port:
                return True
    return False


def read_iptables_input_chain():
    """Returns (policy, [(index, rule)]) for the INPUT chain, or None if unreadable.

    Los indices son 1-based porque es lo que esperan 'iptables -I/-D'.
    """
    chain_listing_result = run_capture_command(["iptables", "-S", "INPUT"], quiet=True)
    if chain_listing_result.returncode != 0:
        print_debug_block("iptables -S INPUT failed", chain_listing_result.stderr or chain_listing_result.stdout)
        return None

    chain_policy = "ACCEPT"
    chain_rules = []
    for listing_line in chain_listing_result.stdout.splitlines():
        if listing_line.startswith("-P INPUT "):
            chain_policy = listing_line.split()[2]
        elif listing_line.startswith("-A INPUT"):
            chain_rules.append(listing_line)

    return chain_policy, list(enumerate(chain_rules, start=1))


def find_iptables_accept_index(chain_rules, tcp_port):
    """Index of the first rule that already accepts TCP traffic for this port."""
    for rule_index, rule_text in chain_rules:
        if "-j ACCEPT" not in rule_text or "-p tcp" not in rule_text:
            continue
        ports_match = re.search(r"--dports?\s+(\S+)", rule_text)
        if ports_match and iptables_ports_cover(ports_match.group(1), tcp_port):
            return rule_index
    return None


def find_iptables_blocking_index(chain_rules, tcp_port):
    """Index of the first REJECT/DROP that would swallow traffic to this port."""
    for rule_index, rule_text in chain_rules:
        target_match = re.search(r"-j\s+(\S+)", rule_text)
        if not target_match or target_match.group(1) not in IPTABLES_BLOCKING_TARGETS:
            continue

        # Una regla de bloqueo acotada a otros puertos no nos afecta; la catch-all si.
        ports_match = re.search(r"--dports?\s+(\S+)", rule_text)
        if ports_match and not iptables_ports_cover(ports_match.group(1), tcp_port):
            continue
        return rule_index
    return None


def persist_iptables_rules():
    """Makes the new rule survive a reboot, with whatever mechanism the distro ships."""
    if which("netfilter-persistent") is not None:
        run_capture_command(["netfilter-persistent", "save"])
        return

    # Debian/Ubuntu con iptables-persistent, y RHEL/Oracle Linux con iptables-services.
    for persistence_path in ("/etc/iptables/rules.v4", "/etc/sysconfig/iptables"):
        if os.path.isdir(os.path.dirname(persistence_path)):
            # shell=True por la redireccion: iptables-save escribe en stdout y no acepta destino.
            print(f"[*] Running: iptables-save > {persistence_path}")
            subprocess.run(f"iptables-save > {persistence_path}", shell=True, text=True, capture_output=True)
            return

    print("[!] No iptables persistence mechanism found: the rule will be lost on reboot.")
    print("    Install iptables-persistent (Debian/Ubuntu) or iptables-services (RHEL).")


def ensure_iptables_port_open(tcp_port):
    """Opens the port in a plain iptables setup, but only when it is really blocked."""
    input_chain = read_iptables_input_chain()
    if input_chain is None:
        return False

    chain_policy, chain_rules = input_chain
    accept_index = find_iptables_accept_index(chain_rules, tcp_port)
    blocking_index = find_iptables_blocking_index(chain_rules, tcp_port)

    # La primera regla que matchea gana, asi que un ACCEPT antes del bloqueo ya alcanza.
    if accept_index is not None and (blocking_index is None or accept_index < blocking_index):
        print(f"[*] iptables already accepts TCP port {tcp_port} (rule {accept_index}).")
        return True

    if blocking_index is None and chain_policy == "ACCEPT":
        print(f"[*] iptables has no rule blocking TCP port {tcp_port} and the INPUT policy is ACCEPT.")
        return True

    # Se inserta justo antes del REJECT/DROP que lo estaba tapando; si lo que bloquea es
    # la policy de la cadena, va al final, que igual queda antes de la policy.
    insert_index = blocking_index if blocking_index is not None else len(chain_rules) + 1
    blocking_reason = f"rule {blocking_index}" if blocking_index is not None else f"INPUT policy {chain_policy}"
    print(f"[*] TCP port {tcp_port} is blocked by {blocking_reason}. Inserting an ACCEPT at position {insert_index}.")

    insert_rule_result = run_capture_command([
        "iptables", "-I", "INPUT", str(insert_index),
        "-p", "tcp", "--dport", str(tcp_port),
        "-m", "state", "--state", "NEW", "-j", "ACCEPT",
    ])
    if insert_rule_result.returncode != 0:
        print(f"[!] Failed to insert the iptables rule for TCP port {tcp_port}.")
        return False

    # Se relee la cadena en vez de confiar en el exit code: confirma que la regla quedo
    # efectivamente por delante de lo que bloqueaba.
    verification_chain = read_iptables_input_chain()
    if verification_chain is None:
        return False

    verified_policy, verified_rules = verification_chain
    verified_accept_index = find_iptables_accept_index(verified_rules, tcp_port)
    verified_blocking_index = find_iptables_blocking_index(verified_rules, tcp_port)
    if verified_accept_index is None or (verified_blocking_index is not None and verified_accept_index > verified_blocking_index):
        print(f"[!] The iptables rule for TCP port {tcp_port} did not take effect.")
        return False

    persist_iptables_rules()
    return True


def ensure_tcp_port_open(tcp_port, service_label=""):
    """Open tcp_port inbound with whichever manager owns this host's firewall.

    Returns True when a manager confirmed the port is open, False when none could. False is not
    necessarily fatal — a host with no filtering at all reports True through the iptables branch,
    so False means "there is filtering here and it was not opened", which every caller should at
    least print. Requires root; the probes fail harmlessly without it.
    """
    described_service = f" ({service_label})" if service_label else ""
    print(f"[*] Ensuring TCP port {tcp_port}{described_service} is open in the host firewall...")

    if which("firewall-cmd") is not None and ensure_firewalld_port_open(tcp_port):
        print(f"[*] Firewall confirmed open for TCP port {tcp_port} via firewalld.")
        return True

    if which("ufw") is not None and ensure_ufw_port_open(tcp_port):
        print(f"[*] Firewall confirmed open for TCP port {tcp_port} via ufw.")
        return True

    # Ultimo recurso: iptables plano, que es lo que traen las imagenes de Oracle Cloud
    # (cadena INPUT terminada en REJECT icmp-host-prohibited, sin firewalld ni ufw).
    if which("iptables") is not None and not is_firewall_frontend_active() and ensure_iptables_port_open(tcp_port):
        print(f"[*] Firewall confirmed open for TCP port {tcp_port} via iptables.")
        return True

    # Saltarse esto en silencio deja el servicio instalado pero inalcanzable, que es
    # justo el sintoma mas caro de diagnosticar.
    print(f"[!] Could not confirm that TCP port {tcp_port} is open in the host firewall.")
    print("    If this host filters inbound traffic, open the port manually before using the service.")
    return False
