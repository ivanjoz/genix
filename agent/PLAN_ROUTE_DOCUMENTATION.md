# PLAN — User-facing route documentation for the Genix RAG agent

## Goal

Create and maintain one `DOCUMENTATION.md` beside every implemented, user-facing
frontend route. The documents are the product knowledge source for an assistant that
answers detailed questions—mostly in Spanish—about what Genix can do, why its business
rules exist, what users must consider, what limitations apply, and where to navigate.

The writing is deliberately mixed English–Spanish. Put the English concept and the
Spanish ERP term in the same sentence so dense retrieval receives both meanings and
lexical retrieval receives the exact words users and the UI use. Do not duplicate the
entire document into separate language versions.

This plan defines the contract for the documentation-writing agent. It does not define
Qdrant ingestion or runtime retrieval; see `PLAN_DOCUMENTATION_RAG.md`.

## Scope

Document directories that represent an implemented page a real user can navigate to:

- Routes declared in `frontend/core/modules.ts` and backed by `+page.svelte`.
- Implemented user routes outside the main menu when users can reach them from another
  documented page.
- SaaS-only administration pages, marked with their visibility restriction.
- Dynamic routes when they are part of a user workflow; describe the route pattern and
  the page that opens it.

Exclude:

- `develop-ui`, tests, showrooms, and engineering-only pages.
- Generic UI packages, services, stores, and internal component folders as independent
  user documents. They may appear as evidence in `FILES`.
- Planned menu entries without an implemented route.
- Technical design Markdown such as implementation plans. These may be evidence only
  when they accurately describe implemented behavior.

## Evidence standard

The agent must trace behavior across layers. Reading only `+page.svelte` is insufficient
for business rules.

1. Read the route entry point and every route-local component that participates in a
   user-visible capability.
2. Read the frontend service calls, request fields, response fields, transformations,
   state labels, and error handling used by those components.
3. Locate and read the backend handlers reached by those services.
4. Follow business functions that validate data, calculate values, change statuses,
   update balances or stock, or create related records.
5. Read the database record/table definitions needed to interpret page-specific statuses,
   relationships, calculations, dates, and lifecycle behavior.
6. Read menu definitions to document navigation. Read access catalogs only when a page or
   capability has an exceptional restriction that differs from normal application access.
7. Read relevant existing product documentation and comments, but verify them against
   current code before treating them as implemented truth.

The agent must list every file that materially supports a documented claim. Do not list
generic dependencies such as `Button.svelte`, `Modal.svelte`, icons, or styling files
unless their behavior is itself relevant to the claim.

## Facts, rationale, and uncertainty

Code often proves **what** happens but not **why** the product requires it. The agent must
not invent business rationale.

- State verified behavior directly.
- Explain rationale only when it is supported by authoritative documentation, an
  explicit code comment, or a human answer.
- Ask the human one focused question when a missing rationale or ambiguous business rule
  would materially affect the documentation.
- Record a non-indexed gap when the document can otherwise be completed:

````markdown
<!-- DOCUMENTATION_GAP: Confirm whether cash reconciliation creates an adjustment
movement or only records the observed difference. This comment is excluded from RAG. -->
````

- Never convert a guess into polished prose. A missing explanation is safer than an
  authoritative-sounding invention.

## User-relevance boundary

Document behavior, procedures, and business rules that are specific to the page and help
an end user decide what to do. Do not fill documents with infrastructure abstractions or
guarantees that apply uniformly across Genix.

Exclude generic facts such as:

- tenant/company isolation;
- ordinary access-profile enforcement;
- the internal integer/cents representation of money;
- generic audit fields, IDs, timestamps, storage types, and API authorization; and
- generic statements that the server validates requests.

Include one only when the page has a non-standard rule with a concrete user consequence,
such as a capability restricted to SaaS administrators, a role-specific approval action,
or a rounding rule that changes the value the user sees. Store universal visibility and
access information as non-embedded retrieval metadata instead of explanatory prose.

## Required document structure

Use repository-relative paths and stable identifiers. `DOC-ID` values are API-like
identifiers: keep them stable when wording or headings change because provenance and
Qdrant chunk IDs depend on them.

````markdown
---
# Machine metadata identifies the page and controls indexing.
schema: 1
page_id: finance.cash-banks
route: /finance/cash-banks
title: Cash & Banks (Cajas y Bancos)
status: implemented
visibility: tenant
---

# Cash & Banks (Cajas y Bancos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Explain the page in mixed English–Spanish, its business scope, and what it does not own.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

