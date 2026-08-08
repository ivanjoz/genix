"""Lectura y escritura puntual de config.toml sin dependencias externas.

El escritor edita líneas en su sitio en vez de reserializar el archivo: los comentarios
son la razón de ser del formato TOML aquí, y un round-trip con tomllib los borraría en
cada ejecución de configure_db.py.
"""
import re
import tomllib
from pathlib import Path


def read_config(config_path):
    """Devuelve el archivo parseado como dict anidado. {} si no existe."""
    config_file = Path(config_path)
    if not config_file.exists():
        return {}
    with open(config_file, "rb") as config_handle:
        return tomllib.load(config_handle)


def get_config_value(config_data, dotted_key, default=None):
    """Lee 'seccion.clave' de un dict anidado. Evita .get().get() encadenados."""
    current_value = config_data
    for key_part in dotted_key.split("."):
        if not isinstance(current_value, dict) or key_part not in current_value:
            return default
        current_value = current_value[key_part]
    return current_value


def format_toml_value(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    escaped_value = str(value).replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped_value}"'


def set_config_values(config_path, value_updates):
    """Escribe {'seccion.clave': valor} en config.toml conservando el resto del archivo.

    Reemplaza la clave dentro de su sección si ya está; si la sección existe pero la clave
    no, la inserta al final de esa sección; si la sección no existe, añade un bloque
    [seccion] al final del archivo.

    Nunca añade claves sueltas al final: en TOML quedarían dentro del último [[endpoints]]
    o [[servers]], que por diseño van al final del archivo.
    """
    config_file = Path(config_path)
    file_lines = config_file.read_text(encoding="utf-8").splitlines()

    for dotted_key, new_value in value_updates.items():
        section_name, _, key_name = dotted_key.rpartition(".")
        formatted_value = format_toml_value(new_value)
        new_line = f"{key_name} = {formatted_value}"

        section_header_pattern = re.compile(r"^\s*\[([^\[\]]+)\]\s*$")
        key_pattern = re.compile(rf"^\s*{re.escape(key_name)}\s*=")

        current_section = ""
        key_line_index = None
        section_last_line_index = None

        for line_index, line_text in enumerate(file_lines):
            header_match = section_header_pattern.match(line_text)
            if header_match:
                current_section = header_match.group(1).strip()
                continue
            if line_text.strip().startswith("[["):
                current_section = "\x00"  # tabla de array: nunca es destino de escritura
                continue
            if current_section != section_name:
                continue
            # Última línea con contenido de la sección: ahí se inserta si falta la clave.
            if line_text.strip():
                section_last_line_index = line_index
            if key_pattern.match(line_text):
                key_line_index = line_index

        if key_line_index is not None:
            file_lines[key_line_index] = new_line
        elif section_last_line_index is not None:
            file_lines.insert(section_last_line_index + 1, new_line)
        elif section_name:
            file_lines.extend(["", f"[{section_name}]", new_line])
        else:
            raise ValueError(f"No se puede añadir la clave raíz {key_name} al final del archivo")

    config_file.write_text("\n".join(file_lines) + "\n", encoding="utf-8")
