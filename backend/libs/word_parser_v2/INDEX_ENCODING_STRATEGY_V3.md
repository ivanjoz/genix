# Index Encoding Strategy V3 (Shape-Grouped Records)

## Goal
Reduce index size and improve compression ratio by removing per-record separator noise (word separators and phrase separators), and by compressing shape metadata with class-specific packing.

## Confirmed Decisions
1. Record reorder is acceptable.
2. `shape_id` is logical `u8`.
3. No word has more than 16 syllables.

## Core Idea
Represent each product name as:
- `shape`: syllable counts per word, example `[3,3,2]`
- `content`: flat syllable ID stream, example `[11,7,4, 9,2,5, 3,8]`

Then sort all records by `(shape_class, shape)` and store:
1. Dictionary (same logical concept as V2).
2. Two shape tables:
- class `0`: all words in `1..4` (2-bit packed word sizes)
- class `1`: words in `1..16` with at least one word in `5..16` (4-bit packed word sizes)
3. One usage counter array per class (`record_count` per shape in class order).
4. Grouped content stream in class+shape order.

## Why Sorting Minimizes Metadata
Sorting by `(shape_class, shape)` guarantees each shape is a contiguous run. This removes per-record and per-run separator bytes.

Structure overhead becomes:
- once per unique shape (shape tables)
- once per shape usage counter
instead of once per record.

## Strict Byte-Level Spec (V3)
Wire format magic: `WPV3IDX1`.

### Byte Order and Primitive Types
- Fixed-width integers use little-endian.
- `u8` = 1 byte unsigned.
- `u32` = 4 bytes unsigned.
- `varuint` = unsigned LEB128.

### File Layout (Exact Order)
1. Header
2. Dictionary section
3. Class-0 shape table section (2-bit packed)
4. Class-1 shape table section (4-bit packed)
5. Class-0 shape usage section
6. Class-1 shape usage section
7. Content section

### 1) Header (Fixed 29 bytes)
| Offset | Size | Field | Type | Notes |
|---|---:|---|---|---|
| 0 | 8 | magic | bytes[8] | ASCII `WPV3IDX1` |
| 8 | 1 | version | u8 | `1` |
| 9 | 1 | header_flags | u8 | bit0: dictionary_delta_mode (`1` enabled) |
| 10 | 4 | record_count | u32 | total encoded records |
| 14 | 1 | dictionary_count | u8 | dictionary token count (`1..255`) |
| 15 | 1 | shape_count_class0 | u8 | class-0 shape count |
| 16 | 1 | shape_count_class1 | u8 | class-1 shape count |
| 17 | 4 | dictionary_bytes | u32 | byte size of dictionary section |
| 21 | 4 | shape_table_class0_bytes | u32 | byte size of class-0 shape table |
| 25 | 4 | shape_table_class1_bytes | u32 | byte size of class-1 shape table |

Notes:
- `shape_usage` section sizes are implicit from parsing exactly `shape_count_classN` `varuint`s.
- `content_bytes` is implicit from file length.

Constraints:
- `shape_count_class0 + shape_count_class1 <= 255` (logical `shape_id` capacity).

### 2) Dictionary Section (Variable)
Dictionary section has two modes selected by `header_flags.bit0`.

Mode A (`dictionary_delta_mode = 0`, raw tokens):
- Repeated `dictionary_count` times in slot order (`slot_id = index + 1`):
1. `token_len` (`u8`)
2. `token_bytes[token_len]` (UTF-8)

Mode B (`dictionary_delta_mode = 1`, delta tokens):
1. Sort dictionary tokens lexicographically for storage.
2. Repeated `dictionary_count` times:
  - `prefix_len` (`u8`): common prefix bytes with previous token (`0` for first token).
  - `suffix_len` (`u8`): number of new bytes appended after prefix.
  - `suffix_bytes[suffix_len]` (UTF-8)
3. Rebuild slot IDs from this sorted dictionary order.
4. Remap every content syllable ID to the new sorted slot ID before writing content bytes.

