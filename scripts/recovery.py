import os, subprocess, json, csv, sys, logging, shutil, traceback, re
from cassandra.cluster import Cluster
from cassandra.auth import PlainTextAuthProvider

# ==========================================
# CONFIGURACIÓN
# ==========================================
KEYSPACE = "genix"
TABLES = ["cajas"] # Puedes agregar todas las que quieras
DATA_DIR = "/var/lib/scylla/data"
# ==========================================

LOG_LEVEL = getattr(logging, os.getenv("RECOVERY_LOG_LEVEL", "INFO").upper(), logging.INFO)
logging.basicConfig(level=LOG_LEVEL, format='[%(asctime)s] %(levelname)s: %(message)s')
log = logging.getLogger("MasterRestore")


def _parse_csv_env_list(environment_value):
    """Parsea una lista CSV desde env; retorna set() si no aplica."""
    if environment_value is None:
        return set()
    normalized = str(environment_value).strip()
    if normalized == "":
        return set()
    return {chunk.strip() for chunk in normalized.split(",") if chunk.strip() != ""}


TRACE_TABLES = _parse_csv_env_list(os.getenv("RECOVERY_TRACE_TABLES"))
TRACE_COLUMNS = _parse_csv_env_list(os.getenv("RECOVERY_TRACE_COLUMNS"))
TRACE_ROWS = int(os.getenv("RECOVERY_TRACE_ROWS", "5") or "5")


def should_trace(table, column=None):
    """Activa logs verbosos solo para tablas/columnas indicadas por env."""
    if TRACE_TABLES and table not in TRACE_TABLES:
        return False
    if column is None:
        return bool(TRACE_TABLES) or bool(TRACE_COLUMNS)
    if TRACE_COLUMNS and column not in TRACE_COLUMNS:
        return False
    return bool(TRACE_TABLES) or bool(TRACE_COLUMNS)


def load_db_credentials():
    """Carga credenciales DB obligatoriamente desde credentials.json."""
    script_directory = os.path.dirname(os.path.abspath(__file__))
    project_root_directory = os.path.dirname(script_directory)
    candidate_paths = [
        os.path.join(os.getcwd(), "credentials.json"),
        os.path.join(project_root_directory, "credentials.json"),
        os.path.join(script_directory, "credentials.json"),
    ]

    credentials_file_path = next((path for path in candidate_paths if os.path.isfile(path)), None)
    if not credentials_file_path:
        raise RuntimeError("[config] credentials.json no encontrado. Debe incluir DB_USER, DB_PASSWORD y DB_PORT.")

    try:
        with open(credentials_file_path, "r") as credentials_file:
            credentials_payload = json.load(credentials_file)
    except Exception as credentials_error:
        raise RuntimeError(f"[config] error leyendo {credentials_file_path}: {credentials_error}")

    username = credentials_payload.get("DB_USER")
    password = credentials_payload.get("DB_PASSWORD")
    port_value = credentials_payload.get("DB_PORT")
    if username in [None, ""] or password in [None, ""] or port_value in [None, ""]:
        raise RuntimeError(
            f"[config] faltan campos obligatorios en {credentials_file_path}. "
            "Requeridos: DB_USER, DB_PASSWORD, DB_PORT."
        )

    try:
        port = int(port_value)
    except (TypeError, ValueError):
        raise RuntimeError(f"[config] DB_PORT inválido en {credentials_file_path}: {port_value!r}. Debe ser entero.")

    log.info(f"[config] credenciales cargadas desde {credentials_file_path}. user={username} port={port}")
    return username, password, port


USER, PASS, PORT = load_db_credentials()

