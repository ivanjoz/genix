#!/usr/bin/env python3

"""Tests for the shared firewall opener, focused on the part that has no second chance.

firewalld and ufw are declarative — asking twice is harmless and the tool reports what it did.
Plain iptables is an ordered list where the first match wins, so a correct rule in the wrong
position does nothing, and "nothing" here means a service that installs cleanly and is
unreachable from every other machine. That placement arithmetic is what these cover.
"""

import importlib.util
import io
import sys
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest import mock

SCRIPTS_DIRECTORY = Path(__file__).resolve().parents[1]


def load_firewall_ports_module():
    if str(SCRIPTS_DIRECTORY) not in sys.path:
        sys.path.insert(0, str(SCRIPTS_DIRECTORY))

    script_path = SCRIPTS_DIRECTORY / "firewall_ports.py"
    module_spec = importlib.util.spec_from_file_location("firewall_ports", script_path)
    firewall_ports = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(firewall_ports)
    return firewall_ports


def build_chain(rule_lines):
    return list(enumerate(rule_lines, start=1))


class PortMatchingTest(unittest.TestCase):
    def test_a_dports_list_and_a_range_both_count_as_covering_the_port(self):
        firewall_ports = load_firewall_ports_module()
        covers = firewall_ports.iptables_ports_cover

        self.assertTrue(covers("14013", 14013))
        self.assertTrue(covers("22,80,14013", 14013))
        self.assertTrue(covers("14000:14100", 14013))
        # Open-ended ranges are legal iptables syntax on both sides.
        self.assertTrue(covers(":14100", 14013))
        self.assertTrue(covers("14000:", 14013))

        self.assertFalse(covers("14012", 14013))
        self.assertFalse(covers("22,80,443", 14013))
        self.assertFalse(covers("14014:14100", 14013))
        # A malformed token must be ignored, not raise, or one odd rule aborts the whole run.
        self.assertFalse(covers("http", 14013))


