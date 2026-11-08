import importlib.util
import io
import os
import pwd
import tempfile
import tomllib
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest import mock


def load_configure_server_module():
    # Load the script directly so the test covers the deployed entrypoint code.
    script_path = Path(__file__).resolve().parents[1] / "configure" / "configure_server.py"
    module_spec = importlib.util.spec_from_file_location("configure_server", script_path)
    configure_server = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(configure_server)
    return configure_server


class ConfigureServerHAProxyTemplateTest(unittest.TestCase):
    def test_main_configuration_without_a_certificate_serves_plain_http_only(self):
        configure_server = load_configure_server_module()

        main_configuration = configure_server.build_haproxy_main_configuration("ubuntu", False, "native")

        self.assertIn("group ubuntu", main_configuration)
        self.assertIn("bind :80", main_configuration)
        # `crt <directory>` refuses an empty directory, so no :443 before the first certificate.
        self.assertNotIn(":443", main_configuration)
        self.assertNotIn("redirect scheme https", main_configuration)
        self.assertIn("use_backend acme_certbot if acme_challenge", main_configuration)
        self.assertIn("server certbot 127.0.0.1:8888", main_configuration)
        self.assertIn("use_backend site_%[req.hdr(host),host_only,lower]", main_configuration)

    def test_main_configuration_with_a_certificate_serves_http3_and_redirects(self):
        configure_server = load_configure_server_module()

        native_configuration = configure_server.build_haproxy_main_configuration("ubuntu", True, "native")
        compat_configuration = configure_server.build_haproxy_main_configuration("ubuntu", True, "compat")
        tcp_only_configuration = configure_server.build_haproxy_main_configuration("ubuntu", True, None)

        self.assertIn("bind :443 ssl crt /etc/haproxy/certs/ alpn h2,http/1.1 allow-0rtt", native_configuration)
        self.assertIn("bind quic4@:443 ssl crt /etc/haproxy/certs/ alpn h3 allow-0rtt", native_configuration)
        self.assertIn('alt-svc "h3=\\":443\\"; ma=86400"', native_configuration)
        self.assertIn("redirect scheme https code 301 if !{ ssl_fc } !acme_challenge", native_configuration)
        # Replayable early data is limited to GET and HEAD.
        self.assertIn("wait-for-handshake if { ssl_fc_has_early } !{ method GET HEAD }", native_configuration)
        self.assertNotIn("limited-quic", native_configuration)
        # The OpenSSL compatibility layer needs limited-quic and cannot do 0-RTT over QUIC.
        self.assertIn("limited-quic", compat_configuration)
        self.assertIn("bind quic4@:443 ssl crt /etc/haproxy/certs/ alpn h3\n", compat_configuration)
        # Without QUIC there is no UDP listener, and h3 is not advertised.
        self.assertNotIn("quic4@", tcp_only_configuration)
        self.assertNotIn("alt-svc", tcp_only_configuration)

    def test_api_site_without_the_agent_forwards_to_the_backend(self):
        configure_server = load_configure_server_module()

        site_configuration = configure_server.build_api_site_configuration(
            "genix-api-4.un.pe", "http", "unix@/run/genix/api.sock", None
        )

        self.assertIn("backend site_genix-api-4.un.pe", site_configuration)
        self.assertIn("server genix unix@/run/genix/api.sock", site_configuration)
        self.assertIn("http-request del-header x-fareward- -m beg", site_configuration)
        self.assertIn("http-request set-header x-real-ip %[src]", site_configuration)
        self.assertIn("if METH_OPTIONS", site_configuration)
        self.assertIn("timeout server 3600s", site_configuration)
        self.assertNotIn("filter spoe", site_configuration)
        self.assertNotIn("mode spop", site_configuration)

    def test_api_site_with_an_https_upstream_uses_tls_to_it(self):
        configure_server = load_configure_server_module()

        site_configuration = configure_server.build_api_site_configuration(
            "genix-api-4.un.pe", "https", "10.0.0.7:8443", None
        )

        self.assertIn("server genix 10.0.0.7:8443 ssl verify none alpn http/1.1", site_configuration)

    def test_api_site_in_shadow_mode_only_logs_the_verdict(self):
        configure_server = load_configure_server_module()

        site_configuration = configure_server.build_api_site_configuration(
            "genix-api-4.un.pe", "http", "unix@/run/genix/api.sock",
            {"socket": "/run/fareward/spoe.sock", "enforce": False},
        )

        self.assertIn(
            "filter spoe engine fareward config /etc/haproxy/spoe/genix-api-4.un.pe.conf", site_configuration
        )
        self.assertIn("http-request capture var(txn.fw.status) id 0", site_configuration)
        self.assertIn("backend spoe_genix-api-4.un.pe", site_configuration)
        self.assertIn("server fareward unix@/run/fareward/spoe.sock", site_configuration)
        self.assertNotIn("status 429", site_configuration)
        self.assertNotIn("x-fareward-admitted", site_configuration)
        # Shadow mode: the backend decides, so the client is shown nothing of the agent's.
        self.assertNotIn("x-fareward-usage", site_configuration)

    def test_api_site_in_enforce_mode_refuses_and_fails_closed(self):
        configure_server = load_configure_server_module()

        site_configuration = configure_server.build_api_site_configuration(
            "genix-api-4.un.pe", "http", "unix@/run/genix/api.sock",
            {"socket": "/run/fareward/spoe.sock", "enforce": True},
        )

        self.assertIn("http-request deny deny_status 503 if { var(txn.fw.error) -m found }", site_configuration)
        self.assertIn("http-request return status 411", site_configuration)
        # A credit refusal: an HTML page worded by the agent, Retry-After and the usage header.
        refusal_429 = next(
            line for line in site_configuration.splitlines() if "status 429" in line
        )
        self.assertIn('content-type "text/html; charset=utf-8" lf-string', refusal_429)
        self.assertIn("%[var(txn.fw.spent)] créditos %[var(txn.fw.window)]", refusal_429)
        self.assertIn("hdr retry-after %[var(txn.fw.retry)]", refusal_429)
        self.assertIn("hdr x-fareward-usage %[var(txn.fw.usage)]", refusal_429)
        self.assertTrue(refusal_429.strip().endswith("if { var(txn.fw.status) -m int eq 429 }"))
        self.assertIn(
            "http-request deny deny_status 403 hdr x-fareward-usage %[var(txn.fw.usage)] if",
            site_configuration,
        )
        # And on what the backend answers.
        self.assertIn(
            "http-response set-header x-fareward-usage %[var(txn.fw.usage)] if { var(txn.fw.usage) -m found }",
            site_configuration,
        )
        self.assertIn("http-request set-header x-fareward-admitted 1", site_configuration)
        # The client's own x-fareward-* headers are deleted before the edge sets its own.
        self.assertLess(
            site_configuration.index("del-header x-fareward-"),
            site_configuration.index("set-header x-fareward-admitted"),
        )

    def test_spoe_engine_skips_preflights(self):
        configure_server = load_configure_server_module()

        spoe_configuration = configure_server.build_api_spoe_configuration("genix-api-4.un.pe")

        self.assertIn("use-backend  spoe_genix-api-4.un.pe", spoe_configuration)
        self.assertIn("event on-backend-http-request if !METH_OPTIONS", spoe_configuration)
        self.assertIn("option       set-on-error    error", spoe_configuration)

    def test_certificate_deploy_hook_joins_the_chain_and_key(self):
        configure_server = load_configure_server_module()

        hook_script = configure_server.build_certificate_deploy_hook()

        self.assertIn('cat "$RENEWED_LINEAGE/fullchain.pem" "$RENEWED_LINEAGE/privkey.pem"', hook_script)
        self.assertIn('mv "/etc/haproxy/certs/$domain.pem.tmp" "/etc/haproxy/certs/$domain.pem"', hook_script)
        self.assertIn("systemctl reload haproxy", hook_script)

    def test_an_nginx_issued_certificate_is_not_renewed_through_haproxy(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            renewal_directory = Path(temporary_directory)
            configure_server.LETSENCRYPT_RENEWAL_DIRECTORY = renewal_directory
            (renewal_directory / "nginx.example.com.conf").write_text(
                "[renewalparams]\nauthenticator = nginx\nkey_type = ecdsa\n", encoding="utf-8"
            )
            (renewal_directory / "haproxy.example.com.conf").write_text(
                "[renewalparams]\nauthenticator = standalone\nhttp01_port = 8888\n"
                "renew_hook = /usr/local/sbin/genix-haproxy-certificate\n",
                encoding="utf-8",
            )

            self.assertFalse(configure_server.certificate_is_renewed_through_haproxy("nginx.example.com"))
            self.assertTrue(configure_server.certificate_is_renewed_through_haproxy("haproxy.example.com"))
            self.assertFalse(configure_server.certificate_is_renewed_through_haproxy("missing.example.com"))


class ConfigureServerSpoeSettingsTest(unittest.TestCase):
    def test_no_socket_means_no_agent(self):
        configure_server = load_configure_server_module()

        with redirect_stdout(io.StringIO()):
            self.assertIsNone(configure_server.resolve_spoe_settings({"spoe": {"mode": "enforce"}}))

    def test_mode_defaults_to_shadow(self):
        configure_server = load_configure_server_module()

        with redirect_stdout(io.StringIO()):
            spoe_settings = configure_server.resolve_spoe_settings({
                "spoe": {"socket": "/run/fareward/spoe.sock"},
                "fareward": {"routes_file": "/var/lib/genix/fareward-routes.toml"},
            })

        self.assertEqual(spoe_settings, {"socket": "/run/fareward/spoe.sock", "enforce": False})

    def test_unusable_settings_stop_the_run(self):
        configure_server = load_configure_server_module()

        for project_credentials in (
            {"spoe": {"socket": "/run/fareward/spoe.sock", "mode": "on"},
             "fareward": {"routes_file": "/var/lib/genix/fareward-routes.toml"}},
            # The agent cannot start without the route table the backend writes.
            {"spoe": {"socket": "/run/fareward/spoe.sock"}},
            # The backend's unit can only write inside /var/lib/genix.
            {"spoe": {"socket": "/run/fareward/spoe.sock"}, "fareward": {"routes_file": "/tmp/routes.toml"}},
        ):
            with redirect_stdout(io.StringIO()), io.StringIO() as error_output, \
                    mock.patch.object(configure_server.sys, "stderr", error_output), \
                    self.assertRaises(SystemExit):
                configure_server.resolve_spoe_settings(project_credentials)

    def test_trust_edge_headers_follows_enforce_mode(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            config_path = Path(temporary_directory) / "config.toml"
            config_path.write_text("[fareward]\nport = 14013\n", encoding="utf-8")

            configure_server.sync_edge_trust_setting({}, config_path, {"socket": "/s", "enforce": True})
            self.assertTrue(tomllib.loads(config_path.read_text())["fareward"]["trust_edge_headers"])

            enforced_credentials = tomllib.loads(config_path.read_text())
            configure_server.sync_edge_trust_setting(enforced_credentials, config_path, None)
            self.assertFalse(tomllib.loads(config_path.read_text())["fareward"]["trust_edge_headers"])


class ConfigureServerCredentialsTest(unittest.TestCase):
    def test_upstream_without_scheme_becomes_http_upstream(self):
        configure_server = load_configure_server_module()

        with redirect_stdout(io.StringIO()):
            edge_settings = configure_server.extract_edge_settings(
                {"server": {"domain": "genix-api-4.un.pe", "upstream": "100.64.0.2:14010"}}
            )

        self.assertEqual(edge_settings["domain"], "genix-api-4.un.pe")
        self.assertEqual(edge_settings["upstream_scheme"], "http")
        self.assertEqual(edge_settings["upstream_server"], "100.64.0.2:14010")

    def test_upstream_keeps_explicit_scheme(self):
        configure_server = load_configure_server_module()

        with redirect_stdout(io.StringIO()):
            edge_settings = configure_server.extract_edge_settings(
                {"server": {"domain": "genix-api-4.un.pe", "upstream": "https://10.0.0.7:8443"}}
            )

        self.assertEqual(edge_settings["upstream_scheme"], "https")
        self.assertEqual(edge_settings["upstream_server"], "10.0.0.7:8443")

    def test_full_mode_forwards_to_the_unix_socket_and_ignores_upstream(self):
        configure_server = load_configure_server_module()

        with redirect_stdout(io.StringIO()):
            edge_settings = configure_server.extract_edge_settings(
                {"server": {"domain": "genix-api-4.un.pe", "upstream": "127.0.0.1:14010"}},
                backend_is_local=True,
            )

        self.assertEqual(edge_settings["upstream_server"], "unix@/run/genix/api.sock")
        self.assertEqual(edge_settings["prompted_values"], {})

    def test_server_port_is_read_from_credentials(self):
        configure_server = load_configure_server_module()

        self.assertEqual(configure_server.extract_server_port({"server": {"port": 14010}}), (14010, {}))
        self.assertEqual(configure_server.extract_server_port({"server": {"port": "14010"}}), (14010, {}))

    def test_validators_reject_unusable_values(self):
        configure_server = load_configure_server_module()

        self.assertIsNone(configure_server.validate_domain("  ")[0])
        self.assertIsNone(configure_server.validate_domain("not a domain")[0])
        self.assertIsNone(configure_server.validate_domain("localhost")[0])
        self.assertEqual(configure_server.validate_domain(" Genix-API-4.un.pe. ")[0], "genix-api-4.un.pe")

        self.assertIsNone(configure_server.validate_upstream("100.64.0.2")[0])
        self.assertIsNone(configure_server.validate_upstream("100.64.0.2:notaport")[0])
        self.assertIsNone(configure_server.validate_upstream("ftp://100.64.0.2:14010")[0])
        self.assertEqual(configure_server.validate_upstream(" 100.64.0.2:14010 ")[0], "100.64.0.2:14010")

        self.assertIsNone(configure_server.validate_server_port("0")[0])
        self.assertIsNone(configure_server.validate_server_port("70000")[0])
        self.assertIsNone(configure_server.validate_server_port("14010a")[0])
        self.assertEqual(configure_server.validate_server_port(" 14010 ")[0], 14010)

    def test_missing_values_are_requested_on_the_terminal(self):
        configure_server = load_configure_server_module()

        # First answer is rejected by the validator, so the prompt must repeat.
        with mock.patch.object(configure_server.sys.stdin, "isatty", return_value=True), \
                mock.patch.object(configure_server, "input", create=True, side_effect=["nope", "14010"]), \
                redirect_stdout(io.StringIO()):
            server_port, prompted_values = configure_server.extract_server_port({})

        self.assertEqual(server_port, 14010)
        self.assertEqual(prompted_values, {"server.port": 14010})

    def test_invalid_stored_value_is_replaced_by_the_prompted_one(self):
        configure_server = load_configure_server_module()

        with mock.patch.object(configure_server.sys.stdin, "isatty", return_value=True), \
                mock.patch.object(configure_server, "input", create=True, side_effect=["100.64.0.2:14010"]), \
                redirect_stdout(io.StringIO()):
            edge_settings = configure_server.extract_edge_settings(
                {"server": {"domain": "genix-api-4.un.pe", "upstream": "100.64.0.2"}}
            )

        self.assertEqual(edge_settings["upstream_server"], "100.64.0.2:14010")
        self.assertEqual(edge_settings["prompted_values"], {"server.upstream": "100.64.0.2:14010"})

    def test_missing_value_without_a_terminal_fails(self):
        configure_server = load_configure_server_module()

        with mock.patch.object(configure_server.sys.stdin, "isatty", return_value=False), \
                redirect_stdout(io.StringIO()), self.assertRaises(SystemExit):
            configure_server.extract_server_port({})

    def test_missing_credentials_file_loads_as_empty(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            missing_path = Path(temporary_directory) / "config.toml"
            self.assertEqual(configure_server.load_project_credentials(missing_path), {})

    def test_prompted_values_are_merged_into_credentials_file(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            config_path = Path(temporary_directory) / "config.toml"
            config_path.write_text('app_name = "genix-3"\n', encoding="utf-8")

            with mock.patch.object(configure_server.sys.stdin, "isatty", return_value=True), \
                    mock.patch.object(configure_server, "input", create=True, return_value="y"), \
                    redirect_stdout(io.StringIO()):
                configure_server.persist_prompted_credentials(
                    config_path, {"server.domain": "genix-api-4.un.pe", "server.port": 14010}
                )

            with open(config_path, "rb") as config_file:
                stored_config = tomllib.load(config_file)

        self.assertEqual(stored_config["app_name"], "genix-3")
        self.assertEqual(stored_config["server"]["domain"], "genix-api-4.un.pe")
        self.assertEqual(stored_config["server"]["port"], 14010)

    def test_declining_the_save_leaves_the_credentials_file_untouched(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            config_path = Path(temporary_directory) / "config.toml"
            config_path.write_text('app_name = "genix-3"\n', encoding="utf-8")

            with mock.patch.object(configure_server.sys.stdin, "isatty", return_value=True), \
                    mock.patch.object(configure_server, "input", create=True, return_value="n"), \
                    redirect_stdout(io.StringIO()):
                configure_server.persist_prompted_credentials(config_path, {"server.port": 14010})

            with open(config_path, "rb") as config_file:
                stored_config = tomllib.load(config_file)

        self.assertEqual(stored_config, {"app_name": "genix-3"})

    def test_empty_placeholder_binary_is_removed_and_none_is_created(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            binary_path = Path(temporary_directory) / "genix_app"
            configure_server.SERVICE_BINARY_PATH = binary_path

            # Nothing deployed yet: the script must not fabricate a file.
            configure_server.remove_empty_placeholder_binary()
            self.assertFalse(binary_path.exists())

            # A zero-byte file can only be a placeholder from an older run, so it is cleaned up.
            binary_path.touch()
            configure_server.remove_empty_placeholder_binary()
            self.assertFalse(binary_path.exists())

            # A real binary is left alone.
            binary_path.write_bytes(b"\x7fELF-not-really")
            configure_server.remove_empty_placeholder_binary()
            self.assertTrue(binary_path.exists())

    def test_only_real_elf_files_count_as_usable_executables(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)

            empty_path = temporary_path / "empty"
            empty_path.touch()
            self.assertFalse(configure_server.is_usable_executable(empty_path))

            script_path = temporary_path / "script.sh"
            script_path.write_text("#!/bin/sh\necho hi\n", encoding="utf-8")
            self.assertFalse(configure_server.is_usable_executable(script_path))

            self.assertFalse(configure_server.is_usable_executable(temporary_path / "missing"))
            self.assertFalse(configure_server.is_usable_executable(temporary_path))

            elf_path = temporary_path / "genix_app"
            elf_path.write_bytes(b"\x7fELF" + b"\x00" * 64)
            self.assertTrue(configure_server.is_usable_executable(elf_path))

    def test_backend_source_is_detected_only_with_go_module_and_main(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            self.assertIsNone(configure_server.detect_backend_source_directory(repository_root))

            backend_directory = repository_root / "backend"
            backend_directory.mkdir()
            (backend_directory / "go.mod").write_text("module app\n", encoding="utf-8")
            self.assertIsNone(configure_server.detect_backend_source_directory(repository_root))

            (backend_directory / "main.go").write_text("package main\n", encoding="utf-8")
            self.assertEqual(
                configure_server.detect_backend_source_directory(repository_root), backend_directory
            )

    def test_prebuilt_binary_search_order_and_crash_when_nothing_is_found(self):
        configure_server = load_configure_server_module()
        architecture = configure_server.resolve_go_architecture()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            configure_server.SERVICE_BINARY_PATH = repository_root / "installed" / "genix_app"
            configure_server.SERVICE_BINARY_PATH.parent.mkdir()

            self.assertIsNone(configure_server.find_prebuilt_binary(repository_root))

            # A deploy artifact under tmp/ is picked up.
            artifact_path = repository_root / "tmp" / f"genix_app_linux_{architecture}"
            artifact_path.parent.mkdir()
            artifact_path.write_bytes(b"\x7fELF" + b"\x00" * 8)
            self.assertEqual(configure_server.find_prebuilt_binary(repository_root), artifact_path)

            # An already-installed binary wins over the artifact.
            configure_server.SERVICE_BINARY_PATH.write_bytes(b"\x7fELF" + b"\x00" * 8)
            self.assertEqual(
                configure_server.find_prebuilt_binary(repository_root),
                configure_server.SERVICE_BINARY_PATH,
            )
            # Explicit precompiled mode must deploy the freshly downloaded latest asset.
            self.assertEqual(
                configure_server.find_prebuilt_binary(repository_root, prefer_downloaded=True),
                artifact_path,
            )

    def test_no_source_and_no_binary_crashes(self):
        configure_server = load_configure_server_module()
        runtime_user_entry = pwd.getpwuid(os.getuid())

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            configure_server.SERVICE_BINARY_PATH = repository_root / "installed" / "genix_app"
            configure_server.SERVICE_BINARY_PATH.parent.mkdir()

            with self.assertRaises(SystemExit):
                configure_server.provide_service_binary(repository_root, runtime_user_entry)

    def test_found_binary_is_staged_and_renamed_into_place(self):
        configure_server = load_configure_server_module()
        runtime_user_entry = pwd.getpwuid(os.getuid())

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            configure_server.SERVICE_INSTALL_DIRECTORY = repository_root / "installed"
            configure_server.SERVICE_BINARY_PATH = configure_server.SERVICE_INSTALL_DIRECTORY / "genix_app"
            configure_server.SERVICE_INSTALL_DIRECTORY.mkdir()

            source_path = repository_root / "genix_app"
            source_path.write_bytes(b"\x7fELF" + b"\x00" * 8)

            configure_server.provide_service_binary(repository_root, runtime_user_entry)

            self.assertTrue(configure_server.SERVICE_BINARY_PATH.is_file())
            self.assertTrue(os.access(configure_server.SERVICE_BINARY_PATH, os.X_OK))
            # The staging file must not survive the rename.
            self.assertFalse((configure_server.SERVICE_INSTALL_DIRECTORY / ".genix_app.staged").exists())

    def test_local_replace_directories_are_parsed_from_go_mod(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory:
            backend_directory = Path(temporary_directory) / "backend"
            backend_directory.mkdir()
            (backend_directory / "go.mod").write_text(
                "module app\n"
                "\n"
                "replace github.com/gocql/gocql v1.6.0 => github.com/scylladb/gocql v1.13.0\n"
                "\n"
                "replace github.com/ivanjoz/genix-orm => ./genix-orm\n"
                "\n"
                "replace github.com/ivanjoz/genix-orm/db => ./genix-orm/db\n"
                "\n"
                "replace (\n"
                "    example.com/blocked => ../sibling\n"
                "    example.com/remote => example.com/other v1.2.3\n"
                ")\n"
                "// replace example.com/commented => ./commented\n",
                encoding="utf-8",
            )

            local_replace_directories = configure_server.parse_local_replace_directories(backend_directory)

        # Only filesystem replacements count: module-to-module and commented ones are ignored.
        self.assertEqual(
            [directory.name for directory in local_replace_directories],
            ["genix-orm", "db", "sibling"],
        )

    def test_no_replace_targets_means_no_work(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            backend_directory = Path(temporary_directory) / "backend"
            backend_directory.mkdir()
            (backend_directory / "go.mod").write_text("module app\n", encoding="utf-8")

            self.assertEqual(configure_server.parse_local_replace_directories(backend_directory), [])
            # Must not fail or shell out when there is nothing to populate.
            configure_server.ensure_local_replace_directories(
                backend_directory, Path(temporary_directory), None
            )

    def test_unpopulated_replace_target_that_is_not_a_submodule_crashes(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            backend_directory = repository_root / "backend"
            backend_directory.mkdir()
            (backend_directory / "go.mod").write_text(
                "module app\n\nreplace github.com/ivanjoz/genix-orm => ./genix-orm\n", encoding="utf-8"
            )
            (backend_directory / "genix-orm").mkdir()

            with self.assertRaises(SystemExit):
                configure_server.ensure_local_replace_directories(backend_directory, repository_root, None)

    def test_ssh_remotes_convert_to_https(self):
        configure_server = load_configure_server_module()

        self.assertEqual(
            configure_server.convert_ssh_remote_to_https("git@github.com:ivanjoz/genix-orm.git"),
            "https://github.com/ivanjoz/genix-orm.git",
        )
        self.assertEqual(
            configure_server.convert_ssh_remote_to_https("ssh://git@github.com/ivanjoz/genix-orm.git"),
            "https://github.com/ivanjoz/genix-orm.git",
        )
        self.assertEqual(
            configure_server.convert_ssh_remote_to_https("git@gitlab.com:group/sub/repo.git"),
            "https://gitlab.com/group/sub/repo.git",
        )
        # Already HTTPS, or not a remote at all: nothing to rewrite.
        self.assertIsNone(
            configure_server.convert_ssh_remote_to_https("https://github.com/ivanjoz/genix-orm.git")
        )
        self.assertIsNone(configure_server.convert_ssh_remote_to_https("../relative/path"))
        self.assertIsNone(configure_server.convert_ssh_remote_to_https(""))

    def test_failed_ssh_clone_retries_over_https(self):
        configure_server = load_configure_server_module()
        submodule_details = {"name": "backend/genix-orm", "url": "git@github.com:ivanjoz/genix-orm.git"}

        class FailedResult:
            returncode = 1

        class SucceededResult:
            returncode = 0

        recorded_commands = []

        def record_command(command_arguments, working_directory=None, stream_output=False, allow_failure=False):
            recorded_commands.append(command_arguments)
            return SucceededResult()

        with mock.patch.object(configure_server, "run_command", side_effect=record_command), \
                mock.patch.object(configure_server.os, "geteuid", return_value=0), \
                mock.patch.object(
                    configure_server, "run_git_submodule_update",
                    side_effect=[FailedResult(), SucceededResult()],
                ) as submodule_update_mock, \
                redirect_stdout(io.StringIO()):
            configure_server.initialise_git_submodule(
                Path("/repo"), "backend/genix-orm", submodule_details, "homelab"
            )

        # The URL override targets .git/config only, never .gitmodules.
        self.assertEqual(
            recorded_commands,
            [[
                "sudo", "-u", "homelab", "-H", "git", "config",
                "submodule.backend/genix-orm.url", "https://github.com/ivanjoz/genix-orm.git",
            ]],
        )
        # First attempt initialises; the retry must not pass --init again.
        self.assertEqual(submodule_update_mock.call_count, 2)
        self.assertTrue(submodule_update_mock.call_args_list[0].kwargs["initialise"])
        self.assertFalse(submodule_update_mock.call_args_list[1].kwargs["initialise"])

    def test_successful_ssh_clone_does_not_rewrite_the_url(self):
        configure_server = load_configure_server_module()
        submodule_details = {"name": "backend/genix-orm", "url": "git@github.com:ivanjoz/genix-orm.git"}

        class SucceededResult:
            returncode = 0

        with mock.patch.object(configure_server, "run_command") as run_command_mock, \
                mock.patch.object(
                    configure_server, "run_git_submodule_update", return_value=SucceededResult()
                ) as submodule_update_mock, \
                redirect_stdout(io.StringIO()):
            configure_server.initialise_git_submodule(
                Path("/repo"), "backend/genix-orm", submodule_details, "homelab"
            )

        self.assertEqual(submodule_update_mock.call_count, 1)
        run_command_mock.assert_not_called()

    def test_git_commands_never_wait_on_a_prompt(self):
        configure_server = load_configure_server_module()

        class SucceededResult:
            returncode = 0

        with mock.patch.object(
            configure_server, "run_command", return_value=SucceededResult()
        ) as run_command_mock:
            configure_server.run_git_submodule_update(
                Path("/repo"), "backend/genix-orm", None, initialise=True
            )

        issued_command = run_command_mock.call_args.args[0]
        self.assertIn("GIT_TERMINAL_PROMPT=0", issued_command)
        self.assertIn("GIT_SSH_COMMAND=ssh -o BatchMode=yes", issued_command)

    def test_unprivileged_command_wrapping(self):
        configure_server = load_configure_server_module()
        base_command = ["git", "status"]

        with mock.patch.object(configure_server.os, "geteuid", return_value=0):
            self.assertEqual(
                configure_server.build_unprivileged_command(base_command, "homelab"),
                ["sudo", "-u", "homelab", "-H", "git", "status"],
            )
            # No account to drop to: run as-is rather than guessing.
            self.assertEqual(configure_server.build_unprivileged_command(base_command, None), base_command)

        with mock.patch.object(configure_server.os, "geteuid", return_value=1000):
            self.assertEqual(
                configure_server.build_unprivileged_command(base_command, "homelab"), base_command
            )

    def test_repository_owner_is_preferred_over_sudo_user(self):
        configure_server = load_configure_server_module()
        current_user = pwd.getpwuid(os.getuid())

        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory)

            with mock.patch.dict(configure_server.os.environ, {"SUDO_USER": "someone-else"}):
                # The directory belongs to the current user, so that account wins.
                self.assertEqual(
                    configure_server.detect_unprivileged_username(repository_root), current_user.pw_name
                )

    def test_missing_go_toolchain_crashes_with_source_present(self):
        configure_server = load_configure_server_module()

        with tempfile.TemporaryDirectory() as temporary_directory, redirect_stdout(io.StringIO()):
            repository_root = Path(temporary_directory)
            backend_directory = repository_root / "backend"
            backend_directory.mkdir()
            (backend_directory / "go.mod").write_text("module app\n", encoding="utf-8")
            (backend_directory / "main.go").write_text("package main\n", encoding="utf-8")

            with mock.patch.object(configure_server, "detect_go_binary", return_value=None), \
                    self.assertRaises(SystemExit):
                configure_server.compile_backend_binary(backend_directory, repository_root)

    def test_execution_modes_match_by_index_and_alias(self):
        configure_server = load_configure_server_module()

        self.assertEqual(configure_server.match_execution_mode("1")["mode"], "full")
        self.assertEqual(configure_server.match_execution_mode("2")["mode"], "systemd")
        self.assertEqual(configure_server.match_execution_mode("3")["mode"], "edge")
        self.assertEqual(configure_server.match_execution_mode(" HAPROXY ")["mode"], "edge")
        self.assertIsNone(configure_server.match_execution_mode("4"))

    def test_systemd_unit_exports_the_credentials_server_port(self):
        configure_server = load_configure_server_module()

        unit_contents = configure_server.build_main_service_contents(
            "ubuntu",
            Path("/home/ubuntu/genix/config.toml"),
            14010,
        )

        self.assertIn("Environment=SERVER_PORT=14010", unit_contents)
        self.assertNotIn("SERVER_SOCKET", unit_contents)
        self.assertIn("Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml", unit_contents)

    def test_full_mode_systemd_unit_binds_the_unix_socket(self):
        configure_server = load_configure_server_module()

        unit_contents = configure_server.build_main_service_contents(
            "ubuntu",
            Path("/home/ubuntu/genix/config.toml"),
            None,
        )

        self.assertIn("RuntimeDirectory=genix", unit_contents)
        # The backend writes [fareward].routes_file into /var/lib/genix.
        self.assertIn("StateDirectory=genix", unit_contents)
        self.assertIn("Environment=SERVER_SOCKET=/run/genix/api.sock", unit_contents)
        self.assertNotIn("SERVER_PORT", unit_contents)


if __name__ == "__main__":
    unittest.main()
