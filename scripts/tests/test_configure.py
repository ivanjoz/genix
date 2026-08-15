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
                ("configure_server_utils.py", ("--binary-source", "precompiled")),
            ],
        )


if __name__ == "__main__":
    unittest.main()
