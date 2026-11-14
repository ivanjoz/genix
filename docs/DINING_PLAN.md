# Dining (Restaurant) Plan

Status: **in progress**: steps 1–3 done, step 4 partly done. See `docs/DINING_STATUS.md` (handoff
and the decisions taken). Built after `docs/PRODUCTION_PLAN.md`,
which brings:
- the pending stock exits (`SaleOrder.PendingStockMovement`) of every sale of a made product
  (PRODUCTION P8, §3);
- the "Cierre del día" that settles them (PRODUCTION §5).

The code review of 2026-11-13 is folded in; its decisions are D15 and D16.

Table service for restaurants: the company draws each physical space it serves in (a **layout**),
places tables and prep stations on it, and takes orders per table, from a waiter's device or from the
customer scanning the table's QR code. The kitchen marks dishes ready with one tap, and the floor map
shows how long each table has waited. The table's bill becomes a regular `SaleOrder`, charged at the
table.

## Settled decisions

| # | Decision |
|---|---|
| D1 | Two ordering modes: **waiter** (in the app) and **customer** (a public page of `frontend/webpage`, reached through a **QR per table**, so the table is known from the QR). |
| D2 | **Prep stations** (kitchen, bar, grill): every product is routed to one. Stations have **pickup points** where waiters collect. |
| D3 | **Virtual stock is the product's "Sobreventa permitida"** (`WebExtraStock`). A made product sells with no stock up to that margin. The "Cierre del día" makes the margin whole again. |
| D4 | The production cost comes from the product's production configuration (PRODUCTION §2). |
| D5 | The statuses stay 0–4. A served table is **Entregado (3/4)**, with **`PendingStockMovement = 1`** while its made lines' stock exits wait for the close (PRODUCTION §3). |
| D6 | Whether a QR round needs a waiter's confirmation is a **configuration of each layout**. A layout is a physical space; a company can have many. |
| D7 | Live updates through the **fareward SSE bridge**. |
| D8 | **One bill per table**. No split, move or merge. |
| D9 | Frontend: a **"Restaurante"** menu group in the Comercial module, after Facturación. |
| D10 | Backend: a new module **`dining`** (`backend/dining/`, `backend/dining/types/`). It reaches sales only through L2 packages (MODULE_BOUNDARIES): `sales/types` and `sales/saleissue/types`, the shared create-sale path (§4). |
| D11 | The customer **pays at the table**: the waiter closes and charges the bill. No payment on the QR page. |
| D12 | Closing and charging is a sub-access **"Cobrar mesa"** of "Atención de Mesas". |
| D13 | New frontend dependency **`qrcode`** (npm). |
| D14 | The regularization is the manual **"Cierre del día"** (PRODUCTION §5). |
| D15 | **The SSE bridge runs on every install, VPS included**, and dining always publishes through it: one code path (§8). |
| D16 | The QR URL is **`/mesa#t=<token>`** on the store's domain: one published page (§6). |

## Overview

```
 DiningLayout (one per physical space, warehouse W)     waiter app ─┐
   shapes: tables, walls, dividers, labels,                          ├─► TableOrder (open, per table)
           station and pickup-point marks                QR page ────┘     reserves units on W
   stations + product routing, QR settings                                  lines: 1 to confirm → 2 ordered
                                                                                  → 3 ready → 4 served
   kitchen display ◄── lines of its station ◄─────┐                          │
   floor map       ◄── colors from line times ◄───┴─ SSE nudge (topic) ◄─────┤ every write
                                                                             │ "Cobrar mesa"
                                                                             ▼
                              SaleOrder 3/4: bought lines leave W now; made lines pending
                                             → PendingStockMovement = 1
                                                                             │ Cierre del día (PRODUCTION §5)
                                                                             ▼
                              production entry + exits → PendingStockMovement = 0, margin restored
```

---

## 1. Table `dining_layout`

`backend/dining/types/dining_layout.go`, `DiningLayout` / `DiningLayoutTable`. One row per physical
space. The plan is edited and saved as one document, so the shapes are a colbin blob, not rows. There
is a precedent: `Warehouse.Layout []WarehouseLayout` (`business/types/sites_warehouses.go:90`).

