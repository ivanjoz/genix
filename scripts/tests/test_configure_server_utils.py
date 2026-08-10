import importlib.util
import io
import os
import pwd
import sys
import tempfile
import unittest
from types import SimpleNamespace
from unittest import mock
from contextlib import redirect_stdout
from pathlib import Path

SCRIPTS_DIRECTORY = Path(__file__).resolve().parents[1]


def load_configure_server_utils_module():
    # Load the script directly so the test covers the deployed entrypoint code. Its own
    # `import configure_server` needs scripts/ importable, which is what this path insert does.
    if str(SCRIPTS_DIRECTORY) not in sys.path:
        sys.path.insert(0, str(SCRIPTS_DIRECTORY))

    script_path = SCRIPTS_DIRECTORY / "configure_server_utils.py"
    module_spec = importlib.util.spec_from_file_location("configure_server_utils", script_path)
    configure_server_utils = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(configure_server_utils)
    return configure_server_utils


class BridgeNginxTemplateTest(unittest.TestCase):
    def test_certificate_produces_an_http3_vhost_that_never_buffers(self):
        configure_server_utils = load_configure_server_utils_module()
        existing_config = (
            "server {\n"
            "    ssl_certificate /etc/letsencrypt/live/genix-sse.un.pe/fullchain.pem; # managed by Certbot\n"
            "    ssl_certificate_key /etc/letsencrypt/live/genix-sse.un.pe/privkey.pem; # managed by Certbot\n"
            "    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot\n"
            "}\n"
        )

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_server_utils.build_bridge_nginx_configuration(
                "genix-sse.un.pe", 14012, existing_config
            )

        self.assertIn("listen 443 quic reuseport;", rendered_config)
        self.assertIn("add_header Alt-Svc 'h3=\":443\"; ma=86400' always;", rendered_config)
        self.assertIn("ssl_certificate /etc/letsencrypt/live/genix-sse.un.pe/fullchain.pem;", rendered_config)
        # One host, one upstream: always the loopback address.
        self.assertIn("proxy_pass http://127.0.0.1:14012;", rendered_config)
        # The streaming settings are the reason this vhost exists.
        self.assertIn("proxy_buffering off;", rendered_config)
        self.assertIn("proxy_read_timeout 3600s;", rendered_config)
        self.assertIn('proxy_set_header Connection "";', rendered_config)
        self.assertIn("gzip off;", rendered_config)
        # Certbot's options include would duplicate ssl_protocols, and CORS is the bridge's job.
        self.assertNotIn("options-ssl-nginx.conf", rendered_config)
        self.assertNotIn("add_header 'Access-Control-Allow-Origin'", rendered_config)

    def test_reuseport_is_dropped_when_another_vhost_already_owns_it(self):
        configure_server_utils = load_configure_server_utils_module()

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_server_utils.build_bridge_nginx_configuration(
                "genix-sse.un.pe",
                14012,
                "ssl_certificate /c/fullchain.pem;\nssl_certificate_key /c/privkey.pem;\n",
                reuseport_is_available=False,
            )

        self.assertIn("listen 443 quic;", rendered_config)
        self.assertNotIn("reuseport", rendered_config)

    def test_without_a_certificate_the_vhost_falls_back_to_plain_http(self):
        configure_server_utils = load_configure_server_utils_module()

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_server_utils.build_bridge_nginx_configuration(
                "genix-sse.un.pe", 14012
            )

        self.assertIn("listen 80;", rendered_config)
        self.assertNotIn("listen 443", rendered_config)
        self.assertNotIn("ssl_certificate", rendered_config)
        # Streaming still has to work over plain HTTP.
        self.assertIn("proxy_buffering off;", rendered_config)

    def test_quic_is_omitted_when_nginx_cannot_speak_it(self):
        """An Nginx without http_v3_module cannot parse `listen ... quic`, and nginx -t covers
        the whole host, so emitting it would break every other site rather than this one."""
        configure_server_utils = load_configure_server_utils_module()

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_server_utils.build_bridge_nginx_configuration(
                "genix-sse.un.pe",
                14012,
                "ssl_certificate /c/fullchain.pem;\nssl_certificate_key /c/privkey.pem;\n",
                http3_is_supported=False,
            )

        self.assertNotIn("quic", rendered_config)
        # Advertising h3 without a QUIC listener points browsers at a port nothing answers on.
        self.assertNotIn("Alt-Svc", rendered_config)
        # TLS itself must still be configured.
        self.assertIn("listen 443 ssl;", rendered_config)
        self.assertIn("ssl_certificate /c/fullchain.pem;", rendered_config)
        self.assertIn("proxy_pass http://127.0.0.1:14012;", rendered_config)

    def test_reuseport_owner_is_found_in_any_nginx_configuration_file(self):
        configure_server_utils = load_configure_server_utils_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            configuration_directory = Path(temporary_directory) / "conf.d"
            configuration_directory.mkdir()
            configure_server_utils.NGINX_CONFIGURATION_DIRECTORY = configuration_directory

            own_configuration_path = configuration_directory / "genix-sse.un.pe.conf"
            self.assertIsNone(configure_server_utils.detect_reuseport_listener_owner(own_configuration_path))

            backend_configuration_path = configuration_directory / "genix-api-4.un.pe.conf"
            backend_configuration_path.write_text(
                "server {\n    listen 443 quic reuseport;\n}\n", encoding="utf-8"
            )
            self.assertEqual(
                configure_server_utils.detect_reuseport_listener_owner(own_configuration_path),
                backend_configuration_path,
            )

            # Our own file claiming reuseport is not a conflict with itself.
            own_configuration_path.write_text("server {\n    listen 443 quic reuseport;\n}\n", encoding="utf-8")
            backend_configuration_path.unlink()
            self.assertIsNone(configure_server_utils.detect_reuseport_listener_owner(own_configuration_path))


