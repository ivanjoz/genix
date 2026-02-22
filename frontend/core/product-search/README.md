# Product Search (Frontend)

TypeScript decoder + search runtime for Genix product search.

This module exists to reduce search latency and improve cache efficiency in the ecommerce search bar.
It is the base layer for product semantic search and decouples:
- fast name matching (local index)
- full product payload retrieval (`getRecordWithCache` / ID-based cache)

By resolving search by product `ID` (for example: `getRecordWithCache<ISearchProductRecord>("productos-ids", selectedProductID)`) instead of caching by word queries, this strategy improves cache hit rate and keeps query-time work small even with large catalogs.
Current target scale:
- up to ~20k active products
- ~300 KB `.idx` payload when `.zstd` compressed (for 20k active products)

It supports two bootstrap modes:
- `idx_plus_delta`: loads `productos.idx` from CDN and then merges online deltas
- `delta_only`: when `.idx` is missing/unavailable, builds an in-memory index from deltas only

This decoder validates:
- text header and section table
- CRC32 checksums per section
- dictionary, aliases, shapes, content, and product_ids payload consistency
- optional taxonomy block (`GIXTAX01`) and row alignment with text rows

## Files

- `decoder.ts`: main decode functions
- `helpers.ts`: shared helper/utility functions
- `encoder.ts`: reusable query normalization + dictionary-syllable encoding helpers
- `product-search.ts`: productID-keyed search/read model built on decoded payload + deltas
- `productos-delta-service.ts`: cached delta service wrapper for `p-productos-index-delta`
- `types.ts`: constants and TypeScript contracts

## Import Style (No Barrel)

This package intentionally avoids `index.ts` re-exports.

Use direct imports:

```ts
// Decode raw idx bytes only when you need low-level inspection/debug.
import { decodeBinary, readBuildSunixTimeFromHeader } from "$core/product-search/decoder";
// Helpers for diagnostics and contract validation.
import { brandEncodingName, sampleDecodedRecords } from "$core/product-search/helpers";
// Query normalization and dictionary syllable encoding utilities.
import { normalizeAndFilterQueryTokens, encodeQueryWordToDictionarySyllables } from "$core/product-search/encoder";
// Main high-level search runtime.
import { ProductSearch } from "$core/product-search/product-search";
// Shared data contracts.
import type { DecodeResult, DecodedRecord, DecodedTaxonomy, IndexedProduct } from "$core/product-search/types";
```

## Main API

```ts
// Optional: low-level decode path for diagnostics.
const decodeResult = decodeBinary(indexBytes);
const updatedSunix = readBuildSunixTimeFromHeader(indexBytes);
const updatedUnix = updatedSunix * 2 + 1_000_000_000;

// High-level runtime: constructor is sync; readiness is async.
const productSearch = new ProductSearch();
await productSearch.readyPromise;

// Runtime status for UX and observability.
const isReady = productSearch.isReady;
const source = productSearch.source; // "idx_plus_delta" | "delta_only"
const error = productSearch.loadError;

// Product lookup/search APIs.
const product = productSearch.get(42); // IndexedProduct | undefined (product ID)
const updated = productSearch.updated; // int32 build/update watermark
const brandMap = productSearch.brandByProductID; // Map<productID, brandID>
const categoryMap = productSearch.categoryByProductID; // Map<productID, categoryIDs[]>
const productIDs = productSearch.productIDs; // ordered product IDs
const hits = productSearch.search("word1 word2 word3"); // ProductSearchHit[] top 20
```

- `indexBytes`: `Uint8Array | ArrayBuffer`
- return: `DecodeResult`

## Notes

- `decodeBinary` returns `taxonomy: null` if payload only contains the text block.
- text decoder expects the current experimental format: 5 mandatory text sections
  (dictionary, shapes, content, aliases, product_ids).
- `readBuildSunixTimeFromHeader` can read `updated` directly from header without full decode.
- `ProductSearch` fetches `.idx` internally from `Env.makeCDNRoute("live", "c1_products.idx")`.
- After `.idx` load (or failure), it fetches deltas through `ProductosDeltaService`.
- If `.idx` cannot be loaded, `ProductSearch` builds dictionary + product rows from deltas only.
- Search returns ranked product IDs/hints; full product data should be resolved by ID for better cache reuse.
- This two-step strategy (search index + ID fetch) is designed to minimize perceived latency in the search bar.
- `ProductSearch` is productID-based (not row-index-based).
- `ProductSearch.search()` ranking model:
  - product-name match priority: `+4` per matched query token in product name
  - brand prefix influence: `+2` per matched query token in brand words
  - character-prefix fallback on decoded word text to match combinations such as query `cho` over syllables `ch` + `oc`
  - light positional tie-breakers (`+1` first word, `+1` second word)
- Decoder error messages are designed for debug visibility and contract validation.