| Column | Type | cb | Notes |
|---|---|---|---|
| CompanyID | int32 | 1 | partition |
| ID | int32 | 2 | autoincrement |
| Name | string | 3 | "Salón principal", "Terraza" |
| WarehouseID | int32 | 4 | Where this space's goods come from: menu stock, reservations, sale exits. It need not be open to online sales. |
| GridWidth, GridHeight | int16 | 5, 6 | Canvas in **cells**. The cell's pixel size (8/12/16 px) is only editor zoom. |
| Shapes | []LayoutShape | 7 | blob |
| Stations | []DiningStation | 8 | blob |
| QRRequiresConfirmation | bool | 9 | D6 |
| QRVersion | int16 | 10 | signed into every QR of the layout (§7) |
| WaitWarnMinutes, WaitLateMinutes | int16 | 11, 12 | floor-map thresholds (§9). Defaults 10 / 20. |
| NextShapeID | int16 | 13 | shape IDs are never reused (§1.2) |
| Status | int8 | ss | 1 active, 0 deleted |
| Updated, Version, Metadata, UpdatedBy | | | standard |

Schema: a delta index on Status. `FixedValues` for Status, 0..1.

### 1.1 Blobs

```go
// LayoutShape is one element of the plan.
type LayoutShape struct {
	ShapeID   int16   `cb:"1"`
	Kind      int8    `cb:"2"` // ShapeTable, ShapeRect, ShapeLine, ShapeLabel, ShapeStation, ShapePickup
	Points    []int16 `cb:"3"` // grid points, x then y (below)
	Label     string  `cb:"4"` // table name "M4", station name, free text
	Seats     int8    `cb:"5"` // tables
	Outline   int8    `cb:"6"` // tables: 1 square/rect, 2 round
	StationID int16   `cb:"7"` // station and pickup marks
}

// DiningStation is a prep station. A product goes to the first station that names it in
// ProductIDs, else the first that names one of its categories, else the default station.
type DiningStation struct {
	ID          int16   `cb:"1"`
	Name        string  `cb:"2"`
	ProductIDs  []int32 `cb:"3"`
	CategoryIDs []int32 `cb:"4"`
	IsDefault   bool    `cb:"5"`
}
```

`Points` is the one geometry encoding, with no separate size or rotation to disagree with it:
- a line (wall, divider) is its two ends, `{x1,y1,x2,y2}`, which must differ;
- every other shape is the four corners of the cells it covers, `{x1,y1,…,x4,y4}`: a grid-aligned
  rectangle of at least one cell, listed **clockwise from the corner that is its content's
  top-left**. The corner order is the orientation: a 90° turn starts the list one corner later
  (`[2,2,5,2,5,5,2,5]` upright, `[5,2,5,5,2,5,2,2]` turned once).

A pure `StationForProduct(stations, product) int16` lives in `dining/types`, unit-tested. The frontend
mirrors it only for display.

### 1.2 Validation on save (`POST dining-layout`)

- Shapes inside the grid, their `Points` as §1.1 describes. Shape IDs unique, and none above `NextShapeID`. A new shape takes
  `NextShapeID` and increments it, so a deleted table's ID is never handed to a new table that old
  orders or printed QRs would then point to.
- Table labels unique within the layout.
- Exactly one default station. Station IDs unique. Every station/pickup shape names an existing
  station.
- `WarehouseID` is an active warehouse of the company.
- **No table shape with an open `TableOrder` may be deleted** (checked against the open orders).
- `WaitWarnMinutes < WaitLateMinutes`.

---

## 2. Table `dining_table_order`

`backend/dining/types/table_order.go`. Modeled on `OnlineSaleOrder`: **not a sale**. It spends no
comprobante number and enters no summary until it is charged.

