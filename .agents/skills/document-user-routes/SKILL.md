---
name: document-user-routes
description: Create or update bilingual `DOCUMENTATION.md` files for implemented Genix frontend routes, covering verified user workflows, navigation, business rules, limitations, and source provenance. Use for new route documentation or reviews after source changes.
---

# Document User Routes

Produce verified product documentation for end users and the Genix Qdrant RAG collection.
Write one `DOCUMENTATION.md` beside each eligible `frontend/routes/**/+page.svelte`.

## Read the contract first

Read `../../../backend/agent/PLAN_ROUTE_DOCUMENTATION.md` completely before examining or
editing a route. It is authoritative for eligibility, evidence, language, provenance, stable
IDs, maintenance behavior, and validation.

Use `assets/DOCUMENTATION.template.md` as the starting structure. Inspect these reviewed
examples when wording or granularity is unclear:

- `../../../frontend/routes/finance/cash-banks/DOCUMENTATION.md`
- `../../../frontend/routes/logistics/purchase-orders/DOCUMENTATION.md`

Read `../../../backend/agent/PLAN_DOCUMENTATION_RAG.md` only when changing chunk construction,
index payloads, retrieval, or ingestion—not for an ordinary route documentation task.

## Scope the work

Document implemented pages real users can navigate to, including user-reachable dynamic routes
and explicitly marked SaaS-only pages. Exclude tests, showrooms, developer UI, internal-only
components, and planned menu entries without an implemented page.

For batch work, inventory eligible routes first and process a small reviewable batch. Keep one
document per route directory and one globally stable `page_id` per page.

## Trace evidence across layers

Do not write from `+page.svelte` alone. Build an evidence graph:

1. Read the route entry point and route-local components controlling visible behavior.
2. Follow imported services, request fields, response mappings, state labels, transformations,
   and user-visible errors.
3. Locate the backend handlers reached by those services.
4. Follow validation, calculations, status transitions, balance/stock/debt changes, and related
   record creation into business functions.
5. Read data models needed to interpret page-specific states and relationships.
6. Read the menu definition for the visible navigation path and labels.
7. Read comments and existing documents only as leads; verify them against implemented code.

Use `rg` for paths, symbols, endpoint names, service methods, UI labels, and status constants.
Record every materially supporting file while researching so the final `FILES` manifest is not
reconstructed from memory.

Do not list generic buttons, icons, styling, or shared components unless their behavior supports
a documented claim.

## Map behavior before prose

For each capability, establish:

- User goal and exact visible trigger.
- Menu, route, view/tab/layer/modal, and UI action label.
- Required existing records, states, and fields.
- Required, optional, derived, fixed, and normalized values.
- Backend rejection conditions and user-understandable errors.
- Calculations and state transitions.
- Created or updated records and effects on balances, stock, debt, reports, or workflows.
- Whether the operation can be edited, canceled, reversed, deleted, copied, or repeated.
- Intentional limitations and operations that do **not** happen.
- Adjacent routes and when the user should use them instead.
- Spanish questions, UI terms, regional synonyms, abbreviations, and useful misspellings.

Keep one capability and its conditions together under one stable `DOC-ID`. Split by user intent,
not by source file or frontend component.

## Write for retrieval and users

Use natural mixed English–Spanish conceptual prose. Pair ambiguous ERP nouns, actions, states,
and visible labels in the same sentence:

> A reconciliation (`cuadre` or `arqueo`) compares the observed cash with the balance Genix
> expects for that account.

Do not duplicate the whole document into separate languages. Preserve connectors and negations
such as `only`, `unless`, `not`, `solo`, `excepto`, `sin`, and `después`; they carry business
meaning and must remain in embeddings and BM25 text.

Explain what each term means inside Genix rather than providing translation pairs alone. Use
exact visible labels and realistic Spanish user questions.

## Keep content page-specific

Exclude generic rules that apply everywhere unless this page has an exceptional behavior users
must understand. In particular, do not spend retrieval space repeating:

- Normal tenant/company isolation.
- Universal permission enforcement.
- Internal cent storage or generic currency representation.
- Generic server validation or authentication.
- Implementation abstractions such as Go fields, JSON keys, endpoint names, or database types.

Translate source evidence into product behavior. Document exceptional page-specific access or
money behavior only where it changes the user's workflow or expectation.

## Separate facts from rationale

State verified behavior directly. Explain why a rule exists only when an authoritative document,
explicit code comment, or human answer supports that rationale.

