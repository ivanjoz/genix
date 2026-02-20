# Index Builder Columnar Taxonomy Migration Plan

## Goal
Migrate `backend/libs/index_builder` to support a columnar binary representation for product taxonomy metadata (brands and categories), while preserving current text-shape indexing behavior.

## Locked Requirements
1. `Build` in `index_builder.go` stays as the generic text index stage (`ID` + `Text` behavior).
2. Taxonomy processing is implemented as a second pass in a separate Go file.
3. Second pass receives one input envelope with 3 arrays: `Products`, `Brands`, `Categories`.
4. All arrays use the same struct type (`RecordInput`), with context-specific field usage.
5. Binary representation must be columnar.
6. Product rows must store taxonomy references as indexes, not original IDs.
7. Indexes must be 0-based.
8. Category dictionary max size is `244` entries total:
9. Entries `0..242`: top `243` categories by frequency.
10. Entry `243`: fixed bucket `"Otros"`.
11. A product referencing a non-existing brand dictionary ID is a build error.
12. Missing/overflow categories are mapped to `"Otros"` index (`243`).
13. Raw product taxonomy IDs must not be stored in product rows in binary.
14. Dictionary arrays must include both names and original IDs (columnar dictionaries).

## Current Gaps in Code
1. `Build` currently accepts `[]RecordInput` only.
2. Binary currently serializes dictionary, shape, and content only.
3. No taxonomy dictionary sections (brand/category names + IDs).
4. No per-product brand/category index columns.
5. Header and decoder are already inconsistent on header byte size and need normalization while migrating version.

## Data Contract Changes
### 1. Stage 2 Build Input Envelope
Create a new input type:
- `type BuildInput struct {`
- `Products []RecordInput`
- `Brands []RecordInput`
- `Categories []RecordInput`
- `}`

Use in signature:
- `func BuildTaxonomySecondPass(sortedProductIDs []int32, input BuildInput) (*TaxonomyBuildResult, error)`

### 2. Two-Stage Execution Strategy
1. Stage 1: call existing `Build(records []RecordInput, options BuildOptions)` from `index_builder.go`.
2. Stage 1 output provides `SortedIDs` aligned with encoded text/shape streams.
3. Stage 2: call taxonomy builder with `SortedIDs` + taxonomy input envelope.
4. Stage 2 constructs columnar brand/category sections aligned to Stage 1 sorted product order.
5. Final payload assembly can append Stage 2 sections after Stage 1 binary sections.

### 3. RecordInput Context Semantics
- Product entry:
- Uses `ID`, `Text`, `BrandID`, `CategoriesIDs`.
- Brand dictionary entry:
- Uses `ID`, `Text` (brand name).
- Category dictionary entry:
- Uses `ID`, `Text` (category name).

### 4. TaxonomyBuildResult Additions (Columnar)
Add stage-2 binary-aligned columnar arrays:
- `BrandIDs []uint16`
- `BrandNames []string`
- `CategoryIDs []uint16`
- `CategoryNames []string`
- `ProductBrandIndexesU8 []uint8`
- `ProductBrandIndexesU16 []uint16`
- `ProductCategoryCount []uint8` (bit-packed 2-bit counters, 4 products per byte)
- `ProductCategoryIndexes []uint8` (flat category index values, read using `ProductCategoryCount`)

Note:
- `BrandIDs` and `BrandNames` are ordered by first appearance in `Products` (through `BrandID`).
- `ProductBrandIndexesU8` stores indexes for the first brand segment.
- `ProductBrandIndexesU16` stores indexes for the remaining brand segment.
- Decoder concatenates both brand-index columns to reconstruct the full per-product brand index sequence.

## Mapping Strategy

### 1. Brand Mapping
1. Build ordered brand dictionary by first appearance in `input.Products` (via `BrandID`).
2. Resolve brand names from `input.Brands` and validate every referenced product brand exists.
3. Assign 0-based `brandIndex` using that first-appearance order.
4. Split brand dictionary/index space into two segments:
5. first segment encoded with `ProductBrandIndexesU8`.
6. remaining segment encoded with `ProductBrandIndexesU16`.
7. Store segment metadata in header so decoder can concatenate both columns in order.

