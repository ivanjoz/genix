<script lang="ts">
  import OrgTree from '#components/charts/OrgTree.svelte'
  import { tr } from '#core/store.svelte.ts'
  import type { EmployeesService, IEmployee } from '#services/hr/employees.svelte.ts'
  import type { OrganizationService } from '#services/hr/organization.svelte.ts'
  import { buildEmployeeTree, employeeFullName, showsDepartmentInOrgChart } from '#routes/hr/employees/employees.ts'

  // The org chart: active employees drawn from each one's immediate boss, laid out compactly by
  // genix-ui's OrgTree (people without reports stack in columns).
  let { organization, employees }: {
    organization: OrganizationService
    employees: EmployeesService
  } = $props()

  const tree = $derived(buildEmployeeTree(employees.records))
</script>

<!-- The department heads the card only where it changes from the boss's, so each area is named once at its top. -->
{#snippet employeeCard(employee: IEmployee, boss: IEmployee | undefined)}
  <div class="flex h-full flex-col overflow-hidden rounded-[10px] border border-slate-300 bg-white text-center shadow-sm"
    aria-label={`${employeeFullName(employee)} in the org chart.`}>
    {#if showsDepartmentInOrgChart(employee, boss)}
      <div class="truncate border-b border-slate-200 bg-indigo-50 px-10 py-3 text-[15px] text-indigo-700">
        {organization.DepartmentsMap.get(employee.DepartmentID)?.Name || ''}
      </div>
    {/if}
    <div class="flex flex-1 flex-col justify-center px-10">
      <div class="ff-bold truncate">{employeeFullName(employee)}</div>
      <div class="truncate text-[14px] text-gray-600">{organization.PositionsMap.get(employee.PositionID)?.Name || ''}</div>
    </div>
  </div>
{/snippet}

{#if tree.length === 0}
  <div class="text-gray-600">{tr('No employees yet: register them in Employees.|Aún no hay trabajadores: regístrelos en Personal.')}</div>
{:else}
  <div class="overflow-auto p-8 pb-16" aria-label="Org chart">
    <OrgTree roots={tree} card={employeeCard} cardWidth={240} cardHeight={84} css="mx-auto" />
  </div>
{/if}
