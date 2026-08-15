# PLAN — Documentation ingestion, embeddings, Qdrant hybrid retrieval, and RAG

## Goal

Turn route-local `DOCUMENTATION.md` files into a precise, incremental knowledge base for
the Genix assistant. Retrieval must work for mostly Spanish questions, mixed-language ERP
terminology, exact UI labels, regional synonyms, business rules, limitations, and
navigation requests.

The baseline combines:

- Dense semantic vectors from `qwen/qwen3-embedding-8b` through OpenRouter.
- Lexical BM25 sparse vectors stored and queried in Qdrant.
- Qdrant Query API fusion using reciprocal rank fusion (RRF).
- Payload filtering for implementation status, document freshness, route visibility, and
  current-user access.

The route-document contract and provenance workflow are defined in
`PLAN_ROUTE_DOCUMENTATION.md`.

## Verified model and platform facts

### Qwen3-Embedding-8B

The authoritative model card states that the model:

- Supports more than 100 languages and cross-language retrieval.
- Has a 32K-token context window.
- Produces up to 4096 dimensions and supports Matryoshka dimensions from 32 to 4096.
- Is instruction-aware.
- Recommends an English, task-specific instruction on queries.
- Does not require an instruction on retrieval documents.
- Reports a retrieval-quality loss of roughly 1–5% in many tasks when the query
  instruction is omitted.

Source: <https://huggingface.co/Qwen/Qwen3-Embedding-8B>

OpenRouter exposes the model through `POST /api/v1/embeddings`, accepts batched string
inputs, an optional output dimension, `input_type`, and float/base64 output encodings.

Sources:

- <https://openrouter.ai/docs/api/api-reference/embeddings/create-embeddings>
- <https://openrouter.ai/qwen/qwen3-embedding-8b/api>

### Qdrant

Qdrant can store named dense and sparse vectors on the same point and fuse their result
lists in the Query API. RRF is the safe initial fusion method when there is no labeled
evaluation set because it combines ranks without assuming comparable dense and BM25 score
scales.

Sources:

- <https://qdrant.tech/documentation/search/text-search/hybrid-search/>
- <https://qdrant.tech/documentation/search/hybrid-queries/>

Qdrant full-text payload indexes tokenize and normalize text for efficient matching and
filtering. They are not a substitute for ranked BM25 retrieval. Qdrant supports BM25 as a
sparse representation either through a compatible Qdrant inference capability or through
sparse vectors generated client-side.

Sources:

- <https://qdrant.tech/documentation/search/text-search/full-text-search/>
- <https://qdrant.tech/documentation/search/text-search/text-filtering/>
- <https://qdrant.tech/documentation/inference/>

## Current project state

`config.toml` declares:

```toml
# Qdrant is installed separately; the backend parses these runtime values.
[qdrant]
http_port = 14014
grpc_port = 14015
public = true

# OpenRouter is the default embedding provider for this model id.
[embedding_model]
id = "qwen/qwen3-embedding-8b"
```

The deployment script supports `qdrant.host` and configures Qdrant to require the root
`internal_apikey`. The backend now parses both configuration sections, uses the official
Qdrant Go client over gRPC, has a separate OpenRouter embeddings client, and implements
documentation parsing, chunking, validation, incremental indexing, and native dense/BM25
RRF retrieval. Agent-tool integration and bounded context assembly remain separate phases.

Do not reuse the chat-completion request types for embeddings. Share HTTP construction,
authentication, provider routing, error handling, and observability where useful, but keep
the wire contracts separate.

## Configuration contract

Add explicit parsed runtime values for:

```toml
# host is written by scripts/configure/configure_db.py when deployment can determine it.
[qdrant]
host = ""
http_port = 14014
grpc_port = 14015
public = true
collection = "genix_user_documentation_v1"

# Full 4096 dimensions favor quality for the initially small documentation corpus.
[embedding_model]
provider = "openrouter"
id = "qwen/qwen3-embedding-8b"
dimensions = 4096
```

Defaults:

- `embedding_model.provider`: `openrouter`.
- `embedding_model.id`: fail startup or indexing when blank; do not silently substitute a
  semantically different model.