### 2. Category Mapping
1. Count category usage from product references (`input.Products[].CategoriesIDs`).
2. Rank categories by frequency desc, tie-break by category ID asc.
3. Select top `243` category IDs.
4. Build dictionary entries:
5. indexes `0..242`: selected category IDs + names.
6. index `243`: synthetic `"Otros"` with synthetic ID (defined in format spec).
7. Build lookup `categoryID -> categoryIndex` for selected top categories.
8. Map each product category:
9. if in lookup => mapped index.
10. else => `243` (`"Otros"`).
11. Deduplicate repeated mapped category indexes per product.
12. Enforce max 4 categories per product after mapping/deduplication.
13. Encode category counts into `ProductCategoryCount` as 2-bit values.
14. Append mapped category indexes into flat `ProductCategoryIndexes`.

## Binary Format Migration

### 1. Versioning
1. Bump `BinaryVersion` (new format version).
2. Keep explicit decoder branch by version.

### 2. Header Extension
Add section lengths/counts for new columnar blocks:
1. brand IDs bytes
2. brand names bytes
3. category IDs bytes
4. category names bytes
5. product brand-index-u8 bytes
6. product brand-index-u16 bytes
7. product category-count bytes
8. product category-index bytes

Add flags:
1. brand split-index mode enabled (`u8` segment + `u16` segment)
2. taxonomy sections enabled

### 3. Section Order
Recommended fixed order after existing sections:
1. dictionary section
2. shape section
3. content section
4. brand IDs section
5. brand names section
6. category IDs section
7. category names section
8. product brand index u8 section
9. product brand index u16 section
10. product category count section
11. product category indexes section

Category index storage rule:
1. `ProductCategoryIndexes` is always `uint8` (no dynamic width mode).
2. `ProductCategoryCount` stores 2-bit counters (`0..3`) for each product.
3. Counter value `3` means 4 categories (encoded as `countMinusOne`).

## Decoder Plan
1. Parse new version header + flags.
2. Decode all taxonomy dictionary sections.
3. Decode `ProductBrandIndexesU8` and `ProductBrandIndexesU16`.
4. Concatenate both decoded brand-index columns to obtain full per-product brand index sequence.
5. Decode per-product 2-bit category counters.
6. Decode flat `uint8` category indexes using per-product counters.
7. Validate index bounds against dictionary lengths.
8. Preserve alignment invariant:
9. product row `i` must map to brand/category columns at `i`.

## Validation Rules
1. Reject build if `input.Products` is empty after normalization.
2. Reject build if any `BrandID` from products is missing in `input.Brands`.
3. Reject build if any dictionary original ID does not fit in `uint16`.
4. Reject build if category dictionary construction fails to place `"Otros"` at index `243`.
5. Reject decode if any product brand/category index is out of dictionary bounds.

## Backward Compatibility Decision
Choose one path before implementation:
1. Strict new-version only decoder (simpler).
2. Dual decoder supporting old and new versions (safer transition).

Recommendation: dual decoder if old `.idx` files are still in use.

## Implementation Phases

### Phase 1: Types and API
1. Add `BuildInput` envelope.
2. Keep `Build` signature unchanged for stage 1.
3. Add `TaxonomyBuildResult` in second-pass file.

### Phase 2: Mapping Engine
1. Implement brand dictionary + strict reference validation.
2. Implement category top-243 + `"Otros"` mapping.
3. Generate per-product mapped taxonomy columns aligned with sorted products.

### Phase 3: Binary Encoder
1. Bump format version.
2. Extend header + flags.
3. Serialize all new sections.

### Phase 4: Decoder
1. Parse new header fields.
2. Decode new sections.
3. Expose decoded taxonomy columns in result type.

### Phase 5: Tooling and Docs
1. Update `backend/libs/index_builder/README.md` input/output docs.
2. Update build/decode CLIs to print taxonomy stats.
3. Add migration notes for binary version change.

### Phase 6: Tests
1. Unit tests for category ranking and `"Otros"` mapping.
2. Unit tests for strict brand-reference validation.
3. Roundtrip encode/decode tests for both brand width modes.
4. Alignment tests ensuring product and taxonomy columns stay index-synced.

## Open Implementation Decisions
1. Synthetic original ID value for `"Otros"` in `CategoryIDs` (example: `0`).
2. Whether to keep old `LoadRecordsFromTextFile` untouched as text-only helper or add a new structured loader for envelope input.

## Suggested Immediate Next Step
1. Confirm the 2 open decisions above.
2. Implement Phase 3 and Phase 4 to serialize/decode stage-2 sections in final binary payload.
