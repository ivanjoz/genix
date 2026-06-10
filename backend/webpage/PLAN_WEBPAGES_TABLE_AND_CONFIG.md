# Plan — `webpages` table + Pages/Config builder admin

## Goal
Add a `webpages` admin screen at `frontend/routes/webpage-builder/pages/` with two top-tabs:
**"Pages|Páginas"** (card list/CRUD of pages) and **"Config"** (SEO metatags + domain).
The page `ID` becomes the `PageID` used by `EcommercePageContent`.

## Decisions (confirmed)
- IDs **1–9 reserved**. **ID 10 = "Inicio" (root `/`)**, always **injected** by the GET handler (not stored, not deletable).
- User-created pages **autoincrement from 11+**.
- `Webpage.ID` is **`int16`** (matches `EcommercePageContent.PageID`).
- `Status`: **1 = active, 2 = published, 0 = removed**.
- APIs: `GET.webpage-pages`, `POST.webpage-page`, `POST.website-domain`.
- SEO metatags + domain saved in the **`parameters`** table, **Group = 10**.

---

## Backend

### 1. New table — `backend/webpage/types/webpages.go`
`Webpage` / `WebpageTable` paired structs:
| Field | Type | Notes |
|---|---|---|
| CompanyID | int32 | partition |
| ID | int16 | autoincrement from 11 |
| Name | string | |
| Route | string | e.g. `/nosotros` |
| Status | int8 `ss` | 0 removed / 1 active / 2 published |
| Updated | int32 `upd` | `core.SUnixTime()` |
| UpdatedBy | int32 | user id = "UserUpdated" on the card |

- `Partition: e.CompanyID`, `Keys: []db.Coln{e.ID.Autoincrement(0)}`.
- Delta-cache view index: `{TypeView, Keys:[Status.DecimalSize(1), Updated.DecimalSize(10)], KeepPart:true}`.
- **Reserved-ID concern:** autoincrement starts at 1. To guarantee user pages land at ≥11
  I'll handle this in the POST handler: reject/clamp so the first stored row uses ID 11
  (seed the sequence). Simplest robust approach: in `PostWebpage`, if a *new* page would
  get an ID ≤ 10, I bump it. **→ open question A below.**

### 2. Handlers — register in `backend/webpage/main.go`
- `GET.webpage-pages` (`webpage_pages.go`): delta-cache list (`upd` watermark, active+published
  on first sync, all statuses on delta). **Always prepend the injected root row**
  `{ID:10, Name:"Inicio", Route:"/", Status:2}` so it's present in every list.
- `POST.webpage-page`: validate `Name` (≥3) and `Route` (starts with `/`, unique per company,
  not `/`), reject edits to ID 10, set `Updated/UpdatedBy`, `db.Insert`.
- `POST.website-domain` (`webpage_config.go`): validate domain, upsert into `parameters`
  `Group=10, Key="domain", Value=<domain>`. (SEO metatags reuse the existing
  `POST.parametros` with `Group=10` — no new endpoint needed; domain gets its own because
  it will later drive Cloudflare/DNS config per `WEBPAGE_DEPLOY_AND_CONFIG_PLAN.md`.)

