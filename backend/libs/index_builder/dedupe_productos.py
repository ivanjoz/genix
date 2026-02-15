#!/usr/bin/env python3
"""Normalize product lines, detect duplicates, and export deduplicated files."""

from __future__ import annotations

import argparse
import logging
import re
import unicodedata
from pathlib import Path


def configure_logging(debug_enabled: bool) -> None:
    """Configure compact logs; debug mode prints row-level details."""
    log_level = logging.DEBUG if debug_enabled else logging.INFO
    logging.basicConfig(level=log_level, format="[%(levelname)s] %(message)s")


def normalize_sentence(raw_sentence: str) -> str:
    """Normalize text to lowercase ASCII and remove non-alnum characters and spaces."""
    lowered_sentence = raw_sentence.casefold()
    decomposed_sentence = unicodedata.normalize("NFKD", lowered_sentence)
    ascii_sentence = "".join(
        character for character in decomposed_sentence if not unicodedata.combining(character)
    )
    alphanumeric_and_spaces = re.sub(r"[^a-z0-9\s]", "", ascii_sentence)
    without_spaces = re.sub(r"\s+", "", alphanumeric_and_spaces)
    return without_spaces


def process_file(
    input_file_path: Path,
    deduplicated_output_path: Path,
    normalized_output_path: Path,
    duplicates_output_path: Path,
    logger: logging.Logger,
) -> None:
    """Read source lines and write deduplicated and diagnostics outputs."""
    normalized_first_original: dict[str, str] = {}
    normalized_occurrence_count: dict[str, int] = {}
    deduplicated_originals: list[str] = []
    normalized_rows: list[tuple[int, str, str]] = []

    with input_file_path.open("r", encoding="utf-8") as input_file:
        for line_number, source_line in enumerate(input_file, start=1):
            original_sentence = source_line.strip()
            if not original_sentence:
                logger.debug("Skipping empty line at line_number=%d", line_number)
                continue

            normalized_sentence = normalize_sentence(original_sentence)
            if not normalized_sentence:
                logger.debug(
                    "Skipping line without normalized content line_number=%d value=%r",
                    line_number,
                    original_sentence,
                )
                continue

            normalized_rows.append((line_number, original_sentence, normalized_sentence))
            normalized_occurrence_count[normalized_sentence] = (
                normalized_occurrence_count.get(normalized_sentence, 0) + 1
            )

            # Keep the first original sentence for each normalized key to preserve a deterministic output.
            if normalized_sentence not in normalized_first_original:
                normalized_first_original[normalized_sentence] = original_sentence
                deduplicated_originals.append(original_sentence)
                logger.debug(
                    "Unique product accepted line_number=%d normalized_key=%s",
                    line_number,
                    normalized_sentence,
                )
            else:
                logger.debug(
                    "Duplicate product detected line_number=%d normalized_key=%s",
                    line_number,
                    normalized_sentence,
                )

    deduplicated_output_path.parent.mkdir(parents=True, exist_ok=True)
    with deduplicated_output_path.open("w", encoding="utf-8") as deduplicated_file:
        for product_sentence in deduplicated_originals:
            deduplicated_file.write(f"{product_sentence}\n")

    normalized_output_path.parent.mkdir(parents=True, exist_ok=True)
    with normalized_output_path.open("w", encoding="utf-8") as normalized_file:
        normalized_file.write("line_number|original|normalized\n")
        for line_number, original_sentence, normalized_sentence in normalized_rows:
            normalized_file.write(
                f"{line_number}|{original_sentence}|{normalized_sentence}\n"
            )

    duplicates_output_path.parent.mkdir(parents=True, exist_ok=True)
    with duplicates_output_path.open("w", encoding="utf-8") as duplicates_file:
        duplicates_file.write("normalized|count|first_original\n")
        for normalized_sentence, occurrence_count in sorted(
            normalized_occurrence_count.items(), key=lambda row: (-row[1], row[0])
        ):
            if occurrence_count <= 1:
                continue
            first_original = normalized_first_original[normalized_sentence]
            duplicates_file.write(
                f"{normalized_sentence}|{occurrence_count}|{first_original}\n"
            )

    total_rows = len(normalized_rows)
    unique_rows = len(deduplicated_originals)
    duplicate_rows = total_rows - unique_rows
    duplicate_keys = sum(
        1 for occurrence_count in normalized_occurrence_count.values() if occurrence_count > 1
    )
    logger.info("Input rows processed: %d", total_rows)
    logger.info("Unique normalized rows: %d", unique_rows)
    logger.info("Duplicate rows removed: %d", duplicate_rows)
    logger.info("Duplicate normalized keys: %d", duplicate_keys)
    logger.info("Deduplicated output: %s", deduplicated_output_path)
    logger.info("Normalized audit output: %s", normalized_output_path)
    logger.info("Duplicate summary output: %s", duplicates_output_path)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Normalize product sentences and create duplicate-free output files."
    )
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("libs/index_builder/productos.txt"),
        help="Source file with one product sentence per line.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("libs/index_builder/productos_deduplicated.txt"),
        help="Output file with duplicates removed using normalized matching.",
    )
    parser.add_argument(
        "--normalized-output",
        type=Path,
        default=Path("libs/index_builder/productos_normalized.tsv"),
        help="Audit file with original and normalized values per input line.",
    )
    parser.add_argument(
        "--duplicates-output",
        type=Path,
        default=Path("libs/index_builder/productos_duplicates.tsv"),
        help="Summary file listing normalized keys that appeared more than once.",
    )
    parser.add_argument("--debug", action="store_true", help="Enable detailed debug logs.")
    arguments = parser.parse_args()

    configure_logging(arguments.debug)
    logger = logging.getLogger("dedupe_productos")

    if not arguments.input.exists():
        logger.error("Input file does not exist: %s", arguments.input)
        raise SystemExit(1)

    process_file(
        input_file_path=arguments.input,
        deduplicated_output_path=arguments.output,
        normalized_output_path=arguments.normalized_output,
        duplicates_output_path=arguments.duplicates_output,
        logger=logger,
    )


if __name__ == "__main__":
    main()
