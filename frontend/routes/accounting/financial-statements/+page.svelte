<script lang="ts">
import { useUI } from '@genix/ui'
import { onMount } from 'svelte'
import Button from '#components/buttons/Button.svelte'
import SearchSelect from '#components/form/SearchSelect.svelte'
import SelectBox from '#components/form/SelectBox.svelte'
import T from '#components/misc/T.svelte'
import Page from '#domain/Page.svelte'
import { summarizeOperations } from '../accounts-consolidation/accounts-consolidation'
import { AccountingPeriodReader } from '../accounts-consolidation/accounts-consolidation.svelte'
import { buildPeriodOptions, periodBounds, periodCode, SINGLE_DAY_PERIOD } from '../books/books'
import {
  buildCashFlow, buildIncomeStatement, operationsOfView, restateCashBalances, statementViewOptions, VIEW_MANAGEMENT,
} from './financial-statements'
import { CashBalancesService } from './financial-statements.svelte'
import StatementTable from './StatementTable.svelte'

// Shell: one period, read like Consolidado & Cuentas (the same entries), shown as the Estado de
// Resultados and the direct Flujo de Caja, over every entry (Gestión) or the official Diario's.
const ui = useUI()
const periodReader = new AccountingPeriodReader()
const cashBalancesService = new CashBalancesService()

const pageOptions = [
  { id: 1, name: "Income statement|Estado de resultados" },
  { id: 2, name: "Cash flow|Flujo de caja" },
]

const today = new Date()
const periodOptions = buildPeriodOptions(today).filter(option => option.ID !== SINGLE_DAY_PERIOD)
let selection = $state({ period: periodCode(today.getFullYear(), today.getMonth()), view: VIEW_MANAGEMENT })

const viewOperations = $derived(operationsOfView(periodReader.operations, selection.view))
// The statements leave out what is unclassified, and an incomplete entry may be missing amounts.
const summary = $derived(summarizeOperations(viewOperations))
const incomeStatementRows = $derived(buildIncomeStatement(viewOperations))
// The opening balance is restated at the rate of the day before the period, the closing at its last day's.
const restatedBalances = $derived.by(() => {
  const { response, loadedPeriod } = periodReader.operationsService
  if (!response || !cashBalancesService.balances) { return null }
  const { firstDay, lastDay } = periodBounds(loadedPeriod)
  return restateCashBalances(cashBalancesService.balances, periodReader.cashBanksService.Cajas,
    periodReader.sellRateOf(firstDay - 1), periodReader.sellRateOf(lastDay))
})
const cashFlowRows = $derived(buildCashFlow(viewOperations, restatedBalances))

const readPeriod = (period: string) => {
  periodReader.read(period)
  cashBalancesService.fetch(period)
}

onMount(() => { readPeriod(selection.period) })
</script>

<Page title="Financial Statements|Estados Financieros" options={pageOptions}>
  <div class="flex flex-wrap items-end gap-8 mb-8">
    <SearchSelect label="Period|Periodo" css="w-190" keyId="ID" keyName="Name"
      options={periodOptions} saveOn={selection} save="period" />
    <Button name="Read|Consultar" color="blue" icon="icon-[fa--search]"
      onClick={() => readPeriod(selection.period)} />
    <SelectBox label="Entries|Asientos" css="w-190 ml-8" keyId="ID" keyName="Name"
      options={statementViewOptions} saveOn={selection} save="view" />
  </div>

  {#if periodReader.operationsService.response}
    {#if summary.unclassified || summary.incomplete}
      <div class="mb-8 text-sm text-yellow-fg" role="status">
        <i class="icon-[fa--exclamation-triangle]"></i>
        <T text="The income statement leaves out the unclassified lines (the cash flow lists them as Unclassified), and incomplete entries may be missing amounts. Review them in Consolidado & Cuentas.|El estado de resultados deja fuera las líneas sin clasificar (el flujo de caja las lista como Sin clasificar), y las operaciones incompletas pueden tener montos faltantes. Revíselas en Consolidado & Cuentas." />
        (<T text="Unclassified|Sin clasificar" />: {summary.unclassified} · <T text="Incomplete|Incompletas" />: {summary.incomplete})
      </div>
    {/if}

    {#if ui.state.pageOptionSelected === 1}
      <StatementTable rows={incomeStatementRows} ariaLabel="Income statement" />
    {/if}
    {#if ui.state.pageOptionSelected === 2}
      {#if restatedBalances?.IsMissingRate}
        <div class="mb-8 text-sm text-red-fg" role="status">
          <T text="A dollar cash bank has no exchange rate: its balance counts as 0.|Una caja en dólares no tiene tipo de cambio: su saldo cuenta como 0." />
        </div>
      {/if}
      <StatementTable rows={cashFlowRows} ariaLabel="Cash flow" />
    {/if}
  {/if}
</Page>