If missing rationale or contradictory behavior materially changes the document, ask one focused
question. If the rest can be completed safely, add a non-indexed gap comment using the format in
the route plan. Never polish a guess into an authoritative statement.

## Preserve stable identities

- Keep `page_id` stable across title or heading changes.
- Give every independently retrievable section one unique `DOC-ID`.
- Keep existing `DOC-ID` values when updating the same semantic capability.
- Create a new ID only for a genuinely new semantic unit.
- Map every `FILES.supports` entry to an existing `DOC-ID`.

Qdrant point identity depends on these values. Renaming them unnecessarily creates deletion and
reinsertion work and breaks stable citations.

## Build provenance last

Place `### FILES` at the end in the exact YAML form from the template.

- Use POSIX repository-relative paths.
- Include every file materially supporting a claim.
- Assign only roles allowed by the route plan.
- Do not include `DOCUMENTATION.md` itself.
- Use whole-file SHA-256 over exact bytes as `sha256:<hex>`.
- Leave `hash: pending` during drafting.
- Replace pending values only after claim-by-claim review of the completed document.
- When updating, never accept a changed source hash before reviewing every supported `DOC-ID`.

The `DOCUMENTATION.md` file hash used for incremental embedding is stored automatically in
Qdrant by the indexer. Do not add that self-hash to `FILES`.

Compute reviewed source hashes with the repository's normal SHA-256 tool and paste exact values.
Do not build a new hash script inside a route folder.

## Review and validate

Review every claim against the complete current files, not only a diff. Confirm exact frontend
labels/navigation and backend rules/side effects. Remove claims supported only by assumptions or
planned code.

Run validation from the repository root:

```bash
# Validate routes, IDs, FILES paths, and exact source hashes without external writes.
./deploy.sh index_documentation -mode validate
```

For one document during iteration:

```bash
# Restrict validation to the route document currently under review.
./deploy.sh index_documentation -mode validate \
  -document frontend/routes/<module>/<feature>/DOCUMENTATION.md
```

Do not run live indexing unless the user asks for ingestion or indexing. Validation must finish
without pending hashes, stale hashes, invalid paths, duplicate IDs, or route mismatches.

## Send the document to Qdrant

After validation, use the repository indexer rather than calling Qdrant directly. Process the
route being documented first so an error does not block unrelated pages:

```bash
# Compare this document with Qdrant without generating embeddings or writing points.
./deploy.sh index_documentation -mode index -dry-run \
  -document frontend/routes/<module>/<feature>/DOCUMENTATION.md \
  -qdrant-host <reachable-host>
```

If the dry-run is correct and the user requested indexing, perform the incremental write:

```bash
# Embed and upsert only new or changed chunks from this document.
./deploy.sh index_documentation -mode index \
  -document frontend/routes/<module>/<feature>/DOCUMENTATION.md \
  -qdrant-host <reachable-host>
```

Omit `-qdrant-host` when `qdrant.host` is already configured. A successful run stores the exact
Markdown SHA-256 as `documentation_file_hash` in Qdrant. Later runs skip an unchanged completed
document; changed content re-embeds only affected chunks, while provenance-only changes update
payloads without generating a new embedding.

Use the same command without `-document` only when the user requests indexing every route.
Live indexing writes to Qdrant and may call OpenRouter, so do not infer authorization from a
documentation-only request.

## Update an existing document

When source files changed:

1. Identify mismatched `FILES` entries and their `supports` IDs.
2. Inspect the Git diff for focus and the full current files for context.
3. Re-review mapped sections plus logically dependent sections.
4. Update capabilities, rules, limitations, navigation, vocabulary, and rationale as required.
5. Discover new or removed evidence dependencies.
6. Recompute source hashes only after review.
7. Validate the document.

A formatting-only source change still requires explicit review before accepting its new hash.
Do not rewrite unrelated sections or rename stable IDs during maintenance.

## Completion checklist

- The page is implemented and eligible.
- Page purpose and business scope are clear.
- Every real user capability is covered.
- Navigation, prerequisites, rules, side effects, limitations, and recovery are page-specific.
- Important Spanish questions and vocabulary appear naturally.
- Unsupported behavior and negations remain explicit.
- Rationale is sourced or marked unknown.
- Every retrievable section has a stable unique `DOC-ID`.
- Every material claim is traceable through `FILES`.
- All source hashes are reviewed and current.
- Generic system-wide abstractions are absent.
- The documentation validator passes.
