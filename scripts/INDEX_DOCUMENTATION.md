# Index user-facing route documentation

The documentation indexer validates and ingests
`frontend/routes/**/DOCUMENTATION.md` into Qdrant. Qdrant stores both the vectors and the
successful indexing hashes; ScyllaDB is not involved.

## Commands

Validate Markdown, `DOC-ID` values, routes, and source-file hashes without contacting
OpenRouter or Qdrant:

```bash
# Validation is the safe default.
./deploy.sh index_documentation -mode validate
```

Compare with Qdrant without embedding or writing:

```bash
# Supply the private host when qdrant.host is intentionally blank in config.toml.
./deploy.sh index_documentation -mode index -dry-run -qdrant-host 100.64.0.2
```

Perform incremental indexing:

```bash
# Only new or changed chunks call OpenRouter; Qdrant BM25 is generated server-side.
./deploy.sh index_documentation -mode index -qdrant-host 100.64.0.2
```

Use `-document frontend/routes/finance/cash-banks/DOCUMENTATION.md` to process one
document and `-batch-size 16` to change the default embedding/upsert batch size.

## Hash behavior

- Source hashes remain in each Markdown `### FILES` manifest.
- `documentation_file_hash` is SHA-256 over exact Markdown bytes and is stored in Qdrant.
- `documentation_hash` covers normalized indexable content without `FILES`.
- `content_hash` covers the exact dense input for one chunk.
- `index_contract_hash` covers parser, chunker, embedding, dimensions, and BM25 settings.

The `page-purpose` chunk is the completion marker and receives `index_state=complete` only
after all other upserts, payload refreshes, and deletions succeed. A later identical run
uses that marker to skip the document without calling OpenRouter.
