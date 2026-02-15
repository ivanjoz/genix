#!/usr/bin/env python3
"""Refine only subdivided main categories in productos.psv using *_procesed files."""

from __future__ import annotations

import argparse
import logging
import re
import unicodedata
from collections import defaultdict
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logging."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def normalize_text(source_text: str) -> str:
    """Normalize text for robust product matching."""
    lowered_text = source_text.casefold()
    decomposed_text = unicodedata.normalize("NFKD", lowered_text)
    ascii_text = "".join(character for character in decomposed_text if not unicodedata.combining(character))
    alphanumeric_text = re.sub(r"[^a-z0-9\s]", "", ascii_text)
    compact_text = re.sub(r"\s+", "", alphanumeric_text)
    return compact_text


def main_category_of(categories_value: str) -> str:
    """Return the first category token, or `Sin categoria`."""
    first_category = categories_value.split(",", 1)[0].strip()
    return first_category if first_category else "Sin categoria"


def parse_productos_row(raw_line: str) -> tuple[str, str, str] | None:
    """Parse `Producto|Brand|Categories` rows."""
    parts = [part.strip() for part in raw_line.split("|", 2)]
    if len(parts) != 3:
        return None
    product_name, brand_name, categories_value = parts
    if not product_name:
        return None
    return product_name, brand_name, categories_value


def load_refinement_mapping(
    productos_sep_dir: Path,
    processed_pattern: str,
    logger: logging.Logger,
) -> tuple[set[str], dict[tuple[str, str], str], dict[tuple[str, str], set[str]]]:
    """
    Build mapping `(normalized_product, main_category) -> refined_categories`.

    Returns:
    - main categories that were subdivided
    - unambiguous mapping
    - ambiguous options for keys with conflicts
    """
    processed_paths = sorted(productos_sep_dir.glob(processed_pattern))
    if not processed_paths:
        logger.error("No processed files found in %s with pattern=%s", productos_sep_dir, processed_pattern)
        raise SystemExit(1)

    subdivided_main_categories: set[str] = set()
    category_options_by_key: dict[tuple[str, str], set[str]] = defaultdict(set)

    for processed_path in processed_paths:
        with processed_path.open("r", encoding="utf-8", errors="replace") as processed_file:
            header = processed_file.readline()
            if not header:
                continue

            for raw_line in processed_file:
                stripped_line = raw_line.strip()
                if not stripped_line:
                    continue
                parsed_row = parse_productos_row(stripped_line)
                if parsed_row is None:
                    continue

                product_name, _brand_name, refined_categories = parsed_row
                normalized_key = normalize_text(product_name)
                if not normalized_key:
                    continue

                main_category = main_category_of(refined_categories)
                subdivided_main_categories.add(main_category)
                category_options_by_key[(normalized_key, main_category)].add(refined_categories)

    refined_mapping: dict[tuple[str, str], str] = {}
    ambiguous_mapping: dict[tuple[str, str], set[str]] = {}
    for mapping_key, options in category_options_by_key.items():
        if len(options) == 1:
            refined_mapping[mapping_key] = next(iter(options))
        else:
            ambiguous_mapping[mapping_key] = options

    logger.info("Processed files loaded: %d", len(processed_paths))
    logger.info("Subdivided main categories found: %d", len(subdivided_main_categories))
    logger.info("Refinement keys (unambiguous): %d", len(refined_mapping))
    logger.info("Refinement keys (ambiguous): %d", len(ambiguous_mapping))
    return subdivided_main_categories, refined_mapping, ambiguous_mapping