class ScyllaRestore:
    def __init__(self):
        self.auth = PlainTextAuthProvider(username=USER, password=PASS)
        self.cluster = Cluster(['127.0.0.1'], port=PORT, auth_provider=self.auth)
        self.session = self.cluster.connect(KEYSPACE)

    def get_column_metadata(self, table):
        """Extrae el orden y los tipos de datos directamente de Scylla."""
        log.info(f"[*] Analizando esquema de {table}...")
        meta = self.cluster.metadata.keyspaces[KEYSPACE].tables[table]
        
        # Obtenemos el orden físico (PKs -> Clusters -> Regulars)
        pks = [c.name for c in meta.partition_key]
        clusters = [c.name for c in meta.clustering_key]
        regulars = [name for name in meta.columns.keys() if name not in pks and name not in clusters]
        
        ordered_cols = pks + clusters + sorted(regulars)
        col_types = {name: col.cql_type for name, col in meta.columns.items()}

        collection_columns = {name: cql for name, cql in col_types.items() if isinstance(cql, str) and cql.lower().strip().startswith(("set<", "list<"))}
        if collection_columns:
            log.info(f"[schema] tabla={table} columnas_coleccion={collection_columns}")
        else:
            log.info(f"[schema] tabla={table} sin columnas de colección detectadas")
        
        return ordered_cols, col_types

    def _parse_collection_type(self, cql_type):
        """Retorna ('set'|'list'|None, tipo_interno|None) para tipos CQL de colección."""
        normalized_cql_type = (cql_type or "").strip().lower()
        matcher = re.match(r"^(set|list)\s*<\s*([^>]+)\s*>$", normalized_cql_type)
        if not matcher:
            return None, None
        return matcher.group(1), matcher.group(2).strip()

    def _cast_scalar_value(self, raw_value, cql_type):
        """Convierte un valor escalar textual a su tipo Python esperado por el driver."""
        if raw_value is None:
            return None

        normalized_value = str(raw_value).strip()
        if normalized_value == "" or normalized_value.lower() in ["none", "null", "f6"]:
            return None

        normalized_cql_type = (cql_type or "").lower()

        # Blobs (Hex a bytes)
        if "blob" in normalized_cql_type:
            return bytes.fromhex(normalized_value)

        # Numéricos enteros
        if any(t in normalized_cql_type for t in ["tinyint", "smallint", "bigint", "int"]):
            return int(float(normalized_value))

        # Numéricos decimales
        if any(t in normalized_cql_type for t in ["float", "decimal", "double"]):
            return float(normalized_value)

        # Booleanos
        if "boolean" in normalized_cql_type:
            return normalized_value.lower() in ["true", "1", "t"]

        return normalized_value

    def smart_cast(self, value, cql_type):
        """Convierte strings de CSV a objetos Python según el tipo CQL."""
        if value is None:
            return None

        normalized_value = str(value).strip()
        if normalized_value == "" or normalized_value.lower() in ["none", "null", "f6"]:
            return None

        normalized_cql_type = (cql_type or "").lower()
        
        try:
            # Colecciones (set/list): parseamos tipo interno (e.g. set<tinyint>)
            collection_kind, collection_inner_type = self._parse_collection_type(normalized_cql_type)
            if collection_kind in ["set", "list"]:
                if normalized_value == "0":
                    return set() if collection_kind == "set" else []

                raw_items = [item.strip() for item in normalized_value.split("|") if item.strip() != ""]
                casted_items = []
                for raw_item in raw_items:
                    casted_item = self._cast_scalar_value(raw_item, collection_inner_type)
                    if casted_item is not None:
                        casted_items.append(casted_item)
                return set(casted_items) if collection_kind == "set" else casted_items

            return self._cast_scalar_value(normalized_value, normalized_cql_type)
        except Exception as cast_error:
            log.debug(f"[smart_cast] valor={value!r} tipo={cql_type!r} error={cast_error}")
            return None

    def extract_sstable_to_csv(self, table, ordered_cols):
        """Fase Forense: Lee el .db y genera un CSV con soporte para colecciones."""
        log.info(f"[*] Iniciando extracción forense de {table}...")
        
        # Localizar archivo
        base_path = os.path.join(DATA_DIR, KEYSPACE)
        source_file = next((os.path.join(base_path, fld, f) 
                            for fld in os.listdir(base_path) if fld.startswith(f"{table}-") 
                            for f in os.listdir(os.path.join(base_path, fld)) 
                            if f.endswith("-Data.db") and os.path.getsize(os.path.join(base_path, fld, f)) > 0), None)
        
        if not source_file:
            log.error(f"[-] No hay data para {table}"); return None

        # Shadow Table Bypass
        shadow_dir = f"/tmp/shadow_{table}"
        if os.path.exists(shadow_dir): shutil.rmtree(shadow_dir)
        os.makedirs(shadow_dir, exist_ok=True)
        prefix = os.path.basename(source_file).split('-Data.db')[0]
        for f in os.listdir(os.path.dirname(source_file)):
            if prefix in f: os.link(os.path.join(os.path.dirname(source_file), f), os.path.join(shadow_dir, f))

        # Dump a JSON
        json_path = f"/tmp/dump_{table}.json"
        subprocess.run(
            f"scylla sstable dump-data {os.path.join(shadow_dir, os.path.basename(source_file))} > {json_path}",
            shell=True,
            check=True
        )

        # Parsear JSON a CSV
        csv_path = f"recovery_{table}.csv"
        with open(json_path, 'r') as f:
            data = json.load(f)
        
        if isinstance(data, list):
            partitions = data
            log.debug(f"[extract] dump root es lista con {len(partitions)} particiones en tabla={table}")
        elif isinstance(data, dict) and "sstables" in data and isinstance(data["sstables"], dict) and data["sstables"]:
            first_sstable_name = list(data["sstables"].keys())[0]
            partitions = data["sstables"][first_sstable_name]
            log.debug(f"[extract] dump root.sstables detectado tabla={table} sstable={first_sstable_name}")
        else:
            log.error(f"[extract] estructura raíz no soportada tabla={table} tipo={type(data).__name__} keys={list(data.keys()) if isinstance(data, dict) else 'N/A'}")
            return None

        if not isinstance(partitions, list):
            log.error(f"[extract] particiones no es lista tabla={table} tipo={type(partitions).__name__}")
            return None

        rows_to_save = []
        log.info(f"[extract] tabla={table} particiones={len(partitions)}")
        
        for partition_index, part in enumerate(partitions):
            if not isinstance(part, dict):
                log.debug(f"[extract] partición ignorada por tipo inválido tabla={table} idx={partition_index} tipo={type(part).__name__}")
                continue

            pk_val = part.get('key', {}).get('value', '') if isinstance(part.get('key'), dict) else str(part.get('key', ''))
            clustering_elements = part.get('clustering_elements', [])
            if not isinstance(clustering_elements, list):
                log.debug(f"[extract] clustering_elements inválido tabla={table} idx={partition_index} tipo={type(clustering_elements).__name__}")
                continue

            for element_index, el in enumerate(clustering_elements):
                if not isinstance(el, dict):
                    log.debug(f"[extract] elemento de clustering inválido tabla={table} part_idx={partition_index} elem_idx={element_index} tipo={type(el).__name__}")
                    continue

                if el.get('type') == 'clustering-row':
                    cols_raw = el.get('columns', {})
                    if not isinstance(cols_raw, dict):
                        log.debug(f"[extract] columns inválido tabla={table} part_idx={partition_index} elem_idx={element_index} tipo={type(cols_raw).__name__}")
                        continue

                    cells_data = {}
                    for name, info in cols_raw.items():
                        if not isinstance(info, dict):
                            log.debug(f"[extract] columna con payload inválido tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} tipo={type(info).__name__}")
                            continue

                        if should_trace(table, name):
                            log.info(f"[trace.extract.col] tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} keys={list(info.keys())}")

                        if 'value' in info:
                            cells_data[name] = str(info['value'])
                            if should_trace(table, name):
                                log.info(f"[trace.extract.value] tabla={table} columna={name} value={cells_data[name]!r}")
                        elif 'cells' in info: # AQUÍ CAPTURAMOS LOS SETS/LISTS
                            raw_cells = info.get('cells', [])
                            if not isinstance(raw_cells, list):
                                log.info(f"[extract] cells inválido tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} tipo={type(raw_cells).__name__} raw_keys={list(info.keys())}")
                                continue

                            collection_values = []
                            for cell_index, cell in enumerate(raw_cells):
                                if not isinstance(cell, dict):
                                    log.debug(f"[extract] cell inválida tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} cell_idx={cell_index} tipo={type(cell).__name__}")
                                    continue

                                # Dump-data para colecciones suele venir como:
                                # - set<T>: {"key": "<elemento>", "value": {"is_live":..., "value": ""}}
                                # - list<T>: {"key": "<idx>", "value": {"...":..., "value": "<elemento>"}}
                                # Por eso preferimos key cuando existe, y si no, extraemos value.value.
                                raw_cell_key = cell.get("key")
                                raw_cell_value = cell.get("value")

                                extracted_element_value = None

                                # 1) Preferimos el elemento en key (string o key.value)
                                if isinstance(raw_cell_key, dict):
                                    extracted_element_value = raw_cell_key.get("value")
                                elif raw_cell_key is not None:
                                    extracted_element_value = raw_cell_key

                                # 2) Fallback: value.value (cuando key es indice o no existe)
                                if extracted_element_value is None:
                                    if isinstance(raw_cell_value, dict):
                                        extracted_element_value = raw_cell_value.get("value")
                                    elif raw_cell_value is not None:
                                        extracted_element_value = raw_cell_value

                                if extracted_element_value is None:
                                    if should_trace(table, name):
                                        log.info(
                                            "[extract] cell sin key/value usable "
                                            f"tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} "
                                            f"cell_idx={cell_index} cell_keys={list(cell.keys())}"
                                        )
                                    continue

                                collection_values.append(str(extracted_element_value))
                            cells_data[name] = "|".join(collection_values)
                            if not collection_values:
                                log.info(f"[extract] colección vacía o no parseada tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} raw_keys={list(info.keys())}")
                            elif should_trace(table, name):
                                preview = collection_values[:10]
                                log.info(f"[trace.extract.cells] tabla={table} columna={name} items={len(collection_values)} preview={preview}")
                        else:
                            log.debug(f"[extract] estructura de columna desconocida tabla={table} part_idx={partition_index} elem_idx={element_index} columna={name} keys={list(info.keys())}")
                    
                    row = {}
                    if len(ordered_cols) < 2:
                        log.debug(f"[extract] tabla sin clustering key o metadata incompleta tabla={table} ordered_cols={ordered_cols}")
                    for col in ordered_cols:
                        if col == ordered_cols[0]: row[col] = pk_val # PK
                        elif len(ordered_cols) > 1 and col == ordered_cols[1]:
                            raw_cluster_key = el.get('key')
                            if isinstance(raw_cluster_key, dict):
                                row[col] = raw_cluster_key.get('value', '')
                            else:
                                row[col] = raw_cluster_key if raw_cluster_key is not None else ''
                        else: row[col] = cells_data.get(col, "")
                    rows_to_save.append(row)
                else:
                    log.debug(f"[extract] elemento clustering ignorado por tipo tabla={table} part_idx={partition_index} elem_idx={element_index} type={el.get('type')}")

        with open(csv_path, 'w', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=ordered_cols)
            writer.writeheader(); writer.writerows(rows_to_save)
        
        log.info(f"[extract] tabla={table} filas_csv={len(rows_to_save)} csv_path={csv_path}")
        shutil.rmtree(shadow_dir)
        return csv_path

    def load_csv_to_db(self, table, csv_path, ordered_cols, col_types):
        """Fase de Carga: Inyecta la data usando el driver nativo y casteo de tipos."""
        log.info(f"[*] Limpiando tabla {table} e iniciando carga...")
        self.session.execute(f"TRUNCATE {table}")
        
        query = f"INSERT INTO {table} ({', '.join(ordered_cols)}) VALUES ({', '.join(['?' for _ in ordered_cols])})"
        prepared = self.session.prepare(query)
        
        success = 0
        with open(csv_path, 'r') as f:
            reader = csv.DictReader(f)
            for row_index, row in enumerate(reader):
                try:
                    if row_index < TRACE_ROWS and should_trace(table):
                        traced = {col: row.get(col) for col in ordered_cols if should_trace(table, col)}
                        if traced:
                            log.info(f"[trace.csv.row] tabla={table} row_idx={row_index} valores={traced}")

                    params = [self.smart_cast(row[col], col_types.get(col, "text")) for col in ordered_cols]

                    if row_index < TRACE_ROWS and should_trace(table):
                        casted_traced = {}
                        for col in ordered_cols:
                            if should_trace(table, col):
                                try:
                                    casted_traced[col] = params[ordered_cols.index(col)]
                                except Exception:
                                    casted_traced[col] = "<index_error>"
                        if casted_traced:
                            log.info(f"[trace.cast.row] tabla={table} row_idx={row_index} valores={casted_traced}")

                    self.session.execute(prepared, params)
                    success += 1
                except Exception as e:
                    log.error(f"[-] Error en fila tabla={table} row_idx={row_index}: {e}")
        
        log.info(f"[+] {table.upper()} FINALIZADA: {success} filas recuperadas.")

    def run_full_recovery(self, table):
        try:
            cols, types = self.get_column_metadata(table)
            csv_path = self.extract_sstable_to_csv(table, cols)
            if csv_path:
                self.load_csv_to_db(table, csv_path, cols, types)
        except Exception:
            log.error(f"Fallo crítico en {table}: {traceback.format_exc()}")

# ==========================================
# EJECUCIÓN
# ==========================================
if __name__ == "__main__":
    restore = ScyllaRestore()
    for t in TABLES:
        restore.run_full_recovery(t)
