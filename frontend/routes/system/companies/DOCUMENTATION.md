---
schema: 1
page_id: system.companies
route: /system/companies
title: Companies (Empresas)
status: implemented
visibility: saas
description_en: >-
  Tenant company management, SaaS only. Create and edit companies registered on the platform with
  company details, API credit usage, and separate CPU and AI credit budgets.
description_es: >-
  Gestión de empresas (tenants), exclusivo SaaS. Crear y editar empresas registradas en la
  plataforma, revisar su consumo y administrar presupuestos separados de créditos CPU e IA.
---

# Companies (Empresas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Companies (`Empresas`) is the SaaS administration page for maintaining tenant company records and
comparing their API credit consumption. It presents every company as a card with its canonical
administrator and latest 30 days of CPU and inference usage.

Open **System → Companies (Sistema → Empresas)** at `/system/companies`. This page is available only
to the company that administers the SaaS platform; it is not the tenant-level **My Company (Mi
Empresa)** configuration page.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A company (`empresa` or `tenant`) is one customer organization registered in the platform.
- The canonical administrator (`administrador`) is user ID 1 inside that company. The card shows
  its first and last name when available, without repeating the normally identical `admin` login.
- CPU credits (`créditos CPU`) and inference or AI credits (`créditos IA`) are independent usage
  pools. The page ranks by one selected pool and never adds both into a synthetic total.
- The daily allowance caps the whole company. Each individual user receives half of that company
  allowance, calculated independently for CPU and AI credits.
- Remaining credits (`créditos restantes`) exist in two independent windows: what is left of today's
  daily allowance, and what is left of the month's balance. Either one at zero blocks new charges
  for the pool it belongs to, so a company with plenty of monthly credit can still be stopped until
  tomorrow.
- The current monthly budget is a remaining balance, while storage keeps an absolute ceiling equal
  to accepted month usage plus that balance. This lets accepted usage remain the only consumption
  source of truth.
- Each card chart covers today and the previous 29 days. Missing daily records display as zero.
- A credit day is the business day in Lima time (UTC-5), not a UTC day: daily usage, the daily
  allowance and the month boundary are all counted on it.

<!-- DOC-ID: capability.review-companies -->
## Review and find companies (Revisar y buscar empresas)

The page orders company cards by **CPU Credits (Créditos CPU)** by default. Select **AI Credits
(Créditos IA)** to reorder by inference usage. The selected pool sorts descending, the other pool
breaks usage ties, and company ID breaks remaining ties. Filtering does not renumber cards, so a
matched company keeps its platform-wide position.

Use the filter to search by company ID, name, legal name (`razón social`), RUC, email,
administrator name, or administrator login. Each card shows:

- company name, with its ID in the top-right edit position;
- administrator name, or **Administrator unavailable (Administrador no disponible)**;
- 30-day CPU total with a green marker;
- 30-day inference total with a purple marker;
- a grouped 30-day chart where green and purple bars are side by side rather than summed. When
  only one pool has usage on a day, that bar expands across the available day width for clarity.
  Its vertical labels use 100-credit base intervals and may skip whole intervals when the compact
  card cannot show every label without overlap. The chart carries no dates along its bottom edge:
  point to a day to read its exact date with its CPU and inference values; and
- two meters below the chart, **Daily (Diario)** and **Credits (Créditos)**, each split into a green
  CPU cell and a purple AI cell. Every number is what is **still available**, not what was spent:
  the daily allowance minus today's usage, and the month's ceiling minus its accumulated usage. The
  colored fill is that remainder as a fraction of its own allowance, so an emptying bar is a company
  running out. A company with no budget for the current month shows both meters at zero on an amber
  track with a warning icon, which is the state the limiter reads as "reject every charge".

Use **Refresh (Actualizar)** to reload the historical report. The company master list continues to
use its normal cached synchronization, while credit history is refreshed on demand rather than
polling continuously. The meters travel with that same report and are always sent in full, never
incrementally, because a remaining balance changes with every charge.

Common questions and vocabulary: `¿qué empresa consume más créditos?`, `buscar empresa por RUC`,
`administrador del tenant`, `ranking de compañías`, `CPU credits`, `AI credits`, `uso de 30 días`,
`créditos restantes`, `cuánto le queda hoy`, `sin presupuesto`.