def refine_productos_categories(
    input_file_path: Path,
    output_file_path: Path,
    subdivided_main_categories: set[str],
    refined_mapping: dict[tuple[str, str], str],
    ambiguous_mapping: dict[tuple[str, str], set[str]],
    ambiguous_report_path: Path,
    logger: logging.Logger,
) -> None:
    """Apply category refinement only for subdivided main categories."""
    with input_file_path.open("r", encoding="utf-8", errors="replace") as input_file:
        all_lines = input_file.readlines()

    if not all_lines:
        logger.error("Input file is empty: %s", input_file_path)
        raise SystemExit(1)

    header = all_lines[0].rstrip("\n")
    output_rows: list[str] = [header]

    total_rows = 0
    subdivided_rows_seen = 0
    category_rows_updated = 0
    skipped_ambiguous_rows = 0
    ambiguous_already_resolved_rows = 0
    skipped_missing_mapping_rows = 0

    ambiguous_report_path.parent.mkdir(parents=True, exist_ok=True)
    with ambiguous_report_path.open("w", encoding="utf-8") as ambiguous_report:
        ambiguous_report.write("Producto|MainCategory|CurrentCategories|CandidateCategories\n")

        for line_number, raw_line in enumerate(all_lines[1:], start=2):
            stripped_line = raw_line.rstrip("\n")
            if not stripped_line.strip():
                continue

            parsed_row = parse_productos_row(stripped_line)
            if parsed_row is None:
                logger.debug("Skipping invalid row line=%d value=%r", line_number, stripped_line)
                output_rows.append(stripped_line)
                continue

            product_name, brand_name, current_categories = parsed_row
            total_rows += 1
            current_main_category = main_category_of(current_categories)

            if current_main_category not in subdivided_main_categories:
                # Keep untouched categories exactly as they are.
                output_rows.append(f"{product_name}|{brand_name}|{current_categories}")
                continue

            subdivided_rows_seen += 1
            normalized_key = normalize_text(product_name)
            mapping_key = (normalized_key, current_main_category)

            if mapping_key in ambiguous_mapping:
                candidate_values = sorted(ambiguous_mapping[mapping_key])
                # If current value is already one valid candidate, keep it and do not report as pending conflict.
                if current_categories in ambiguous_mapping[mapping_key]:
                    ambiguous_already_resolved_rows += 1
                    output_rows.append(f"{product_name}|{brand_name}|{current_categories}")
                    continue

                skipped_ambiguous_rows += 1
                ambiguous_report.write(
                    f"{product_name}|{current_main_category}|{current_categories}|{', '.join(candidate_values)}\n"
                )
                output_rows.append(f"{product_name}|{brand_name}|{current_categories}")
                continue

            refined_categories = refined_mapping.get(mapping_key)
            if refined_categories is None:
                skipped_missing_mapping_rows += 1
                output_rows.append(f"{product_name}|{brand_name}|{current_categories}")
                continue

            if refined_categories != current_categories:
                category_rows_updated += 1
            output_rows.append(f"{product_name}|{brand_name}|{refined_categories}")

    output_file_path.parent.mkdir(parents=True, exist_ok=True)
    with output_file_path.open("w", encoding="utf-8") as output_file:
        for output_row in output_rows:
            output_file.write(f"{output_row}\n")

    logger.info("Rows scanned: %d", total_rows)
    logger.info("Rows in subdivided main categories: %d", subdivided_rows_seen)
    logger.info("Category updates applied: %d", category_rows_updated)
    logger.info("Rows skipped due to ambiguous mapping: %d", skipped_ambiguous_rows)
    logger.info("Rows already resolved with one ambiguous option: %d", ambiguous_already_resolved_rows)
    logger.info("Rows skipped due to missing mapping: %d", skipped_missing_mapping_rows)
    logger.info("Ambiguous report: %s", ambiguous_report_path)
    logger.info("Output written: %s", output_file_path)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Refine only subdivided categories in productos.psv using *_procesed files."
    )
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Input productos PSV file.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("libs/index_builder/productos.psv"),
        help="Output productos PSV file (can be same as input).",
    )
    parser.add_argument(
        "--productos-sep-dir",
        type=Path,
        default=Path("libs/productos_sep"),
        help="Directory containing *_procesed.psv files.",
    )
    parser.add_argument(
        "--processed-pattern",
        default="*_procesed.psv",
        help="Glob pattern to find processed files.",
    )
    parser.add_argument(
        "--ambiguous-report",
        type=Path,
        default=Path("libs/productos_sep/refine_ambiguous_report.psv"),
        help="Report file for products with conflicting subcategories.",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logging.")
    args = parser.parse_args()

    configure_logging(args.debug)
    logger = logging.getLogger("refine_subdivided_categories")

    if not args.input.exists():
        logger.error("Input file does not exist: %s", args.input)
        raise SystemExit(1)
    if not args.productos_sep_dir.exists():
        logger.error("productos_sep directory does not exist: %s", args.productos_sep_dir)
        raise SystemExit(1)

    subdivided_main_categories, refined_mapping, ambiguous_mapping = load_refinement_mapping(
        args.productos_sep_dir, args.processed_pattern, logger
    )
    refine_productos_categories(
        args.input,
        args.output,
        subdivided_main_categories,
        refined_mapping,
        ambiguous_mapping,
        args.ambiguous_report,
        logger,
    )


if __name__ == "__main__":
    main()
