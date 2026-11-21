# Dining Menu ("Carta") Builder Plan

Status: **steps 1–7 implemented (2026-11-16)**; step 8 (the QR page serving the carta) waits for DINING_PLAN step 6. Not yet exercised in a running app. Extends `docs/DINING_PLAN.md` (the restaurant module). The QR page it feeds
(DINING_PLAN §6) is not built yet; this plan changes what that page shows.

A restaurant composes its **carta** (menu) by dragging blocks onto a phone-sized preview: all its
products, one category, a hand-picked list of products, images, titles, subtitles and text. Each
dining layout (salón) picks the carta its customers see when they scan a table's QR code.

## Settled decisions

| # | Decision |
|---|---|
| M1 | **Its own typed blocks**, not the ecommerce builder's HTML/AST engine. A block is data (kind + a few fields) bound to the catalog, so prices, photos and availability always come from the product. From the ecommerce builder we reuse only the drag-and-drop pattern, the image picker and the phone-frame look. No AI agent in v1. |
| M2 | **Many cartas per company; each layout picks one** (`DiningLayout.MenuID`). Availability is decided by the layout's warehouse, as for the rest of the dining menu (DINING_PLAN §3). |
| M3 | **Consumer: the QR page `/mesa`** (DINING_PLAN §6). It renders the layout's carta, and the customer orders from it. The waiter's `MenuPicker` is unchanged. |
| M4 | **Styling: a carta-wide theme + presets per block.** Theme = accent color, background color, font from a short list. Product blocks have display presets (compact list / list with photo / 2-column cards). No free CSS, no per-block colors. |
| M5 | **One renderer, two hosts.** `DiningMenuView.svelte` draws the carta in the builder and on `/mesa`, so the preview is exactly what the customer sees. |
| M6 | **The preview is a 390px phone frame, not an iframe.** The ecommerce builder needs an iframe so Tailwind's `md:`/`lg:` media queries resolve at phone width. The carta is mobile-only by design and has no breakpoints, so a plain 390px container is enough. Drag-and-drop then stays in one document (dragging into an iframe crosses documents and was disabled in the ecommerce mobile view for that reason). |
| M7 | Menu entry **"Carta"** in the Restaurante group, after "Cierre del día". Route `dining/menu-builder`. |
| M8 | A **new access "Carta" (id 67)**, separate from "Diseño de Salones" (§2). |
| M9 | A layout with **`MenuID` = 0 shows the default carta** (one "Todos los productos" block, default theme) (§1.2). |
| M10 | **`RestOnly` ("Todos los demás")** is in v1 (§1.1, §3.2). |
| M11 | The **divider block** is in v1 (§1.1). |

## Overview

```
 left panel (drag sources)        center: phone frame 390px             right panel
 ┌───────────────────┐           ┌────────────────────────┐          ┌──────────────────────┐
 │ Bloques           │  drag ──► │ [Title]  "Entradas"    │ ◄─click─ │ selected block props │
 │  Todos los prod.  │           │ [Category] Ceviches    │          │  or, with none:      │
 │  Categoría        │           │ [Products] 3 picked    │          │  carta name + theme  │
 │  Productos        │           │ [Image]                │          └──────────────────────┘
 │  Imagen / Texto   │           │ [All products, rest]   │
 │  Título/Subtítulo │           └────────────────────────┘
 │ Productos (list)  │  drag a product → new Products block, or onto one → appended
 │ Categorías (list) │  drag a category → new Category block
 └───────────────────┘
          save ──► POST dining-menu ──► dining_menu row (blocks + theme as blobs)
                                            ▲
            DiningLayout.MenuID ────────────┘ ──► GET p-dining-menu?t= ──► /mesa renders it
```

---

## 1. Table `dining_menu`

`backend/dining/types/dining_menu.go`, `DiningMenu` / `DiningMenuTable`. The carta is edited and
saved as one document, so its blocks are a colbin blob, as `DiningLayout.Shapes` is.

| Column | Type | cb | Notes |
|---|---|---|---|
| CompanyID | int32 | 1 | partition |
| ID | int32 | 2 | autoincrement |
| Name | string | 3 | "Carta principal", "Bebidas". Required, at most 80 characters |
| Theme | DiningMenuTheme | 4 | blob |
| Blocks | []DiningMenuBlock | 5 | blob, in display order |
| Status | int8 | ss | 1 active, 0 deleted |
| Updated, UpdatedBy, Version, Metadata | | | standard |

