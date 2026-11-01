<script lang="ts">
import { tr } from '#core/store.svelte.ts'
import { formatN } from '#libs/helpers.ts'
import {
  accountRowName, STATEMENT_ROW_ACCOUNT, STATEMENT_ROW_BALANCE, STATEMENT_ROW_SECTION, STATEMENT_ROW_SUBTOTAL,
  type IStatementRow,
} from './financial-statements'

// One statement as a report: section headers with their totals, the accounts under them, and
// the running results. A negative amount (an expense, cash out) reads in parentheses.
interface Props {
  rows: IStatementRow[]
  ariaLabel: string
}

const { rows, ariaLabel }: Props = $props()

const formatAmount = (cents: number) => cents < 0 ? `(${formatN(-cents / 100, 2)})` : formatN(cents / 100, 2)
</script>

<table class="w-full max-w-760 text-sm" aria-label={ariaLabel}>
  <tbody>
    {#each rows as row, rowIndex (rowIndex)}
      {#if row.Kind === STATEMENT_ROW_ACCOUNT}
        <tr class="border-b border-line-soft">
          <td class="py-4 pl-24 {row.AccountCode ? '' : 'text-red-fg'}">
            {#if row.AccountCode}<span class="ff-mono">{row.AccountCode}</span>{/if}
            {tr(accountRowName(row.AccountCode))}
          </td>
          <td class="py-4 text-right ff-mono">{formatAmount(row.Amount)}</td>
        </tr>
      {:else if row.Kind === STATEMENT_ROW_SECTION}
        <tr class="border-b border-line">
          <td class="pt-12 pb-4 font-semibold">{tr(row.Label)}</td>
          <td class="pt-12 pb-4 text-right ff-mono font-semibold">{formatAmount(row.Amount)}</td>
        </tr>
      {:else if row.Kind === STATEMENT_ROW_SUBTOTAL || row.Kind === STATEMENT_ROW_BALANCE}
        <tr class="border-b-2 border-line {row.Kind === STATEMENT_ROW_BALANCE ? 'bg-surface-soft' : ''}">
          <td class="py-8 font-bold">{tr(row.Label)}</td>
          <td class="py-8 text-right ff-mono font-bold {row.Amount < 0 ? 'text-red-fg' : ''}">{formatAmount(row.Amount)}</td>
        </tr>
      {/if}
    {/each}
  </tbody>
</table>
