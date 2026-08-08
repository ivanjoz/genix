import importlib.util
import io
import os
import pwd
import sys
import tempfile
import tomllib
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest import mock

SCRIPTS_DIRECTORY = Path(__file__).resolve().parents[1]


def load_configure_sse_bridge_module():
    # Load the script directly so the test covers the deployed entrypoint code. Its own
    # `import configure_server` needs scripts/ importable, which is what this path insert does.
    if str(SCRIPTS_DIRECTORY) not in sys.path:
        sys.path.insert(0, str(SCRIPTS_DIRECTORY))

    script_path = SCRIPTS_DIRECTORY / "configure_sse_bridge.py"
    module_spec = importlib.util.spec_from_file_location("configure_sse_bridge", script_path)
    configure_sse_bridge = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(configure_sse_bridge)
    return configure_sse_bridge


class BridgeNginxTemplateTest(unittest.TestCase):
    def test_certificate_produces_an_http3_vhost_that_never_buffers(self):
        configure_sse_bridge = load_configure_sse_bridge_module()
        existing_config = (
            "server {\n"
            "    ssl_certificate /etc/letsencrypt/live/genix-sse.un.pe/fullchain.pem; # managed by Certbot\n"
            "    ssl_certificate_key /etc/letsencrypt/live/genix-sse.un.pe/privkey.pem; # managed by Certbot\n"
            "    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot\n"
            "}\n"
        )

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_sse_bridge.build_bridge_nginx_configuration(
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
        configure_sse_bridge = load_configure_sse_bridge_module()

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_sse_bridge.build_bridge_nginx_configuration(
                "genix-sse.un.pe",
                14012,
                "ssl_certificate /c/fullchain.pem;\nssl_certificate_key /c/privkey.pem;\n",
                reuseport_is_available=False,
            )

        self.assertIn("listen 443 quic;", rendered_config)
        self.assertNotIn("reuseport", rendered_config)

    def test_without_a_certificate_the_vhost_falls_back_to_plain_http(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with redirect_stdout(io.StringIO()):
            rendered_config = configure_sse_bridge.build_bridge_nginx_configuration("genix-sse.un.pe", 14012)

        self.assertIn("listen 80;", rendered_config)
        self.assertNotIn("listen 443", rendered_config)
        self.assertNotIn("ssl_certificate", rendered_config)
        # Streaming still has to work over plain HTTP.
        self.assertIn("proxy_buffering off;", rendered_config)

    def test_reuseport_owner_is_found_in_any_nginx_configuration_file(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            configuration_directory = Path(temporary_directory) / "conf.d"
            configuration_directory.mkdir()
            configure_sse_bridge.NGINX_CONFIGURATION_DIRECTORY = configuration_directory

            own_configuration_path = configuration_directory / "genix-sse.un.pe.conf"
            self.assertIsNone(configure_sse_bridge.detect_reuseport_listener_owner(own_configuration_path))

            backend_configuration_path = configuration_directory / "genix-api-4.un.pe.conf"
            backend_configuration_path.write_text(
                "server {\n    listen 443 quic reuseport;\n}\n", encoding="utf-8"
            )
            self.assertEqual(
                configure_sse_bridge.detect_reuseport_listener_owner(own_configuration_path),
                backend_configuration_path,
            )

            # Our own file claiming reuseport is not a conflict with itself.
            own_configuration_path.write_text("server {\n    listen 443 quic reuseport;\n}\n", encoding="utf-8")
            backend_configuration_path.unlink()
            self.assertIsNone(configure_sse_bridge.detect_reuseport_listener_owner(own_configuration_path))


class BridgeCredentialsTest(unittest.TestCase):
    def test_domain_is_taken_from_the_url_in_any_accepted_shape(self):
        configure_sse_bridge = load_configure_sse_bridge_module()
        config_path = Path("/repo/config.toml")

        with redirect_stdout(io.StringIO()):
            for configured_url in ("https://genix-sse.un.pe/", "http://Genix-SSE.un.pe", "genix-sse.un.pe"):
                self.assertEqual(
                    configure_sse_bridge.resolve_bridge_domain(
                        {"sse_bridge": {"url": configured_url}}, config_path
                    ),
                    "genix-sse.un.pe",
                )

    def test_an_unusable_url_stops_the_run_instead_of_prompting(self):
        configure_sse_bridge = load_configure_sse_bridge_module()
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
                configure_sse_bridge.resolve_bridge_domain(project_credentials, config_path)

    def test_port_defaults_to_the_value_compiled_into_the_bridge(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with redirect_stdout(io.StringIO()):
            self.assertEqual(configure_sse_bridge.resolve_bridge_port({}), 14012)
            self.assertEqual(
                configure_sse_bridge.resolve_bridge_port({"sse_bridge": {"port": "15000"}}), 15000
            )

            # A typo must stop the run: the Nginx upstream is built from this number.
            with self.assertRaises(SystemExit):
                configure_sse_bridge.resolve_bridge_port({"sse_bridge": {"port": "0"}})

    def test_a_stored_key_under_either_name_is_used_without_asking(self):
        configure_sse_bridge = load_configure_sse_bridge_module()
        config_path = Path("/repo/config.toml")

        with mock.patch.object(configure_sse_bridge.getpass, "getpass") as getpass_mock, \
                redirect_stdout(io.StringIO()):
            self.assertEqual(
                configure_sse_bridge.resolve_bridge_api_key(
                    {"sse_bridge": {"apikey": "abcd1234efgh"}}, config_path
                ),
                ("abcd1234efgh", False),
            )
            # A developer machine has the backend's full file: same value, original name.
            self.assertEqual(
                configure_sse_bridge.resolve_bridge_api_key(
                    {"secret_phrase": "abcd1234efgh"}, config_path
                ),
                ("abcd1234efgh", False),
            )

        getpass_mock.assert_not_called()

    def test_a_missing_key_is_asked_for_without_echoing_it_and_then_stored(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            config_path = Path(temporary_directory) / "config.toml"
            config_path.write_text(
                '[sse_bridge]\nurl = "https://genix-sse.un.pe/"\n', encoding="utf-8"
            )

            # The first answer is too short to be a shared secret, so the prompt repeats.
            with mock.patch.object(configure_sse_bridge.sys.stdin, "isatty", return_value=True), \
                    mock.patch.object(
                        configure_sse_bridge.getpass, "getpass", side_effect=["short", "abcd1234efgh"]
                    ), \
                    redirect_stdout(io.StringIO()) as captured_output:
                bridge_api_key, api_key_was_prompted = configure_sse_bridge.resolve_bridge_api_key(
                    {"sse_bridge": {"url": "https://genix-sse.un.pe/"}}, config_path
                )
                configure_sse_bridge.store_bridge_api_key(config_path, bridge_api_key)

            with open(config_path, "rb") as config_file:
                stored_config = tomllib.load(config_file)

        self.assertEqual((bridge_api_key, api_key_was_prompted), ("abcd1234efgh", True))
        self.assertEqual(stored_config["sse_bridge"]["apikey"], "abcd1234efgh")
        # Unrelated keys survive, and the secret is never printed to the terminal.
        self.assertEqual(stored_config["sse_bridge"]["url"], "https://genix-sse.un.pe/")
        self.assertNotIn("abcd1234efgh", captured_output.getvalue())

    def test_a_missing_key_without_a_terminal_fails(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with mock.patch.object(configure_sse_bridge.sys.stdin, "isatty", return_value=False), \
                redirect_stdout(io.StringIO()), self.assertRaises(SystemExit):
            configure_sse_bridge.resolve_bridge_api_key({}, Path("/repo/config.toml"))


class BridgeUnitAndBinaryTest(unittest.TestCase):
    def test_service_unit_reads_the_key_from_the_credentials_file(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        unit_contents = configure_sse_bridge.build_bridge_service_contents(
            "ubuntu", Path("/home/ubuntu/genix/config.toml"), 14012
        )

        self.assertIn("Description=Genix SSE Bridge", unit_contents)
        self.assertIn("Environment=SSE_BRIDGE_PORT=14012", unit_contents)
        self.assertIn("Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml", unit_contents)
        self.assertIn("ExecStart=/usr/local/bin/genix/sse_bridge", unit_contents)
        self.assertIn("User=ubuntu", unit_contents)
        # The unit is world-readable, so the secret itself must never be exported here.
        self.assertNotIn("Environment=SSE_BRIDGE_APIKEY=", unit_contents)

    def test_restart_watcher_units_target_the_bridge_only(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        self.assertIn(
            "PathChanged=/usr/local/bin/genix/sse_bridge",
            configure_sse_bridge.build_bridge_restart_path_contents(),
        )
        self.assertIn(
            "ExecStart=/usr/bin/systemctl restart genix-sse-bridge.service",
            configure_sse_bridge.build_bridge_restart_service_contents(),
        )

    def test_bridge_source_is_detected_only_with_go_module_and_main(self):
        configure_sse_bridge = load_configure_sse_bridge_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            self.assertIsNone(configure_sse_bridge.detect_bridge_source_directory(repository_root))

            bridge_directory = repository_root / "sse_bridge"
            bridge_directory.mkdir()
            (bridge_directory / "go.mod").write_text("module genix/sse_bridge\n", encoding="utf-8")
            self.assertIsNone(configure_sse_bridge.detect_bridge_source_directory(repository_root))

            (bridge_directory / "main.go").write_text("package main\n", encoding="utf-8")
            self.assertEqual(
                configure_sse_bridge.detect_bridge_source_directory(repository_root), bridge_directory
            )

    def test_prebuilt_binary_is_staged_and_renamed_into_place(self):
        configure_sse_bridge = load_configure_sse_bridge_module()
        runtime_user_entry = pwd.getpwuid(os.getuid())

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            configure_sse_bridge.SERVICE_INSTALL_DIRECTORY = repository_root / "installed"
            configure_sse_bridge.BRIDGE_BINARY_PATH = configure_sse_bridge.SERVICE_INSTALL_DIRECTORY / "sse_bridge"
            configure_sse_bridge.SERVICE_INSTALL_DIRECTORY.mkdir()

            # No source to compile and no artifact anywhere: that must stop the run.
            with self.assertRaises(SystemExit):
                configure_sse_bridge.provide_bridge_binary(repository_root, runtime_user_entry)

            artifact_path = (
                repository_root / "tmp" / f"sse_bridge_linux_{configure_sse_bridge.resolve_go_architecture()}"
            )
            artifact_path.parent.mkdir()
            artifact_path.write_bytes(b"\x7fELF" + b"\x00" * 8)

            configure_sse_bridge.provide_bridge_binary(repository_root, runtime_user_entry)

            self.assertTrue(configure_sse_bridge.BRIDGE_BINARY_PATH.is_file())
            self.assertTrue(os.access(configure_sse_bridge.BRIDGE_BINARY_PATH, os.X_OK))
            self.assertFalse((configure_sse_bridge.SERVICE_INSTALL_DIRECTORY / ".sse_bridge.staged").exists())


if __name__ == "__main__":
    unittest.main()
