# PLAN — Editable asset acquisition data

Status: **implemented.** Verified end to end against the real database; see §4. The four open decisions were settled by the user; see §2.3, §2.1.3, §2.2 and §5.

Goal: from the asset detail panel (`Layer id=2`), an **Editar** button reopens the acquisition
form — as a `Modal`, not a side `Layer` — with only five fields writable:

| Field | Column | Consequence of changing it |
| --- | --- | --- |
| Monto de Compra | `PurchaseAmount` | payment state; conflicts with payments already registered |
| Valor en Libros | `AcquisitionValue` | **rewrites the whole depreciation ledger** |
| Fecha de Adquisición | `AcquisitionDate` | **rewrites the whole depreciation ledger** (period 1 = month after acquisition) |
| Vencimiento del Pago | `DueDate` | none — a reminder date only |
| Número de Serie | `SerialNumber` | it is the key of the `ProductStockDetail` row the asset shadows |

Everything else on the form (Material, Almacén, Proveedor, Moneda, Cantidad) renders **disabled**.
Warehouse already has its own path, `PUT.asset-transfer`; the rest are the asset's identity.

---

## 1. What makes this non-trivial

`DepreciationSchedule(asset, throughDate)` (`backend/accounting/depreciation.go`) is a **pure
function of `AcquisitionValue`, `AcquisitionDate` and `DepreciationMonths`**. It has no notion of a
mid-life basis change. So the moment either of the first two changes, every Type-4 `Expense` row
already posted for the asset, plus `AccumulatedDepreciation` and the `LastDepreciationDate`
watermark, describe a schedule that no longer exists.

Three further facts constrain the fix:

- **The ORM has no `Delete`.** `backend/db/` exposes Insert / Update / Merge only. A posted
  depreciation row can only be retired by setting `Expense.Status = 0` (the removed slot, already
  declared in its `FixedValues`).
- **`ProductStockDetail`'s key is `(ProductStockID, LotID, SerialNumber)`** — a serial cannot be
  updated in place; the units have to move.
- **`Asset.PaidAmount` is server-maintained** from the `CashBankMovement` ledger, so an edit can
  never touch it. It is the number a new `PurchaseAmount` has to be checked against.

---

## 2. Backend — `PUT.asset`

New handler `PutAssetEdit` in `backend/accounting/asset_api.go`, payload:

```go
type AssetEditPayload struct {
	AssetID          int32
	SerialNumber     string
	AcquisitionDate  int16
	DueDate          int16
	AcquisitionValue int32   // asset-row total, in cents — see §5
	PurchaseAmount   int32
}
```

### 2.1 Validation (in order, all server-side)

1. Asset exists in the company; `Status != Removed`.
2. `AcquisitionValue > 0`, `PurchaseAmount >= 0`, `AcquisitionDate > 0`.
3. `PurchaseAmount >= PaidAmount` — **rejected otherwise**, naming the amount already paid. This is
   the payments case: the correct move is to reverse the payment first, and reversing cash is not
   something an asset edit gets to do implicitly.
4. `DisposalDate > 0 && AcquisitionDate > DisposalDate` → rejected.
5. `DueDate > 0 && DueDate < AcquisitionDate` → rejected.
6. Serial changed → must not collide with another non-removed asset of the same `ProductID`
   (there is a local index on `SerialNumber` to query).
7. Serial changed **and** the asset is disposed → rejected. Its units are already out of stock;
   there is nothing to move.

### 2.2 Serial change moves the stock

Two `logistics.InternalMovement`s through `ApplyMovimientos` — the same engine acquisition and
disposal use: `-Quantity` on the old serial, `+Quantity` on the new one, both carrying the asset ID
as `DocumentID`. Going serial → empty, or empty → serial, is the same pair (an empty serial is the
no-detail bucket).

### 2.3 Depreciation rebuild — the rows are rewritten in place

Only when `AcquisitionValue` or `AcquisitionDate` actually changed. `rewriteAssetDepreciation`
reconciles the posted ledger against the schedule the *new* values produce, rather than retiring
the old ledger wholesale:

1. Read every `Expense` with `AssetID = asset.ID`, keep `Type = 4` and `Status != 0`, sort by `Date`.
2. `newSchedule := DepreciationSchedule(asset, core.FechaUnix())` with the new values already on the
   asset — the full elapsed schedule, not the pending tail.
3. For the first `min(len(old), len(new))` rows: overwrite `Date` and `Amount` on the existing row.
4. Surplus new periods (`len(new) > len(old)`): insert them, same shape as `PostAssetDepreciationRun`.
5. Surplus old rows (`len(old) > len(new)`): `Status = 0`, the removed slot.
6. `AccumulatedDepreciation` / `LastDepreciationDate` take the last new period's values, or 0 when
   the new schedule is empty (the acquisition date moved into the current month).

Chosen over retiring the whole ledger and regenerating it: no garbage rows accumulate per edit, the
row IDs stay stable, and the expense register sees an amount change rather than a delete plus an
insert. It costs the reconciliation loop — about twenty lines more than a wholesale rebuild.

The update writes `Date`, `Amount`, `Status`, `Updated`, `UpdatedBy` in one call. `Status` rides
along on every row, changed or not, for the same reason `PostAssetPayment` does it: it shares the
delta view's composite key with `UpdatedVersion`.