Constraints:
- Raw mode: `token_len` in `[1,255]`.
- Delta mode: reconstructed token length must be in `[1,255]`.
- Delta mode: `prefix_len <= len(previous_token)` and `suffix_len >= 1`.
- Every content byte must reference slot ID `[1, dictionary_count]`.

### 3) Class-0 Shape Table (2-bit Packed)
Repeated `shape_count_class0` times (`shape_id_local = index + 1`):
1. `word_count` (`varuint`, must be `>=1`)
2. `packed_word_sizes_len` (`varuint`)
3. `packed_word_sizes[packed_word_sizes_len]` (2-bit packed)

2-bit packed rule:
- Word size domain is `1..4`.
- Encode each size as `(size - 1)` in 2 bits.
- Pack 4 words per byte, high bits first.

Derived:
- `shape_total_syllables = sum(decoded_word_sizes)`.

### 4) Class-1 Shape Table (4-bit Packed)
Repeated `shape_count_class1` times (`shape_id_local = index + 1`):
1. `word_count` (`varuint`, must be `>=1`)
2. `packed_word_sizes_len` (`varuint`)
3. `packed_word_sizes[packed_word_sizes_len]` (4-bit packed)

4-bit packed rule:
- Word size domain is `1..16`.
- Encode each size as `(size - 1)` in 4 bits.
- Pack 2 words per byte, high nibble first.

Class-1 membership rule:
- At least one word size must be in `5..16`.

Derived:
- `shape_total_syllables = sum(decoded_word_sizes)`.

### 5) Class-0 Shape Usage Section
Repeated `shape_count_class0` times in shape-table order:
1. `shape_record_count` (`varuint`)

### 6) Class-1 Shape Usage Section
Repeated `shape_count_class1` times in shape-table order:
1. `shape_record_count` (`varuint`)

Usage constraints:
- Total of all class-0 and class-1 `shape_record_count` values must equal `record_count`.

### 7) Content Section
Write records in this exact order:
1. Class-0 shapes in ascending `shape_id_local`.
2. Class-1 shapes in ascending `shape_id_local`.

For each shape:
- Let `run_count = shape_record_count`.
- Let `record_len = shape_total_syllables`.
- Write `run_count` records, each exactly `record_len` bytes.
- Each byte is one syllable dictionary ID (`u8`).

No separators:
- Phrase boundaries are implicit via `run_count` and fixed `record_len`.
- Word boundaries are implicit via decoded shape word sizes.

## Shape Classification Rules
Given a shape:
- If every word size is `<=4`, class = `0`.
- Else if every word size is `<=16`, class = `1`.
- Else reject record (`word_size > 16` invalid).

## Deterministic Build Rules
1. Encode records and compute shapes.
2. Classify each shape into class-0 or class-1.
3. Sort unique shapes lexicographically inside each class.
4. If `dictionary_delta_mode=1`, sort dictionary lexicographically and remap content IDs to sorted slot IDs.
5. Sort records by `(class, shape_id_local)`.
6. Keep stable input order within each final shape run.
7. Emit sections exactly in file-layout order.

## Decoder Validation Checklist
Reject file if any check fails:
1. Header magic/version mismatch.
2. Section byte lengths exceed file bounds.
3. `dictionary_count == 0` or `dictionary_count > 255`.
4. `shape_count_class0 + shape_count_class1 > 255`.
5. Invalid dictionary delta reconstruction (`prefix_len` overflow, invalid token length, invalid UTF-8 bytes).
6. Any decoded class-0 word size outside `1..4`.
7. Any decoded class-1 word size outside `1..16`.
8. Any shape has `word_count == 0`.
9. Usage totals do not equal `record_count`.
10. Content length mismatch against expected bytes.
11. Any content syllable ID byte is `0` or `> dictionary_count`.

## Compression Effect Summary
- Removes per-record separator metadata.
- Removes per-run `shape_id` metadata (implicit by sorted section order).
- Uses tighter shape-table packing (class-0: 2 bits per word size, class-1: 4 bits per word size).
- Adds optional dictionary delta mode (prefix/suffix) to reduce dictionary section bytes.

## Suggested Next Step
Implement prototype writer/reader for `WPV3IDX1` with this dual-class spec and benchmark against V2 on the same input file.