Define ambiguous ERP entities and distinguish them from related concepts.

<!-- DOC-ID: capability.create-account -->
## Create a cash or bank account (Crear una caja o banco)

### User intention (Intención del usuario)

Explain when and why a user chooses this operation.

### Where to find it (Dónde encontrarlo)

State the menu path, route, view/tab, and exact visible action label.

### Required information and prerequisites (Requisitos previos)

Describe records, page-specific statuses, and fields that must already exist.

### Business rules and rationale (Reglas y razón de negocio)

Explain validation, calculations, lifecycle transitions, and the verified reason.

### Result and side effects (Resultado y efectos)

Explain records created or updated and effects on balances, stock, debt, reports, or
related workflows.

### Limitations (Limitaciones)

Explain intentionally unsupported behavior and common mistaken expectations.

### Common questions and vocabulary (Preguntas y vocabulario)

Include realistic Spanish questions, English equivalents, UI terms, abbreviations, and
regional synonyms that users may search.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

Document only page-specific rules shared by several capabilities without duplicating
them. Omit universal platform behavior such as tenant isolation, generic permissions,
internal money representation, and audit storage.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

List only confirmed causes, the checks a user can perform, and safe recovery actions.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

Connect the page to other routes and explain when the user should go there instead.

### FILES

```yaml
# The hashing tool fills hashes only after the documentation has been reviewed.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/finance/cash-banks/+page.svelte
    role: page
    hash: pending
    supports:
      - page-purpose
      - capability.create-account
  - path: backend/finance/cajas.go
    role: backend-handler
    hash: pending
    supports:
      - capability.create-account
      - rules
```
````

The final document may add, remove, or rename human-facing headings to match the page,
but it must preserve the concepts above and assign one unique `DOC-ID` to every section
that can be embedded independently.

## Mixed-language writing rules

Write conceptual bilingual prose, not line-by-line translation.

Good:

> A cash register (`caja`) is the financial account where Genix records cash
> collections (`cobros en efectivo`), payments (`pagos`), and the available balance
> (`saldo disponible`).

Avoid:

> Cash register. Spanish: caja.

Rules:

- Pair business nouns, actions, states, and actual UI labels in both languages.
- Prefer terminology used by Spanish-speaking small businesses: `caja`, `caja chica`,
  `cuadre`, `arqueo`, `proveedor`, `almacén`, and similar regional vocabulary.
- Explain what a term means specifically inside Genix; translation alone is not enough.
- Keep relationships and negations in natural prose. Words such as `only`, `unless`,
  `not`, `solo`, `excepto`, and `sin` often carry the business rule.
- Use exact visible labels as `English (Español)` when both labels exist in the UI.
- Do not expose Go field names, JSON keys, database types, or endpoint names unless the
  user must know them. Translate implementation evidence into product behavior.

## Capability completeness checklist

For every user action, answer all applicable questions:

- What user goal does it solve?
- Where is it found: module, menu, route, view, layer/modal, and action label?
- What must exist first?
- Which values are required, optional, derived, or fixed?
- What does the backend reject, and what error should the user understand?
- Which calculation or normalization occurs?
- Which state transition occurs?
- Which records, balances, stock quantities, debts, or reports change?
- Can it be edited, canceled, reversed, deleted, or repeated afterward?
- Does a page-specific restriction control it, beyond ordinary application access?
- What deliberately does not happen?
- What related page should the user use for an adjacent task?
- Which Spanish questions, synonyms, and common misspellings might refer to it?

## Provenance contract

Place `### FILES` at the end so the RAG parser can remove it before embedding. Use a YAML
block rather than a Markdown table because the maintenance script must parse it safely.

Allowed roles:

- `page`: route composition, views, and page options.
- `user-interface`: forms, actions, displayed values, and interaction conditions.
- `frontend-service`: API requests, response mapping, and frontend transformations.
- `backend-handler`: server validation and endpoint behavior.
- `business-logic`: calculations, transitions, and side effects.
- `data-model`: persisted statuses, relationships, and value meaning.
- `permissions`: exceptional page-specific access or visibility rules only.
- `shared-domain`: shared business constants or domain components.
- `reference-document`: verified existing documentation.

For each file:

- Use a POSIX path relative to the repository root.
- List the whole-file SHA-256 over exact bytes as `sha256:<hex>`.
- Map `supports` to existing `DOC-ID` values.
- Do not include `DOCUMENTATION.md` itself; hashing it would be self-referential.
- Do not refresh a changed hash until the affected sections have been reviewed.

The maintenance tool requires three separate operations:

1. `discover`: compare route imports, services, backend handlers, and route-directory
   inventory with the manifest; report likely new or removed evidence files.
2. `check`: compare stored hashes with current files without modifying the document;
   report the affected `DOC-ID` values from `supports`.
3. `refresh`: update hashes only after a human or documentation agent has completed the
   review.

## Documentation-agent workflow

### 1. Inventory

- Build the list of eligible routes.
- Classify implemented, dynamic, SaaS-only, public, and internal routes.
- Detect existing route Markdown that must be migrated or preserved as internal design
  documentation.

### 2. Build an evidence graph

- Start at `+page.svelte`.
- Follow route-local components and business-aware shared components.
- Follow services to backend handlers.
- Follow handlers to validation, business functions, and record types. Follow permissions
  only for exceptional page-specific restrictions.
- Log each file examined and why it matters.

### 3. Build a behavior map before prose

For each capability, record the trigger, inputs, prerequisites, validation, calculation,
state transition, side effects, limitations, navigation target, evidence files, and open
questions. This map may be temporary; the final `DOCUMENTATION.md` is the deliverable.

### 4. Resolve important uncertainty

Ask the human concise questions about missing rationale, ambiguous product intent, or
behavior that code paths contradict. Continue documenting independent verified sections
while waiting when possible.

### 5. Write the document

- Use mixed English–Spanish conceptual prose.
- Give every retrievable section a stable `DOC-ID`.
- Keep one capability and its conditions together.
- Add the provenance manifest with `hash: pending`.

### 6. Verify against source

Read the completed document claim by claim. Confirm navigation labels in the frontend and
business rules in the backend. Remove statements supported only by assumption.

### 7. Hash and validate

Run the provenance tool in `refresh` mode, then validate unique IDs, valid paths, supported
roles, complete hashes, eligible route metadata, and absence of unresolved gaps in indexed
sections.

## Incremental review behavior

When a source file changes:

1. Run `check` and identify affected documents and `DOC-ID` values.
2. Inspect the Git diff and the complete current file; the diff gives focus, while the full
   file preserves context.
3. Review every mapped section and any logically dependent section.
4. Update product behavior, rationale, limitations, navigation, and vocabulary as needed.
5. Run `discover` to find new dependencies.
6. Run `refresh` only after the review.
7. Re-index only content chunks whose normalized embedding text changed; update provenance
   payload on unchanged chunks without paying for another embedding.

A formatting-only source change may leave prose unchanged, but it still requires an
explicit review before the new hash is accepted. Correctness is more important than
avoiding a small amount of review work.

## Validation and observability

The scripts must log concise, structured events containing document path, route, source
path, hash status, affected `DOC-ID` values, and final counts. Never log source contents,
credentials, or embedding vectors.

Validation must fail on:

- Duplicate `page_id` or `DOC-ID` values.
- A route that does not match its directory or route pattern.
- Missing, absolute, escaping, or unreadable source paths.
- Unknown file roles or references to nonexistent `DOC-ID` values.
- `pending` hashes when attempting production indexing.
- Hash mismatches when attempting to mark documentation current.
- User-facing claims labeled implemented while their evidence is planned-only.

## Delivery phases

1. Define the final Markdown parser and provenance schema.
2. Implement `discover`, `check`, `refresh`, and `validate` as deterministic project
   scripts.
3. Turn this workflow into a project skill named `document-user-routes`, using this plan
   as a reference and a canonical template as an asset.
4. Document and human-review a representative route from each major domain: Business,
   Sales, Logistics, Finance, Security, Configuration, and Website.
5. Validate retrieval quality with the RAG plan before generating every remaining route.
6. Generate the remaining documents in small reviewable batches.
7. Add the provenance check and documentation validation to CI.

## Acceptance criteria

- Every eligible route has exactly one `DOCUMENTATION.md`.
- Every document explains user-relevant capabilities, prerequisites, page-specific
  business rules, rationale when known, side effects, limitations, navigation,
  troubleshooting, and related workflows.
- Documents omit universal implementation abstractions that do not help the end user act,
  including tenant isolation, generic permissions, and internal cents storage.
- Mixed English–Spanish terminology reads naturally and includes the words real users
  search.
- Every retrievable section has a stable unique `DOC-ID`.
- Every material claim is traceable through `FILES` to examined source evidence.
- Hash checks identify the affected document and sections without modifying stored hashes.
- The agent asks rather than invents when business rationale is absent from the code.
- Internal plans and unimplemented behavior cannot enter the production RAG collection.