Schema: next free schema ID (127 today; confirm with `static-project-validation`), a delta index on
Status, `FixedValues` Status 0..1.

### 1.1 Blobs

```go
// DiningMenuTheme is the carta-wide look (M4).
type DiningMenuTheme struct {
	AccentColor     string `cb:"1"` // "#b45309": titles, prices, separators
	BackgroundColor string `cb:"2"` // "#fffbeb"; the text color is derived from it by contrast
	Font            int8   `cb:"3"` // MenuFont*: a short fixed list (sans, serif, rounded)
}

// DiningMenuBlock is one block of the carta. One struct for every kind: each kind reads only
// the fields its row in the table below names, the rest stay empty.
type DiningMenuBlock struct {
	Kind        int8    `cb:"1"` // MenuBlock*
	Text        string  `cb:"2"`
	CategoryID  int32   `cb:"3"`
	ProductIDs  []int32 `cb:"4"`
	ImageURL    string  `cb:"5"`
	Variant     int8    `cb:"6"` // product blocks: MenuProductsCompact, MenuProductsWithPhoto, MenuProductsCards
	Align       int8    `cb:"7"` // text blocks: 1 left, 2 center
	RestOnly    bool    `cb:"8"` // all-products block only
}
```

| Kind | Fields | Renders |
|---|---|---|
| `MenuBlockAllProducts` (1) | Variant, RestOnly | Every restaurant product, grouped under a heading per category (§3.2). With `RestOnly` ("Todos los demás") it leaves out the products another block of the carta already shows. |
| `MenuBlockCategory` (2) | CategoryID, Text, Variant | One category's products under a heading: `Text`, or the category's name when empty. |
| `MenuBlockProducts` (3) | ProductIDs, Variant | Hand-picked products, in the order given. |
| `MenuBlockTitle` (4) | Text, Align | A title in the accent color. |
| `MenuBlockSubtitle` (5) | Text, Align | A smaller heading. |
| `MenuBlockText` (6) | Text, Align | A paragraph (line breaks kept, no markup). |
| `MenuBlockImage` (7) | ImageURL, Text | A full-width image, `Text` as its optional caption. |
| `MenuBlockDivider` (8) | — | A thin line in the accent color. |

Product presets:
- **Compact**: name ··· price, one line each.
- **With photo**: a thumbnail, the name, the description (2 lines) and the price.
- **Cards**: a 2-column grid of photo cards with the name and price.

A new carta starts as **one `MenuBlockAllProducts` block** (Variant "with photo"): the user sees all
their products at once and builds from there.

### 1.2 `DiningLayout.MenuID`

`MenuID int32 cb:"19"` on `DiningLayout`: the carta the layout's QR page shows. Picked in the
floor-layout editor's settings panel (a SelectBox over the active cartas). 0 = no carta chosen: the
QR page renders the default carta (`DefaultMenuBlocks()`, the one block of §1.1) with the default
theme.

### 1.3 Validation on save (`POST dining-menu`)

Pure `ValidateDiningMenu(menu, categoryIDs, productIDs) error` in `dining/types`, unit-tested; the
handler reads the company's categories and the referenced products and calls it.

- Name required, at most 80 characters. At most 200 blocks.
- Theme: both colors `#rrggbb`, `Font` a known `MenuFont*`.
- Every block's `Kind` is known, and the fields its kind needs are set and in range:
  - title and subtitle: `Text` 1–120 characters; text: 1–1000; image caption: at most 200;
  - category: `CategoryID` is a category of the company;
  - products: 1–100 `ProductIDs`, no repeats, every one an active product of the company;
  - image: `ImageURL` non-empty and on the company's CDN (as the webpage gallery's URLs are);
  - `Variant` and `Align` within their constants.
- Fields a kind does not use are cleared on save, so the stored blob holds only what renders.
- **Deleting** (Status 0) a carta that a layout's `MenuID` names is refused, naming the layouts.
- `POST dining-layout` checks that `MenuID` is 0 or an active carta of the company.

---

## 2. Endpoints

