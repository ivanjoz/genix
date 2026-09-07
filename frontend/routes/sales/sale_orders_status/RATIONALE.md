# RATIONALE — sale_orders_status

## Annulling a sale reverses the ledgers, it does not re-derive from Status
**Context** — Annulment has to give back the money, return the delivered stock, and take the
sale out of the day summary. The obvious implementation reads `Status` to decide what
happened (paid? delivered?) and rebuilds the reversal from the sale's own detail lines, which
is what `POST.sale-order` does in the forward direction.

**Decision** — `POST.sale-order-annul` reads the two ledgers instead, by `DocumentID = sale.ID`
(both `cash_bank_movements` and `warehouse_product_movement` index it), and nets every
reversal already recorded against what was originally posted. `netCashRefundAmount` and
`netStockReturns` in `backend/sales/sale_order_annul.go` are pure functions over those rows.
Only the summary rollback still reads `Status`, because the summary has no per-sale ledger to
read back.

**Rationale** — Three properties fall out of it that the re-derivation does not have. It
returns each unit to the exact lot, serial and presentation it left from, and to the warehouse
it *actually* left — `sale.WarehouseID` is editable after the fact, so it can name a different
one. It is idempotent: a retry after a partial failure gives back only what never left, which
matters because there are no transactions here and the alternative failure mode is refunding
twice. And it needs no `if paid / if delivered` ladder — a sale that was never paid simply has
no collection rows and the sum is zero. The cost is two extra queries per annulment.

## The status flips last
**Context** — The three reversals and the status write cannot be atomic.

**Decision** — Cash, stock and summary are applied first (in an `errgroup`), and
`Status = 0` is written only after all three succeed.

**Rationale** — A failure before the last step leaves the sale un-annulled and retryable, and
the netting above makes the retry safe. The reverse order would leave a sale marked void with
the money still taken — an inconsistency nobody can see from the UI.

## The summary rollback is the one step that cannot be made idempotent
**Context** — Cash and stock are netted against their ledgers, so re-running them is a no-op.
The day summary has no per-sale rows — only per-day, per-product totals — so nothing records
that a given sale was already taken out of it.

**Decision** — It runs sequentially *after* both ledger reversals have succeeded, and before
the status write. Not inside the `errgroup` with them.

**Rationale** — It removes the common failure mode: an annulment whose stock reversal fails no
longer subtracts the sale from the day totals on every retry. What is left is a narrow window —
a crash between the summary write and the status write — which can under-count a day. The
counters clamp at zero (`applySummaryChangeToStats`), so it can distort a report but never
invent stock or money. The proper fix is `SaleOrderReprocess`, which rebuilds a day from its
sales and already skips annulled ones, but its cron handler is currently a stub that returns
without calling it (`sale_summary_reprocess.go`), so there is no rebuild to fall back on.

## The operator chooses the refund cash bank; the amount comes from the ledger
**Context** — The money may not be able to leave the register that collected it: the sale may
have been paid days ago, or in cash at a register that is now short.

**Decision** — `RefundCashBankID` is a required field of the annul request when the sale
collected anything, and the panel offers the same active-cash-bank selector as the payment
panel, defaulted to the collecting one. The amount is never sent by the client.

**Rationale** — Destination is an operational choice; amount is not, and a client-supplied
amount would be a way to refund more than was ever paid. Idempotency survives the split
because `netCashRefundAmount` sums across cash banks — a refund paid from a different bank
still settles the sale. The backend rejects an inactive cash bank, which `ApplyCashBankMovement`
does not check on its own. There is deliberately no balance check: none exists for any other
outflow type, and adding one only here would be inconsistent.

## Annulling is a sub-access, and this is the first `HasSubAcceso` caller
**Context** — Annulment is the destructive action on this page. Access 11 "Gestión Ventas"
gates the whole route at one level, so anyone who can register a payment could also void a sale.

**Decision** — `sub_accesses_ids = [2]` / `["Anular Venta"]` on access 11, checked in the
handler with `req.User.HasSubAcceso` and mirrored in the UI with `security.checkSubAcceso` to
decide whether the button is offered.

**Rationale** — It is the mechanism the catalog already has for exactly this. Two consequences
worth knowing: this is the **first handler in the codebase to call `HasSubAcceso`**, so it sets
the pattern for the next one; and because an access lives in exactly one grant payload, every
existing profile holding "Gestión Ventas" must have "Anular Venta" (or "Todos") ticked before
anyone can annul — the capability is not inherited from the access.

## An invoiced sale cannot be annulled
**Context** — A sale with a live SUNAT document cannot simply disappear; the legal instrument
is a credit note, and nothing in this system emits one yet (`DocTypeCreditNote` exists as a
constant only).

**Decision** — The handler refuses, naming the document number. `FindBySaleOrder` moved from
`invoicing/emitter.go` to `invoicing/types/invoice_lookup.go` so `sales` can call it without a
module body importing another module body.

**Rationale** — Refusing is the only correct answer until credit notes exist. The move mirrors
why `ApplyCashBankMovement` lives in `finance/types`: the function touches only `db` and its
own tables, so it is a leaf and belongs where every module can reach it. The reverse guard
already existed — `invoice_api.go` refuses to invoice an annulled sale.

## `AnnulReason` is the only new column
**Context** — The annulment reason is mandatory and had nowhere to live. The obvious companion
fields are `AnnulledTime` and `AnnulledUser`.

**Decision** — Only `AnnulReason` was added to `sale_order`. The panel shows "Anulada el / por"
from `Updated` / `UpdatedBy`.

**Rationale** — Annulment is terminal: nothing writes to the row afterwards, so `Updated` and
`UpdatedBy` already *are* when and by whom. A second pair could only drift from the first.

## The Anuladas tab needed a presence check, not a truthiness check
**Context** — The tab groups map 1:1 to backend query params, and annulled is `Status = 0`.
`GetSaleOrders` rejected `order-status <= 0`, so the tab could not be expressed.

**Decision** — The backend reads `req.Query["order-status"]` for presence rather than testing
the parsed value, and the removal list is now built as "every status this tab does not show".

**Rationale** — Keeps one parameter for all status-pinned tabs instead of inventing an
`annulled=1` flag. The removal-list rewrite fixed a live bug on the way past: the Finalizadas
tab used to list `OrderStatusCompleted` in both the query and the removal set, telling the
client to delete the very rows it had just been sent. The client-side `ss > 0` filter in
`SaleOrdersService.handler` was removed for the same reason — it would have hidden every record
in the new tab — so `useCache.ver` was bumped to 11 to retire caches written under it.
