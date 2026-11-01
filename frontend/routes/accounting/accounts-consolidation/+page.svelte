<script lang="ts">
import { useUI } from '@genix/ui'
import { onMount } from 'svelte'
import Button from '#components/buttons/Button.svelte'
import FilterInput from '#components/form/FilterInput.svelte'
import SearchSelect from '#components/form/SearchSelect.svelte'
import SelectBox from '#components/form/SelectBox.svelte'
import T from '#components/misc/T.svelte'
import Page from '#domain/Page.svelte'
import { formatN } from '#libs/helpers.ts'
import { buildPeriodOptions, periodCode, SINGLE_DAY_PERIOD } from '../books/books'
import { STATUS_VIEW_ALL, statusViewOptions, summarizeOperations } from './accounts-consolidation'
import { AccountingPeriodReader } from './accounts-consolidation.svelte'
import ChartOfAccounts from './ChartOfAccounts.svelte'
import JournalTable from './JournalTable.svelte'
import OperationsTable from './OperationsTable.svelte'

// Shell: one period, read once, shown three ways — the operations to classify, the Diario de
// gestión, and the period's balance per account. Every tab reads the same built operations.
const ui = useUI()
const periodReader = new AccountingPeriodReader()

const pageOptions = [
  { id: 1, name: "Operations|Movimientos" },
  { id: 2, name: "Management journal|Diario de gestión" },
  { id: 3, name: "Chart of accounts|Plan de cuentas" },
]

const today = new Date()
// A month is the unit the page classifies, so the books' single-day option is left out.
const periodOptions = buildPeriodOptions(today).filter(option => option.ID !== SINGLE_DAY_PERIOD)
let selection = $state({ period: periodCode(today.getFullYear(), today.getMonth()) })
// The tables' filters sit in this top row, beside the period: the status view (Movimientos) and
// the search text, shared by Movimientos and the Diario.
let operationsFilter = $state({ statusView: STATUS_VIEW_ALL, filterText: "" })

const summary = $derived(summarizeOperations(periodReader.operations))

onMount(() => { periodReader.read(selection.period) })
</script>

<Page title="Consolidation & Accounts|Consolidado & Cuentas" options={pageOptions}>
  <div class="flex flex-wrap items-end gap-8 mb-8">
    <SearchSelect label="Period|Periodo" css="w-190" keyId="ID" keyName="Name"
      options={periodOptions} saveOn={selection} save="period" />
    <Button name="Read|Consultar" color="blue" icon="icon-[fa--search]"
      onClick={() => periodReader.read(selection.period)} />
    {#if ui.state.pageOptionSelected === 1}
      <SelectBox label="Show|Mostrar" css="w-190 ml-8" keyId="ID" keyName="Name"
        options={statusViewOptions} saveOn={operationsFilter} save="statusView" />
    {/if}
    {#if ui.state.pageOptionSelected === 1 || ui.state.pageOptionSelected === 2}
      <FilterInput css="w-220" icon="icon-[fa--search]" bind:value={operationsFilter.filterText} />
    {/if}
  </div>

  {#if periodReader.operationsService.response}
    <!-- How accounting-aligned the period is, by count and by amount (PLAN.md §6). -->
    <div class="flex flex-wrap items-center gap-x-20 gap-y-4 mb-8 text-sm" aria-label="Period classification status">
      <div class="text-green-fg">
        <T text="Classified|Clasificadas" />: <span class="font-semibold">{summary.classified}</span>
        ({summary.count ? Math.round(summary.classified * 100 / summary.count) : 0}%)
      </div>
      <div class="text-red-fg"><T text="Unclassified|Sin clasificar" />: <span class="font-semibold">{summary.unclassified}</span></div>
      <div class="text-yellow-fg"><T text="Incomplete|Incompletas" />: <span class="font-semibold">{summary.incomplete}</span></div>
      <!-- Only the gap between the sides matters: a balanced period is a green check. -->
      <div class="flex items-center gap-6">
        <T text="Debit − Credit|Debe − Haber" />:
        {#if summary.debitTotal === summary.creditTotal}
          <i class="icon-[fa--check] text-green-fg"></i>
        {:else}
          <span class="ff-mono text-red-fg">{formatN((summary.debitTotal - summary.creditTotal) / 100, 2)}</span>
        {/if}
      </div>
      <div class="text-fg-muted">
        <T text="No comprobante|Sin comprobante" />: {summary.undocumentedCount} · S/ {formatN(summary.undocumentedAmount / 100, 2)}
      </div>
    </div>
  {/if}

  {#if ui.state.pageOptionSelected === 1}
    <OperationsTable operationsService={periodReader.operationsService} sources={periodReader.sources}
      operations={periodReader.operations}
      statusView={operationsFilter.statusView} filterText={operationsFilter.filterText} />
  {/if}
  {#if ui.state.pageOptionSelected === 2}
    <JournalTable operations={periodReader.operations} filterText={operationsFilter.filterText} />
  {/if}
  {#if ui.state.pageOptionSelected === 3}
    <ChartOfAccounts operations={periodReader.operations} />
  {/if}
</Page>