| Column | Type | cb | Notes |
|---|---|---|---|
| CompanyID | int32 | 1 | partition |
| ID | int32 | 2 | `Autoincrement(0)`, as `OnlineSaleOrder` |
| LayoutID | int32 | 3 | |
| TableShapeID | int16 | 4 | |
| WarehouseID | int32 | 5 | the layout's, copied when opened; its reservations live there |
| Lines | []TableOrderLine | 6 | blob |
| Guests | int8 | 7 | optional |
| WaiterID | int32 | 8 | the user who opened or confirmed it |
| SaleOrderID | int64 | 9 | set when charged |
| ExchangeRate | int32 | 10 | the rate (×1000) the last round converted a USD-priced product with, 0 when none did; the sale inherits it, as from a web order |
| Status | int8 | ss | 1 open, 2 charged, 0 annulled |
| Created, Updated, Version, Metadata, UpdatedBy | | | standard |
| AnnulReason | string | 17 | why an open order was annulled (§4), at most 200 characters |

Schema:
- a delta index on **(LayoutID, Status)**, so a floor map syncs the open orders of its layout only;
- `FixedValues` Status 0..2.

At most **one open order per table**. `POST dining-table-order-round` opens it if there is none, and
takes a per-table action lock (`core.AcquireLock`, a new `ActionDiningTable`) so two devices cannot
open two.

A table order is always in **soles**. Each round prices its lines with `sales/types.PriceSaleOrderLine`,
the till's pricing, extracted from the `sales` body together with `NormalizeCurrency` and
`ResolveSaleExchangeRate` (the company flag values are passed in, since `sales/types` cannot import
`cloud`). A USD-priced product is converted at the day's rate when the round is added.

```go
type TableOrderLine struct {
	LineID         int16  `cb:"1"`
	Round          int8   `cb:"2"`
	ProductID      int32  `cb:"3"`
	PresentationID int16  `cb:"4"`
	Quantity       int32  `cb:"5"` // packed, as SaleOrderLine
	SubDivisor     int16  `cb:"6"`
	Price          int32  `cb:"7"` // fixed when ordered, from the catalog, in soles (a USD price at the day's rate)
	SubPrice       int32  `cb:"8"`
	StationID      int16  `cb:"9"`
	Note           string `cb:"10"` // "sin cebolla", max 120 chars
	FromQR         bool   `cb:"11"`
	// 0 cancelled, 1 to confirm (QR, when the layout requires it), 2 ordered, 3 ready, 4 served
	Status      int8  `cb:"12"`
	OrderedAt   int32 `cb:"13"` // SUnixTime it reached the kitchen
	ReadyAt     int32 `cb:"14"`
	ServedAt    int32 `cb:"15"`
	UpdatedBy   int32 `cb:"16"`
}
```

### 2.1 Line transitions

| Transition | Who | Rule |
|---|---|---|
| new → 1 | QR round, layout requires confirmation | |
| new → 2 | waiter round, or QR when not required | `OrderedAt` = now |
| 1 → 2 | waiter (confirm) | `OrderedAt` = now |
| 2 → 3 | kitchen (line or whole ticket = round × station) | `ReadyAt` = now |
| 3 → 4 | waiter | `ServedAt` = now |
| 1/2 → 0 | waiter | not after ready |
| 3 → 2 | kitchen | undo a mis-tap, within 2 min |

Wait times are **derived, not stored** (pure functions in `dining/types` and
`routes/dining/dining.ts`):
- a table's wait is now − the oldest `OrderedAt` among status-2 lines;
- a pickup's wait is now − the oldest `ReadyAt` among status-3 lines.

### 2.2 Reservation

Every line in status 1–4 holds its units on `WarehouseID`'s `ProductStock.PendingDeliveryStore`:
- +1 when a round is added, −1 when a line is cancelled;
- applied with `PendingDeliveryChanges` + `ApplyPendingDeliveryIn` in the same transaction as the
  order write.

The till, the web store and the restaurant then see one figure for that warehouse.

`exec/pending_delivery_recompute.go` also adds the units of open `TableOrder` lines. Otherwise a
recompute would wipe the restaurant's reservations.

---

## 3. The menu and the sellable rule

