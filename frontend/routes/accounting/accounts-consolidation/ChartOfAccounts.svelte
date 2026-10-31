<script lang="ts">
import VTable from '#components/vTable/VTable.svelte'
import type { ITableColumn } from '#components/vTable/types.ts'
import { tr } from '#core/store.svelte.ts'
import { findPCGEAccount } from '#core/pcge.ts'
import { formatN } from '#libs/helpers.ts'
import { buildTrialBalance, type IAccountingOperation, type ITrialBalanceRow } from './accounts-consolidation'

// Tab Plan de cuentas: the PCGE accounts the period moved, as a tree, with their Debe, Haber and
// saldo — the period's balance de comprobación. Read-only.
interface Props {
  operations: IAccountingOperation[]
}

const { operations }: Props = $props()

const trialBalanceRows = $derived(buildTrialBalance(operations))
const formatMoney = (cents: number) => cents ? formatN(cents / 100, 2) : ""

const columns: ITableColumn<ITrialBalanceRow>[] = [
  {
    header: "Account|Cuenta", width: "minmax(260px, 2fr)",
    // A grouping account reads in bold; each digit below the 2-digit level indents one step.
    render: row => {
      if (!row.AccountCode) { return `<span class="text-red-fg">— ${tr("Unclassified|Sin clasificar")}</span>` }
      const accountName = findPCGEAccount(row.AccountCode)?.name || ""
      const text = `<span class="ff-mono">${row.AccountCode}</span> ${accountName}`
      return `<span style="padding-left: ${row.Depth * 16}px">${row.IsPostable ? text : `<span class="font-bold">${text}</span>`}</span>`
    },
  },
  { header: "Debit|Debe", width: "120px", align: "right", css: "ff-mono", getValue: row => formatMoney(row.Debit) },
  { header: "Credit|Haber", width: "120px", align: "right", css: "ff-mono", getValue: row => formatMoney(row.Credit) },
  {
    header: "Balance|Saldo", width: "130px", align: "right", css: "ff-mono",
    getValue: row => row.Balance ? formatN(row.Balance / 100, 2) : "",
  },
]
</script>

<VTable data={trialBalanceRows} {columns}
  maxHeight="calc(100vh - var(--header-height) - 200px)"
  emptyMessage="No movements in the period.|No hay movimientos en el periodo."
/>
