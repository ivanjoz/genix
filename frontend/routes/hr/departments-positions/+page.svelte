<script lang="ts">
  import Page from '#domain/Page.svelte'
  import { useUI } from '@genix/ui'
  import { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { OrganizationService } from '#services/hr/organization.svelte.ts'
  import OrgChartTab from './OrgChartTab.svelte'
  import OrganizationTab from './OrganizationTab.svelte'

  const ui = useUI()
  const employees = new EmployeesService(true)
  const organization = new OrganizationService(true)

  const VIEW_ORGANIZATION = 1
  const VIEW_ORG_CHART = 2
  const pageOptions = [
    { id: VIEW_ORGANIZATION, name: 'Departments & Positions|Áreas y Cargos' },
    { id: VIEW_ORG_CHART, name: 'Org chart|Organigrama' },
  ]
</script>

<Page title="Departments & Positions|Áreas y Cargos" options={pageOptions}>
  {#if ui.state.pageOptionSelected === VIEW_ORG_CHART}
    <OrgChartTab {organization} {employees} />
  {:else}
    <OrganizationTab {organization} {employees} />
  {/if}
</Page>
