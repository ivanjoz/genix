# RATIONALE — purchase-management

Design decisions for the supply (abastecimiento) page, newest first.

## The import preview renders in VTable, not TableGrid

**Context** — the Excel sheet groups four sub-columns under each supplier, and the preview modal
should show the rows the way the file does. The Productos import preview uses `TableGrid`, the
obvious thing to copy.

**Decision** — the preview uses `VTable`, which is the only one of the two that renders `subcols`
as a second header row. `TableGrid` ignores the field entirely.

**Rationale** — with `TableGrid` the twelve provider cells would have flattened into twelve
unrelated columns, and "Proveedor / Capacidad / Entrega / Precio" repeated three times reads as
noise without the group header above it. `VTable` is virtualized too, so a 2000-row preview costs
the same either way.

## One column definition drives export, import and preview

**Context** — the sheet shape, the import mapping and the preview table are three views of the same
columns, and the importer matches sheet headers against column headers by normalized name.

**Decision** — `makeSupplyExcelColumns(providerGroupCount)` returns the single definition all three
consume. The count differs per use: export takes the widest product's supplier count (minimum 3),
import takes that plus three spare groups, and the preview takes whatever the parsed rows needed.

**Rationale** — a separate import mapping would drift from the export headers and break the
round-trip silently, since an unmatched header is reported as ignored rather than as an error. The
spare groups on import exist so a sheet somebody widened by hand still parses in full instead of
quietly dropping the extra suppliers.

## A row's suppliers are positional, and the sheet is authoritative

**Context** — the sheet exposes suppliers as ordered slots, but `ProviderSupply` is a list. The
import has to decide what "changed" means for it.

**Decision** — the diff compares slot by slot, so moving a supplier from column 2 to column 1 counts
as a change. The imported list replaces the saved one outright: blanking a supplier's cells removes
it from the product.

**Rationale** — positional comparison is what the person editing the file sees; a set comparison
would call a reordered row unchanged and then silently persist the new order anyway. Replace rather
than merge because there is no other way to express a deletion in a spreadsheet — an additive import
would make removing a supplier impossible from the file.

## An ambiguous name is an error, not a best guess

**Context** — products and suppliers are resolved from the sheet by `normalizeStringN`, which
collapses case and accents. Two catalog entries can normalize to the same string.

**Decision** — the name index marks such a name as ambiguous and every row using it fails with
"hay más de un producto llamado …" instead of resolving to either candidate.

**Rationale** — picking the first match writes a replenishment policy onto the wrong product, and
nothing downstream would ever flag it. The failure is noisy and the fix (rename one of them) is
obvious.

## Export follows the page's filter

**Context** — the toolbar's search box filters the table; the export button sits next to it.

**Decision** — the export writes the filtered rows, so searching "néctar" and exporting produces a
sheet of just those products.

**Rationale** — exporting the full 10k catalog when the screen shows 40 rows contradicts what the
button appears to do. Clearing the filter is the way to get everything, and that is already how the
record counter beside it behaves.
