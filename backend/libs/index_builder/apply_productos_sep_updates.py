#!/usr/bin/env python3
"""Apply brand/category updates from productos_sep processed files to productos.psv."""

from __future__ import annotations

import argparse
import logging
import re
import unicodedata
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logging for update runs."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def normalize_text(source_text: str) -> str:
    """Normalize product names for resilient matching."""
    lowered_text = source_text.casefold()
    decomposed_text = unicodedata.normalize("NFKD", lowered_text)
    ascii_text = "".join(character for character in decomposed_text if not unicodedata.combining(character))
    alphanumeric_text = re.sub(r"[^a-z0-9\s]", "", ascii_text)
    compact_text = re.sub(r"\s+", "", alphanumeric_text)
    return compact_text


def is_missing_category(category_value: str) -> bool:
    """Check if category should be considered missing."""
    normalized_value = category_value.strip().casefold()
    return normalized_value in {"", "sin categoria", "sincategoria"}


def is_missing_brand(brand_value: str) -> bool:
    """Check if brand should be considered missing-like."""
    normalized_value = brand_value.strip().casefold()
    return normalized_value in {"", "sin marca", "(sin marca)", "(sinmarca)"}


def parse_raw_row(raw_line: str) -> tuple[str, str] | None:
    """Parse `id | product` rows from raw split files."""
    if "|" not in raw_line:
        return None
    row_id, product_name = raw_line.split("|", 1)
    normalized_row_id = row_id.strip()
    normalized_product_name = product_name.strip()
    if not normalized_row_id or not normalized_product_name:
        return None
    return normalized_row_id, normalized_product_name


def parse_processed_row(processed_line: str) -> tuple[str, str, str] | None:
    """Parse `id | brand | categories` rows from processed files."""
    parts = [part.strip() for part in processed_line.split("|", 2)]
    if len(parts) != 3:
        return None
    row_id, brand_name, categories = parts
    if not row_id:
        return None
    normalized_brand_name = brand_name if brand_name else "Sin marca"
    normalized_categories = categories if categories else "Sin categoria"
    return row_id, normalized_brand_name, normalized_categories


def load_update_mapping(
    productos_sep_dir: Path,
    processed_pattern: str,
    logger: logging.Logger,
) -> dict[str, tuple[str, str]]:
    """Build `normalized_product -> (brand, categories)` map from processed pairs."""
    processed_files = sorted(productos_sep_dir.glob(processed_pattern))
    if not processed_files:
        logger.error("No processed files found in %s with pattern %s", productos_sep_dir, processed_pattern)
        raise SystemExit(1)

    update_mapping: dict[str, tuple[str, str]] = {}
    conflict_count = 0
    mapped_rows = 0

    for processed_file_path in processed_files:
        raw_file_path = processed_file_path.with_name(processed_file_path.name.replace("_procesed.psv", ".psv"))
        if not raw_file_path.exists():
            logger.warning("Skipping processed file without raw pair: %s", processed_file_path)
            continue

        raw_products_by_id: dict[str, str] = {}
        with raw_file_path.open("r", encoding="utf-8") as raw_file:
            for raw_line in raw_file:
                parsed_raw_row = parse_raw_row(raw_line)
                if parsed_raw_row is None:
                    continue
                row_id, product_name = parsed_raw_row
                raw_products_by_id[row_id] = product_name

        with processed_file_path.open("r", encoding="utf-8") as processed_file:
            for processed_line in processed_file:
                parsed_processed_row = parse_processed_row(processed_line)
                if parsed_processed_row is None:
                    continue
                row_id, brand_name, categories = parsed_processed_row
                product_name = raw_products_by_id.get(row_id)
                if product_name is None:
                    logger.debug("Processed row without raw product id=%s file=%s", row_id, processed_file_path)
                    continue

                normalized_product_key = normalize_text(product_name)
                if not normalized_product_key:
                    continue

                new_value = (brand_name, categories)
                existing_value = update_mapping.get(normalized_product_key)
                if existing_value is not None and existing_value != new_value:
                    conflict_count += 1
                    logger.warning(
                        "Conflict for product=%r existing=%r new=%r file=%s",
                        product_name,
                        existing_value,
                        new_value,
                        processed_file_path,
                    )
                    continue

                update_mapping[normalized_product_key] = new_value
                mapped_rows += 1

    logger.info("Processed files used: %d", len(processed_files))
    logger.info("Update mapping keys: %d", len(update_mapping))
    logger.info("Processed rows mapped: %d", mapped_rows)
    logger.info("Conflicts skipped: %d", conflict_count)
    return update_mapping


