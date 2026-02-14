# Index Builder

This package builds a compact binary index from product records.

## Goal

Given input records:

- `ID int32`
- `Text string`

produce:

- reordered record IDs (`SortedIDs`) aligned with index storage order
- shape stream bytes (`Shapes`)
- content stream bytes (`Content`)
- dictionary section bytes
- build stats
- a binary `.idx` payload
- decoder support to reconstruct records from `.idx`

## Public API

### Input

```go
type RecordInput struct {
    ID   int32
    Text string
}
```

### Options

```go
type BuildOptions struct {
    MaxWordsPerRecord   int32
    MaxSyllablesPerWord int32
    MaxDictionarySlots  int32
    ConnectorWords      []string
}
```

Use `DefaultOptions()` for a working baseline.

### Build

```go
func Build(records []RecordInput, options BuildOptions) (*BuildResult, error)
```

### Decode

```go
func DecodeBinary(indexBytes []byte) (*DecodeResult, error)
func SampleDecodedRecords(records []DecodedRecord, sampleCount int32, seed int64) []DecodedRecord
```

### Input Loader

```go
func LoadRecordsFromTextFile(inputPath string) ([]RecordInput, error)
```

### Result

```go
type BuildResult struct {
    SortedIDs []int32
    Shapes    []byte
    Content   []byte

    HeaderFlags       uint8
    DictionaryTokens  []string
    DictionarySection []byte

    Stats BuildStats
}
```

### Serialization

```go
func (r *BuildResult) MarshalBinary() ([]byte, error)
func (r *BuildResult) WriteBinaryFile(outputPath string) error
```

### Syllable Utilities

```go
type SyllableFrequency struct {
    Syllable string
    Count    int32
}

func SplitTokenIntoSyllables(token string) []string
func ExtractTopSyllableFrequencies(records []RecordInput, limit int32) []SyllableFrequency
```

## Processing Pipeline

1. Normalize text
- lowercases text
- normalizes accented characters to ASCII
- keeps `a-z`, `0-9`, and spaces

2. Token filtering
- splits by spaces
- removes connector words
- applies numeric compaction for numeric-prefix tokens
- drops single-character tokens at this stage

3. Dictionary construction
- starts from fixed canonical tokens and aliases
- computes syllable frequency on normalized records
- fills dictionary up to `MaxDictionarySlots` (max `255`)

4. Record encoding
- each word is encoded as 1..N syllable IDs
- each word contributes its syllable count to record shape
- each record shape is encoded as fixed 24-bit value (`8 words * 3 bits`)

5. Reorder records
- sorts records by shape value ascending
- keeps stable order for ties
- emits `SortedIDs` aligned with final storage order

6. Shape stream
- delta-encodes sorted shape values
- token format:
  - `0 + 8 bits` for delta `<=255`
  - `1 + 16 bits` for delta `<=65534`
  - `1 + 0xFFFF + 24 bits` escape for larger deltas

7. Content stream
- concatenates encoded record content bytes in sorted order

## Binary Format

The binary payload written by `MarshalBinary()` is:

1. Header
2. Dictionary section
3. Shape stream section
4. Content section

### Header Layout

- magic: `GIXIDX01` (8 bytes)
- version: `u8`
- flags: `u8`
- record_count: `u32` little-endian
- dictionary_count: `u8`
- dictionary_bytes: `u32` little-endian
- shapes_bytes: `u32` little-endian
- content_bytes: `u32` little-endian

### Flags

- `bit0`: dictionary delta mode
- `bit1`: shape delta stream enabled
- `bit2`: numeric compaction enabled

## Stats

`BuildStats` exposes:

- record counts (input and encoded)
- dictionary size/count
- shape/content/total byte sizes
- shape delta bucket counts (`8/16/24`)

All numeric stats use `int32`.

## Example

```go
records := []index_builder.RecordInput{
    {ID: 101, Text: "Vino tinto reserva 750 ml"},
    {ID: 102, Text: "Queso curado unidad"},
}

opts := index_builder.DefaultOptions()
result, err := index_builder.Build(records, opts)
if err != nil {
    panic(err)
}

err = result.WriteBinaryFile("productos.idx")
if err != nil {
    panic(err)
}
```

## CLI + Script

Build CLI:

```bash
go run ./cmd/index_builder_build_idx \
  -input libs/index_builder/productos.txt \
  -output libs/index_builder/productos.idx \
  -max-words 8 \
  -max-syllables-per-word 7 \
  -slots 255
```

Convenience script:

```bash
libs/index_builder/run_build_and_stats.sh
```

Decode CLI:

```bash
go run ./cmd/index_builder_decode_idx \
  -input libs/index_builder/productos.idx \
  -sample 10 \
  -seed 0
```

Decode script:

```bash
libs/index_builder/run_decode_and_stats.sh
```

Script parameters:

1. input path (default: `libs/index_builder/productos.txt`)
2. output path (default: `libs/index_builder/productos.idx`)
3. max words per record (default: `8`)
4. max syllables per word (default: `7`)
5. max dictionary slots (default: `255`)

Decode script parameters:

1. input path (default: `libs/index_builder/productos.idx`)
2. sample count (default: `10`)
3. seed (default: `0`, random each run)
