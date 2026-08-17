---
schema: 1
page_id: system.companies
route: /system/companies
title: Companies (Empresas)
status: implemented
visibility: saas
description_en: >-
  Tenant company management, SaaS only. Create and edit companies registered on the platform with
  name, tax ID, legal name, email, phone, representative, and address.
description_es: >-
  Gestión de empresas (tenants), exclusivo SaaS. Crear y editar empresas registradas en la
  plataforma con nombre, RUC, razón social, email, teléfono, representante y dirección.
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
- Each card chart covers today and the previous 29 days. Missing daily records display as zero.

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
- 30-day inference total with a purple marker; and
- a grouped 30-day chart where green and purple bars are side by side rather than summed. When
  only one pool has usage on a day, that bar expands across the available day width for clarity.
  Its vertical labels use 100-credit base intervals and may skip whole intervals when the compact
  card cannot show every label without overlap. Point to a day to see its exact CPU and inference
  values; each bottom date is aligned with that exact sampled day rather than the following date
  interval.

Use **Refresh (Actualizar)** to reload the historical report. The company master list continues to
use its normal cached synchronization, while credit history is refreshed on demand rather than
polling continuously.

Common questions and vocabulary: `¿qué empresa consume más créditos?`, `buscar empresa por RUC`,
`administrador del tenant`, `ranking de compañías`, `CPU credits`, `AI credits`, `uso de 30 días`.

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
usage. The report includes user records with zero usage and does not hide a legacy administrator
merely because its older record lacks the current active-status marker. The user report loads only
when this tab is opened and can be reused in memory until **Actualizar** clears it.

Common questions and vocabulary: `consumo por día`, `qué API gastó créditos`, `detalle por endpoint`,
`CPU por ruta`, `créditos por usuario`, `quién consume más créditos`, `créditos de una compañía`,
`API usage by tenant`, `user credit usage`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- All catalog companies, including zero-usage companies, can appear in the 30-day report.
- Administrator identity on each card is deliberately limited to the display name; cards do not
  expose the repeated login, credentials, profiles, access lists, documents, or other user data.
- CPU and AI card charts and the daily calendar use the same time window but remain separately
  identifiable. Ranking always uses the selected independent pool.
- Company credit totals come from the absolute daily company aggregate, not from the signed-in
  administrator's personal usage.
- User cards come from each user's own absolute daily snapshots; their totals are not estimated
  from the company aggregate or apportioned from API totals.

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
    hash: sha256:fe1d2925d1f24e8de2f270fcc0cd95a28ade1223209c6c5a5fa93a2a76bbe63b
    supports: [page-purpose, capability.configure-company]
  - path: frontend/routes/system/companies/CompanyCards.svelte
    role: user-interface
    hash: sha256:21fafc1b759178232a33233ba18285d06c711e012f72a81ee927e561a6a6545c
    supports: [capability.review-companies, capability.review-credit-detail, rules, troubleshooting]
  - path: frontend/routes/system/companies/CompanyRouteCreditCards.svelte
    role: user-interface
    hash: sha256:e773bd76d6d20f3d773d55e79a6111c3423fe9e5c12a40f1805967347bcb4d02
    supports: [capability.review-credit-detail, troubleshooting]
  - path: frontend/routes/system/companies/CompanyUserCreditCards.svelte
    role: user-interface
    hash: sha256:7c9411c8ddcf77ef4ef31950b3e0771b07f109eb606a98e84a436f5146d65ede
    supports: [capability.review-credit-detail, rules, troubleshooting]
  - path: frontend/routes/system/companies/CompanyCreditCalendar.svelte
    role: user-interface
    hash: sha256:fe5ee633a885063ef6b77e5c299f52e32537d278b2cb6fef5e6c81f45dfdea05
    supports: [capability.review-credit-detail, rules]
  - path: frontend/routes/system/companies/company-credit-calendar.ts
    role: business-logic
    hash: sha256:d4268261cdea63d794de8e6f0fb3f98fd5d8314a0db5282d4238351fe46a7de7
    supports: [concepts, capability.review-credit-detail, rules]
  - path: frontend/routes/system/companies/CompanyCreditCard.svelte
    role: user-interface
    hash: sha256:4959a59c8e451ea641381a3139422ee755aea519c3e2af4a2b48a57f91b7fb7c
    supports: [concepts, capability.review-companies, capability.configure-company, troubleshooting]
  - path: frontend/routes/system/companies/empresas.svelte.ts
    role: frontend-service
    hash: sha256:4d916555b86fb688c1926b602c8bdf63e20f7fbe0789c84fd2832c47d062444d
    supports: [capability.configure-company]
  - path: frontend/routes/system/companies/company-credit-usage.ts
    role: frontend-service
    hash: sha256:c67d405bb05563cb4e5fb5f697bb3e9bcb150edba8af5acb1bf81572fd66cd34
    supports: [capability.review-companies, capability.review-credit-detail, troubleshooting]
  - path: frontend/routes/system/companies/company-credit-usage.model.ts
    role: business-logic
    hash: sha256:b540f473489453fb4a8c51c5dea2f123396f0d37e1a6a6bf827d3fb7d41e4082
    supports: [concepts, capability.review-companies, capability.review-credit-detail, rules]
  - path: frontend/packages/genix-ui/charts/ChartCanvas.svelte
    role: shared-domain
    hash: sha256:53d785220f015d7130aaa7e12cb362c09e212016ae3f647562d6b85d2269a584
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
    hash: sha256:db87d9d0fb5a6d9603204168a92a22049d7928aa90947ae784269c0da46c3381
    supports: [concepts, capability.review-companies, capability.review-credit-detail, rules, troubleshooting]
  - path: backend/config/credit_usage.go
    role: business-logic
    hash: sha256:68d46b30cb8e4bc857c39dc86281308101d2c67e9f6891c00a1fa1c7e05363a8
    supports: [concepts, capability.review-companies, capability.review-credit-detail, rules]
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
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [page-purpose]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [page-purpose]
```