| Route | Who | What |
|---|---|---|
| `GET dining-menus` | staff | delta of the company's cartas |
| `POST dining-menu` | Carta | create, update or delete (Status 0) one carta (§1.3) |
| `GET p-dining-menu?t=` | public | **changed** (DINING_PLAN §3, §11): besides the products, categories and sellable units, it answers the layout's carta (`Theme`, `Blocks`; the default when `MenuID` = 0). |

`backend/dining/dining_menus.go` holds the two staff handlers; registered in `dining/main.go`.

`backend/access.toml`, group 3, a new access **"Carta" (id 67)**: `frontend_routes =
"dining/menu-builder"`, `backend_apis = "POST.dining-menu"`. Separate from "Diseño de Salones"
because whoever writes the carta is often not whoever draws the floor plan. Picking a layout's carta
stays under "Diseño de Salones" (it is a field of `POST dining-layout`).

---

## 3. Resolving a carta into what renders

Pure, in `frontend/core/dining-menu.ts` (both the admin app and `frontend/webpage` import `#core`),
tested in `dining-menu.test.ts`.

### 3.1 `resolveDiningMenu(menu, catalog, mode)`

Input: the carta, and a catalog of `{ products, categories, sellableByProductID? }` where each
product carries its name, description, price in soles and main image. Output: the list of blocks to
draw, with product blocks already holding their products.

- A product is listed only if it is sold in the restaurant (`isProductSoldIn(...,
  PRODUCT_CHANNEL_RESTAURANT)`) and active.
- `mode = 'public'` (QR page): also only if it has sellable units (DINING_PLAN §3: a product that
  cannot be ordered now is left out, not shown at 0). A product block left with no products is
  dropped, and so is a category heading with nothing under it.
- `mode = 'builder'`: every restaurant product is shown whatever its stock, and an empty product
  block renders a placeholder ("Sin productos — elige algunos en el panel") so it can be selected.
- Prices: `convertUnitPrice` into soles with the day's rate, as `diningMenu` does; a USD product
  with no rate is left out. `diningMenu` (`floor-service.ts`) and this function share the
  per-product price/sellable step instead of duplicating it.

### 3.2 Grouping for "Todos los productos"

- One group per category, in the shared list's order; a product goes to its **first** category.
  Products with no category go last, under "Otros".
- Within a group, products by name.
- `RestOnly`: first collect the product IDs every other product block of the carta lists (category
  blocks included), then leave them out.

---

## 4. Frontend

### 4.1 Menu and page

- `db/modules.toml`, Restaurante: `routes = "dining/floor-service, dining/kitchen-display,
  dining/floor-layout, production/daily-close, dining/menu-builder"`.
- `db/pages.toml`: `route = "dining/menu-builder"`, `name = "Menu|Carta"`, `icon = "icon-[fa--book]"`,
  description "Arma la carta que ven tus clientes al escanear el QR de la mesa: arrastra productos,
  categorías, imágenes y textos sobre la vista del celular."

### 4.2 Files

```text
frontend/routes/dining/menu-builder/
  +page.svelte              Page shell: carta picker (CardsStrip) + "Nueva carta" + Guardar/Eliminar,
                            three columns: MenuBlocksPalette | phone frame | MenuBlockEditor
  MenuBlocksPalette.svelte  Left panel, OptionsStrip "Bloques" / "Productos" / "Categorías":
                            drag sources (a block kind, a product, a category) with a search box
  MenuCanvas.svelte         The phone frame: DiningMenuView in edit mode + drop placeholder,
                            reorder by dragging a block, click to select, delete on the selection
  MenuBlockEditor.svelte    Right panel: the selected block's fields, or the carta's name and theme
  menu-builder.svelte.ts    DiningMenusService (GetHandler, delta), the edited carta ($state),
                            selection, unsaved-changes check, save/delete calls
  menu-builder.ts           Pure edits: newBlock(kind), insertBlock, moveBlock, removeBlock,
                            dropProduct (new Products block vs append to the one dropped on),
                            dropCategory
  menu-builder.test.ts
  DOCUMENTATION.md          (skill: document-user-routes)

frontend/core/dining-menu.ts        §3: constants mirrored from Go, resolveDiningMenu, grouping
frontend/core/dining-menu.test.ts

frontend/webpage/components/dining-menu/
  DiningMenuView.svelte     The renderer (M5). Props: resolved blocks, theme, mode. In 'builder'
                            mode each block is wrapped with the selection outline and drag handle;
                            in 'public' mode each product gets the "agregar" action of the QR cart.
  MenuProductItem.svelte    One product in the three presets

frontend/domain-components/PhoneFrame.svelte
                            The device look (390px, rounded, dark outline, stage gradient), pulled
                            out of webpage-builder/builder/MobilePreviewFrame.svelte, which then
                            wraps its iframe in it.
```