- `embedding_model.dimensions`: `4096` for this model.
- `qdrant.collection`: `genix_user_documentation_v1`.
- Qdrant API key: reuse `internal_apikey`, matching the deployment contract.
- Qdrant host: loopback when private; require an explicit reachable host when public.

The embedding provider is independent from `providers.model`, which selects the chat model
provider. When `providers.model = "meta"` and `embedding_model.provider = "openrouter"`, the
chat loop uses `agent.meta_key` but documentation indexing and every RAG query still require
`agent.openrouter_key`. Validate the embedding key when the RAG subsystem starts; do not let
the chat-provider startup rule incorrectly treat it as optional.

Reflect these sections in `config.example.toml`, the backend configuration structs, and
deployment documentation. Validate ports, host, model ID, dimension, API key, and
collection name at startup of the indexing/retrieval subsystem.

`public = true` is appropriate only when a remote backend such as Lambda must reach the
database. A public raw HTTP/gRPC endpoint must be protected by Qdrant authentication plus
private networking, a tunnel/VPN, TLS termination, or strict source firewall rules. Never
send `internal_apikey` over an untrusted plaintext network.

## Collection schema

Use one versioned collection rather than mutating an incompatible schema in place.

Named vectors:

- `dense`: 4096-dimensional float vector, cosine distance.
- `lexical`: sparse BM25 vector with Qdrant IDF modifier.

Required payload:

- `schema_version`: parser/chunker schema.
- `document_id`: stable page identifier such as `finance.cash-banks`.
- `section_id`: stable `DOC-ID` plus optional part suffix.
- `point_key`: stable `document_id + section_id + part_index` key.
- `route`: exact frontend route or route pattern.
- `module`: Business, Sales, Logistics, Finance, Security, Configuration, Website, or
  System.
- `title`: mixed-language page title.
- `section_title`: mixed-language section title.
- `section_type`: purpose, concept, capability, rule, exceptional-permission, limitation,
  troubleshooting, or related-page.
- `content`: exact normalized chunk text returned to the LLM.
- `context_header`: short repeated page context prepended for embedding.
- `keywords`: explicit aliases, UI labels, abbreviations, and regional synonyms.
- `status`: `implemented` only in the production collection.
- `visibility`: tenant, SaaS-only, public, or other explicit scope.
- `access_resources`: access identifiers required to expose or navigate to the section.
- `documentation_path`: repository-relative `DOCUMENTATION.md` path.
- `documentation_file_hash`: SHA-256 of the exact `DOCUMENTATION.md` bytes; this is the
  fast whole-file skip key.
- `documentation_hash`: SHA-256 of normalized indexable Markdown.
- `content_hash`: SHA-256 of the exact dense input.
- `document_chunk_count`: expected total points for this document.
- `source_hash_digest`: deterministic digest of the reviewed source-file manifest.
- `index_contract_hash`: digest of model, dimensions, parser/chunker, and lexical settings.
- `index_state`: `chunk` normally and `complete` only on the final completion marker.
- `documentation_current`: whether the point belongs to reviewed current documentation.
- `source_files`: provenance paths supporting this section.
- `source_hashes`: hashes captured when the document was reviewed.
- `indexed_at`: operational timestamp.

`visibility` and `access_resources` are operational filter metadata derived from trusted
route/access catalogs. Do not repeat or embed generic permission enforcement, tenant
isolation, or other system-wide guarantees. Only page-specific exceptional restrictions
belong in `content`.

Create payload indexes for fields used in filters: `document_id`, `route`, `module`,
`section_type`, `status`, `visibility`, `access_resources`, `documentation_path`, and
`content_hash`. Add a full-text payload index to `content` with lowercase, multilingual
tokenization, ASCII folding, and phrase matching for diagnostic/exact-match filtering.

The full-text payload index is supplemental. Ranked lexical retrieval comes from the
`lexical` sparse vector.

## Point identity and collection migration

Generate a deterministic UUID from:

`collection schema version + document_id + section_id + part_index`

Content changes update the same point instead of creating duplicates. Store the content
hash separately to decide whether a new embedding is required.

