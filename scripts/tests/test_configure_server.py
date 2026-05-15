import importlib.util
import unittest
from pathlib import Path


def load_configure_server_module():
    # Load the script directly so the test covers the deployed entrypoint code.
    script_path = Path(__file__).resolve().parents[1] / "configure_server.py"
    module_spec = importlib.util.spec_from_file_location("configure_server", script_path)
    configure_server = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(configure_server)
    return configure_server


class ConfigureServerNginxTemplateTest(unittest.TestCase):
    def test_nginx_template_preserves_existing_certbot_tls_lines(self):
        configure_server = load_configure_server_module()
        existing_config = (Path(__file__).resolve().parent / "nginx.conf").read_text()

        rendered_config = configure_server.build_http3_nginx_configuration(
            "genix-dev-api-2.un.pe",
            existing_config,
        )

        self.assertIn("ssl_certificate /etc/letsencrypt/live/genix-dev-api-2.un.pe/fullchain.pem;", rendered_config)
        self.assertIn("ssl_certificate_key /etc/letsencrypt/live/genix-dev-api-2.un.pe/privkey.pem;", rendered_config)
        self.assertIn("listen 443 quic reuseport;", rendered_config)
        self.assertIn("add_header Alt-Svc 'h3=\":443\"; ma=86400';", rendered_config)
        self.assertIn("proxy_http_version 1.1;", rendered_config)
        self.assertIn("proxy_set_header Upgrade $http_upgrade;", rendered_config)
        self.assertIn("proxy_set_header Connection $connection_upgrade;", rendered_config)
        self.assertIn("proxy_read_timeout 3600s;", rendered_config)


if __name__ == "__main__":
    unittest.main()
