---
# Machine metadata must match the route directory and remain stable.
schema: 1
page_id: <module>.<feature>
route: /<module>/<feature>
title: English Page Name (Nombre en Español)
status: implemented
visibility: tenant
---

# English Page Name (Nombre en Español)

<!-- DOC-ID: page-purpose -->
## Page purpose

Explain the page's user-facing purpose in mixed English–Spanish prose, its business scope, and
the adjacent responsibility it does not own.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

Define ambiguous entities, statuses, and workflows as Genix uses them. Distinguish concepts
that users may confuse.

<!-- DOC-ID: capability.replace-with-stable-name -->
## Capability title (Acción en español)

### User intention (Intención del usuario)

Explain the goal this capability solves and when a user chooses it.

### Where to find it (Dónde encontrarlo)

State the menu path, route, view/tab/layer/modal, and exact visible action label.

### Required information and prerequisites (Requisitos previos)

Describe what must exist first and which inputs are required, optional, derived, normalized, or
fixed.

### Business rules and rationale (Reglas y razón de negocio)

Describe page-specific validation, calculations, states, conditions, and supported rationale.
Do not add generic tenant, permission, currency-storage, or server-validation abstractions.

### Result and side effects (Resultado y efectos)

Explain records and states created or updated and effects on balances, stock, debt, reports, or
related workflows.

### Limitations (Limitaciones)

State unsupported operations, irreversible behavior, and common mistaken expectations.

### Common questions and vocabulary (Preguntas y vocabulario)

Include realistic Spanish questions, English equivalents where helpful, UI terms, abbreviations,
regional synonyms, and useful misspellings.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

Include only page-specific rules shared by several capabilities. Delete this section when it
would merely repeat generic system behavior.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

List confirmed causes, checks the user can perform, and safe recovery actions.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

Connect this page to other routes and explain when the user should use each one.

### FILES

```yaml
# Replace pending only after the completed documentation is reviewed against exact file bytes.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/<module>/<feature>/+page.svelte
    role: page
    hash: pending
    supports:
      - page-purpose
      - capability.replace-with-stable-name
  - path: frontend/routes/<module>/<feature>/<service>.svelte.ts
    role: frontend-service
    hash: pending
    supports:
      - capability.replace-with-stable-name
  - path: backend/<domain>/<handler>.go
    role: backend-handler
    hash: pending
    supports:
      - capability.replace-with-stable-name
      - rules
```
