<script lang="ts">
import VTable from '#components/vTable/VTable.svelte'
import type { ITableColumn } from '#components/vTable/types.ts'
import { tr } from '#core/store.svelte.ts'
import { formatTime } from '#libs/helpers.ts'
import { entrySides, type IAccountingOperation, type IGroupAccountLine } from './accounts-consolidation'
import { accountLinesCell, amountLinesCell, ENTRY_TABLE_CELL_CSS } from './accounts-consolidation.cells'

// Tab Diario de gestión: every entry of the period, one row per operation, its Debe and Haber
// side by side as in Movimientos — with and without comprobante. The official Diario is these
// same entries filtered by InOfficialBook (phase 6). The search text is set in the page's top row.
interface Props {
  operations: IAccountingOperation[]
  filterText: string
}

const { operations, filterText }: Props = $props()

interface IJournalEntry {
  operation: IAccountingOperation
  DebitLines: IGroupAccountLine[]
  CreditLines: IGroupAccountLine[]
}
const journalEntries = $derived(operations.map((operation): IJournalEntry => ({ operation, ...entrySides(operation) })))

const columns: ITableColumn<IJournalEntry>[] = [
  { header: "Date|Fecha", width: "70px", getValue: entry => formatTime(entry.operation.Date, "M-d") as string },
  { header: "CUO", width: "150px", css: "ff-mono", getValue: entry => entry.operation.Key },
  {
    header: "Description|Glosa", width: "minmax(160px, 1.2fr)",
    render: entry => entry.operation.InOfficialBook ? tr(entry.operation.Description)
      : `${tr(entry.operation.Description)} <span class="px-6 rounded bg-yellow-bg text-yellow-fg">${tr("no comprobante|sin comprobante")}</span>`,
  },
  { header: "Doc. Nº|Nº Doc", width: "120px", css: "ff-mono", getValue: entry => entry.operation.DocumentLabel },
  {
    header: "Debit|Debe", subcols: [
      { header: "Account|Cuenta", render: entry => accountLinesCell(entry.DebitLines, entry.operation.Status, [entry.operation]) },
      { header: "Amount|Monto", align: "right", css: "ff-mono", render: entry => amountLinesCell(entry.DebitLines) },
    ],
  },
  {
    header: "Credit|Haber", subcols: [
      { header: "Account|Cuenta", render: entry => accountLinesCell(entry.CreditLines, entry.operation.Status, [entry.operation]) },
      { header: "Amount|Monto", align: "right", css: "ff-mono", render: entry => amountLinesCell(entry.CreditLines) },
    ],
  },
]
</script>

<VTable data={journalEntries} {columns}
  cellCss={ENTRY_TABLE_CELL_CSS}
  {filterText}
  getFilterContent={(entry) => `${entry.operation.Key} ${tr(entry.operation.Description)} ${entry.operation.DocumentLabel} ${
    [...entry.DebitLines, ...entry.CreditLines].map(line => line.AccountCode).join(" ")}`}
  maxHeight="calc(100vh - var(--header-height) - 108px)"
  emptyMessage="No entries in the period.|No hay asientos en el periodo."
/>
