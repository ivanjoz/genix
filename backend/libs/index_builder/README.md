# Index Builder
This package builds the product search binary used by Genix.

Current output is one combined `productos.idx` payload:

1. Text block (encoded product text index)
2. Taxonomy block (brand/category dictionaries and product mappings)

`decoder.go` exists for decode/debug and is not part of the build path.

## Purpose

The implementation is optimized for:

1. Small payload size
2. Deterministic output
3. Strict row alignment between text and taxonomy
4. Operationally visible build stats

## Build Path Files

Every `.go` file under `libs/index_builder` is in the build path except `decoder.go`.

- `index_builder.go`
  - text index encoding
  - dictionary generation
  - shape/content stream generation
  - text binary serialization
- `builder.go`
  - sequential orchestration entrypoint (`BuildProductosIndex`)
  - aggressive product text optimization integration
  - optimization metrics aggregation
- `taxonomy_pass.go`
  - taxonomy dictionaries and product taxonomy columns
  - brand index encoding mode selection (`uint12` or `uint16`)
  - validation for binary writing
- `taxonomy_binary.go`
  - serializes taxonomy block
  - concatenates text + taxonomy into one payload
  - writes file
- `packing.go`
  - reusable low-level numeric packing helpers
  - `int -> uint16` conversion
  - `uint12` packing
- `syllable_generator.go`
  - canonical alias rules
  - syllable split/frequency support used by text build

## Main Data Contracts

### `RecordInput`

```go
type RecordInput struct {
    ID            int32
    CategoriesIDs []int32
    BrandID       int32
    Text          string
}
```

Context usage:

- product rows use all fields
- brand rows use `ID`, `Text`
- category rows use `ID`, `Text`

### `BuildInput`

```go
type BuildInput struct {
    Products   []RecordInput
    Brands     []RecordInput
    Categories []RecordInput
}
```

### `ProductosIndexBuild`

`BuildProductosIndex` returns:

- Stage-1 text fields (`SortedIDs`, `Shapes`, `Content`, `DictionaryTokens`, `Stats`, etc.)
- Stage-2 taxonomy fields (`BrandIDs`, `CategoryIDs`, packed indexes, etc.)
- `OptimizationStats ProductTextOptimizationAggregate`

## End-to-End Flow

`BuildProductosIndex(buildInput)` runs a sequential process:

1. product text optimization
2. text index build
3. taxonomy build aligned to text `SortedIDs`

Writing combined bytes is done with:

- `buildResult.ToBytes()`

### Phase 1: Product Text Optimization

For each product:

1. Resolve brand name using `BrandID`
2. Inline-remove brand token sequence and duplicate tokens from normalized product text
3. Collect stats
4. If optimized text becomes empty, fallback to original product text
5. Keep same IDs/relations, only replace `Text`

Why this exists:

- brand words are duplicated across many products
- removing them reduces text entropy and content bytes
- duplicate token removal avoids redundant syllable encoding
- fallback preserves product visibility in text index

### Phase 2: Text Index Build (`Build`)

Core behavior:

1. normalize text
2. remove connector words
3. compact numeric tokens
4. build dictionary from fixed aliases + frequency
5. encode words into syllable IDs
6. encode record shapes
7. sort encoded records deterministically
8. emit `SortedIDs`, `Shapes`, `Content`, stats

`SortedIDs` is the row-order contract consumed by taxonomy.

### Phase 3: Taxonomy Build (`BuildTaxonomySecondPass`)

Taxonomy rows are built in `SortedIDs` order:

1. reorder products by `SortedIDs`
2. brand dictionary by first appearance in sorted sequence
3. category ranking by usage frequency
4. fixed category dictionary with `"Otros"` bucket
5. per-product brand index encoding
6. per-product category count and category index payload

Brand encoding mode:

- `uint12` packed mode when dictionary cardinality fits
- `uint16` otherwise

Category storage:

- packed per-product count (2-bit counters)
- flat category index payload consumed by packed counts

## Alignment Invariant

Critical invariant:

- row `i` in text block and row `i` in taxonomy block represent the same product.

How it is guaranteed:

- text builder defines `SortedIDs`
- taxonomy builder uses the same `SortedIDs` to reorder products
- taxonomy serialization validates row-consistency assumptions

## Text Block Binary

Text block is serialized by `ProductosIndexBuild.MarshalBinary` with:

1. header (`GIXIDX01`, version, flags, record count, build_sunix_time, dictionary counters)
2. section table entries (section id, offset, length, item count, CRC32 checksum)
3. dictionary section
4. shape stream section
5. content section

Shape stream uses delta encoding with compact paths for small deltas.

## Taxonomy Block Binary

Taxonomy block is appended after text block.

Taxonomy header contains:

- magic (`GIXTAX01`)
- taxonomy version
- brand encoding flag
- sorted product count
- seven section table entries (section id, offset, length, item count, CRC32 checksum)

Sections (fixed order):

1. brand IDs (`uint16` LE)
2. brand names (len-prefixed strings)
3. category IDs (`uint16` LE)
4. category names (len-prefixed strings)
5. brand indexes (`uint12` packed or `uint16`)
6. packed category counts
7. category indexes

## Validation in Build Path

Before taxonomy serialization, `ValidateForBinary` enforces:

- non-nil and non-empty sorted products
- dictionary ID/name column length parity
- valid brand index encoding flag
- payload consistency with selected brand mode
- category-count packed byte length sanity
- category index payload minimum size sanity

These checks fail early before writing bytes.

## Determinism Rules

To preserve deterministic binaries:

1. keep tie-break ordering explicit
2. avoid map iteration as output order source
3. keep section write order fixed
4. keep normalization rules stable

## Operational Integration

Typical runtime integration:

1. handler fetches products, brands, categories
2. map source rows into `BuildInput`
3. call `BuildProductosIndex`
4. call `indexBuild.ToBytes()` and write bytes to disk
5. log stage stats and optimization aggregate

Output path used by current integration:

- `libs/index_builder/productos.idx`

## Extension Guidelines

When adding fields or sections:

1. keep existing section order unless version changes
2. add explicit validation checks
3. expose stats for observability
4. update this README in same PR
5. ensure tests cover size/consistency regressions

## Testing Guidance

Current tests focus on text optimization and taxonomy validation sanity.
For binary contract changes, add validation and payload-length assertions.
