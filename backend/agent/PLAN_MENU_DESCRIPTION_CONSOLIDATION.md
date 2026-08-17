# Plan — Consolidate route menu descriptions into `DOCUMENTATION.md`

Status: **awaiting approval** · Created 2026-08-16

## Goal

Remove the per-route `<slug>.md` stub files (`empresas.md`, `cajas.md`, …) by moving their
bilingual menu description into the route's `DOCUMENTATION.md`, so each route directory holds one
documentation file instead of two.

## Current state (verified)

Two independent pipelines read markdown under `frontend/routes`:

| | Menu description index | RAG index |
|---|---|---|
| Input | any `.md` containing `## DESCRIPTION::ES` / `::EN` | file named exactly `DOCUMENTATION.md` |
| Reader | `scripts/generate_menu_descriptions.go:50` | `backend/agent/ragdocs/parser.go:41` |
| Output | `tmp/menu_description.json` | Qdrant chunks |
| Consumer | `AttachMenuDescriptions()` → `GET /agent?get=menu` (`backend/agent/agent.go:113`) | RAG retrieval |

Neither file replaces the other. Coverage today — 23 menu entries, 4 `DOCUMENTATION.md`:

- **Both files (3):** `system/companies`, `finance/cash-banks`, `logistics/purchase-orders`
- **Menu stub only (20):** the remaining routes
- **`DOCUMENTATION.md` only (1):** `system/observability` — **currently missing from the agent's
  menu index entirely**
- **Neither, but referenced elsewhere (4):** `finance/expenses/EXPENSES.md`, `welcome/welcome.md`,
  `webpage-builder/docs/BUILDER_INSTRUCTIONS.md`, `webpage-builder/templates/HOW_TO_CREATE_TEMPLATE.md`

So consolidation deletes 3 files today; the rest follows as `DOCUMENTATION.md` coverage grows.

## Spike results

Inserted `## DESCRIPTION::` blocks into `system/companies/DOCUMENTATION.md` between the frontmatter
and the first `<!-- DOC-ID: page-purpose -->`, ran both pipelines, reverted.

**Confirmed safe for RAG.** `parseSections` (`parser.go:157`) only reads from the first DOC-ID
comment onward, so text above it is never a section. Validation stayed green at `sections=8
chunks=8`, identical to baseline.

**Confirmed zero embedding cost.** Chunks are built strictly from `document.Sections` +
frontmatter (`chunker.go:21`), and re-embedding is decided per chunk by `ContentHash`
(`indexer.go:112`). Only `Document.FileHash` changes, which costs one payload-update pass with
`Embedded=0` — no OpenRouter spend.

**Blocker A — description text gets polluted.** `parseDescriptionBlocks`
(`scripts/generate_menu_descriptions.go:112`) only terminates a block on a `## ` heading. An HTML
comment is not one, so the trailing DOC-ID marker leaked into the generated English description:

```json
"description": "Tenant company management, SaaS only.\n\n<!-- DOC-ID: page-purpose -->"
```

**Blocker B — duplicate routes resolve silently.** With both files present the generator emitted
*two* entries for `/system/companies` with no warning (23 → 24), and `loadMenuDescriptions`
(`menu_descriptions.go:90`) writes them into a map keyed by route, so the last one silently wins.
A half-finished migration would corrupt a description with no visible error.

**Pre-existing failure, unrelated but blocking.** `documentation-index -mode=validate` currently
exits 1: `system/observability/DOCUMENTATION.md` has a stale evidence hash for
`backend/main-handlers.go` (stored `10902f13…`, current `2474ede3…`). The validator aborts the
whole run on the first bad document, so this must be fixed before the migration can be verified.

**Pre-existing bad entry.** `webpage-builder/builder/builder-store.md` sits in a folder with no
`+page.svelte`, so `routeFromMarkdownPath` (`generate_menu_descriptions.go:153`) falls back to the
filename and publishes a non-existent route `/webpage-builder/builder/builder-store`.

## Design decision

### Option A (recommended) — carry the description in YAML frontmatter

Add `description_en` / `description_es` to the `DOCUMENTATION.md` frontmatter and teach the menu
generator to read it:

```yaml
---
schema: 1
page_id: system.companies
route: /system/companies
title: Companies (Empresas)
status: implemented
visibility: saas
description_en: Tenant company management, SaaS only. Create and edit companies …
description_es: Gestión de empresas (tenants), exclusivo SaaS. Crear y editar empresas …
---
```

Why this over Option B:

- **Blocker A disappears.** YAML parsing has real delimiters; nothing can bleed into the value.
- Structured identity metadata belongs in frontmatter, next to `route` and `title`.
- No dependence on body layout or DOC-ID ordering.
- `canonicalDocumentation` (`parser.go:263`) only serializes the six known frontmatter fields, so
  `DocumentationHash` and every chunk `ContentHash` stay byte-identical — same zero-embed property.
- `yaml.Unmarshal` ignores unknown keys, so existing documents keep validating during migration.

Cost: the generator must parse frontmatter in addition to `## DESCRIPTION::` blocks. Both formats
coexist until the 20 stub-only routes get a `DOCUMENTATION.md`.

### Option B — keep `## DESCRIPTION::` blocks, move them above the first DOC-ID

Verified to work, but requires fixing the block terminator (Blocker A) and keeps a
whitespace-sensitive convention that breaks if anyone reorders the file.

## Steps

Assumes Option A. Each step is independently committable.

1. **Unblock validation.** Refresh the stale `backend/main-handlers.go` evidence hash in
   `system/observability/DOCUMENTATION.md` and confirm `documentation-index -mode=validate` exits 0
   for all 4 documents. *(Independent fix — worth doing regardless of this plan.)*

2. **Teach the generator to read frontmatter.** In `scripts/generate_menu_descriptions.go`, before
   the `## DESCRIPTION::` scan, parse a leading `---` YAML block and use `description_en` /
   `description_es` when both are present. Keep the existing both-or-error rule.

3. **Hard-error on duplicate routes** (Blocker B). In `collectMenuDescriptions`, fail with both
   contributing paths if two files map to the same route. This makes a half-done migration loud
   instead of silent, and is the safety net for every later step.

4. **Migrate the 3 overlapping routes.** For `system/companies`, `finance/cash-banks`,
   `logistics/purchase-orders`: copy the ES/EN text into frontmatter, delete `empresas.md`,
   `cajas.md`, `purchase-orders.md`. Verify the entry count holds at 23 and each description text
   is unchanged from the pre-migration `menu_description.json`.

5. **Close the observability gap.** Add `description_en` / `description_es` to
   `system/observability/DOCUMENTATION.md`, bringing the index to 24 and giving the agent a
   description for a route it currently cannot describe.

6. **Fix the bogus builder route.** Decide whether `/webpage-builder/builder/builder-store` should
   exist. If not, delete `builder-store.md` or drop its DESCRIPTION blocks; if it should point at a
   real page, correct the mapping. *(Independent of the consolidation — flagged, not assumed.)*

7. **Update the conventions.** Record the frontmatter fields in
   `scripts/GENERATE_MENU_DESCRIPTIONS.md`, the `document-user-routes` skill, and
   `assets/DOCUMENTATION.template.md`, so new routes ship one file. Note that the 20 stub-only
   routes keep `## DESCRIPTION::` until they get a `DOCUMENTATION.md`.

## Verification

- `./app.sh generate_menu_descriptions` — entry count 23 → 23 (step 4) → 24 (step 5), no duplicates.
- Diff `tmp/menu_description.json` before/after step 4: only the source path changes, never text.
- `go run ./agent/cmd/documentation-index -mode=validate -root=..` — exits 0, `chunks` per document
  unchanged (`companies=8`, `cash-banks=9`, `purchase-orders=12`).
- `go test ./agent/ragdocs/...`.
- `-mode=index -dry-run` on a migrated document: expect `embed=0`, payload-only updates.

## Out of scope

- Writing `DOCUMENTATION.md` for the 20 stub-only routes.
- Changing chunking, embedding, or retrieval.
- The 4 non-indexed `.md` files — all referenced from code or other docs; keep them.