<!-- DOC-ID: capability.configure-company -->
## Create, edit, or deactivate a company (Crear, editar o desactivar una empresa)

Use the green add button to open a blank company form. On desktop, move over a company card to
replace its top-right ID with the pencil; keyboard focus provides the same replacement. Touch
layouts keep the ID and pencil visible together. The pencil opens the existing edit modal without
opening the usage detail.

The form includes name, legal name, RUC, email, phone, representative, city, and address. Before
saving, the visible form requires at least three characters in the company name and eight in the
RUC. The server additionally requires a name of at least five characters and an email of at least
four characters, so email is operationally required even though the current form does not mark it
with a required indicator.

A successful save updates the company card collection and reloads the credit report. Deleting from
the edit modal deactivates/removes the company from the current list; it does not erase its historical
credit records.

Creating a company from this page does not create its administrator account. Until canonical user
ID 1 exists through the account-registration/user workflow, the card displays **Administrador no
disponible**.

Common questions and vocabulary: `¿cómo creo una empresa?`, `editar tenant`, `cambiar razón social`,
`desactivar compañía`, `RUC`, `representante`, `administrador no disponible`.

<!-- DOC-ID: capability.manage-credit-budget -->
## Manage a company credit budget (Administrar el presupuesto de créditos)

Open an existing company's edit modal to find **Credit budget (Presupuesto de créditos)** below the
company fields. CPU and AI are always configured separately. The summary shows the accumulated usage
of the current month and the remaining balance derived from it.

Use the three controls according to the intended change:

- **Set daily (Establecer diario)** replaces the company's daily CPU and AI allowances. Each user
  can consume up to 50% of the corresponding company allowance per business day.
- **Set current (Establecer actual)** activates the current month and replaces its remaining
  CPU and AI balances with the entered values. Already accepted usage is not erased.
- **Add credits (Agregar créditos)** adds the entered CPU and AI amounts to the active month's
  current balances. This action is disabled until the current month has been activated.

Values must be non-negative whole numbers within JavaScript's safe-integer range. Setting one pool
to zero blocks charges that consume that pool. A newly created company has zero daily allowance and
no active month, so both the daily allowance and current balance must be configured before its
credit-consuming APIs can run.

Month activation is manual for now: when a new month starts, the previous monthly ceiling no
longer authorizes requests. Use **Set current** to activate and assign the new month's balance. If
the credit-limit service cannot be reached, mutations and credit-consuming backend requests fail
closed instead of bypassing the budget.

Accumulated usage is the figure the credit service itself decides on: every flush writes its own
daily and monthly counters next to the entitlement they are compared against, so the screen subtracts
the same numbers the limiter would. Recent accepted requests can therefore take roughly one flush
interval to appear; enforcement in the running limiter is immediate. A company the service has never
flushed for falls back to adding up its persisted daily rows, which is how the service itself
recovers those counters when it restarts. If a mutation reply is lost, the screen reloads the durable record before another action,
but an **Add credits** request is deliberately not retried automatically because repeating it could
double the increase.

Common questions and vocabulary: `presupuesto mensual`, `saldo actual`, `aumentar créditos`,
`límite diario`, `50% por usuario`, `activar el mes`, `CPU budget`, `AI budget`.

<!-- DOC-ID: capability.review-credit-detail -->
## Review daily, per-API, and per-user credits (Revisar créditos diarios, por API y por usuario)

Select the body of a company card to open its read-only credit layer. **By day (Por día)** shows the
30-day CPU and AI totals followed by a Monday-first calendar whose columns are **L, M, X, J, V, S,
D**. Each calendar row is labeled with the month containing that row's Monday, including weeks that
cross into a new month. A day inside the 30-day window shows its day number, a green CPU line and
value, and a purple AI line and value. A zero remains visible as `0` without a colored fill, while
padding days outside the report window remain blank. CPU and AI line lengths are scaled independently
against the largest daily value of their own pool. The calendar data is already included in the
summary, so opening a company does not make another historical-summary request.

