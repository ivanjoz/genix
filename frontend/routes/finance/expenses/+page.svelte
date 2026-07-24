<script lang="ts">
import Page from '$domain/Page.svelte'
import { useUI } from '@genix/ui'
import ExpensesRegister from './ExpensesRegister.svelte'
import ExpensesSchedule from './ExpensesSchedule.svelte'

const pageOptions = [
  { id: 1, name: "Register|Registro" },
  { id: 2, name: "Scheduled|Programados" },
]
const ui = useUI()
</script>

<Page title="Expenses|Gastos" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 1}
    <ExpensesRegister />
  {/if}
  {#if ui.state.pageOptionSelected === 2}
    <ExpensesSchedule />
  {/if}
</Page>
