<script lang="ts">
import Page from '$domain/Page.svelte'
import { Core } from '$core/store.svelte'
import ExpensesRegister from './ExpensesRegister.svelte'
import ExpensesSchedule from './ExpensesSchedule.svelte'

const pageOptions = [
  { id: 1, name: "Register|Registro" },
  { id: 2, name: "Scheduled|Programados" },
]
</script>

<Page title="Expenses|Gastos" options={pageOptions}>
  {#if Core.pageOptionSelected === 1}
    <ExpensesRegister />
  {/if}
  {#if Core.pageOptionSelected === 2}
    <ExpensesSchedule />
  {/if}
</Page>