class ConfigurationResolutionTest(unittest.TestCase):
    def test_domain_is_taken_from_the_url_in_any_accepted_shape(self):
        configure_server_utils = load_configure_server_utils_module()
        config_path = Path("/repo/config.toml")

        with redirect_stdout(io.StringIO()):
            for configured_url in ("https://genix-sse.un.pe/", "http://Genix-SSE.un.pe", "genix-sse.un.pe"):
                self.assertEqual(
                    configure_server_utils.resolve_bridge_domain(
                        {"sse_bridge": {"url": configured_url}}, config_path
                    ),
                    "genix-sse.un.pe",
                )

    def test_an_unusable_url_stops_the_run_instead_of_prompting(self):
        configure_server_utils = load_configure_server_utils_module()
        config_path = Path("/repo/config.toml")

        unusable_credentials = [
            {},
            {"sse_bridge": {"url": "   "}},
            {"sse_bridge": {"url": "localhost"}},
            {"sse_bridge": {"url": "ftp://genix-sse.un.pe"}},
            # A function URL is how the project says "no bridge", so it can never be the vhost.
            {"sse_bridge": {"url": "https://abc.lambda-url.us-east-1.on.aws/"}},
        ]

        for project_credentials in unusable_credentials:
            with redirect_stdout(io.StringIO()), self.assertRaises(SystemExit):
                configure_server_utils.resolve_bridge_domain(project_credentials, config_path)

    def test_port_defaults_to_the_value_compiled_into_the_binary(self):
        configure_server_utils = load_configure_server_utils_module()

        with redirect_stdout(io.StringIO()):
            self.assertEqual(configure_server_utils.resolve_bridge_port({}), 14012)
            self.assertEqual(
                configure_server_utils.resolve_bridge_port({"sse_bridge": {"port": "15000"}}), 15000
            )

            # A typo must stop the run: the Nginx upstream is built from this number.
            with self.assertRaises(SystemExit):
                configure_server_utils.resolve_bridge_port({"sse_bridge": {"port": "0"}})

    def test_both_secrets_must_be_present_and_are_never_prompted_for(self):
        configure_server_utils = load_configure_server_utils_module()
        config_path = Path("/repo/config.toml")

        with redirect_stdout(io.StringIO()):
            # Both present: passes without asking anything.
            configure_server_utils.verify_required_secrets(
                {"secret_phrase": "abcd1234", "internal_apikey": "efgh5678"}, config_path
            )

            # Either one missing or blank stops the run, naming the key.
            for incomplete_credentials in (
                {},
                {"secret_phrase": "abcd1234"},
                {"internal_apikey": "efgh5678"},
                {"secret_phrase": "abcd1234", "internal_apikey": "   "},
            ):
                with self.assertRaises(SystemExit):
                    configure_server_utils.verify_required_secrets(incomplete_credentials, config_path)

    def test_the_script_never_imports_an_interactive_prompt(self):
        # The old bridge script asked for sse_bridge.apikey. Both secrets are now root-level keys
        # written by initial setup, so there is nothing left to ask and no getpass to import.
        script_source = (SCRIPTS_DIRECTORY / "configure_server_utils.py").read_text(encoding="utf-8")
        self.assertNotIn("getpass", script_source)
        self.assertNotIn("input(", script_source)


