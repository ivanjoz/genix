# Generate Menu Descriptions

Creates `tmp/menu_description.json` from the `DOCUMENTATION.md` files under `frontend/routes`.

## Usage

```bash
./app.sh generate_menu_descriptions
```

## Markdown Format

`DOCUMENTATION.md` is the only source. Every other markdown file under `frontend/routes` is
ignored, including the `## DESCRIPTION::` stub files this generator used to read. A route describes
itself in that file's YAML frontmatter:

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

## Validation

- A document omitting both fields is skipped, so a route may be documented for retrieval before it
  earns a menu entry.
- A document with only one language fails, so the generated menu data stays complete.
- A `DOCUMENTATION.md` with no `+page.svelte` beside it fails, because the resulting route could
  never match a menu option.

## Route mapping

The route is the document's own directory relative to `frontend/routes` — never the file name. One
`DOCUMENTATION.md` per directory therefore means a route can never be described twice.

## Adding a route to the menu

Write the route's `DOCUMENTATION.md` with the `document-user-routes` skill; the description fields
are part of that file. A route without one has no menu description at all, so the agent can
navigate to it but cannot describe it.