Select a day to open **APIs for day (APIs del día)**. This view loads only that company/day and lists
compact route cards in two columns. Each card shows only the actual HTTP method, API path, and CPU
credits; its green band represents that route's share of the day's total CPU usage. The cards do not
show route ID, inference credits, or a numeric CPU percentage. A day with no persisted usage returns
an empty API list; it does not copy values from another day. Reopening a day can reuse the page's
in-memory detail until **Actualizar** clears it.

Open **Users (Usuarios)** to compare the selected company's user consumption across the same 30-day
window. User cards appear in two columns and are ordered by CPU credits, then AI credits, from most
to least usage. Each card shows the user's display name, login when available, ID, 30-day CPU and AI
totals, and a compact 40-pixel grouped chart whose green and purple bars represent daily CPU and AI
usage. Only users that actually consumed credits in the window appear: the list is built from usage
records, not from the company's user roster. Names are resolved separately and cached, so a user
whose record is not resolved yet shows as **User #ID**.

Common questions and vocabulary: `consumo por día`, `qué API gastó créditos`, `detalle por endpoint`,
`CPU por ruta`, `créditos por usuario`, `quién consume más créditos`, `créditos de una compañía`,
`API usage by tenant`, `user credit usage`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- All catalog companies, including zero-usage companies, appear in the 30-day report; the company
  catalog drives the card list and usage is joined onto it.
- The user breakdown works the other way: it lists only users with recorded usage in the window.
- Administrator identity on each card is deliberately limited to the display name; cards do not
  expose credentials, profiles, access lists, documents, or other user data. Names are resolved
  through a by-IDs endpoint that returns only login, first name and last name.
- Usage reads are incremental: the client keeps the days it already holds and the server returns
  only the current day plus any day that has since started.
- CPU and AI card charts and the daily calendar use the same time window but remain separately
  identifiable. Ranking always uses the selected independent pool.
- Company credit totals come from the absolute daily company total, which is stored separately from
  per-user totals, not from the signed-in administrator's personal usage.
- User cards come from each user's own absolute daily snapshots; their totals are not estimated
  from the company total or apportioned from API totals.
- The API breakdown is company-wide for the selected day. It is not available per user: only the
  company total records which endpoint spent what.
- Daily and monthly enforcement uses the Lima business day (UTC-5), the same day the usage rows are
  bucketed by, so a daily allowance resets at local midnight. CPU and AI balances are checked
  independently, and one exhausted pool rejects only requests that charge that pool.
- The two card meters are read as "what remains" in each window, and both windows are labeled with
  the day and month they belong to: a window the credit service has not touched yet reads as unused
  rather than as the previous window's spending.
- Budget administration endpoints and the SaaS-only company catalog/summary reads are not
  themselves charged credits, so the SaaS administrator can reach the editor and restore an
  exhausted company's allowance. Detail reports and company writes remain charged.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **Administrador no disponible:** the company has no canonical user ID 1 yet. Creating the company
  record alone does not create that account.
- **The save rejects a short name after the form accepted it:** use at least five characters; server
  validation is stricter than the visible three-character check.
- **The save rejects a company with an empty email:** enter an email with at least four characters;
  the current form does not visually mark it as required.
- **A card shows zero credits:** no company-level daily aggregate exists in the latest 30-day window.
- **The API detail is empty:** select a day whose CPU or AI total is greater than zero, or confirm
  that the company had no recorded usage that day.
- **The Users tab is empty:** confirm that the company has user records. Users with zero credits
  still appear, so an empty list means the company user catalog itself returned no identities.
- **Cards do not update automatically:** this historical report does not poll. Select **Actualizar**.
- **APIs remain blocked after setting the current balance:** also configure a positive daily
  allowance for every credit pool the company needs.
- **No budget is active for the current month:** use **Set current**; adding credits cannot activate
  a month.
- **The displayed balance is briefly higher than the value just set:** recently accepted usage has
  not reached its persisted daily row yet. It converges after the limiter flushes usage.
- **A card's Daily meter is at zero while Credits still shows a balance:** the company spent its
  whole daily allowance. Its charges resume at local midnight, or immediately after raising the
  daily allowance with **Set daily**.
- **Both meters show zero on an amber track:** no budget is active for the current month, so every
  charge is rejected. Use **Set current** in the company's edit modal.