Store the indexing hashes and completion state in Qdrant payloads, not ScyllaDB. They
describe rebuildable vector-index state and must remain transactionally adjacent to the
points they validate. The `page-purpose` part-zero point is the document completion marker:
write or refresh it only after all other upserts, payload refreshes, and deletions succeed.
An exact-file skip is allowed only when its raw file hash, source digest, index contract,
expected chunk count, and actual returned point count all match.

When the dense model, output dimension, chunk construction, or sparse tokenization changes,
create a new versioned collection, populate it completely, run evaluation, then switch a
small alias/config pointer. Do not mix vectors generated by different model contracts in
one named vector.

## What text to embed

Embed the full natural-language content of each semantic chunk. Do not embed only extracted
keywords.

Keywords alone lose:

- Conditions such as “only after delivery.”
- Negation such as “does not change the balance.”
- Relationships between sales, payments, cash movements, stock, and debt.
- Business rationale and limitations.
- The difference between a capability and an adjacent unsupported operation.

Use explicit keywords as additional payload and lexical vocabulary, not as a replacement
for prose.

Do not remove connectors from dense text. Natural syntax is how the embedding represents
conditions and business logic. Never strip `not`, `only`, `unless`, `without`, `before`,
`after`, `no`, `solo`, `excepto`, `sin`, `antes`, or `después`.

For BM25, do not run an LLM keyword extractor over every chunk. Index the complete chunk so
exact UI labels and unanticipated user terms remain searchable. BM25 already down-weights
very common terms through IDF. A small explicit `keywords` list may repeat high-value
aliases, but the original text stays authoritative.

## Chunking strategy

### Semantic boundaries first

Parse Markdown by stable `DOC-ID`, not by a fixed number of characters. Each initial chunk
should express one complete concept:

- Page purpose and scope.
- One capability with its prerequisites, rules, effects, and limitations.
- A coherent group of cross-capability rules.
- An exceptional page-specific restriction, when one exists.
- One troubleshooting group.
- Related navigation and workflows.

Never embed YAML frontmatter, HTML maintenance comments, `DOCUMENTATION_GAP` comments, or
the `### FILES` provenance block.

### Size targets

Use these as starting bounds, then tune against the evaluation set:

- Preferred: 250–600 model tokens including the context header.
- Soft maximum: 700 tokens.
- Hard maximum: 900 tokens.
- Minimum: approximately 100 tokens; merge smaller neighboring material when it remains
  semantically coherent.

Qwen supports much longer inputs, but the model context limit is not the retrieval-optimal
chunk size. Very large chunks combine unrelated capabilities and reduce the precision of
both dense similarity and BM25 length normalization. Very small chunks lose prerequisites,
rationale, and exceptions.

When a `DOC-ID` exceeds the hard maximum:

1. Split at its `###` subsections or paragraph groups.
2. Keep prerequisites, rule, rationale, result, and limitation together when they describe
   the same action.
3. Assign deterministic part indices.
4. Add a maximum 50-token overlap only when a necessary sentence relationship crosses the
   split. Section-based chunks normally need no overlap.

### Context header on every chunk

Yes: every chunk should contain the title and one short page description. A Qdrant point is
retrieved independently and cannot assume that the preceding chunk was also retrieved.

Construct a concise 30–80 token header rather than repeating the full introduction:

```text
# Repeated context disambiguates the independently retrieved section.
Genix ERP — Finance (Finanzas). Page: Cash & Banks (Cajas y Bancos).
Route: /finance/cash-banks. Scope: financial accounts, balances, movements, and cash
reconciliation (cuadre o arqueo de caja).
Section: Register a manual movement (Registrar un movimiento manual).
```

Dense document input is:

`context_header + section content`

Returned LLM context may contain the same text, or present the header as metadata and the
section as prose. Do not repeat a long page summary in every point; it wastes tokens and can
make all chunks from a page artificially similar.

### Neighbor expansion

Store `part_index`, `part_count`, previous point ID, and next point ID. After retrieval,
include one adjacent part only when a split capability needs it. Do not automatically return
the entire `DOCUMENTATION.md`.

