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


def find_new_section_index(file_lines):
    """Dónde insertar una sección nueva: antes de las tablas de array, no al final del archivo.

    Las tablas [[...]] van al final por convención de este config y llevan encima el bloque de
    comentarios que explica por qué; colocar una sección después de ellas deja ese bloque
    separado de lo que documenta. Se retrocede sobre los comentarios y líneas en blanco que
    preceden a la primera [[...]] para insertar antes del bloque entero.
    """
    for line_index, line_text in enumerate(file_lines):
        if not line_text.strip().startswith("[["):
            continue
        block_start_index = line_index
        while block_start_index > 0:
            preceding_line = file_lines[block_start_index - 1].strip()
            if preceding_line and not preceding_line.startswith("#"):
                break
            block_start_index -= 1
        return block_start_index
    return len(file_lines)


def set_config_values(config_path, value_updates):
    """Escribe {'seccion.clave': valor} en config.toml conservando el resto del archivo.

    Reemplaza la clave dentro de su sección si ya está; si la sección existe pero la clave
    no, la inserta al final de esa sección; si la sección no existe, añade un bloque
    [seccion] antes de las tablas de array.

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
        section_header_index = None
        section_last_key_index = None

        for line_index, line_text in enumerate(file_lines):
            header_match = section_header_pattern.match(line_text)
            if header_match:
                current_section = header_match.group(1).strip()
                if current_section == section_name:
                    section_header_index = line_index
                continue
            if line_text.strip().startswith("[["):
                current_section = "\x00"  # tabla de array: nunca es destino de escritura
                continue
            if current_section != section_name:
                continue
            # Sólo cuentan las líneas con clave: los comentarios que preceden a la siguiente
            # sección todavía caen dentro de ésta al recorrer el archivo, y insertar después de
            # ellos separaría el comentario de lo que documenta.
            if not line_text.strip() or line_text.lstrip().startswith("#"):
                continue
            section_last_key_index = line_index
            if key_pattern.match(line_text):
                key_line_index = line_index

        if key_line_index is not None:
            file_lines[key_line_index] = new_line
        elif section_last_key_index is not None:
            file_lines.insert(section_last_key_index + 1, new_line)
        elif section_header_index is not None:
            # Sección declarada pero vacía: la clave va inmediatamente bajo su cabecera.
            file_lines.insert(section_header_index + 1, new_line)
        elif section_name:
            new_section_index = find_new_section_index(file_lines)
            file_lines[new_section_index:new_section_index] = ["", f"[{section_name}]", new_line]
        else:
            raise ValueError(f"No se puede añadir la clave raíz {key_name} al final del archivo")

    config_file.write_text("\n".join(file_lines) + "\n", encoding="utf-8")