class IptablesPlacementTest(unittest.TestCase):
    # The real chain from the Oracle Cloud image: a catch-all REJECT at the end, with the ports
    # opened so far sitting in front of it.
    ORACLE_CLOUD_CHAIN = [
        "-A INPUT -p tcp -m tcp --dport 14008 -m conntrack --ctstate NEW -j ACCEPT",
        "-A INPUT -m state --state RELATED,ESTABLISHED -j ACCEPT",
        "-A INPUT -p icmp -j ACCEPT",
        "-A INPUT -i lo -j ACCEPT",
        "-A INPUT -p tcp -m tcp --dport 14011 -m state --state NEW -j ACCEPT",
        "-A INPUT -p tcp -m state --state NEW -m tcp --dport 22 -j ACCEPT",
        "-A INPUT -p tcp -m tcp --dport 80 -j ACCEPT",
        "-A INPUT -p tcp -m tcp --dport 443 -j ACCEPT",
        "-A INPUT -j REJECT --reject-with icmp-host-prohibited",
    ]

    def test_the_blocking_rule_is_found_and_an_unrelated_reject_is_not(self):
        firewall_ports = load_firewall_ports_module()
        chain = build_chain(self.ORACLE_CLOUD_CHAIN)

        self.assertIsNone(firewall_ports.find_iptables_accept_index(chain, 14013))
        self.assertEqual(firewall_ports.find_iptables_blocking_index(chain, 14013), 9)
        # An already-open port finds its ACCEPT ahead of that REJECT.
        self.assertEqual(firewall_ports.find_iptables_accept_index(chain, 14011), 5)

        # A REJECT scoped to other ports must not be mistaken for the one in the way.
        narrow_chain = build_chain([
            "-A INPUT -p tcp -m tcp --dport 3306 -j REJECT --reject-with icmp-port-unreachable",
        ])
        self.assertIsNone(firewall_ports.find_iptables_blocking_index(narrow_chain, 14013))
        # A DROP counts too — it just shows up as a timeout instead of 'no route to host'.
        drop_chain = build_chain(["-A INPUT -j DROP"])
        self.assertEqual(firewall_ports.find_iptables_blocking_index(drop_chain, 14013), 1)

    def test_the_accept_is_inserted_ahead_of_the_reject_that_was_swallowing_the_port(self):
        firewall_ports = load_firewall_ports_module()
        inserted_commands = []
        chain_state = list(self.ORACLE_CLOUD_CHAIN)

        def fake_run(command_arguments, quiet=False):
            if command_arguments[:3] == ["iptables", "-S", "INPUT"]:
                listing = "-P INPUT ACCEPT\n" + "\n".join(chain_state) + "\n"
                return mock.Mock(returncode=0, stdout=listing, stderr="")
            inserted_commands.append(command_arguments)
            position = int(command_arguments[3])
            chain_state.insert(position - 1, "-A INPUT " + " ".join(command_arguments[4:]))
            return mock.Mock(returncode=0, stdout="", stderr="")

        with mock.patch.object(firewall_ports, "run_capture_command", side_effect=fake_run), \
                mock.patch.object(firewall_ports, "persist_iptables_rules") as persist:
            with redirect_stdout(io.StringIO()):
                opened = firewall_ports.ensure_iptables_port_open(14013)

        self.assertTrue(opened)
        # Position 9 is the REJECT: the new rule has to land in front of it, not appended after.
        self.assertEqual(
            inserted_commands,
            [["iptables", "-I", "INPUT", "9", "-p", "tcp", "--dport", "14013",
              "-m", "state", "--state", "NEW", "-j", "ACCEPT"]],
        )
        # A rule lost on reboot is the same outage a week later.
        persist.assert_called_once()

    def test_an_already_open_port_is_left_alone(self):
        firewall_ports = load_firewall_ports_module()
        listing = "-P INPUT ACCEPT\n" + "\n".join(self.ORACLE_CLOUD_CHAIN) + "\n"

        with mock.patch.object(
            firewall_ports, "run_capture_command",
            return_value=mock.Mock(returncode=0, stdout=listing, stderr=""),
        ) as run_command:
            with redirect_stdout(io.StringIO()):
                self.assertTrue(firewall_ports.ensure_iptables_port_open(14011))

        # Only the read; re-adding a duplicate ACCEPT on every provision grows the chain forever.
        self.assertEqual(len(run_command.call_args_list), 1)

    def test_a_host_with_no_filtering_reports_success_without_adding_a_rule(self):
        firewall_ports = load_firewall_ports_module()

        with mock.patch.object(
            firewall_ports, "run_capture_command",
            return_value=mock.Mock(returncode=0, stdout="-P INPUT ACCEPT\n", stderr=""),
        ) as run_command:
            with redirect_stdout(io.StringIO()):
                self.assertTrue(firewall_ports.ensure_iptables_port_open(14013))

        self.assertEqual(len(run_command.call_args_list), 1)


class ManagerSelectionTest(unittest.TestCase):
    def test_iptables_is_skipped_while_ufw_is_active(self):
        # Both drive the same netfilter chains, and ufw rewrites them on reload: a hand-inserted
        # rule would work until the next `ufw reload` and then vanish with no trace of why.
        firewall_ports = load_firewall_ports_module()

        def fake_which(binary_name):
            return "/usr/sbin/ufw" if binary_name == "ufw" else None

        with mock.patch.object(firewall_ports, "which", side_effect=fake_which), \
                mock.patch.object(
                    firewall_ports, "run_capture_command",
                    return_value=mock.Mock(returncode=0, stdout="Status: active\n", stderr=""),
                ):
            with redirect_stdout(io.StringIO()):
                self.assertTrue(firewall_ports.is_firewall_frontend_active())

    def test_a_missing_optional_binary_is_an_exit_code_not_a_traceback(self):
        # firewall-cmd does not exist on Ubuntu and ufw does not exist on Oracle Linux. Probing
        # for one has to be routine, not a crash after the service is already installed.
        firewall_ports = load_firewall_ports_module()

        with redirect_stdout(io.StringIO()):
            result = firewall_ports.run_capture_command(["genix-binary-that-does-not-exist"])
        self.assertEqual(result.returncode, 127)

    def test_no_manager_available_reports_failure_rather_than_claiming_success(self):
        firewall_ports = load_firewall_ports_module()

        with mock.patch.object(firewall_ports, "which", return_value=None):
            with redirect_stdout(io.StringIO()) as captured_output:
                self.assertFalse(firewall_ports.ensure_tcp_port_open(14013, "server_utils raw TCP"))

        self.assertIn("Could not confirm", captured_output.getvalue())
        self.assertIn("server_utils raw TCP", captured_output.getvalue())


if __name__ == "__main__":
    unittest.main()
