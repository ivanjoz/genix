#!/usr/bin/env python3
"""Export one PSV file per main category when category size is above a threshold."""

from __future__ import annotations

import argparse
import logging
import re
import unicodedata
from collections import defaultdict
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logging for export runs."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def slugify_category_name(category_name: str) -> str:
    """Convert category to a filesystem-safe lowercase filename token."""
    lowered_name = category_name.casefold()
    decomposed_name = unicodedata.normalize("NFKD", lowered_name)
    ascii_name = "".join(character for character in decomposed_name if not unicodedata.combining(character))
    normalized_name = re.sub(r"[^a-z0-9]+", "_", ascii_name).strip("_")
    return normalized_name or "sin_categoria"


def extract_main_category(categories_field: str) -> str:
    """Take the first category as the main category."""
    first_category = categories_field.split(",", 1)[0].strip()
    return first_category if first_category else "Sin categoria"


def parse_product_row(raw_line: str) -> tuple[str, str, str] | None:
    """Parse `Producto|Brand|Categories` rows from productos.psv."""
    parts = [part.strip() for part in raw_line.split("|", 2)]
    if len(parts) != 3:
        return None
    product_name, brand_name, categories_field = parts
    if not product_name:
        return None
    normalized_brand = brand_name if brand_name else "Sin marca"
    normalized_categories = categories_field if categories_field else "Sin categoria"
    return product_name, normalized_brand, normalized_categories


def load_rows_grouped_by_main_category(
    input_file_path: Path,
    logger: logging.Logger,
) -> dict[str, list[tuple[str, str, str]]]:
    """Load products grouped by main category."""
    grouped_rows: dict[str, list[tuple[str, str, str]]] = defaultdict(list)
    total_rows = 0
    invalid_rows = 0

    with input_file_path.open("r", encoding="utf-8") as input_file:
        header_line = input_file.readline()
        if not header_line:
            logger.error("Input file is empty: %s", input_file_path)
            raise SystemExit(1)

        for line_number, raw_line in enumerate(input_file, start=2):
            stripped_line = raw_line.strip()
            if not stripped_line:
                continue

            parsed_row = parse_product_row(stripped_line)
            if parsed_row is None:
                invalid_rows += 1
                logger.debug("Skipping invalid row line=%d value=%r", line_number, stripped_line)
                continue

            product_name, brand_name, categories_field = parsed_row
            main_category = extract_main_category(categories_field)
            grouped_rows[main_category].append((product_name, brand_name, categories_field))
            total_rows += 1

    logger.info("Rows loaded: %d", total_rows)
    logger.info("Invalid rows skipped: %d", invalid_rows)
    logger.info("Main categories found: %d", len(grouped_rows))
    return grouped_rows


def write_category_files(
    grouped_rows: dict[str, list[tuple[str, str, str]]],
    output_dir_path: Path,
    min_products_exclusive: int,
    logger: logging.Logger,
) -> int:
    """Write one PSV file per category with product count > min threshold."""
    output_dir_path.mkdir(parents=True, exist_ok=True)
    exported_file_count = 0

    for category_name, rows in sorted(grouped_rows.items(), key=lambda item: (-len(item[1]), item[0].casefold())):
        category_size = len(rows)
        if category_size <= min_products_exclusive:
            logger.debug(
                "Skipping category=%r size=%d because it does not exceed threshold=%d",
                category_name,
                category_size,
                min_products_exclusive,
            )
            continue

        category_slug = slugify_category_name(category_name)
        output_file_path = output_dir_path / f"{category_slug}.psv"

        with output_file_path.open("w", encoding="utf-8") as output_file:
            output_file.write("Producto|Brand|Categories\n")
            for product_name, brand_name, categories_field in rows:
                output_file.write(f"{product_name}|{brand_name}|{categories_field}\n")

        exported_file_count += 1
        logger.info(
            "Exported category file=%s category=%r rows=%d",
            output_file_path,
            category_name,
            category_size,
        )

    logger.info("Category files exported: %d", exported_file_count)
    return exported_file_count


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Create one PSV file per main category with more than N products."
    )
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Input productos PSV with columns Producto|Brand|Categories.",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("libs/productos_sep"),
        help="Output directory where category PSV files will be created.",
    )
    parser.add_argument(
        "--min-products",
        type=int,
        default=120,
        help="Only export categories with product count strictly greater than this value.",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logs.")
    args = parser.parse_args()

    configure_logging(args.debug)
    logger = logging.getLogger("export_large_categories")

    if not args.input.exists():
        logger.error("Input file does not exist: %s", args.input)
        raise SystemExit(1)
    if args.min_products < 0:
        logger.error("min-products must be >= 0. Got: %d", args.min_products)
        raise SystemExit(1)

    grouped_rows = load_rows_grouped_by_main_category(args.input, logger)
    write_category_files(grouped_rows, args.output_dir, args.min_products, logger)


if __name__ == "__main__":
    main()
