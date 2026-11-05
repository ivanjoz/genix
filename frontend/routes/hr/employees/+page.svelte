<script lang="ts">
  import Page from '#domain/Page.svelte'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { OrganizationService } from '#services/hr/organization.svelte.ts'
  import { ContractsService } from '#services/hr/contracts.svelte.ts'
  import EmployeesTab from './EmployeesTab.svelte'

  /** backend/access.toml "Planilla": contracts carry salaries, so only its holders load them. */
  const ACCESS_ID_PAYROLL = 41

  const employees = new EmployeesService(true)
  const organization = new OrganizationService(true)
  const contracts = security.checkAcceso(ACCESS_ID_PAYROLL) ? new ContractsService(true) : undefined
</script>

<Page title="Employees|Personal">
  <EmployeesTab {employees} {organization} {contracts} />
</Page>
