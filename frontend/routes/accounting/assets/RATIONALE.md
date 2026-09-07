# RATIONALE — accounting/assets

Design decisions for the Activos page, newest first.

## The detail panel's four figures moved to `LabelCell`

**Context** — The acquisition/book-value block was four copies of the same label/value markup, with a
page-local `.stat-label` class colouring the label with `--input-label-color` so it read as a field
label. The users page needed the identical pattern.

**Decision** — Extracted as `$components/form/LabelCell.svelte` and used here; `.stat-label` and the
page's `<style>` block are gone. `valueCss="h4"` keeps the two smaller figures as they were.

**Rationale** — A straight 1:1 swap, so the panel is unchanged. Note it was **not** verified in the
browser: the detail layer only opens from the table's `onRowClick`, which the agent's `selectRow`
does not invoke, so the check was `svelte-check` plus reading the diff.

## The acquisition form is a component, and the edit shell is a Modal

**Context** — Once an asset was created there was no way back into it: the detail panel is a read
surface, and the acquisition form lived inline in `Layer id=1`. Correcting a mistyped purchase
amount or serial meant touching the database.

**Decision** — The field grid moved to `AssetForm.svelte`, rendered by both shells: `Layer id=1`
for creation, `Modal id=12` for editing, driven by a `mode: "create" | "edit"` prop. `mode` decides
three things and nothing else — which fields are `disabled`, whether the `Info` hints render, and
whether serials are a one-per-line textarea or a single `Número de Serie` input. Both shells write
into the same `IAssetForm` shape, held in two separate `$state` objects.

**Rationale** — A Modal rather than reusing the side layer: the detail panel stays on screen behind
it, so the numbers being edited and the ledger they produced are both visible, and the create form
is not disturbed mid-typing. The `Info` boxes are onboarding for a form that has not been submitted
— "el activo debe registrarse como material" is advice about a decision already made, and the
donated-asset note explains a choice that is already fixed — so `edit` drops all three. Cost: two
form objects on the page instead of one, and `IAssetForm` is the union of both payloads, so it
carries the locked fields (Material, Almacén, Proveedor, Moneda) that the edit handler ignores.

## The edit dialog names its own focus target

**Context** — `Modal` focuses the dialog's first enabled control on open. In edit mode the first
enabled field is Fecha de Adquisición — Material and Almacén are locked — and `DateInput` opens
its calendar on focus, so the dialog appeared with its own form half-covered. The same trap the
payment dialog's field order works around, one entry below.

**Decision** — `Valor en Libros` carries `focusOnOpen={isEdit}`. That prop is new: `Modal`'s
`[autofocus]` preference had never worked (one comma-separated `querySelector` returns document
order, not selector priority), so fixing it was part of this change — see genix-ui's `RATIONALE.md`.

**Rationale** — Preferred over reordering the fields in edit mode, which was the alternative: the
edit form is meant to read as the same form the asset was created with, and a conditional field
order would break that to work around a focus rule. Naming the target is also more honest about
intent — the book value is the field an edit is usually about.

## Only five fields are editable, and the rest render locked rather than hidden

**Context** — Almost everything on an asset is derived from or pointed at by something else: the
material is the catalog row, the warehouse is where the stock is, the currency is what payments are
matched against, the lot size is how many units the stock movement carried.

**Decision** — `PurchaseAmount`, `AcquisitionValue`, `SerialNumber`, `AcquisitionDate` and `DueDate`
are writable. Material, Almacén, Proveedor, Moneda and Cantidad render `disabled`, in place.

**Rationale** — Locked and visible beats hidden: the user is looking at the asset, and a form that
drops half its fields on edit reads as a different asset. A warehouse change already has its own
path (`PUT.asset-transfer`), which moves the stock; letting this form write `WarehouseID` would
change where the asset says it is without moving anything.

## The edit form works in row totals, the create form in per-unit values

**Context** — `PostAsset` multiplies the per-unit `AcquisitionValue` and `PurchaseAmount` by the
quantity, so a lot of five stores the total. The detail panel displays that total.

**Decision** — `assetEditForm(asset)` seeds the edit form straight from the columns, and the labels
drop their "(por unidad)" suffix in `edit` mode. `PUT.asset` writes the values through unchanged.

