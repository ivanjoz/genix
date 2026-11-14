# Manufacturing Plan: templates, production orders, labor, subcontracting, lots

Status: **approved, not implemented** (2026-11-13). Every decision is settled (§14).

This is the general production module for small manufacturers: shoes, garments, food, furniture.
It **shares its base with the restaurant daily close** of `docs/PRODUCTION_PLAN.md`. That close is
already implemented in the working tree (uncommitted, table `production_close` ID 96). Here it
becomes one kind of production order (§3).

Running example: a shoe workshop.
- Template **"Zapatos Verano"** holds 3 models, each with its size run of 36–41.
- An order made from it is released for 40 series.
- The leather is cut in-house. The stitching ("aparado") goes to an outside workshop.
- The finished pairs are received into stock in two partial receipts. Shoemakers and machine
  operators log their hours on the order.

## Settled decisions

| # | Decision |
|---|---|
| M1 | The reusable list of products and quantities is a **Plantilla de producción**, not a "lote". "Lote" stays the stock lot (`product_stock_lot`). |
| M2 | **A template is one reusable list** ("Zapatos Verano" is a template, not a group of them). An order is created from a template, and from then on it is independent of it. |
| M3 | Sizes are the product's **Presentations**. Order lines are per (product, presentation). |
| M4 | **One base for every production.** A manual order and a daily close are both a `production_order`. They share the recipe, the consumption and valuation, movement types 15/16 and the accounting. `production_close` is deleted. |
| M5 | **Waste per supply** in the recipe, plus an optional **adjustment per order** (the user's "margen de eficiencia"). |
| M6 | **Actual consumption and actual output** are recorded on each receipt. Planned vs. actual is the efficiency measure. |
| M7 | **Purchase suggestions** (requirements explosion, MRP) are in scope. |
| M8 | **Partial receipts.** One order can be received in several parts. |
| M9 | **Labor types** (Zapatero, Maquinista, Limpiador…) have a cost per hour **for estimates**. **Hours are logged per employee**, and the **same person may log different types** on different days. |
| M10 | **Subcontracting (maquila)** is in scope: **a subcontracted job is a production order with a workshop assigned**. |
| M11 | **Semi-finished products** are **supplies (Status 2) with `Origin = 2`** and a recipe of their own, booked to 231 / 713. |
| M12 | **Lot traceability, lot status and expiry** are in scope, **only for products that track lots**. |
| M13 | A **simple status board** (Kanban by stage). No routings, work centers or capacity planning. |
| M14 | Recipe stays **one per product**, applied to every presentation (PRODUCTION P17). |
| M15 | Pre-alpha: `production_close` rows are **wiped and reseeded**, not migrated. |
| M16 | **Templates store size runs ("series")**: units per size for one series. The order says how many series. |
| M17 | **Supplies leave the warehouse at each receipt.** While the order is open they are **reserved in a stored counter on `ProductStock`**, and **nothing may reserve more than is available**. |
| M18 | **Labor is valued at the employee's real cost per hour**, known beforehand from the contract and payroll. An employee's labor booked in a month **never exceeds their estimated monthly payroll cost**. When the actual payroll comes out lower, **one accounting adjustment** corrects it. Stock is never revalued. |
| M19 | A workshop invoice (Expense) **links to its production order**. |
| M20 | A lot's **expiry date** is entered on the lot, or derived from the product's **shelf life in days** since its production / entry date. |
| M21 | **Make-to-order is v2**: a suggestion like "make these products to fulfill these pending sales". Nothing is generated automatically. |
| M22 | A lot enters **Cuarentena** in two ways: **automatically**, when its product has `QualityHoldOnEntry`, and **by hand**, from the lot detail at any time. |
| M23 | An **expired lot is warned, not blocked**: it shows red and is consumed last. |
| M24 | An employee's **monthly hours = `ContractAgreement.DailyHours × 26`** (a 6-day week). |
| M25 | **Workers not on payroll** (recibo por honorarios, temporary) are **registered in HR** with the employees, so they can be picked on a labor entry. They have **no cap and no adjustment** (§9.5). |
| M26 | The daily close values labor at **standard**: recipe minutes × the type's rate. |
| M27 | The monthly labor adjustment is **Dr 7111 · Cr 2111**. |
| M28 | `EngagementType` has three values: **Planilla**, **Recibo por honorarios**, **Temporal sin planilla**. A non-payroll worker has an **agreement**: honorarios are paid **per hour**; a temporary worker is paid **monthly, weekly (with a pay day of the week) or per hour**. |
| M29 | A non-payroll worker's hourly cost comes from **their own agreement**, kept in the employee form. A worker with none falls back to the **labor type's rate**. |
| M30 | Payroll contracts and non-payroll agreements are **one table, `hr_contracts_agreements`** (the former `hr_contracts`, ID 73). |
| M31 | The recipe is **flat `Production*` columns on `Product`**, not a nested `ProductProduction` struct. The daily close work does the flattening (cb 48–51, cb 47 retired). |

---

## 1. Concepts and lifecycle

```
Plantilla ──create──▶ Orden (Borrador) ──release──▶ Liberada ──first work──▶ En proceso
                                       (reserves)         │                       │
                                                          └────── receipts (1..n) ┘
                                                                          │
                                                                    Terminada
Daily close ─────────────────────────────────────────────▶ Terminada (born closed, one receipt)
```

| Status | Value | Meaning |
|---|---|---|
| Anulada | 0 | Annulled. Allowed only with no receipt and no material at a workshop. Its reservations are released. |
| Borrador | 1 | Editable. Nothing is planned or reserved. |
| Liberada | 2 | The recipe is exploded and frozen on the order (§4.2), and its supplies are **reserved** (§5). |
| En proceso | 3 | Set by the first receipt, material shipment or labor entry. |
| Terminada | 4 | Completed by the user, or after the last planned unit is received. Unconsumed reservations are released, and labor entries are locked. |

Receipts cannot be annulled in v1, the same as the daily close (PRODUCTION P11). A mistake is
corrected with manual stock movements.

---

## 2. The recipe (flat `Production*` columns on `Product`, extended)

The recipe is **flat columns on `Product`** (M31), not a nested struct. The flattening is done in
the daily close work first, cb 48–51, with cb 47 retired. This plan adds `ProductionLabor`. The daily
close reads the recipe too.

```go
// on Product (backend/production/types/product.go)
ProductionYieldPortions    int16              `cb:"48"`
ProductionSupplies         []ProductionSupply `cb:"49"`
ProductionExtraCostPerUnit int32              `cb:"50"` // labeled "Costos indirectos por unidad" (overhead)
ProductionStandardCost     int32              `cb:"51"`
ProductionLabor            []ProductionLabor  `cb:"52"` // new: estimate only (M18)

type ProductionSupply struct {
	SupplyID   int32 `cb:"1"`
	Quantity   int32 `cb:"2"`
	SubDivisor int16 `cb:"3"`
	// WasteBasisPoints is the expected waste on top of Quantity: 1500 = 15% (leather), 0 = none.
	WasteBasisPoints int16 `cb:"4"` // new
}

// ProductionLabor is the work ProductionYieldPortions units are expected to need, of one labor type. It
// estimates the cost and plans the hours. The actual labor value comes from the hours logged (§9).
type ProductionLabor struct {
	LaborTypeID int32 `cb:"1"`
	Minutes     int32 `cb:"2"`
}
```

**Estimated unit cost.** One pure function, used by the product form, the template and the release.
It is `ProductionStandardCost` when that is > 0. Otherwise:

```
Σ supplies (quantity × (1 + waste) × average cost) ÷ yield
+ Σ labor (minutes × the labor type's CostPerHour) ÷ yield
+ ProductionExtraCostPerUnit
```

### 2.1 Semi-finished products (M11)

- A semi-finished product (e.g. "capellada aparada") is a **supply (Status 2) with `Origin = 2`** and
  its own recipe.
  - It is stocked and consumed like any supply.
  - It is never sold, and a purchase order refuses it, as it refuses every made product
    (`purchase-order-management.go:366`).
- `ValidateProductProduction` gains a **cycle check** (a recipe may not reach itself through its
  semi-finished supplies) and a depth limit of 5.
- The supply form (`supplies-materials`) gets the same "Producción" tab as the product form.
  - `PostSupplyMaterial` keeps excluding the `Production*` columns from its Merge for normal
    supplies (PRODUCTION §2.4).
  - It writes them for a supply with `Origin = 2`.
- Its stock account is **231 Productos en proceso**, and its variation account is **713 Variación
  de productos en proceso** (§11).
- PRODUCTION P12 ("supplies only, no sub-recipes") stays true for the restaurant close: the close
  consumes a semi-finished supply from stock and never explodes it.

### 2.2 Validation additions (`ValidateProductProduction`)

- `0 ≤ WasteBasisPoints ≤ 10000`.
- Every `LaborTypeID` is an active labor type, none appears twice, and every `Minutes > 0`.
- A recipe with labor and no supplies is allowed (pure assembly). `HasProductionConfig()` and the
  cost-source check count `ProductionLabor`.

---

## 3. One table for every production: `production_order` (M4)

`backend/production/types/production_order.go`, table ID **97**. It replaces `production_close`
(ID 96, deleted and retired).

| Column | Type | Notes |
|---|---|---|
| CompanyID | int32 | partition |
| ID | int32 | autoincrement. The `DocumentID` of every stock movement of the order. |
| Kind | int8 | 1 manual order, 2 daily close |
| TemplateID | int32 | the template it came from, 0 if none |
| Series | int32 | how many series of the template (M16), 0 if not from a template |
| WarehouseID | int32 | where the finished products enter (the close: where the sales left) |
| SuppliesWarehouseID | int32 | where the supplies come from. A subcontracted order uses the **workshop's warehouse** (§8). |
| SubcontractorID | int32 | `client_provider` (Type 2), 0 = made in-house (§8) |
| SubcontractPricePerUnit | int32 | PEN cents net of IGV, agreed with the workshop |
| Date | int16 | creation (UnixDay). The close: the day closed. |
| PlannedStart, PlannedEnd | int16 | UnixDay, optional |
| StageID | int32 | the board column (§9.4), 0 = none |
| SupplyAdjustmentBasisPoints | int16 | signed. +1000 = use 10% more supplies than the recipe, −500 = 5% less (M5). |
| Lines | []ProductionOrderLine | blob |
| Supplies | []ProductionOrderSupply | blob, frozen at release |
| Labor | []ProductionOrderLabor | blob, planned at release, recorded from the entries |
| LaborRecordedAmount | int64 | Σ `Amount` of the order's labor entries (§9.2), PEN cents |
| Receipts | []ProductionReceipt | blob, one per partial receipt (§6) |
| SaleOrderIDs | []int64 | the sales it settled (close only in v1) |
| IsStrict | bool | close only: company flag 15 when it ran |
| Notes | string | |
| Status | int8 | §1 |
| Created, CreatedBy, Updated, Version, Metadata | | standard |

```go
type ProductionOrderLine struct {
	ProductID      int32 `cb:"1"`
	PresentationID int16 `cb:"2"`
	PlannedUnits   int32 `cb:"3"`
	ProducedUnits  int32 `cb:"4"` // Σ receipts
	ScrappedUnits  int32 `cb:"5"` // Σ receipts: made but defective, never entered stock
	Value          int32 `cb:"6"` // Σ receipts, PEN cents
	CostSource     int8  `cb:"7"` // 1 standard, 2 supplies (the close's constants)
	ShortSupplies  bool  `cb:"8"` // some receipt consumed less than the recipe asked
}

type ProductionOrderSupply struct {
	SupplyID   int32 `cb:"1"`
	SubDivisor int16 `cb:"2"`
	Planned    int32 `cb:"3"` // sub-units, waste and adjustment included
	Available  int32 `cb:"4"` // snapshot at release / close
	Consumed   int32 `cb:"5"` // Σ receipts
	Reserved   int32 `cb:"6"` // what this order still holds on ProductStock (§5)
}

type ProductionOrderLabor struct {
	LaborTypeID     int32 `cb:"1"`
	PlannedMinutes  int32 `cb:"2"`
	PlannedCost     int32 `cb:"3"` // at the type's CostPerHour when released
	RecordedMinutes int32 `cb:"4"` // Σ production_labor_entry
	RecordedCost    int32 `cb:"5"` // Σ their Amount
}
```

Indexes:
- delta on `(Status)`: the order list and the board;
- delta on `(Kind, WarehouseID)`: the close history.

**What happens to the daily close.** `PlanDailyClose` keeps its algorithm, because the proportional
sharing of short supplies is specific to it. Its write changes:

- It inserts a `production_order` with `Kind = 2`, `Status = 4` and **one receipt**.
  - `ProductionCloseProduct` maps to `Lines`.
  - `ProductionCloseSupply` maps to `Supplies`.
  - `SaleOrderIDs` and `IsStrict` keep their meaning.
- Its supply availability becomes `physical − reserved` (§5). A close cannot eat the supplies a
  released order holds.
- Its labor is valued at **standard** (recipe minutes × type rate). Cooks do not log hours per close
  (M26).
- The consumption and entry movements go through the **shared receipt function** (§6.2). Only the
  close's sale exits (type 8) and their reservation release stay in `daily_close.go`.
- `GET.production-closes` becomes a delta read of `Kind = 2` by warehouse.
- The DINING plan depends on the close's behavior, not its table, so it is not affected.

---

## 4. Templates (M1, M2, M16)

### 4.1 Table `production_template` (ID **98**)

| Column | Type | Notes |
|---|---|---|
| CompanyID, ID | int32 | autoincrement |
| Name | string | "Zapatos Verano" |
| Lines | []ProductionTemplateLine | blob |
| DefaultSeries | int32 | prefilled on a new order |
| WarehouseID, SuppliesWarehouseID | int32 | defaults copied to the order |
| Notes, Status, Created, CreatedBy, Updated, Version, Metadata | | standard |

```go
// ProductionTemplateLine is one product in its size run: how many units of each presentation one
// series holds. 36×1, 37×2, 38×3, 39×3, 40×2, 41×1 is a series of 12 pairs.
type ProductionTemplateLine struct {
	ProductID int32                  `cb:"1"`
	Run       []PresentationQuantity `cb:"2"` // {PresentationID int16, Units int32}
}
```

- Validation: each product is made (`Origin = 2`) and has a recipe. Each presentation is active and
  appears once. Units > 0.
- The template page shows the **estimated cost of one series** and the **supplies one series needs**
  (§2's estimate).

### 4.2 From template to order

- "Nueva orden desde plantilla" asks for the **number of series** and creates one line per product
  and presentation: `PlannedUnits = Run.Units × Series`. Lines can still be edited while the order is
  a draft. An order can also be made without a template.
- "Guardar como plantilla" saves the order's lines back as a run (units ÷ series).
- **Release** (Borrador → Liberada) explodes the recipes, freezes them on the order and **reserves**
  (§5):

  ```
  planned[s] = ceil( Σ_lines PlannedUnits × recipe.quantity[s] ÷ recipe.yield
                     × (1 + waste[s]) × (1 + SupplyAdjustmentBasisPoints) )   // sub-units of s
  labor[t]   = Σ_lines PlannedUnits × recipe.minutes[t] ÷ recipe.yield
  ```

- **Release is refused if any supply is short** (M17). The error lists the shortages and links to
  the requirements page (§7). The user buys first, or lowers the series count.
- Later recipe edits never change a released order. This is a pure function in
  `production/types/production_order.go`, with tests.

---

## 5. Reservations: a stored counter on `ProductStock` (M17)

`ProductStock` (`logistics/types/product-stock.go`) gains:

```go
// ReservedProduction is what released production orders hold of this row and have not consumed
// yet, at the row's SubDivisor. Only an order's own consumption, transfer or release decreases it.
ReservedProductionQuantity    int32 `cb:"28"`
ReservedProductionSubQuantity int32 `cb:"29"`
```

It is a (Units, Sub) pair because supplies are measured in sub-units (50 g). The
`PendingDelivery*` counters are whole units only. Wherever the row's divisor is refined, the pair is
converted the same way `Quantity`/`SubQuantity` are.

**Writing it.** A new `ApplyProductionReservationIn(transaction, companyID, userID,
[]ProductionReservationChange)` in `logistics/types`, modeled on `ApplyPendingDeliveryIn`.
- Changes are merged per row and rows are locked in key order.
- **Increase:** refused unless `free + released lots − blocked − reserved ≥ requested`. This is the
  M17 rule. Lots in quarantine (§10) do not count.
- **Decrease:** refused below 0.

**Who changes it.**

| Event | Change |
|---|---|
| Release | + planned, per supply, in `SuppliesWarehouseID` |
| Receipt consumption (type 15 of the order) | − min(consumed, the order's `Reserved`). Consumption above the reservation must fit in **unreserved** stock. |
| Shipment to a workshop (type 17) | − at the source, + at the workshop warehouse, for the order's share |
| Completion, annulment | − whatever the order still holds |

**The engine honors it.** Every outflow that is not the owning order's own consumption or transfer
must leave `balance ≥ reserved`:
- manual exits (2), internal consumption (13), shrinkage (14);
- other orders' consumption (15);
- the daily close;
- transfers (17).

It is one extra comparison on rows the engine already locks. The owning order's movements carry
their reservation change in the same transaction, and the check runs after both.

**Recompute.** `exec/production_reservation_recompute.go` (like `pending_delivery_recompute.go`)
rebuilds the counters from the open orders' `Supplies[].Reserved`, for repair.

---

## 6. Receipts (M6, M8)

### 6.1 What a receipt records

```go
type ProductionReceipt struct {
	N            int16                          `cb:"1"` // 1, 2, 3… within the order
	Date         int16                          `cb:"2"`
	Lines        []ProductionReceiptLine        `cb:"3"` // {ProductID, PresentationID, Units, Scrapped, LotID, Value}
	Consumptions []ProductionReceiptConsumption `cb:"4"` // {SupplyID, LotID, SerialNumber, SubUnits, SubDivisor, Value}
	SuppliesValue    int64 `cb:"5"`
	LaborValue       int64 `cb:"6"` // the labor logged since the previous receipt (§9.2)
	SubcontractValue int64 `cb:"7"` // units × SubcontractPricePerUnit
	OverheadValue    int64 `cb:"8"` // units × ProductionExtraCostPerUnit
	CreatedBy        int32 `cb:"9"`
}
```

### 6.2 Flow (`POST production-order-receipt`, one transaction)

1. The user enters the **good** and **scrapped** units per presentation.
2. The page proposes the consumption:
   - a normal receipt: `Planned[s] × (good + scrapped) ÷ total planned units`;
   - the last receipt ("Completar"): the remainder.
   The user edits the **actual** quantity of each supply.
3. Lock the order, then re-read it and check its status and version.
4. **Consumption exits** (type 15, from `SuppliesWarehouseID`) go through `ApplyMovimientosWithLedgerIn`,
   together with the order's reservation decrease (§5).
   - Each supply is drawn from the free bucket first, then from the **released** lots by earliest
     expiry, then oldest (§10).
   - The bucket walker moves out of `daily_close.go` into a shared function. The close uses it too.
5. **Value** = Σ the consumption values the engine stamped + labor + subcontract + overhead, or
   `ProductionStandardCost × units`.
   - **Labor:** `LaborValue = LaborRecordedAmount − Σ LaborValue of the previous receipts`. Each
     receipt absorbs the hours logged since the last one.
   - The value is spread over the presentations by good units. The last presentation takes the
     rounding remainder, as the close does (`daily_close.go:374-390`). Scrapped units have no value:
     the good units absorb the whole cost.
6. **Production entries** (type 16, into `WarehouseID`, `InflowValue`). A product that tracks lots
   gets the lot `OP<orderID>-<N>` (§10).
7. Append the receipt, then update `Lines`, `Supplies.Consumed/Reserved` and `Status` (3, or 4 when
   completed). Completion releases what is left and locks labor entries, so no logged hour is left
   unabsorbed.

The daily close calls steps 4–6 with its own consumption plan, then adds its type 8 exits.

---

## 7. Purchase suggestions (MRP, M7)

Page `routes/production/requirements/`, with the pure logic in `requirements.ts`. **No new backend
endpoint.** The page reads only what the services already load.

1. **Input:** draft orders, or a template × series count, and a supplies warehouse.
2. **Multi-level explosion:** finished lines → supplies by recipe. A semi-finished supply first nets
   against its stock, then its shortfall explodes through its own recipe.
3. **Netting per supply:**
   ```
   need − (stock − reserved − blocked) − on order
   ```
   "On order" means the lines of purchase orders in Confirmed (2), which are not yet received.
4. **Output:**
   - **To buy:** supply, shortfall rounded up to whole units, provider, price.
     - The providers come from `ProductSupply.ProviderSupply`, with the lowest price preselected.
     - "Crear órdenes de compra" groups the lines by provider and calls the **existing**
       `POST purchase-orders` once per provider. The orders are created Pending.
   - **To make:** semi-finished shortfalls. "Crear órdenes" creates **draft production orders**, and
     subcontracted products keep their last workshop.
5. With the box "reponer al mínimo", the need also includes `ProductSupply.MinimunStock`.

A purchase line has no sub-quantity, so a shortfall of 300 g is suggested as 1 kg.

**v2 (M21):** a "Para pedidos pendientes" tab. Pending sales of made products, minus finished stock,
become a suggested draft order. It is never created automatically.

---

## 8. Subcontracting / maquila (M10, M19)

The typical case: the uppers are stitched by a workshop, as an order for the semi-finished product
"capellada aparada". The main order then consumes the stitched uppers. A whole product made by a
workshop works the same way.

| Piece | Change |
|---|---|
| Workshop warehouse | `Warehouse` (`business/types/sites_warehouses.go`) gains `SubcontractorID int32`. A warehouse with it is "material at workshop X". It is excluded from sales and the web (`ExcludeFromOnlineSales` forced). |
| The order | `SubcontractorID`, `SubcontractPricePerUnit`. `SuppliesWarehouseID` = that workshop's warehouse. |
| Sending material | "Enviar materiales" creates **type 17 movements**, a new type ("Traslado de producción", document `production_order`, two-leg with `DestWarehouseID`), from the company's warehouse to the workshop's. They default to `Planned − already at the workshop`. The reservation moves with them (§5). |
| Returning leftovers | The same type 17, workshop → company warehouse. Allowed until completion. Completion warns if material is left at the workshop. |
| Receipt | Consumes from the workshop warehouse and enters into the order's `WarehouseID`. `SubcontractValue = units × price`. |
| The workshop's invoice | A normal **Expense** to **633 Producción encargada a terceros** (already in the chart). It gains **`ProductionOrderID int32`**: the order shows agreed vs. invoiced, and the expense form offers the provider's open subcontracted orders. |
| Annul | Refused while material is at the workshop. |

Type 11 stays reserved for the generic transfer screen, which is not part of this plan.

---

## 9. Labor (M9, M18) and the status board (M13)

### 9.1 Labor types: table `production_labor_type` (ID **99**)

`CompanyID, ID, Name ("Zapatero"), CostPerHour int32 (PEN cents), Status, Updated, Version,
Metadata, audit`. Delta cache.
- `CostPerHour` is used for **estimates**: recipe, template, release.
- It is also the fallback rate for a worker not on payroll who has no agreement (§9.5, M29).
- The settings page shows the average real rate of the employees who logged that type last month,
  as a hint.

### 9.2 Hours: table `production_labor_entry` (ID **100**)

| Column | Type | Notes |
|---|---|---|
| CompanyID, ID | int32 | autoincrement |
| OrderID | int32 | |
| EmployeeID | int32 | `hr_employees`, payroll or not (§9.5). Always set. |
| LaborTypeID | int32 | picked on each entry (M9) |
| Date | int16 | UnixDay |
| Period | int16 | YYMM |
| Minutes | int32 | |
| CostPerHour | int32 | the employee's real rate when logged (below) |
| Amount | int32 | the valued cost, after the monthly cap |
| Status, Created, CreatedBy, Updated, Version, Metadata | | |

Indexes:
- delta on `(OrderID)`: the order's "Mano de obra" tab;
- cache-groups on `(Period)`: the monthly report and the adjustment (skill `cache-groups-api`).

**The employee's real cost per hour, known beforehand (M18).**
- A new pure function `EstimatedMonthlyEmployerCost(employee, contract, parameters)` in `hr/types`.
  It runs the existing payroll engine (`CalculateMonthlyPayslip`) for a full month with no absences,
  and returns `EmployerCost + Σ provision lines (Kind 4)`: EsSalud, gratificación, CTS, vacation.
- `CostPerHour = estimated monthly cost ÷ monthly hours`. Monthly hours = `ContractAgreement.DailyHours × 26`
  (M24).
- A worker not on payroll uses their own hourly rate instead, with no cap (§9.5).
- Production imports `hr/types`, never the hr body, so the module boundary holds.

**The monthly cap.** When an entry is saved:

```
Amount = min( Minutes × CostPerHour,
              estimated monthly cost − Σ Amount of the employee's other entries in the Period )
```

An employee's labor booked in a month never exceeds their estimated payroll cost.

Each write (insert, edit, delete) refreshes the order's `Labor[].Recorded*` and
`LaborRecordedAmount` in the same transaction. Entries are refused on a Terminada or Anulada order.

Entry happens on the order's "Mano de obra" tab: an editable VTable with employee, type, date and
hours.

### 9.3 Monthly adjustment and the "Horas vs planilla" report

When the period's payroll is run, per employee:

```
booked = Σ Amount of their entries in the Period
actual = their Payslip.EmployerCost + Σ provision lines (Kind 4)
excess = max(0, booked − actual)
```

- The **Σ excess** of the month becomes **one accounting operation** (§11). Stock values are never
  touched (M18).
- A shortfall (`booked < actual`, idle hours) needs nothing: it is payroll expense that never
  entered a product.
- In July/December and May/November the benefit earnings are charged against provisions already
  accrued. `actual` uses each month's **provision lines** and ignores the paid benefit, so nothing
  is counted twice.

Page `routes/production/labor-hours/`, per employee and month:

| Figure | Source |
|---|---|
| Hours logged on orders | Σ Minutes, split by labor type |
| Hours paid | `Payslip.WorkedDays × ContractAgreement.DailyHours` + overtime hours (`hr_employee_records`, concepts 3/4) |
| Utilization, idle hours | logged ÷ paid, paid − logged |
| Labor booked to products | Σ Amount |
| Actual payroll cost | `EmployerCost` + provisions |
| Adjustment | excess (above) |

- Employees appear if they logged hours or belong to a department with `CostFunction = 3`
  (Producción). A production worker with no hours shows 0%.
- A month without a payroll run shows the hours and the booked labor only, flagged "planilla
  pendiente".
- It has its own access (§12), because it shows pay per hour.

### 9.4 Stages and the board

- Table `production_stage` (ID **101**): `CompanyID, ID, Name ("Corte", "Aparado", "Armado",
  "Acabado"), Position int16, Status…`. Delta cache.
- The orders page has a **"Tablero"** view built on genix-ui `Kanban.svelte` (as `crm/pipeline` uses
  it).
  - Columns are the stages. Cards are the orders in Status 2–3.
  - Dragging a card calls `PUT production-order-stage`.
  - A card shows the products, the % received and the workshop.
- The stage is information only: it moves no stock and changes no cost.

### 9.5 Workers not on payroll (HR change, M25)

Today an employee without a contract in force is still visited by every payroll run. They come back
in its "skipped" list with "No tiene un contrato vigente" (`payroll_calculate.go:224-228`,
`payroll_engine.go:307`), so a non-payroll worker would be reported every month.

| Change | Detail |
|---|---|
| `Employee.EngagementType int8` (cb 31) | 1 **Planilla** (default), 2 **Recibo por honorarios** (4ta categoría), 3 **Temporal sin planilla** (M28). A selector on the employee form. Existing rows are reseeded/backfilled to 1. |
| Payroll | `calculateRunPayslips` skips anything that is not Planilla **silently**. The PLAME/T-Registro exports leave them out. |
| Their pay terms | A version of **`hr_contracts_agreements`** (below): the same table as payroll contracts, with the fields that do not apply to them left empty. Edited in the employee form's existing contracts tab, renamed **"Contratos y acuerdos"**. |
| Employee lists | The employees page shows everyone, with an engagement column and filter. The time-off, payslip and payroll-records pages list Planilla only. |
| Labor entry | `CostPerHour` = the hourly cost of their agreement in force on the entry's date. A worker with none falls back to the labor type's `CostPerHour` (M29). `Amount = Minutes × CostPerHour`, with **no cap**. |
| Monthly adjustment, report | No adjustment. "Horas vs planilla" shows them in a separate block: hours, rate, labor booked. There is no payroll to compare with. |
| Their payment | An ordinary **Expense** with the receipt as its document (a recibo por honorarios, or none). It is classified to the expense account the accountant chooses. Booking their labor to products credits 71, as for everyone else, so there is nothing more to post. The 4ta-categoría withholding is out of scope. |

**Table `hr_contracts` becomes `hr_contracts_agreements`** (M30). It keeps table ID **73** and its
versioning by `ValidFrom`: a raise is a new row, and the row in force on a day applies. One row is
the terms of one worker from one date, whether a payroll contract or a non-payroll agreement.
- The Go types `Contract` / `ContractTable` (`hr/types/contracts.go`) become `ContractAgreement` /
  `ContractAgreementTable` in `hr/types/contracts_agreements.go`.
- The routes `GET/POST.hr-contracts` become `GET/POST.hr-contracts-agreements`.
- The renames follow in `access.toml`, `frontend/services/hr/contracts.svelte.ts` (→
  `contracts-agreements.svelte.ts`), `EmployeeContracts.svelte` (→ `EmployeeContractsAgreements.svelte`)
  and `seeds/hr/staff.go`.
- Pre-alpha: the table is wiped and reseeded under its new name (M15). No migration.

Changes to the columns:

| Column | Change |
|---|---|
| `MonthlySalary int32` (cb 8) | **Renamed `PayAmount`**: PEN cents per month, week or hour, by `PayBasis`. The payroll engine reads `PayAmount` wherever it read `MonthlySalary` (19 references, all mechanical). |
| `PayBasis int8` (new, cb 25) | 1 **Mensual**, 2 **Semanal**, 3 **Por hora** |
| `PayDayOfWeek int8` (new, cb 26) | weekly only: 1 Monday … 7 Sunday |
| `DailyHours` (cb 9) | unchanged. It also turns a monthly or weekly agreement into an hourly cost. |
| `ContractType`, `PensionSystem`, `CUSPP`, `AfpCommissionType`, `HasEPS`, `CtsBankID`, `CtsAccount` | payroll only: 0 / empty on a non-payroll agreement |
| `StartDate`, `EndDate`, `SalaryBankID`, `SalaryAccount` | shared by both |

Validation (`PostContractAgreement`) follows the employee's `EngagementType`:

| EngagementType | Rule |
|---|---|
| Planilla | `PayBasis = 1`. A T-Registro `ContractType` is required. Today's payroll rules all hold. |
| Recibo por honorarios | `PayBasis = 3`. `ContractType` and the pension, EPS and CTS fields are empty. |
| Temporal sin planilla | `PayBasis` is 1, 2 or 3. The payroll fields are empty, as above. |
| all | `PayAmount > 0`. `PayDayOfWeek` is in 1..7 when weekly and 0 otherwise. `DailyHours` is in 1..12 when monthly or weekly. |

A worker who changes from Temporal to Planilla gets a new `EngagementType` on the employee, plus a
new agreement version from that date.

The form shows only the fields that apply to the employee's `EngagementType`.

**Hourly cost**, a pure function `ContractAgreementHourlyCost(employee, agreement, parameters)` in
`hr/types`:

| Case | Cost per hour |
|---|---|
| Planilla | `EstimatedMonthlyEmployerCost ÷ (DailyHours × 26)` (§9.2, M24) |
| Mensual, not payroll | `PayAmount ÷ (DailyHours × 26)` |
| Semanal | `PayAmount ÷ (DailyHours × 6)` |
| Por hora | `PayAmount` |

`PayDayOfWeek` is shown on the employee list and in the agreement. **Generating the weekly payment**
(e.g. as a scheduled expense) **is out of scope** for this plan.

---

## 10. Lots, traceability, status and expiry (M12, M20)

Today:
- `product_stock_lot` (ID 20) has `Date, Name, SupplierID` and no status, expiry or origin.
- `ProductStockDetail.ExpirationDate` exists but is never written.

**Product** gains:
- `TracksLots bool` (cb 53):
  - a made product with it gets one lot per receipt, `OP<orderID>-<N>`;
  - a supply with it must be received with a lot.
- `ShelfLifeDays int16` (cb 54): the default expiry, counted from the lot's production date
  (receipt) or entry date (purchase).
- `QualityHoldOnEntry bool` (cb 55): its new lots enter in Cuarentena (M22). Any lot can also be
  moved by hand between Liberado / Cuarentena / Rechazado from the lot detail.

**Lot** (`ProductStockLot`) gains:

| Field | Meaning |
|---|---|
| `OriginMovementType int8`, `OriginDocumentID int64` | stamped by the engine when `resolveLotIDsForMovements` creates the lot: a purchase (3, the PO) or a production (16, the order) |
| `ExpirationDate int16` | from `InternalMovement.LotExpirationDate` when given, else `entry date + ShelfLifeDays`, else 0 (none). It stays editable on the lot. `ProductStockDetail.ExpirationDate` is **deleted**. |
| `QualityStatus int8` | 1 Liberado, 2 Cuarentena, 3 Rechazado |

**Quarantine, blocked stock and the engine.**
- `ProductStock` gains `BlockedQuantity` / `BlockedSubQuantity` (cb 30/31): the stock of this row in
  lots that are not Liberado.
  - The engine maintains it for movements into or out of such lots.
  - The status-change endpoint maintains it for every detail row of the lot. `ProductStockDetail`
    gains an index on `LotID` for that.
- A lot that is not Liberado is refused as the source of a sale (8), a consumption (15) or a
  transfer (17). It may leave only by manual exit (2) or shrinkage (14), to dispose of rejected
  goods.
- `web_sellable.go` subtracts `Blocked*` from the net. Reservations (§5) and the MRP exclude it too.
- The consumption walker (§6.2) skips unreleased lots and uses the **earliest expiry first**.
- An expired lot is **warned, not blocked** (M23): it shows red in stock, and the walker takes it
  last.

**Traceability: no new table.** A lot's ledger rows are already indexed by `LotID`.
- **Forward:** supply lot → its type 15 rows → their `DocumentID` (the order) → the receipts → the
  produced lots.
- **Backward:** produced lot → `OriginDocumentID` → the receipt's `Consumptions`, each with its
  `LotID`.
- A "Trazabilidad" view sits in the order and in the products-stock lot detail. The lot detail also
  shows and changes its status.

---

## 11. Accounting (to be confirmed with the accountant)

The PCGE by-nature model, which `ACCOUNTING_MODEL_PLAN.md` already uses (no class 9, no 79):
- labor stays where payroll books it (62x);
- the workshop invoice stays in 633;
- overhead stays in its 6x;
- the **production entry credits 71** at the full cost.

Unsold production therefore nets to zero in the P&L, and the stock carries it. This replaces
PRODUCTION §7, which is still not implemented (15/16 are not in `operationStockTypes`). Both kinds
of order are booked once, here.

| Event | Entry |
|---|---|
| 15, consumption of a bought supply | Dr its variation account · Cr its stock account (6032 → Dr 6132 · Cr 252) |
| 15, consumption of a semi-finished supply | Dr **713** · Cr **231** |
| 16, entry of a finished product | Dr **2111** · Cr **7111** |
| 16, entry of a semi-finished product | Dr **231** · Cr **713** |
| 17, transfer to or from a workshop | none (same account) |
| Workshop invoice | Expense: Dr 633 · Dr 40111 · Cr 4212 (existing path) |
| Sale of a produced unit | Dr 6922 · Cr 2111 (existing) |
| **Monthly labor adjustment** (§9.3) | Dr **7111** · Cr **2111** for Σ excess (M27). The ledger's 2111 then differs from the stock's value by the adjustments, because stock is never revalued. |

- One operation per order and receipt: key `PR-<orderID>-<N>`, kind `OPERATION_PRODUCTION`.
- One labor adjustment per period: key `LA-<period>`, kind `OPERATION_LABOR_ADJUSTMENT`. The
  accounting API sends the per-employee booked totals of the period, and the browser compares them
  with the payslips it already receives.
- `operationStockTypes` gains 15, 16 and 17.
- The dispatch in `accounts-consolidation.ts` becomes explicit:
  - 3/4 → reception;
  - 8/9 → cost of sales;
  - 15/16 → production;
  - 17 → none;
  - anything else → a visible warning instead of a silent skip.
- New constants: 7111, 713, 231, 633.
- Tests:
  - a two-supply receipt;
  - a semi-finished chain;
  - a standard-cost product;
  - a labor adjustment.

---

## 12. Pages, menus and access

| Route | Page | Access |
|---|---|---|
| `production/orders` | Órdenes de producción. **Lista** \| **Tablero** views. A detail Layer with tabs: Productos (size grid), Insumos, Mano de obra, Maquila, Ingresos, Trazabilidad, Resumen. | **61 "Órdenes de producción"** |
| `production/requirements` | Necesidades de compra (MRP) | 61 |
| `production/templates` | Plantillas de producción | **62 "Configuración de producción"** |
| `production/settings` | Tipos de trabajo, Etapas (two tabs) | 62 |
| `production/labor-hours` | Horas vs planilla | **63 "Costos de mano de obra"** |
| `production/daily-close` | unchanged | 60 |

- All three new accesses are in group 9 (Producción).
- The lot status action sits in `logistics/products-stock`, under its existing access.
- `db/modules.toml`, Operaciones › Producción:
  `production/orders, production/requirements, production/templates, production/merchandise,
  production/products, production/supplies-materials, production/daily-close, production/labor-hours,
  production/settings`.
- Folder layout per CLAUDE.md §5. For example, `orders/` holds:
  - `+page.svelte`, `OrderBoard.svelte`, `OrderLayer.svelte`, `OrderReceipt.svelte`;
  - `production-order.svelte.ts`;
  - `production-order.ts`: the explosion, the proposed consumption and the efficiency figures, all
    pure;
  - `production-order.test.ts`, `DOCUMENTATION.md`.
- The **Resumen** tab compares, per order:
  - planned vs. produced units (yield %);
  - planned vs. consumed per supply;
  - planned vs. recorded minutes and cost per labor type;
  - estimated vs. booked cost per unit.

## 13. Order of work

0. The daily close (PRODUCTION §1–§6) is committed as it is in the working tree. Another session is
   writing it now.
1. **Recipe:**
   - waste, labor lines, semi-finished supplies plus the cycle check;
   - the supply form's Producción tab;
   - `production_labor_type` and `production_stage`, plus the `settings` page.
2. **`production_order`:**
   - replaces `production_close`: wipe, reseed, refactor the close;
   - the shared receipt function;
   - movement types point to `production_order`.
3. **Reservations:** `ProductStock` counters, `ApplyProductionReservationIn`, the engine check, the
   recompute exec. The close starts honoring them.
4. **Accounting** (§11) for 15/16/17, which also closes PRODUCTION §7.
5. **Templates** and **orders**: draft, release, receipts, completion, annul, list and board.
6. **Requirements / MRP**, with purchase-order creation through the existing endpoint.
7. **Labor:**
   - HR: `EngagementType`, the `hr_contracts` → `hr_contracts_agreements` merge and rename, payroll skipping non-payroll
     workers (§9.5);
   - `EstimatedMonthlyEmployerCost`;
   - labor entries with the cap;
   - receipts absorbing labor;
   - the "Horas vs planilla" report;
   - the monthly adjustment operation.
8. **Subcontracting:** the workshop warehouse, type 17, send/return, the subcontract value,
   `Expense.ProductionOrderID`.
9. **Lots:**
   - `TracksLots`, `ShelfLifeDays`, `QualityHoldOnEntry`;
   - lot origin/expiry/status, `Blocked*` counters, the engine checks;
   - production lots, traceability.
10. **v2:** make-to-order suggestions (M21).

Each step leaves the system working. Steps 1–4 change nothing visible for a restaurant except the
history types.

## 14. Pending decisions

| # | Question | Proposal |
|---|---|---|
None. Q1–Q8 are settled as M22–M29, plus M30.