class UnitAndBinaryTest(unittest.TestCase):
    def test_service_unit_reads_both_secrets_from_the_config_file(self):
        configure_server_utils = load_configure_server_utils_module()

        unit_contents = configure_server_utils.build_service_contents(
            "ubuntu", Path("/home/ubuntu/genix/config.toml"), 14012
        )

        self.assertIn("Description=Genix Server Utilities", unit_contents)
        self.assertIn("Environment=SSE_BRIDGE_PORT=14012", unit_contents)
        self.assertIn("Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml", unit_contents)
        self.assertIn("ExecStart=/usr/local/bin/genix/genix-server-utils", unit_contents)
        self.assertIn("User=ubuntu", unit_contents)
        # The unit is world-readable, so neither secret may be exported here.
        self.assertNotIn("Environment=SECRET_PHRASE=", unit_contents)
        self.assertNotIn("Environment=INTERNAL_APIKEY=", unit_contents)

    def test_the_c_linker_is_installed_only_when_it_is_missing(self):
        """rustc shells out to `cc` to link, so a host without one cannot build at all — not even
        the build scripts of pure-Rust crates, which are compiled into real executables."""
        configure_server_utils = load_configure_server_utils_module()
        installed_commands = []

        def fake_run_command(command_arguments, **_keyword_arguments):
            installed_commands.append(command_arguments)
            return SimpleNamespace(returncode=0, stdout="", stderr="")

        # A host that already has a compiler must not have packages installed behind its back.
        with mock.patch.object(configure_server_utils.shutil, "which", lambda name: "/usr/bin/cc"):
            with mock.patch.object(configure_server_utils, "run_command", fake_run_command):
                with redirect_stdout(io.StringIO()):
                    configure_server_utils.ensure_c_linker()
        self.assertEqual(installed_commands, [])

        # A host without one gets the compiler and the libc headers, and nothing more: this
        # project has no C++ and no make-driven build scripts.
        available_tools = {"apt-get": "/usr/bin/apt-get"}
        with mock.patch.object(
            configure_server_utils.shutil, "which", lambda name: available_tools.get(name)
        ):
            with mock.patch.object(configure_server_utils, "run_command", fake_run_command):
                with redirect_stdout(io.StringIO()):
                    with self.assertRaises(SystemExit):
                        configure_server_utils.ensure_c_linker()

        flattened = [" ".join(command) for command in installed_commands]
        self.assertTrue(any("apt-get update" in command for command in flattened))
        self.assertTrue(any("install -y gcc libc6-dev" in command for command in flattened))
        self.assertFalse(any("build-essential" in command for command in flattened))

    def test_service_unit_raises_the_file_descriptor_limit(self):
        """systemd defaults the soft limit to 1024, which is below the daemon's own
        max_connections of 1024 once the ScyllaDB session, both listeners and every open SSE
        stream are counted: it would hit EMFILE before its configured ceiling."""
        configure_server_utils = load_configure_server_utils_module()

        service_contents = configure_server_utils.build_service_contents(
            "genix", Path("/repo/config.toml"), 14012
        )

        self.assertIn("LimitNOFILE=65536", service_contents)

    def test_restart_watcher_units_target_the_merged_service(self):
        configure_server_utils = load_configure_server_utils_module()

        self.assertIn(
            "PathChanged=/usr/local/bin/genix/genix-server-utils",
            configure_server_utils.build_restart_path_contents(),
        )
        self.assertIn(
            "ExecStart=/usr/bin/systemctl restart genix-server-utils.service",
            configure_server_utils.build_restart_service_contents(),
        )

    def test_source_is_detected_only_with_a_cargo_manifest_and_main(self):
        configure_server_utils = load_configure_server_utils_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            self.assertIsNone(configure_server_utils.detect_source_directory(repository_root))

            source_directory = repository_root / "server_utils"
            (source_directory / "src").mkdir(parents=True)
            (source_directory / "Cargo.toml").write_text(
                '[package]\nname = "genix-server-utils"\n', encoding="utf-8"
            )
            self.assertIsNone(configure_server_utils.detect_source_directory(repository_root))

            (source_directory / "src" / "main.rs").write_text("fn main() {}\n", encoding="utf-8")
            self.assertEqual(
                configure_server_utils.detect_source_directory(repository_root), source_directory
            )

    def test_prebuilt_binary_is_staged_and_renamed_into_place(self):
        configure_server_utils = load_configure_server_utils_module()
        runtime_user_entry = pwd.getpwuid(os.getuid())

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            configure_server_utils.SERVICE_INSTALL_DIRECTORY = repository_root / "installed"
            configure_server_utils.BINARY_PATH = (
                configure_server_utils.SERVICE_INSTALL_DIRECTORY / "genix-server-utils"
            )
            configure_server_utils.SERVICE_INSTALL_DIRECTORY.mkdir()

            # No source to compile and no artifact anywhere: that must stop the run.
            with self.assertRaises(SystemExit):
                configure_server_utils.provide_binary(repository_root, runtime_user_entry)

            artifact_path = (
                repository_root
                / "tmp"
                / f"genix-server-utils_linux_{configure_server_utils.resolve_go_architecture()}"
            )
            artifact_path.parent.mkdir()
            artifact_path.write_bytes(b"\x7fELF" + b"\x00" * 8)

            configure_server_utils.provide_binary(repository_root, runtime_user_entry)

            self.assertTrue(configure_server_utils.BINARY_PATH.is_file())
            self.assertTrue(os.access(configure_server_utils.BINARY_PATH, os.X_OK))
            self.assertFalse(
                (configure_server_utils.SERVICE_INSTALL_DIRECTORY / ".genix-server-utils.staged").exists()
            )

    def test_cargo_is_looked_up_where_sudo_cannot_see_it(self):
        configure_server_utils = load_configure_server_utils_module()

        # sudo strips ~/.cargo/bin from PATH, so an explicit override has to win.
        with tempfile.TemporaryDirectory() as temporary_directory:
            fake_cargo = Path(temporary_directory) / "cargo"
            fake_cargo.write_text("#!/bin/sh\n", encoding="utf-8")
            fake_cargo.chmod(0o755)

            os.environ["CARGO_BINARY"] = str(fake_cargo)
            try:
                self.assertEqual(configure_server_utils.detect_cargo_binary(None), fake_cargo)
            finally:
                del os.environ["CARGO_BINARY"]

        # A non-executable override is a configuration error, not a silent fallback.
        os.environ["CARGO_BINARY"] = "/nonexistent/cargo"
        try:
            with redirect_stdout(io.StringIO()), self.assertRaises(SystemExit):
                configure_server_utils.detect_cargo_binary(None)
        finally:
            del os.environ["CARGO_BINARY"]


if __name__ == "__main__":
    unittest.main()
