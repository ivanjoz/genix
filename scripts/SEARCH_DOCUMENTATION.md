# Search user-facing route documentation

The documentation search command performs read-only hybrid retrieval against
`genix_user_documentation_v1`. It embeds the original user question with the configured
Qwen query instruction, prefetches dense and Qdrant BM25 candidates, and uses native
reciprocal rank fusion (RRF).

## Commands

Search one Spanish or mixed-language question:

```bash
# Results include the frontend route and a stable documentation citation.
./deploy.sh search_documentation \
  -question "¿Cómo registro un ingreso manual en caja?" \
  -qdrant-host 100.64.0.2
```

Run the built-in Spanish examples:

```bash
# This is a read-only retrieval check, but each question calls OpenRouter once.
./deploy.sh search_documentation -examples -limit 3 -qdrant-host 100.64.0.2
```

Return the full integration structure as JSON:

```bash
# JSON contains evidence, route, page, section, hashes, and fused score; vectors are omitted.
./deploy.sh search_documentation \
  -question "¿Qué pasa si el arqueo tiene una diferencia?" \
  -json \
  -qdrant-host 100.64.0.2
```

Optional exact filters are `-route`, `-module`, and `-visibility`. `-candidates` controls
the candidates from each retriever and `-limit` controls final fused results.

## Retrieval contract

- The dense branch uses `embedding.Client.EmbedQuery`, which adds the Qwen retrieval
  instruction without translating or keyword-reducing the question.
- The lexical branch sends the original question to `qdrant/bm25` with multilingual
  tokenization, ASCII folding, and no language-specific stemmer or stopword removal.
- Both branches require `status=implemented` and `documentation_current=true` before RRF.
- Results include `route`, page and section identifiers, evidence text, provenance hashes,
  part links, and a stable citation ID.
- Dense vectors and API credentials are never returned or logged.

The current examples cover navigation, manual cash movements, cash reconciliation,
purchase-order creation, supplier payments, editing, and cancellation. They are smoke tests,
not a replacement for the labeled retrieval evaluation dataset.