## Dense embedding contract

### Documents

Send the constructed document text without a retrieval instruction. Batch multiple strings
through OpenRouter while respecting request/token limits.

Use float output and omit `input_type` unless the selected OpenRouter provider is verified to
apply the intended Qwen document behavior. The explicit document construction is the stable
contract; provider-specific implicit transformations are not.

### Queries

Preserve the user's original question except for whitespace normalization. Prefix this exact
English instruction, as recommended by the Qwen model card:

```text
# The instruction is part of the query embedding, never the document embedding.
Instruct: Given a mostly Spanish question from a small-business ERP user, retrieve Genix
documentation passages that answer what the software can do, required conditions, business
rules, limitations, side effects, or where the user should navigate.
Query: {user question}
```

The instruction remains English even when the user asks in Spanish. Do not translate,
summarize, or keyword-reduce the user question before dense encoding.

### Dimension

Use the full 4096 dimensions initially. The corpus is small enough that the storage cost is
modest, and accuracy is the stated priority. Assert that every OpenRouter response contains
exactly the configured dimension before upserting it.

Qwen supports smaller Matryoshka dimensions. Evaluate 1024 or 2048 later only through a new
collection and the same labeled queries; never change a live collection's dimension.

## Lexical search contract

### Recommended baseline

Use Qdrant BM25 sparse vectors over the same complete mixed-language chunk text. Configure
ingestion and query identically.

Because the documents deliberately mix English and Spanish, begin with:

- Multilingual tokenizer.
- Lowercasing enabled.
- ASCII folding enabled so `gestion` can match `gestión`.
- No language-specific stemmer.
- No automatic English-only stopword list.
- No manual connector stripping.

This preserves exact bilingual UI and business terminology. A single-language English or
Spanish stemmer would normalize only half of the corpus and may remove meaningful negations.
BM25 IDF already reduces the effect of words common across the collection.

Test a conservative bilingual stopword list later, but never remove negations, conditions,
temporal connectors, or exception words. Accept it only if retrieval evaluation improves.

### Capability gate for the deployed Qdrant

The deployment installs a self-hosted Qdrant binary and currently tracks the latest release.
Before committing to server-side `qdrant/bm25` document inference:

1. Pin a minimum supported Qdrant version in deployment documentation.
2. Run an integration test against the same self-hosted binary and configuration used in
   production.
3. Create a temporary collection with an IDF sparse vector.
4. Upsert and query a `qdrant/bm25` document with the required multilingual options.
5. Verify Spanish accents, English terms, and mixed-language text.

If the deployed binary does not provide that inference contract, choose explicitly:

- Generate compatible sparse vectors client-side and send them to Qdrant; or
- Use the existing GenixSearch service for ranked lexical candidates and fuse its ranks
  with Qdrant dense results in Go.

Do not call a payload full-text filter “BM25.” It can enforce exact token/phrase conditions,
but it is not the same ranked lexical retriever.

### Keywords and exact terms

Index full text, plus a compact list of curated terms already present in each documentation
section:

- Exact UI labels in both languages.
- ERP synonyms and regional vocabulary.
- Abbreviations.
- Route names.
- Common user phrasings and important misspellings.

Do not use an LLM to invent an opaque keyword-only representation. It increases ingestion
cost, is difficult to reproduce, and may silently omit the future query term.

## Ingestion pipeline

1. Discover only `frontend/routes/**/DOCUMENTATION.md`.
2. Parse and validate frontmatter, `DOC-ID` values, and `FILES` YAML.
3. Recalculate provenance hashes. Reject production ingestion when hashes are pending,
   missing, or stale.
4. Remove non-indexable metadata and comments.
5. Split each `DOC-ID` by the semantic chunking rules.
6. Construct the repeated context header.
7. Normalize only line endings and redundant whitespace; preserve punctuation, accents,
   case in stored content, and business connectors.
