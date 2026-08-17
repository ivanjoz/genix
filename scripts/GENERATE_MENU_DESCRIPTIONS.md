# Generate Menu Descriptions

Creates `tmp/menu_description.json` from route markdown files under `frontend/routes`.

## Usage

```bash
./app.sh generate_menu_descriptions
```

## Markdown Format

A route describes itself in one of two ways. Both produce the same entry; the frontmatter form
takes precedence when present.

### Preferred: `DOCUMENTATION.md` frontmatter

Routes that have a `DOCUMENTATION.md` carry the description in its YAML frontmatter, so the route
keeps a single documentation file:

```yaml
---
schema: 1
page_id: system.companies
route: /system/companies
title: Companies (Empresas)
status: implemented
visibility: saas
description_en: >-
  Tenant company management, SaaS only.
description_es: >-
  Gestión de empresas (tenants), exclusivo SaaS.
---
```

Use a folded block scalar (`>-`) so the text may contain colons and apostrophes without quoting.
These fields sit above the first `<!-- DOC-ID: -->` marker's territory, so they never reach the RAG
index: `backend/agent/ragdocs` builds chunks only from DOC-ID sections and the six identity fields.

### Legacy: `## DESCRIPTION::` stub file

Routes without a `DOCUMENTATION.md` keep a separate `<slug>.md` stub beside `+page.svelte`:

```md
## DESCRIPTION::ES
Texto en espanol.

## DESCRIPTION::EN
English text.
```

When a route gains a `DOCUMENTATION.md`, move both descriptions into its frontmatter and delete the
stub.

## Validation

- Files carrying neither form are skipped.
- Files with only one language fail, so the generated menu data stays complete.
- Two files describing the same route fail. `AttachMenuDescriptions` keys descriptions by route, so
  a duplicate would otherwise silently overwrite the other — this is what catches a stub that was
  left behind after migrating to frontmatter.

## Route mapping

The route comes from the folder when a `+page.svelte` sits beside the markdown file; otherwise it
falls back to the file path minus `.md`. Keep description files next to a real page, or the
generator will publish a route that no menu option can match.
