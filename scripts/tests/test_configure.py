import importlib.util
import io
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from types import SimpleNamespace
from unittest import mock


def load_configure_module():
    script_path = Path(__file__).resolve().parents[1] / "configure.py"
    module_spec = importlib.util.spec_from_file_location("configure", script_path)
    configure = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(configure)
    return configure


class SelectionTest(unittest.TestCase):
    def test_238_selects_backend_and_server_utils_from_precompiled_binaries(self):
        configure = load_configure_module()

        selected_components, binary_source = configure.parse_selection("238")

        self.assertEqual(selected_components, {"2", "3"})
        self.assertEqual(binary_source, "precompiled")

    def test_selection_requires_components_and_exactly_one_binary_source(self):
        configure = load_configure_module()

        for invalid_selection in ("", "23", "78", "2378", "2238", "239"):
            with self.subTest(invalid_selection=invalid_selection):
                with self.assertRaises(ValueError):
                    configure.parse_selection(invalid_selection)


class ReleaseVerificationTest(unittest.TestCase):
    def test_release_asset_must_match_its_manifest_entry(self):
        configure = load_configure_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            asset_path = Path(temporary_directory) / "genix_app_linux_amd64"
            asset_path.write_bytes(b"verified release bytes")
            expected_checksum = configure.hashlib.sha256(asset_path.read_bytes()).hexdigest()

            with redirect_stdout(io.StringIO()):
                configure.verify_release_asset(
                    asset_path, {asset_path.name: expected_checksum}
                )
            self.assertTrue(asset_path.exists())

            asset_path.write_bytes(b"tampered bytes")
            with self.assertRaises(RuntimeError):
                configure.verify_release_asset(
                    asset_path, {asset_path.name: expected_checksum}
                )
            self.assertFalse(asset_path.exists())

    def test_matching_latest_release_asset_is_reused_without_downloading(self):
        configure = load_configure_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            download_directory = Path(temporary_directory)
            asset_name = "genix_app_linux_amd64"
            asset_path = download_directory / asset_name
            asset_path.write_bytes(b"cached latest release")
            expected_checksum = configure.hashlib.sha256(asset_path.read_bytes()).hexdigest()
            manifest_path = download_directory / "SHA256SUMS"
            manifest_path.write_text(f"{expected_checksum}  {asset_name}\n", encoding="utf-8")

            def download_manifest_only(requested_name):
                if requested_name == "SHA256SUMS":
                    return manifest_path
                self.fail(f"Unexpected binary download: {requested_name}")

            with mock.patch.object(configure, "DOWNLOAD_DIRECTORY", download_directory), \
                    mock.patch.object(configure, "resolve_release_architecture", return_value="amd64"), \
                    mock.patch.object(
                        configure,
                        "download_latest_release_file",
                        side_effect=download_manifest_only,
                    ), redirect_stdout(io.StringIO()):
                configure.download_selected_binaries({"2"}, "2")


class DispatchTest(unittest.TestCase):
    def test_238_downloads_then_runs_backend_before_server_utils(self):
        configure = load_configure_module()
        executed_configurers = []

        with mock.patch.object(
            configure, "parse_command_arguments", return_value=SimpleNamespace(selection="238")
        ), mock.patch.object(configure.os, "geteuid", return_value=0), mock.patch.object(
            configure, "input", create=True, return_value="2"
        ), mock.patch.object(configure, "download_selected_binaries") as download_mock, mock.patch.object(
            configure,
            "run_configurer",
            side_effect=lambda script_name, *arguments: executed_configurers.append(
                (script_name, arguments)
            ),
        ), redirect_stdout(io.StringIO()):
            configure.main()

        download_mock.assert_called_once_with({"2", "3"}, "2")
        self.assertEqual(
            executed_configurers,
            [
                ("configure_server.py", ("2", "--binary-source", "precompiled")),
                (
                    "configure_server_utils.py",
                    ("--binary-source", "precompiled", "--service-only"),
                ),
            ],
        )

    def run_server_utils_only(self, backend_unit_exists, typed_answer):
        """Selection '37': Server Utils alone, built from source. Returns the arguments it passed."""
        configure = load_configure_module()
        executed_configurers = []

        with mock.patch.object(
            configure, "parse_command_arguments", return_value=SimpleNamespace(selection="37")
        ), mock.patch.object(configure.os, "geteuid", return_value=0), mock.patch.object(
            configure, "input", create=True, return_value=typed_answer
        ), mock.patch.object(
            configure,
            "BACKEND_SERVICE_UNIT_PATH",
            SimpleNamespace(is_file=lambda: backend_unit_exists, name="genix.service"),
        ), mock.patch.object(
            configure,
            "run_configurer",
            side_effect=lambda script_name, *arguments: executed_configurers.append(
                (script_name, arguments)
            ),
        ), redirect_stdout(io.StringIO()):
            configure.main()

        self.assertEqual(len(executed_configurers), 1)
        return executed_configurers[0][1]

    # Installing Server Utils on its own used to demand sse_bridge.url on every host, because
    # --service-only was only passed when the Backend component happened to be selected in the same
    # run. On a box that already runs the backend that is a bridge nobody wants.
    def test_server_utils_alone_skips_the_bridge_when_the_backend_is_installed_here(self):
        arguments = self.run_server_utils_only(backend_unit_exists=True, typed_answer="")
        self.assertIn("--service-only", arguments)

    # The case the bridge exists for: a Lambda backend cannot hold the connection, so this host has
    # to, and the vhost plus sse_bridge.url are required.
    def test_server_utils_alone_keeps_the_bridge_when_no_backend_is_installed_here(self):
        arguments = self.run_server_utils_only(backend_unit_exists=False, typed_answer="")
        self.assertNotIn("--service-only", arguments)

    # Detection is a default, not a verdict: a host staged before the backend is installed looks
    # exactly like a Lambda deployment, so the typed answer has to win.
    def test_a_typed_answer_overrides_what_was_detected(self):
        self.assertIn(
            "--service-only",
            self.run_server_utils_only(backend_unit_exists=False, typed_answer="y"),
        )
        self.assertNotIn(
            "--service-only",
            self.run_server_utils_only(backend_unit_exists=True, typed_answer="n"),
        )

    def test_an_unreadable_answer_stops_without_a_traceback(self):
        with self.assertRaises(SystemExit) as exit_context:
            self.run_server_utils_only(backend_unit_exists=True, typed_answer="maybe")
        self.assertEqual(
            str(exit_context.exception),
            "[!] Answer the backend host question with y or n.",
        )

    def test_configurer_failure_has_no_python_traceback(self):
        configure = load_configure_module()
        command_error = configure.subprocess.CalledProcessError(1, ["python3", "child.py"])

        with mock.patch.object(configure.subprocess, "run", side_effect=command_error), \
                redirect_stdout(io.StringIO()), self.assertRaises(SystemExit) as exit_context:
            configure.run_configurer("child.py")

        self.assertEqual(str(exit_context.exception), "[!] child.py stopped with exit code 1.")


if __name__ == "__main__":
    unittest.main()