- The products: `Status = 1` and `WebHidden = false`, the same set as the web store.
- A product is sellable by the **web store's rule** applied to **the layout's warehouse alone**:
  - `logistics.ReadWebStockRows(company, []int32{W}, presentations)`;
  - then `WebSellableStock` / `WebOrderFits`, which already take the warehouse list as a parameter.

```
net        = Quantity + DetailQuantity − PendingDeliveryStore − PendingDeliveryOnline
oversold   = max(−net, 0)        (summed over presentations)
marginLeft = max(WebExtraStock − oversold, 0)
sellable   = max(net, 0) + marginLeft
```

- **Bought** (a Coke) sells its real stock, plus a margin if one is set.
- **Made** (a dish) has no stock. Each reserved portion pushes `net` below zero and uses up the
  margin. The close (production entry + exits + release) brings `net` back to 0 and the margin back
  to whole.

Every round is validated with the same function on the backend, so the menu and the order cannot
disagree.

In the product form, "Sobreventa permitida" moves out of the "Tienda Web" box; its label stays.

**Endpoints:**
- `GET dining-menu?layout-id` (staff);
- `GET p-dining-menu?t=<token>` (QR page). It answers the products, categories and the sellable units
  per presentation. It is small (one warehouse), so it has no file snapshot. It can get one later,
  as the e-commerce catalog did.
- Only products with at least one sellable presentation are listed: a product that cannot be ordered
  now is left out, not shown at 0.

---

## 4. Charging the table → `SaleOrder`

`POST dining-table-order-charge`, sub-access "Cobrar mesa" (D12). Body: `OrderID`, `CashBankID`
(0 = unpaid: the client owes the bill), `IssueSeriesID`, and the client as at the till (`ClientID`
or `ClientInfo`). A payment collects the whole bill.

**One create-sale path.** Every sale is born through `sales/saleissue/types`, the package the
till's create (`PostSaleOrder`), the web order's settlement and this charge share:
- `PrepareNewSale(req, NewSaleRequest{Sale, Products, IssueSeries, ReservedWarehouseID,
  IsOnlineReservation})` decides the debt and the collection, resolves the client, the
  reservations and the early stock check, stamps payment and delivery, and mints the sale id with
  its comprobante. Everything that can refuse the sale runs before the number is spent.
- `WriteNewSaleIn(transaction, req, &prepared)` writes the sale, its document, the cash and stock
  movements, the reservation changes and the client week row in the caller's transaction, so the
  caller adds its own writes (here, the order) to the same commit.
- `ScheduleNewSaleDocument` sends the comprobante after the commit.
- `ResolveSaleIssueSeries` (series and buyer identity) and `ResolveSaleClientID` live beside it.

It is its own L2 package, not `sales/types`, because it needs `invoicing/types`, `config/types` and
`crm/types`, which all import `sales/types`. The building blocks it calls (`ReadSaleProducts`,
`ValidateSaleStock`, `ValidateSaleCurrency`, the payment/delivery stamps and movement builders,
`ApplyClientWeekSaleChangesIn`) moved from the `sales` body to `sales/types`.

Steps (`backend/dining/table_order_charge.go`):
1. Read the order, take the table lock (`ActionDiningTable`, as a round does), read it again.
   Refuse if any line is in status 1, 2 or 3 (`ChargeSaleLines`): every live line must be served or
   cancelled, and at least one served.
2. Build one `SaleOrder` from the served lines at their stored prices, in soles, with the order's
   `ExchangeRate` and `WarehouseID`. The till reprices from the catalog; here the order's prices are
   kept, as the web order's settlement does. `ActionsIncluded` = delivered, plus paid with a cash bank.
3. `ValidateSaleCurrency` (the cash bank must be in soles), `ResolveSaleIssueSeries`,
   `ReadSaleProducts`, then `PrepareNewSale` with `ReservedWarehouseID` = the order's warehouse.
4. **Bought lines** leave W now (type 8) and release their reservation. **Made lines** get
   `StockExitPending` and keep theirs. The reservation passes from the table order to the sale with
   no counter change.