### 3. Register table controller
Add `makeDBController[webpageTypes.Webpage]()` — regenerate via
`scripts/controllers/controllers_generator.go` (don't hand-edit the generated file).

---

## Frontend

### 4. `frontend/routes/webpage-builder/pages/+page.svelte`
- `Page` shell + `OptionsStrip` `[[1,"Pages|Páginas"],[2,"Config"]]`, `FilterInput`, "Nuevo" button.
- **Pages view (cards)** — clone the Categorías card pattern (`CategoriasMarcas.svelte`):
  each card shows Name, Route, formatted Updated, UpdatedBy (resolved to user name), Status badge.
  Click → Modal to edit; "Nuevo" → Modal to create. Fields: Name, Route, Status (select).
  Root card (ID 10) is read-only.
- **Config view** — `WebpageConfig.svelte`: SEO metatag inputs (title, description, keywords,
  ogImage…) + domain input. Save → `POST.parametros` (Group 10) for metatags,
  `POST.website-domain` for domain.

### 5. Frontend service — `frontend/services/webpage/pages.svelte.ts`
`WebpagesService extends GetHandler<IWebpage>` (`route="webpage-pages"`,
`useCache`, `inferRemoveFromStatus`, `keyID:"ID"`). `postAndSync` for save.

### 6. Menu — `frontend/core/modules.ts`
Add `{ name: "Pages|Páginas", route: "/webpage-builder/pages" }` under the Website module.

---

## Confirmed answers
- **A.** Enforce "user pages start at ID ≥ 11" in `PostWebpage` (bump any new ID landing in the
  reserved range). **Yes.**
- **B.** SEO keys: `title`, `description`, `keywords`, `ogTitle`, `ogDescription`, `ogImage`,
  `favicon`. **Yes.**
- **C.** SEO metatags are **global** (one Group-10 set for the whole site). **Yes.**

## Injected (always-present, non-stored) pages
Mirror the existing Website menu. User-created pages autoincrement from **15**.
```
10  /         Home|Inicio
11  /about    About Us|Nosotros
12  /store     Store|Tienda
13  /product   Product|Producto
14  /cart      Shopping Cart|Carrito Compra
```
> Single commented constant in the GET handler — trim/extend if the set should differ.

## As-built notes
- Table `webpages` uses **two narrow views** (`{Status}` + `{Updated}`) — matching the
  Product delta pattern — since the initial fetch filters by Status only and the delta by
  Updated only.
- The existing `GET/POST.parametros` handlers don't scope by company / set `CompanyID`
  server-side, so to keep config secure & self-contained the webpage module added:
  `GET.website-config` (loads domain + SEO, company-scoped), `POST.website-seo`
  (saves the known SEO keys to params group 10, tenant-set server-side), plus the
  requested `POST.website-domain`. SEO + domain all live in **parameters group 10**.
- **Open follow-up:** the new POST routes (`webpage-page`, `website-seo`, `website-domain`)
  are not in `access_list.yml` — same as the existing `POST.ecommerce-page-content`. They
  work for the admin user; add access entries before non-admin users need them.

## Iteration 2 — card thumbnail + per-page builder (as-built)
- **`webpages.Image int32`** column added (numeric image-ID reference; upload/resolution
  is a later task — the column only stores the id for now).
- **Cards**: removed the `#ID` badge; added a thumbnail placeholder area at the top; added
  a hover lift effect; added a hover-revealed **pencil button** (`icon-pencil`, top-right
  corner). Card body click → metadata modal (unchanged); pencil → `goto('/webpage-builder/<ID>')`.
  The pencil works on system pages too (their *content* is editable; only name/route metadata is read-only).
- **Per-page builder route**: new `routes/webpage-builder/[pageID]/+page.svelte`. Both it and
  the bare `/webpage-builder` route render a shared `BuilderPage.svelte` that loads/saves a
  specific page. `BuilderPage` resets the singleton `editorStore` and re-loads on `pageID`
  change (a load token discards stale fetches), since SvelteKit reuses the component across
  `[pageID]` navigations.
- **PageID threading**: `getPageContent(pageID)` / `savePageContent(sections, pageID)` now take
  a page id; a module-level `currentPageID` (set via `setCurrentPageID`) lets the deeply-nested
  editor Save button (`SectionEditorLayer`) target the open page without prop-drilling. The
  backend `ecommerce-page-content` handlers already accepted `?page-id=`.
- **`defaultPageID` 11 → 10**: the bare `/webpage-builder` route (Home/Inicio menu) now maps to
  Inicio = page **10**, matching the webpages scheme so the Home menu and the Inicio card edit
  the same content. ⚠️ Any content previously saved at page 11 in a dev DB is now orphaned.

## Top tabs
NOT `OptionsStrip` — use the `Page` component's own `options` prop
(`[{id:1,name:"Pages|Páginas"},{id:2,name:"Config"}]`) and switch on `Core.pageOptionSelected`.