`ImagePickerModal.svelte` (gallery + stock images, with upload) moves from
`routes/webpage-builder/components/` to `domain-components/` since two features use it now; the
image block opens it.

### 4.3 Builder behavior

- **Drag sources** (left):
  - *Bloques*: one tile per kind. Dropped → a new block at the drop point, selected, so its fields
    open on the right.
  - *Productos*: the restaurant products (delta-cached `ProductsService`), with photo and price.
    Dropped between blocks → a new Products block with that product; dropped **on** a Products
    block → appended to it (ignored if already there).
  - *Categorías*: dropped → a new Category block.
  - The drag payload uses its own MIME types (`application/x-genix-menu-block`,
    `-product`, `-category`), as the ecommerce builder does.
- **Canvas**: the drop index comes from the pointer against each block's midpoint (as
  `EcommerceBuilder.handleDragOver`). Blocks reorder by dragging their handle. Click selects;
  Delete/Backspace or the trash button removes the selected block.
- **Right panel**:
  - Title / Subtitle / Text: an Input or textarea + alignment.
  - Category: a SearchSelect over the categories, optional heading override, preset.
  - Products: preset + an **editable VTable** of the picked products (add with a product
    SearchSelect, remove, reorder).
  - All products: preset + "Solo los que no están en otro bloque" (`RestOnly`).
  - Image: the image picker + caption.
  - No block selected: carta name, accent color, background color, font.
- Changes are local until **Guardar**; leaving with unsaved changes asks to confirm (the
  ecommerce builder's baseline/hash check, applied to the one carta document).
- The builder's catalog is the delta-cached products and categories, so it paints from IndexedDB at
  once. No stock is needed in builder mode (§3.1).

### 4.4 Floor layout

`routes/dining/floor-layout/`: the layout settings panel gains a "Carta" SelectBox (active cartas,
plus "Predeterminada" = 0), saved with the layout.

### 4.5 The QR page (`/mesa`, DINING_PLAN §6.3, when built)

"Menu by category (sellable only)" becomes: `DiningMenuView` in `public` mode over
`resolveDiningMenu(carta, catalog from p-dining-menu, 'public')`. The table cart, notes, "Enviar
pedido" and "Tu mesa" stay as DINING_PLAN §6.3 describes. The theme's background and font apply to
the whole page.

---

## 5. Seeds and docs

- `backend/seeds/dining`: one carta per seeded company — a Title, two Category blocks, an Image and
  "Todos los demás" — assigned to the seeded layout.
- `docs/DINING_PLAN.md`: §3 and §6.3 point at this plan; §10/§11 list the route, access and
  endpoints.
- `routes/dining/menu-builder/DOCUMENTATION.md` and the floor-layout one (the new setting).

---

## 6. Order of work

1. `dining_menu` table, `DiningLayout.MenuID`, `ValidateDiningMenu` + tests, `GET dining-menus` /
   `POST dining-menu`, the layout check, access 67, menu and page entries.
2. `core/dining-menu.ts` (resolve + grouping) + tests; share the price/sellable step with
   `diningMenu`.
3. `PhoneFrame` and `ImagePickerModal` moved to `domain-components/`; the ecommerce builder still
   works (check its mobile view).
4. `DiningMenuView` + `MenuProductItem` (the three presets, the theme).
5. The builder page: palette, canvas drag-and-drop, block editor, save/delete; `menu-builder.ts`
   tests.
6. The carta picker in the floor-layout settings.
7. Seeds and docs.
8. With DINING_PLAN step 6 (QR page): `p-dining-menu` answers the carta, `/mesa` renders it.

Verify each step in the running app (skill `agent-browser`): build a carta from scratch by dragging,
save, reload and get the same carta back.

Out of scope for v1: AI agent, per-block colors or spacing, printing the carta (PDF), cartas by
schedule (breakfast/lunch), translations, the waiter's `MenuPicker` following the carta, a public
`/carta` page without a table, undo/redo.

---

## 7. Pending decisions

None. M8–M11 settled them (2026-11-16).
