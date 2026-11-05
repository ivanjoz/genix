<script lang="ts">
  import Page from '#domain/Page.svelte'
  import { useUI } from '@genix/ui'
  import { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { OrganizationService } from '#services/hr/organization.svelte.ts'
  import { PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import AdvancesTab from './AdvancesTab.svelte'
  import RunsTab from './RunsTab.svelte'

  const ui = useUI()
  const config = new PayrollConfigService(true)
  const employees = new EmployeesService(true)
  const organization = new OrganizationService(true)

  const pageOptions = [
    { id: 1, name: 'Payroll|Planilla' },
    { id: 2, name: 'Weekly payments|Pagos semanales' },
  ]
</script>

<!-- The payroll tab is edge to edge: a gray months column at full height beside a white detail. -->
<Page title="Payroll|Planilla" options={pageOptions} background="white"
  noPadding={ui.state.pageOptionSelected !== 2} fixedFullHeight={ui.state.pageOptionSelected !== 2}>
  {#if ui.state.pageOptionSelected === 2}
    <AdvancesTab {employees} />
  {:else}
    <RunsTab {config} {employees} {organization} />
  {/if}
</Page>
