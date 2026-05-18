#!/usr/bin/env python3
"""Split remaining broad 'Limpieza del hogar' rows into 4 subcategories."""

from __future__ import annotations

import argparse
import logging
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logging."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def classify_limpieza_subcategory(product_name: str) -> str:
    """Assign one of four subcategories based on product intent."""
    lowered_name = product_name.casefold()

    # Products that control odors, humidity, or pests belong to environment-control group.
    environment_keywords = (
        "ambientador",
        "antihumedad",
        "absorbeolor",
        "absorbeolores",
        "antiolor",
        "incienso",
        "insecticida",
        "antimosquitos",
        "cucarachas",
        "hormigas",
        "antipolillas",
    )
    if any(keyword in lowered_name for keyword in environment_keywords):
        return "Ambientadores e Insecticidas"

    # Toilet-focused cleaning products and tools are isolated for bathroom maintenance.
    toilet_keywords = (
        "wc",
        "inodoro",
        "cisterna",
        "desatascador",
        "escobilla",
        "bloq wc",
        "block wc",
    )
    if any(keyword in lowered_name for keyword in toilet_keywords):
        return "WC y Bano"

    # Reusable tools or consumables for cleaning/storage are grouped as supplies.
    supplies_keywords = (
        "bayeta",
        "fregona",
        "mopa",
        "escoba",
        "estropajo",
        "guante",
        "bolsa",
        "papel",
        "film",
        "aluminio",
        "servilleta",
        "panuelo",
        "cubo",
        "pinza",
        "cepillo",
        "gamuza",
        "plumero",
    )
    if any(keyword in lowered_name for keyword in supplies_keywords):
        return "Utiles y Consumibles"

    # Default bucket for chemicals and formulas used on surfaces, floors, or fabrics.
    return "Superficies y Desinfeccion"


def refine_limpieza_categories(
    input_file_path: Path,
    output_file_path: Path,
    logger: logging.Logger,
) -> None:
    """Update only broad `Limpieza del hogar` rows in-place-safe way."""
    with input_file_path.open("r", encoding="utf-8", errors="replace") as input_file:
        lines = input_file.readlines()

    if not lines:
        logger.error("Input file is empty: %s", input_file_path)
        raise SystemExit(1)

    header = lines[0].rstrip("\n")
    updated_lines: list[str] = [header]

    total_rows = 0
    target_rows = 0
    unchanged_rows = 0
    distribution: dict[str, int] = {
        "Ambientadores e Insecticidas": 0,
        "WC y Bano": 0,
        "Utiles y Consumibles": 0,
        "Superficies y Desinfeccion": 0,
    }

    for line_number, line in enumerate(lines[1:], start=2):
        raw_line = line.rstrip("\n")
        if not raw_line.strip():
            continue

        parts = [part.strip() for part in raw_line.split("|", 2)]
        if len(parts) != 3:
            logger.debug("Skipping malformed row line=%d value=%r", line_number, raw_line)
            updated_lines.append(raw_line)
            continue

        product_name, brand_name, categories = parts
        total_rows += 1

        if categories != "Limpieza del hogar":
            unchanged_rows += 1
            updated_lines.append(f"{product_name}|{brand_name}|{categories}")
            continue

        target_rows += 1
        subcategory = classify_limpieza_subcategory(product_name)
        distribution[subcategory] += 1
        updated_lines.append(f"{product_name}|{brand_name}|{subcategory}")

    output_file_path.parent.mkdir(parents=True, exist_ok=True)
    with output_file_path.open("w", encoding="utf-8") as output_file:
        for updated_line in updated_lines:
            output_file.write(f"{updated_line}\n")

    logger.info("Rows scanned: %d", total_rows)
    logger.info("Rows updated from broad Limpieza del hogar: %d", target_rows)
    logger.info("Rows left unchanged: %d", unchanged_rows)
    for category_name, category_count in distribution.items():
        logger.info("Assigned %-30s -> %d", category_name, category_count)
    logger.info("Output written: %s", output_file_path)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Split remaining broad Limpieza del hogar rows into 4 subcategories."
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
    parser.add_argument("--debug", action="store_true", help="Enable debug logs.")
    args = parser.parse_args()

    configure_logging(args.debug)
    logger = logging.getLogger("process_limpieza_hogar_residual")

    if not args.input.exists():
        logger.error("Input file does not exist: %s", args.input)
        raise SystemExit(1)

    refine_limpieza_categories(args.input, args.output, logger)


if __name__ == "__main__":
    main()