5. Status: **3/4**, paid or not by the payment given. `PendingStockMovement = 1` if any line is
   pending.
6. In one transaction: lock the order row and check its lines are still the ones billed,
   `WriteNewSaleIn`, then `TableOrder.Status = 2` and `SaleOrderID` set.

An open order is annulled (`POST dining-table-order-annul`, same sub-access, with a reason of at
most 200 characters) by releasing every live line's reservation and storing `AnnulReason`. Nothing
was sold, so nothing else moves.

---

## 5. Stations, kitchen display, pickup

- `StationID` is resolved once, when the line is ordered. Later routing changes affect only new lines.
- **Kitchen display:**
  - Pick a layout + station.
  - Tickets (round × station) with their status-2 lines, oldest first, each with a running timer.
  - Large targets: one marks a line ready, one marks the ticket ready.
  - Timer colors use the layout's thresholds.
- **Pickup:** the floor-service page has a "Listos para recoger" strip. It lists the status-3 lines of
  every table, grouped by the pickup point of their station ("M4 · 2 lomo saltado · listo hace 3
  min"). One tap marks them served.
- A shared kitchen tablet uses an ordinary user session. Today the browser keeps a session alive
  while it is open and online (`reload-login` every 40 min). A tablet asleep or offline for more
  than ~2h40m is logged out on the client. There is no kiosk concept, and v1 adds none.

---

## 6. The QR page (customer)

### 6.1 The path

- The storefront pre-renders each published path into `websites-html/<host>/<path>/index.html`, and
  CloudFront has **no SPA fallback** (`cloud/storefront.yml:97-105`). So `/mesa/<token>` with a
  token per table would 404.
- Instead: **one published path `/mesa`**, with the token in the **hash**: `/mesa#t=<token>`.
  - The hash never reaches the CDN, so every table shares one cached HTML.
  - `/mesa` joins the paths `webpage_render.go:45-73` publishes for every company (with `/` and
    `/about`).
- The page reads the company from the rendered `<meta name="company-id">`, as the rest of the
  storefront does, and the backend checks it against the token's.

### 6.2 The token

- Bytes: `uvarint(companyID) ‖ uvarint(layoutID) ‖ uvarint(shapeID) ‖ uvarint(QRVersion) ‖ tag`.
- `tag` = BLAKE2s-128 keyed by SHA-256 of `SECRET_PHRASE`, domain `diningTable:v1`, truncated to
  8 bytes. This is the construction of `core/web-customer-token.go:34-48`, with its own domain.
- base64url. `core/dining-table-token.go` has `MakeDiningTableToken` / `ParseDiningTableToken`, with
  tests.
- A token is valid while its layout is active, its shape is still a table and its `QRVersion` is
  current. "Regenerar QRs" bumps the version.

### 6.3 The page

`frontend/webpage/routes/components/mesa/`:
- Header with the table label.
- Menu by category (sellable only).
- A table cart: its own small store keyed `genixTableCart-<cid>-<token>`. The web cart is capped by
  the online warehouses, and `CartMenu.svelte` is tied to web checkout (address, Culqi, shipping), so
  neither is reused. `ProductCard`-style tiles are reused.
- Notes per line. "Enviar pedido".
- **"Tu mesa"**: the table's lines and their states (to confirm / en cocina / listo / servido).
  It polls `GET p-dining-table-order?t=` every 15 s while the page is visible. The page has no
  session, so it cannot use the bridge (§8).

### 6.4 Abuse (public routes have no fareward limit)

`p-` routes skip fareward entirely (`fareward_routes.go:54-59`), so the handlers protect themselves,
as `web_customers.go:194-205` does:
- per IP, `core.AcquireLock(ActionDiningQRByIP, req.ClientIPKey())`: at most 1 round per 20 s;
- per table, at most 6 QR rounds per open order while unconfirmed;
- per round, at most 20 lines and 10 units per line;
- `QRRequiresConfirmation` (D6) is the real guard against a photographed QR.

---

## 7. The QR section

`routes/dining/table-qr/`:
- Pick a layout. A card per table shows its label and its QR (`qrcode` npm, rendered to SVG) for
  `https://<store host>/mesa#t=<token>`.
- "Imprimir" lays the cards on A4 (CSS print).
- "Regenerar QRs" asks for confirmation, then bumps `QRVersion`. Every printed QR of the layout stops
  working.
- The tokens are made by the backend (`GET dining-table-qr-tokens?layout-id`), because the browser
  does not hold the secret.

---

## 8. Live updates (SSE bridge)

What exists (review):
- the bridge is **one channel per browser tab**, with no fan-out and no topics;
- **the main backend has no bridge client** (the agent's is unexported, in the `genix-agent` module);
- the bridge is enabled only under Lambda; on a VPS the agent streams by itself.

Changes:

1. **fareward (submodule, committed and pushed there):**
   - A tab subscribes to topics when it opens its stream: `GET /sse?ch=<channel>&topics=dining.<cid>.<layoutID>,…`.
   - The bridge refuses a topic whose company differs from the session token's.
   - New `POST /publish-topic {Topic, Message}` (backend-authenticated, `X-Bridge-Auth`) fans out to
     every subscribed tab and answers `{Delivered: n}`.
   - Topics live in the same in-memory registry as the channels and die with them.
2. **Go client in `fareward/go/`** (`bridge.go`): `PublishTopic(topic, message)` and `Publish(channel, message)`.
   - The backend already imports `fareward/go` for the limiter.
   - The agent's `bridge.go` moves onto it, so there is one client.
3. **Deployment: the bridge runs on every install (VPS too, D15).** Today `BridgeEnabled()`
   (`agent/bridge.go:56`) requires `IS_SERVERLESS`, and the VPS deploy does not configure the bridge.
   - The deployer and `config.example.toml` gain the bridge for VPS installs.
   - Dining publishes whenever `SSE_BRIDGE_URL` is set.
   - The agent keeps its own choice of transport; it is not in scope.
4. **The message is a nudge only**: `{Type: "dining", LayoutID, Updated}`.
   - The tab then pulls `GET dining-table-orders?layout-id` through the delta cache.
   - A frame missed while disconnected is healed by the next nudge or by the reconnect sync, so no
     data travels through the bridge.
   - Every write that touches a `TableOrder` publishes after its transaction commits. A publish
     failure is logged and never fails the write.
5. **Frontend:**
   - The stream is opened with the existing fetch-based client (`frontend/core/agent/sse.ts`
     pattern, or `libs/sse-client.ts`), which sends `Authorization` and reconnects with backoff.
   - The 20 s bridge ping keeps it alive under the 60 s HAProxy timeouts.
   - A fallback poll every 30 s covers a bridge that is down.

---

## 9. Floor map colors

Per table, from its open order (pure, `routes/dining/dining.ts`, tested). The first rule that matches
wins:

| State | Rule |
|---|---|
| Late | a status-2 line older than `WaitLateMinutes` |
| Ready to pick up | a status-3 line |
| QR to confirm | a status-1 line |
| Waiting | a status-2 line older than `WaitWarnMinutes` |
| Ordered | a status-2 line |
| Eating | open, everything served |
| Free | no open order |

A one-second ticker re-renders the timers. It recomputes no data.

---

## 10. Frontend

`db/modules.toml`, Comercial, after Facturación:

```toml
[[modules.menus]]
name = "Restaurant|Restaurante"
min_name = "RES"
routes = "dining/floor-service, dining/kitchen-display, dining/floor-layout, dining/table-qr, production/daily-close"
```

`db/pages.toml`: one entry per route.

`backend/access.toml`, group 3, new ids (61+; 60 goes to "Cierre de Producción"):

| Access | Routes | APIs |
|---|---|---|
| Atención de Mesas | dining/floor-service | POST.dining-table-order-round, POST.dining-table-order-lines; sub-access 2 "Cobrar mesa": POST.dining-table-order-charge, POST.dining-table-order-annul |
| Cocina | dining/kitchen-display | POST.dining-table-order-lines (ready / undo only, checked in the handler by the caller's access) |
| Diseño de Salones | dining/floor-layout, dining/table-qr | POST.dining-layout, POST.dining-layout-qr-regenerate, GET.dining-table-qr-tokens |

| Route | Content |
|---|---|
| `routes/dining/floor-layout/` | **Editor.** `LayoutCanvas.svelte` in edit mode: an SVG grid with pointer events (pointerdown/move/up, pointer capture), snap to cells. **Tools:** select (click, shift-click, rubber band), move (drag, arrow keys = 1 cell), resize handles, table (square/round, seats), rectangle, line (wall/divider on grid points), label, station mark, pickup mark. **Edit:** duplicate (Ctrl+D or button, offset 1 cell), delete, rotate 90°, undo/redo (snapshot stack), zoom 8/12/16 px. **Side panel:** the selected shape's properties; the layout's settings (name, warehouse, QR confirmation, thresholds); stations with their routing (products and categories, editable VTable). Geometry in `floor-layout.ts` (snap, hit-test, bounds, overlap warning, duplicate offset, shape-ID allocation), tested. |
| `routes/dining/floor-service/` | **Live map.** `LayoutCanvas` read-only, colored by §9. A table opens a side `Layer` with its order by round and line state, wait times, add a round (product picker over `dining-menu`), confirm QR lines, mark served, cancel a line, charge (Cobrar mesa). The "Listos para recoger" strip. |
| `routes/dining/kitchen-display/` | §5. Wall-tablet layout, no small text. |
| `routes/dining/table-qr/` | §7. |
| `routes/dining/LayoutCanvas.svelte`, `dining.ts`, `dining.svelte.ts` | shared by the four pages: the renderer, the pure rules (§2.1, §9), the services (`DiningLayoutsService`, `TableOrdersService` as `GetHandler`s, the SSE subscription) |
| `frontend/webpage/routes/components/mesa/` | §6.3 |

Following the `online-orders` pattern:
- `TableOrdersService` uses `route = 'dining-table-orders?layout-id=N'`, so each layout has its own
  cache key.
- Charged and annulled orders arrive as removed IDs.

---

## 11. Endpoints

| Route | Who | What |
|---|---|---|
| `GET dining-layouts` | staff | delta of the company's layouts |
| `POST dining-layout` | Diseño de Salones | create/update one layout (§1.2) |
| `POST dining-layout-qr-regenerate` | Diseño de Salones | bump `QRVersion` |
| `GET dining-table-qr-tokens?layout-id` | Diseño de Salones | the token of each table |
| `GET dining-menu?layout-id` | staff | §3 |
| `GET dining-table-orders?layout-id` | staff | delta of the layout's open orders |
| `POST dining-table-order-round` | Atención de Mesas | open if none, add a round (validated by §3, reserves) |
| `POST dining-table-order-lines` | Atención de Mesas / Cocina | line transitions (§2.1) |
| `POST dining-table-order-charge` | Cobrar mesa | §4 |
| `POST dining-table-order-annul` | Cobrar mesa | annul an open order |
| `GET p-dining-menu?t=` | public | layout name, table label, menu |
| `POST p-dining-table-order-round?t=` | public | a QR round (§6.4) |
| `GET p-dining-table-order?t=` | public | the table's open order, as the customer sees it |

Every write publishes the layout's topic (§8).

---

## 12. Order of work (after PRODUCTION §8 steps 1–5)

1. `dining_layout` + `POST`/`GET` + the editor and canvas.
2. `dining_table_order`, rounds and reservation, line transitions, the floor-service page (polling
   first).
3. Charging (with the `sales/types` extraction of §4).
4. fareward topics + Go client + backend publish + the frontend subscription.
5. Kitchen display.
6. QR: tokens, the QR section, `/mesa` publication, the public routes and the page.

Out of scope: split bills, moving or merging tables, tips, reservations, kitchen printers, payment
from the QR page.

---

## 13. Pending decisions

None. D15 and D16 settled the review's points.