8. Calculate deterministic point ID, documentation hash, and content hash.
9. Compare with the existing Qdrant points for `documentation_path`:
   - Exact raw file hash, source digest, contract, and complete point count: skip the file
     without calling OpenRouter.
   - Unchanged content hash: keep dense/sparse vectors and update payload if needed.
   - Changed/new content hash: regenerate dense and lexical representations.
   - Removed point key: delete the obsolete point.
10. Batch OpenRouter dense requests and validate response count, ordering, dimension, and
    finite numeric values.
11. Generate or request the BM25 representation using the verified lexical path.
12. Upsert with `wait=true`; commit the `page-purpose` completion marker last.
13. Record an ingestion summary without logging document contents or vectors.

The index operation must be idempotent. An interrupted run can restart without duplicates.
Use bounded retries with jitter for OpenRouter 429 and transient 5xx responses, but fail on
authentication, unsupported model, dimension mismatch, invalid Markdown, or stale evidence.

## Retrieval pipeline

1. Receive the original user question plus authenticated user/access context.
2. Build the instructed Qwen query input and request one dense vector from OpenRouter.
3. Build the BM25 query using the unchanged user text and the exact ingestion tokenizer
   options.
4. Apply payload filters before vector search:
   - `status = implemented`.
   - Documentation is current.
   - Visibility is compatible with the deployment and company.
   - `access_resources` intersects the user's accessible resources when navigation or
     protected operations are involved.
5. Prefetch approximately 25 dense and 25 lexical candidates.
6. Fuse with unweighted RRF as the initial safe default.
7. Return approximately 8 fused candidates.
8. Deduplicate near-identical parts and limit repeated chunks from one page unless the
   question is specifically about that page.
9. Expand one adjacent chunk when a retrieved split capability needs its continuation.
10. Assemble a bounded context, initially 3,500–5,000 tokens, preserving route, title,
    section, source document, and freshness metadata.
11. Instruct the answering agent to ground claims in retrieved text, distinguish limitations
    from capabilities, and cite or navigate to the documented route.

Do not apply a fixed weighted sum directly to cosine and BM25 scores; their scales differ.
RRF uses rank and avoids that mismatch. Tune weighted RRF only after building a labeled
evaluation set.

Do not set a production similarity threshold by intuition. Log score distributions during
evaluation, measure false positives and misses, then choose thresholds per retriever if they
improve results.

## Agent integration

Add a retrieval operation to the agent loop before it attempts page navigation or answers a
product-capability question. Keep the accessible menu as compact orientation context, but use
RAG for detailed rules and workflows.

The retrieval result supplied to the LLM should contain:

- Page title and route.
- Section title and content.
- Implementation/freshness status.
- Required access resources.
- Documentation path for diagnostics.
- A stable citation identifier.

The LLM must still inspect the live page after navigation. Documentation explains product
behavior and target actions; the live page snapshot supplies the current interactive handles.

## Failure behavior

- OpenRouter unavailable: return a clear temporary knowledge-search error; do not substitute
  an embedding from a different model into the collection.
- Qdrant unavailable: the agent may still use its compact accessible menu for basic route
  navigation, but must not claim detailed undocumented business rules.
- BM25 unavailable but dense healthy: log degraded retrieval and allow dense-only search only
  if explicitly configured; expose this state in health diagnostics.
- Stale documentation: exclude stale sections by default and report the affected route to
  maintainers.
- No confident result: say that the documentation does not establish the answer, then ask a
  focused question or offer the closest relevant route without fabricating behavior.

## Logging and metrics

Log structured diagnostics extensively while protecting content and secrets:

- Model ID, configured dimension, request item count, token usage, latency, retry count, and
  response dimension.
- Qdrant host label—not credentials—collection, operation, candidate counts, filters, latency,
  and result point IDs.
- Parser counts, chunk token-size histogram, changed/unchanged/deleted points, and stale-file
  failures.
- Retrieval mode: hybrid, dense-only degraded, or unavailable.
- Dense and lexical rank positions and fused ranks for evaluation IDs.

Never log API keys, full vectors, full user questions in production, or complete retrieved
passages. Use request IDs and hashes for correlation.

## Evaluation strategy

Build a labeled dataset before tuning. Include at least:

- Natural Spanish questions from small-business users.
- Mixed English–Spanish questions and exact UI labels.
- Regional synonyms: `cuadre`, `arqueo`, `caja chica`, `despacho`, `abono`, and similar.
- Missing accents and common misspellings.
- Questions about prerequisites, rationale, limitations, calculations, and side effects.
- Navigation questions.
- Confusable routes such as Cash & Banks versus Cash Movements, Sales Management versus
  Sales Report, or Products versus Stock Changes.
- Negative questions: “¿esto cambia el saldo?”, “¿puedo pagar sin caja?”, and “¿se puede
  eliminar después de usarlo?”.
- Unanswerable or planned-feature questions that must not produce confident claims.

Measure separately:

- Chunk recall@5 and recall@10.
- Correct-route recall and mean reciprocal rank.
- Correct capability/business-rule retrieval.
- Answer groundedness and unsupported-claim rate.
- Navigation accuracy.
- Spanish and mixed-language subsets.
- Latency and OpenRouter token cost.

Run controlled comparisons using the same corpus and questions:

1. Dense only versus BM25 only versus hybrid RRF.
2. Full natural text versus keyword-only lexical text.
3. Context header present versus absent.
4. Approximate 300-, 600-, and 900-token chunks.
5. Full 4096 dimensions versus 2048/1024 in separate collections.
6. No stopword removal versus a conservative bilingual list.

The recommended baseline remains full text, context headers, semantic 250–600-token chunks,
4096 dense dimensions, multilingual no-stem BM25, and unweighted RRF. Change it only when the
labeled evaluation demonstrates a better configuration.

## Implementation phases

### Phase 1 — Contracts and capability tests

- Finalize Markdown/provenance parsing from `PLAN_ROUTE_DOCUMENTATION.md`.
- Parse and validate `[qdrant]` and `[embedding_model]`.
- Add the official Qdrant Go client.
- Add an OpenRouter embeddings client and tests with a fake HTTP server.
- Pin/verify Qdrant version and run the self-hosted BM25 capability test.

### Phase 2 — Deterministic ingestion

- Implement Markdown parsing, semantic splitting, context headers, stable IDs, and hashes.
- Create the versioned collection and payload indexes.
- Implement incremental dense/BM25 generation and idempotent upsert/delete.
- Provide a dry-run mode that reports changes without OpenRouter or Qdrant writes.

### Phase 3 — Retrieval

- Implement instructed query embedding, lexical query construction, permission filters,
  RRF, deduplication, neighbor expansion, and bounded context assembly.
- Add a retrieval tool/service to the agent loop.
- Preserve current menu-based navigation as a compact fallback.

### Phase 4 — Evaluation and tuning

- Create the labeled Spanish/mixed-language retrieval set.
- Record baseline metrics.
- Tune chunk bounds, candidate counts, RRF weights, thresholds, and optional dimensions only
  from measured results.

### Phase 5 — Operations

- Add health checks for OpenRouter embeddings, Qdrant, collection schema, and lexical mode.
- Add CI documentation/provenance validation.
- Add an administrative index command with dry-run, incremental, rebuild, and collection
  migration modes.
- Document backups and restore of the versioned Qdrant collection.

## Acceptance criteria

- The configured OpenRouter model produces validated 4096-dimensional document and query
  vectors using the correct asymmetric instruction contract.
- Large Markdown files are divided at semantic `DOC-ID` boundaries; no production chunk
  exceeds the hard limit without an explicit validator error.
- Every chunk repeats concise page, route, scope, and section context.
- Full natural text—not keywords alone—is used for dense and BM25 retrieval.
- No manual connector removal changes conditions, negations, or rationale.
- Qdrant hybrid queries fuse dense and ranked lexical candidates with RRF.
- Payload filters prevent planned, stale, invisible, or unauthorized content from guiding
  the agent.
- Incremental indexing embeds only changed chunks and deletes obsolete deterministic IDs.
- Logs make ingestion and retrieval failures diagnosable without exposing credentials,
  vectors, or document contents.
- A labeled Spanish/mixed-language evaluation demonstrates the final settings and protects
  them from unmeasured regressions.