- **The credit service is unavailable:** budget changes return an error and ordinary charged APIs
  remain blocked until `server_utils` is reachable; the backend does not fail open.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Observability (Observabilidad)** monitors recent backend API health, failures, and
  repeated error messages across the platform. It no longer owns the company credit ranking.
- **System → Server Panel** monitors machine CPU, memory, disk, and network behavior.
- Tenant users edit their own operational company configuration in **My Company (Mi Empresa)**;
  use this page for SaaS-wide company administration and comparison.

### FILES

```yaml
# Exact source hashes are filled after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: permissions
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/system/companies/+page.svelte
    role: page
    hash: sha256:a0bdb9bf2c5439353729698c660b85cd6f025aa6be603658636da83b273481ed
    supports: [page-purpose, capability.configure-company, capability.manage-credit-budget]
  - path: frontend/routes/system/companies/CompanyCreditBudget.svelte
    role: user-interface
    hash: sha256:4c92b7f692fb9dd99d759a6948feb4cd70bd60a4dfe4b3e2468b7b699a518fc4
    supports: [concepts, capability.manage-credit-budget, rules, troubleshooting]
  - path: frontend/routes/system/companies/company-credit-budget.ts
    role: frontend-service
    hash: sha256:13ac2e1efb545d509fe1782e78c9d14dc2b85b6b8c1e2b2e31d999785c9e1a6f
    supports: [capability.manage-credit-budget]
  - path: frontend/routes/system/companies/CompanyCards.svelte
    role: user-interface
    hash: sha256:cba7956e874c5cee5cf986aa318bf298643fb302ff6459b9e0ef1a75589320a1
    supports: [capability.review-companies, capability.review-credit-detail, rules, troubleshooting]
  - path: frontend/routes/system/companies/CompanyRouteCreditCards.svelte
    role: user-interface
    hash: sha256:a2c26c04bcac6c07f864b150d983e0eff47474cf221c49ad29338d0af4225045
    supports: [capability.review-credit-detail, troubleshooting]
  - path: frontend/routes/system/companies/CompanyUserCreditCards.svelte
    role: user-interface
    hash: sha256:2efb480f3c68a8237f8764424733eca595e30f400aaf6d65e8c1850d1130326d
    supports: [capability.review-credit-detail, rules, troubleshooting]
  - path: frontend/routes/system/companies/CompanyCreditCalendar.svelte
    role: user-interface
    hash: sha256:2a950920fb1057b9b4b5816a4c778afe240f3e5a6497203fc88cd92e1d58ca52
    supports: [capability.review-credit-detail, rules]
  - path: frontend/routes/system/companies/company-credit-calendar.ts
    role: business-logic
    hash: sha256:a75a865c97771a335a73239db346207b07b41bb7fdcb7e747df036a84e18176d
    supports: [concepts, capability.review-credit-detail, rules]
  - path: frontend/routes/system/companies/CompanyCreditCard.svelte
    role: user-interface
    hash: sha256:f495d1f9a30e9ef38c93af0a96a9145170c10fc9d99b1eff4b0b4f2aa5385270
    supports: [concepts, capability.review-companies, capability.configure-company, troubleshooting]
  - path: frontend/routes/system/companies/CompanyCreditMeters.svelte
    role: user-interface
    hash: sha256:8f2c5c3d3c9ae127cbe6b3fe2e4b18179698e547d671fb5e60d8f468c2243a2e
    supports: [concepts, capability.review-companies, rules, troubleshooting]
  - path: frontend/routes/system/companies/empresas.svelte.ts
    role: frontend-service
    hash: sha256:4d916555b86fb688c1926b602c8bdf63e20f7fbe0789c84fd2832c47d062444d
    supports: [capability.configure-company]
  - path: frontend/routes/system/companies/company-credit-usage.svelte.ts
    role: frontend-service
    hash: sha256:01c376d2cda2ab12782db3f3e8f4c46945e7e092217479dde56490d7c5bb921e
    supports: [capability.review-companies, capability.review-credit-detail, troubleshooting]
  - path: frontend/routes/system/companies/company-credit-usage.model.ts
    role: business-logic
    hash: sha256:3b1b11cd1ca3e37c252c393a6ca704a4f0d1c1fa16fd2ffb580aa586df4a8627
    supports: [concepts, capability.review-companies, capability.review-credit-detail, rules]
  - path: frontend/packages/genix-ui/charts/ChartCanvas.svelte
    role: shared-domain
    hash: sha256:accb366d779fd2499af4f74978a40b7e9f275c791d7ab995074f08e096b58658
    supports: [capability.review-companies]
  - path: frontend/packages/genix-ui/charts/chart-axis-layout.ts
    role: shared-domain
    hash: sha256:e55bbb070e16c8153ba865af1fd7955f89c82aefcb317fec337e1041e139bce6
    supports: [capability.review-companies]
  - path: frontend/packages/genix-ui/charts/chart-bar-layout.ts
    role: shared-domain
    hash: sha256:d551d1d903cd78f3c9c9f3d04eb72210222a06f8a6a036e88208113414ea4803
    supports: [concepts, capability.review-companies]
  - path: backend/config/empresas.go
    role: backend-handler
    hash: sha256:d4c5e8f6e6b247497423e07247d326252913709c3b0cb0a362f02ce3e7488752
    supports: [capability.configure-company, troubleshooting]
  - path: backend/config/company_credit_usage.go
    role: backend-handler
    hash: sha256:a2a76cc3418b8b52f94085009dc6bebcda200c987d4a8d33ec7770b721928a12
    supports: [concepts, capability.review-companies, capability.review-credit-detail, rules, troubleshooting]
  - path: backend/config/credit_usage.go
    role: business-logic
    hash: sha256:4b30c32fafd9c3a84eaa2b21cccd8c9e67f7d8f526c3fa9623cddb1b5a22597d
    supports: [concepts, capability.review-companies, capability.review-credit-detail, capability.manage-credit-budget, rules]
  - path: backend/config/company_credit_budget.go
    role: backend-handler
    hash: sha256:08f7c37e23a822e34f5aab0975187640272b53033a17e8b05ec30dad300ecd43
    supports: [concepts, capability.manage-credit-budget, rules, troubleshooting]
  - path: backend/core/server_utils/budgets.go
    role: business-logic
    hash: sha256:4f300cce6d5a808b480f4e9f291ef4c4463943f22dbc1d789e9a928542efb1a5
    supports: [capability.manage-credit-budget, troubleshooting]
  - path: backend/core/types/company_credit_budget.go
    role: data-model
    hash: sha256:7c7db01450949828b8450a5678dff023a22a0b0b93cde31bd56b2bbc2a68fccc
    supports: [concepts, capability.manage-credit-budget]
  - path: server_utils/src/limiter/storage.rs
    role: business-logic
    hash: sha256:59c3857cea6c81b30153d4eeee37af6ac2634df97f9fff62b4ddbd99cbb79269
    supports: [concepts, capability.manage-credit-budget, rules]
  - path: server_utils/src/limiter/time_frame.rs
    role: business-logic
    hash: sha256:010fbb19a95da09186006dc5d2042e8608236f79925f1d897b86a92b7a75be60
    supports: [concepts, rules]
  - path: server_utils/src/limiter/quota.rs
    role: business-logic
    hash: sha256:8f45bcd7011451cf77cbf0a72b4d15ca838f69e07db844cefc22b75c1462911e
    supports: [concepts, capability.manage-credit-budget, rules, troubleshooting]
  - path: backend/config/types/empresas.go
    role: data-model
    hash: sha256:c751f3a161094d85cb735e64eadeb9f4f2f243b8391449675c1bb1dfeaf2de25
    supports: [concepts, capability.configure-company, rules]
  - path: backend/core/types/users.go
    role: data-model
    hash: sha256:b6d7a7c08f228d3315d5c1d831dcec406094d4ae9c6218a3e8b51c19bca2ca9c
    supports: [concepts, capability.review-companies, rules, troubleshooting]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:f11c349be832baa870463eb18658a8f637436c84f39e80bb5bcb47d9118c7f2a
    supports: [page-purpose, capability.manage-credit-budget]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:5c82fc28d43e45327b373a75c6901e5a141cc993e47a998dc57f6ac0b89a0d75
    supports: [page-purpose, capability.manage-credit-budget, rules]
```