def apply_updates_to_productos(
    input_file_path: Path,
    output_file_path: Path,
    update_mapping: dict[str, tuple[str, str]],
    logger: logging.Logger,
) -> None:
    """Apply category/brand updates to productos PSV and write final file."""
    with input_file_path.open("r", encoding="utf-8") as input_file:
        all_lines = input_file.readlines()

    if not all_lines:
        logger.error("Input productos file is empty: %s", input_file_path)
        raise SystemExit(1)

    header = all_lines[0].rstrip("\n")
    updated_rows: list[str] = [header]
    total_rows = 0
    category_updates = 0
    brand_updates = 0
    matched_rows = 0
    unmatched_missing_category_rows = 0

    for line_number, raw_line in enumerate(all_lines[1:], start=2):
        line_value = raw_line.rstrip("\n")
        if not line_value.strip():
            continue

        parts = [part.strip() for part in line_value.split("|", 2)]
        if len(parts) != 3:
            logger.debug("Skipping invalid productos row line=%d value=%r", line_number, line_value)
            updated_rows.append(line_value)
            continue

        product_name, brand_name, category_name = parts
        total_rows += 1
        normalized_product_key = normalize_text(product_name)
        mapped_values = update_mapping.get(normalized_product_key)

        new_brand_name = brand_name
        new_category_name = category_name

        if mapped_values is not None:
            matched_rows += 1
            mapped_brand_name, mapped_categories = mapped_values
            if is_missing_category(category_name):
                new_category_name = mapped_categories
                category_updates += 1
            if is_missing_brand(brand_name):
                new_brand_name = mapped_brand_name
                brand_updates += 1
        else:
            if is_missing_category(category_name):
                unmatched_missing_category_rows += 1

        updated_rows.append(f"{product_name}|{new_brand_name}|{new_category_name}")

    output_file_path.parent.mkdir(parents=True, exist_ok=True)
    with output_file_path.open("w", encoding="utf-8") as output_file:
        for row in updated_rows:
            output_file.write(f"{row}\n")

    logger.info("Rows scanned: %d", total_rows)
    logger.info("Rows matched with update mapping: %d", matched_rows)
    logger.info("Category updates applied: %d", category_updates)
    logger.info("Brand updates applied: %d", brand_updates)
    logger.info("Rows still missing category without mapping: %d", unmatched_missing_category_rows)
    logger.info("Output written to: %s", output_file_path)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Apply productos_sep processed rows to update Brand and Categories in productos.psv."
    )
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Input productos PSV (Producto|Brand|Categories).",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Output productos PSV path (can be same as input for in-place update).",
    )
    parser.add_argument(
        "--productos-sep-dir",
        type=Path,
        default=Path("libs/productos_sep"),
        help="Directory with productos_xxx.psv and productos_xxx_procesed.psv files.",
    )
    parser.add_argument(
        "--processed-pattern",
        default="productos_*_procesed.psv",
        help="Glob pattern used to find processed files in productos_sep dir.",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logs.")
    args = parser.parse_args()

    configure_logging(args.debug)
    logger = logging.getLogger("apply_productos_sep_updates")

    if not args.input.exists():
        logger.error("Input file does not exist: %s", args.input)
        raise SystemExit(1)
    if not args.productos_sep_dir.exists():
        logger.error("productos_sep directory does not exist: %s", args.productos_sep_dir)
        raise SystemExit(1)

    update_mapping = load_update_mapping(args.productos_sep_dir, args.processed_pattern, logger)
    apply_updates_to_productos(args.input, args.output, update_mapping, logger)


if __name__ == "__main__":
    main()
