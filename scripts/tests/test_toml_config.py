import importlib.util
import tempfile
import tomllib
import unittest
from pathlib import Path


def load_toml_config_module():
    script_path = Path(__file__).resolve().parents[1] / "configure" / "toml_config.py"
    module_spec = importlib.util.spec_from_file_location("toml_config", script_path)
    toml_config = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(toml_config)
    return toml_config


class SetConfigValuesTest(unittest.TestCase):
    def setUp(self):
        self.toml_config = load_toml_config_module()
        self.temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_dir.cleanup)

    def write_config(self, contents):
        config_path = Path(self.temp_dir.name) / "config.toml"
        config_path.write_text(contents, encoding="utf-8")
        return config_path

    def test_replacing_existing_key_preserves_comments(self):
        config_path = self.write_config(
            "# comentario de archivo\n"
            "[db]\n"
            "# comentario de seccion\n"
            "host = \"old-host\"\n"
            "port = 9042\n"
        )

        self.toml_config.set_config_values(config_path, {"db.host": "new-host"})

        rendered_text = config_path.read_text(encoding="utf-8")
        self.assertIn("# comentario de archivo", rendered_text)
        self.assertIn("# comentario de seccion", rendered_text)
        self.assertIn('host = "new-host"', rendered_text)

    def test_adding_key_to_existing_section_stays_inside_it(self):
        config_path = self.write_config(
            "[search]\n"
            "url = \"127.0.0.1:14446\"\n"
            "\n"
            "[logs]\n"
            "full = false\n"
        )

        self.toml_config.set_config_values(config_path, {"search.password": "secret"})

        parsed = tomllib.loads(config_path.read_text(encoding="utf-8"))
        self.assertEqual(parsed["search"]["password"], "secret")
        self.assertNotIn("password", parsed["logs"])

    def test_adding_new_section_does_not_contaminate_trailing_array_table(self):
        config_path = self.write_config(
            "[logs]\n"
            "full = false\n"
            "\n"
            "[[servers]]\n"
            "host = \"127.0.0.1\"\n"
        )

        self.toml_config.set_config_values(config_path, {"search.url": "127.0.0.1:14446"})

        parsed = tomllib.loads(config_path.read_text(encoding="utf-8"))
        self.assertEqual(parsed["search"]["url"], "127.0.0.1:14446")
        self.assertEqual(list(parsed["servers"][0].keys()), ["host"])

    def test_values_with_quotes_and_backslashes_survive_round_trip(self):
        config_path = self.write_config("[smtp]\npassword = \"old\"\n")

        tricky_value = 'weird\\path"with"quotes'
        self.toml_config.set_config_values(config_path, {"smtp.password": tricky_value})

        parsed = tomllib.loads(config_path.read_text(encoding="utf-8"))
        self.assertEqual(parsed["smtp"]["password"], tricky_value)

    def test_integers_are_written_without_quotes(self):
        config_path = self.write_config("[server]\nport = 1\n")

        self.toml_config.set_config_values(config_path, {"server.port": 14010})

        rendered_text = config_path.read_text(encoding="utf-8")
        self.assertIn("port = 14010", rendered_text)
        self.assertNotIn('"14010"', rendered_text)


if __name__ == "__main__":
    unittest.main()