`ResolveAssetStatus` then lands the asset back on Active or FullyDepreciated on its own.

**`GetAssetDepreciation` must start filtering `Status != 0`** — today it returns everything for the
asset, so rows retired by step 5 would show up in the panel's ledger.

### 2.4 Known limitation, deliberately not fixed

The inbound stock movement written at acquisition carries `Price = AcquisitionValue`. An edit leaves
that stale. Rewriting a historical movement to correct a price is a bigger change than this task,
and the movement ledger is not what the balance sheet reads. Recorded in `RATIONALE.md`.

### 2.5 Registration

- `accounting/main.go`: `"PUT.asset": PutAssetEdit`.
- `backend/access_list.yml` id 25 `backend_apis`: append `PUT.asset`.
- `cd scripts && go run . generate_route_ids` to assign its `int16`.

---

## 3. Frontend — the form becomes a component

New `frontend/routes/accounting/assets/AssetForm.svelte`:

```ts
interface Props {
  form: IAssetAcquisition | IAssetEdit
  mode: "create" | "edit"
  serialsInput: { text: string }      // create only — one serial per line
  supplies; warehouses; providers     // option lists, passed in
}
```

It renders the shared field grid. `mode` decides three things and nothing else:

- **Disabled**: in `edit`, Material / Almacén / Proveedor / Moneda / Cantidad get `disabled`.
- **The `Info` hints disappear.** "El activo debe registrarse como material para ser adquirido"
  and the serials note are onboarding for a form that has not been submitted yet; on an existing
  asset they are noise. The donated-asset note also goes: whether it was donated is already decided.
- **Serials**: `create` shows Cantidad + a one-per-line textarea; `edit` shows one
  `Número de Serie` input, because one asset row is one serial.

`+page.svelte` keeps both shells: `Layer id=1` for create (unchanged), `Modal id=12` for edit, both
rendering `<AssetForm>`. `openAssetEditModal(asset)` seeds the edit form from the selected asset;
on success it patches `selectedAsset`, refetches `assets`, reloads `depreciationEntries` (the
ledger the panel shows was just rewritten) and closes the modal.

`assets.svelte.ts` gains `IAssetEdit` + `putAssetEdit(data)` with
`refreshRoutes: ["assets", "expenses"]`.

The edit form writes **the asset row's totals**, not per-unit values: it drops the "(por unidad)"
suffix and sends `AcquisitionValue` / `PurchaseAmount` straight through. Dividing a stored total by
`Quantity` would not round-trip for a lot of 3.

The **Editar** button goes in the panel's existing button row, next to Registrar Pago / Dar de Baja,
gated on `asset.ss !== AssetStatus.DISPOSED`.

---

## 4. Verification — what actually ran

- `go build ./...`, `go vet ./...`, `go test ./accounting/...` — pass. Four new tests in
  `asset_test.go` cover `ResolveAssetPaymentStatus` and the three schedule-rewrite shapes
  (same length, shortened, emptied).
- `bun run check` — no errors in `routes/accounting/assets/` or in the two genix-ui files.
  Ten pre-existing errors remain in five untouched files.
- **`PUT.asset` exercised against the live database** through a throwaway `exec` entrypoint,
  since removed. Asset 1 (2,000.00 over 48 months, 14 periods posted, accumulated 58324):
  - value → 1,500.00: 14 rows **updated in place, 0 inserted**, amounts 4166 → 3125,
    dates unchanged, `AccumulatedDepreciation` 43750 = 3125 × 14. Correct.
  - acquisition date → Oct 2026: schedule shrank to 4 periods, the 10-row tail retired
    with `Status = 0`, the 4 survivors took the new dates.
  - back to the original date and value: **all 14 rows reused, 0 inserted** — the retired
    tail was revived, and the ledger came back byte-identical (same IDs, dates, 4166, 58324).
    This round trip is what §2.3's reuse rule buys; the first implementation would have
    left 10 dead rows and inserted 10 replacements.
  - serial `""` → `SN-TEST-001` → `""`: the stock moved both ways. Movement ledger shows
    `-1` on the free bucket and `+1` on the serial, then the reverse; the free bucket
    holds the unit again. A zero-quantity `warehouse_product_stock_detail` row for the
    serial remains, which is how the stock layer works everywhere.
  - `AcquisitionValue: 0` → rejected 400, *"El valor en libros debe ser mayor a 0."*
  - Asset 1 was left exactly as it was found.
- `agent-browser`: the create Layer renders identically to before the extraction, and the
  edit Modal renders with Material / Almacén / Proveedor / Moneda / Cantidad greyed out,
  focus on Valor en Libros, and no onboarding hints.

Not verified: the `PurchaseAmount < PaidAmount` rejection has no unpaid-asset fixture on this
database (asset 1 has `PaidAmount = 0`), so only the pure rule behind it is unit-tested.

**Noise found, not caused here:** `ApplyMovimientos` prints `Error: Value is not an integer: <nil>`
several times per call, from the ORM's `sequences` accessor. It is unrelated to this change and
harmless — the movements are written correctly — but it is real and worth a look someday.

## 5. Settled: the edit form writes totals

See §3. The create form keeps its per-unit labels and its `× Quantity`; only the edit form works in
row totals, because that is what the column holds and what the detail panel already displays.
