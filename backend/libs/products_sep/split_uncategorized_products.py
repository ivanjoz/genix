#!/usr/bin/env python3
"""Split uncategorized products from an enriched PSV into productos_xxx.psv chunks."""

from __future__ import annotations

import argparse
import logging
import math
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logs with optional debug verbosity."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def normalize_category(raw_category: str) -> str:
    """Normalize category value for missing-category detection."""
    return raw_category.strip().casefold()


def has_missing_category(raw_category: str, missing_tokens: set[str]) -> bool:
    """Return True when category should be considered missing."""
    normalized_value = normalize_category(raw_category)
    return normalized_value in missing_tokens


def parse_product_row(product_row: str) -> tuple[str, str, str] | None:
    """Parse `Producto|Brand|Categories` row and ignore malformed entries."""
    parts = [part.strip() for part in product_row.split("|", 2)]
    if len(parts) != 3:
        return None
    product_name, brand_name, category_name = parts
    if not product_name:
        return None
    return product_name, brand_name, category_name


def collect_uncategorized_products(
    input_file_path: Path,
    missing_tokens: set[str],
    logger: logging.Logger,
) -> list[str]:
    """Collect product names that have missing categories in the source file."""
    uncategorized_products: list[str] = []
    total_rows = 0

    with input_file_path.open("r", encoding="utf-8") as input_file:
        header_line = input_file.readline()
        if not header_line:
            logger.error("Input file is empty: %s", input_file_path)
            raise SystemExit(1)

        for line_number, source_line in enumerate(input_file, start=2):
            parsed_row = parse_product_row(source_line.strip())
            if parsed_row is None:
                logger.debug("Skipping invalid row line=%d value=%r", line_number, source_line.rstrip("\n"))
                continue

            total_rows += 1
            product_name, _brand_name, category_name = parsed_row
            if has_missing_category(category_name, missing_tokens):
                uncategorized_products.append(product_name)
                logger.debug(
                    "Uncategorized product found line=%d product=%r category=%r",
                    line_number,
                    product_name,
                    category_name,
                )

    logger.info("Rows scanned: %d", total_rows)
    logger.info("Uncategorized products found: %d", len(uncategorized_products))
    return uncategorized_products


def write_chunk_files(
    uncategorized_products: list[str],
    output_dir_path: Path,
    chunk_size: int,
    logger: logging.Logger,
) -> int:
    """Write `productos_xxx.psv` files with `global_id | product` format."""
    output_dir_path.mkdir(parents=True, exist_ok=True)
    if not uncategorized_products:
        logger.warning("No uncategorized products to split. No output files created.")
        return 0

    total_files = math.ceil(len(uncategorized_products) / chunk_size)
    for file_index in range(total_files):
        file_start = file_index * chunk_size
        file_end = min(file_start + chunk_size, len(uncategorized_products))
        output_file_path = output_dir_path / f"productos_{file_index:03d}.psv"

        with output_file_path.open("w", encoding="utf-8") as output_file:
            for global_position in range(file_start, file_end):
                # Keep the historical `id | product` shape expected by existing processing scripts.
                product_identifier = global_position + 1
                output_file.write(f"{product_identifier} | {uncategorized_products[global_position]}\n")

        logger.info(
            "Wrote chunk file=%s rows=%d first_id=%d last_id=%d",
            output_file_path,
            file_end - file_start,
            file_start + 1,
            file_end,
        )

    logger.info("Total chunk files written: %d", total_files)
    return total_files


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Extract products without categories and split them into productos_xxx.psv files."
    )
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Input enriched PSV file with Producto|Brand|Categories columns.",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("libs/productos_sep"),
        help="Directory where productos_xxx.psv files will be written.",
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=200,
        help="Maximum number of products per generated file.",
    )
    parser.add_argument(
        "--missing-category-values",
        default=",sin categoria,sincategoria",
        help="Comma-separated normalized category values to treat as missing.",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logs.")
    args = parser.parse_args()

    configure_logging(args.debug)
    logger = logging.getLogger("split_uncategorized_products")

    if not args.input.exists():
        logger.error("Input file does not exist: %s", args.input)
        raise SystemExit(1)
    if args.chunk_size <= 0:
        logger.error("Chunk size must be greater than zero. Got: %d", args.chunk_size)
        raise SystemExit(1)

    missing_tokens = {
        token.strip().casefold() for token in args.missing_category_values.split(",")
    }
    if "" not in missing_tokens:
        missing_tokens.add("")

    uncategorized_products = collect_uncategorized_products(args.input, missing_tokens, logger)
    write_chunk_files(uncategorized_products, args.output_dir, args.chunk_size, logger)


if __name__ == "__main__":
    main()