**Rationale** — Dividing a stored total by the quantity does not round-trip: a lot of three at 1000
has no exact per-unit figure, and the edit would silently change the total it was meant to preserve.
The suffix change is what keeps the two modes from reading as the same field.


## The acquisition form points at supplies & materials when the material is missing

**Context** — The Material selector only lists supplies whose `DepreciationMonths > 0`, so a user
looking for an asset that was never created as a material finds an empty dropdown with no
explanation and no way forward from this page.

**Decision** — An `Info` box next to the selector states that an asset must exist as a material
first, with "Regístrelo aqui" linking to `/production/supplies-materials`. The two existing hints
on the layer — the donated-asset note and the serials note — moved to `Info` as well, so every
advisory line on the form reads the same.

**Rationale** — Cheaper than duplicating material creation inside the acquisition layer, and it
keeps the single creation path for supplies. Cost: the user leaves the half-filled acquisition
form behind, since the layer state is not preserved across navigation.

## The payment form is a Modal, and it leads with the amount

**Context** — The payment fields sat inline in the asset detail panel, under a "Compra" heading,
with the Dispose button on its own line below. The panel is mostly a read surface — balances,
depreciation ledger — so a live form in the middle of it competed with the numbers it changes.

**Decision** — Payment moved into `Modal id=11`, opened by a button that now shares a row with
Dispose. The "Compra" subtitle is gone. Inside the dialog the amount input comes first, not the
cash-register select.

**Rationale** — The field order is load-bearing, not cosmetic: `Modal.focusDialogContent()` focuses
the dialog's first `input`, and `SearchSelect` opens its dropdown on focus, so leading with the
register covered the date field with the option list. Leading with the amount also lands focus on
the field users actually edit, since it opens pre-filled with the outstanding balance. The dialog
carries `css="min-h-0!"` because `Modal` hard-codes `min-h-460`, which left ~300px of dead space
under a two-row form.

## The depreciation period label is composed here, not read off the row

**Context** — The ledger rows arrive from `expenses` with no `Name`: the backend stopped storing
`"Depreciación 7/60"` because a persisted label cannot be translated.

**Decision** — `depreciationSchedule(entries, totalMonths)` in `assets.ts` sorts the rows by
`Date` and numbers them, returning a `Depreciation n/N|Depreciación n/N` label per row.

**Rationale** — Sorting on `Date` rather than trusting arrival order means the number refers
to the period, not to when the row happened to be inserted — a re-run that interleaved entries
would still number them correctly. It costs one array copy per render of the panel.

## Serials decide granularity, so the form asks for them instead of asking the question

**Context** — Buying five identical laptops is either five assets or one. Both are legitimate: it
depends on whether the business tracks them individually.

**Decision** — The acquisition form has a Quantity field and a Serial Numbers textarea, one serial
per line. Entering serials produces one asset per serial; leaving it empty produces one grouped
asset with the quantity on it.

**Rationale** — The user already knows whether their laptops have asset tags. Asking "track these
individually?" as a separate checkbox is asking them to answer the same question twice, and lets the
two answers disagree. The presence of serials *is* the answer, and it matches the stock layer, where
a serial is exactly what promotes a unit to its own `ProductStockDetail` row.

## The page runs depreciation on load

**Context** — Depreciation is generated lazily by `POST.asset-depreciation-run`. Something has to
call it or the register shows stale book values.

**Decision** — An `$effect` calls it on mount, refetches the assets if anything was posted, and
otherwise says nothing. Failures log to the console rather than raising a toast.

**Rationale** — The run is idempotent — the backend filters each period against the asset's own
watermark — so calling it on every visit is safe, and it removes the need for a cron. A failure here
must not block the page: the register is still readable with values that are merely a month behind,
and a red toast on every load would be worse than a slightly stale number.

## Pure math lives in `assets.ts`, not in the service

**Context** — Book value, percentage depreciated and remaining months are needed by the table, the
detail panel and (eventually) the balance sheet.

**Decision** — They are plain functions in `assets.ts` with no Svelte and no fetch; `assets.svelte.ts`
holds only the reactive service and the API calls.

**Rationale** — The project convention, and it pays off here specifically: remaining months is
derived from accumulated depreciation rather than from the calendar, so it agrees with the ledger
even before a run has happened today. That rule is worth being able to test without a component.
