<script lang="ts">
  import { untrack } from 'svelte'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { ProductionWorkersService } from '#services/production/production-settings.svelte.ts'
  import { PRODUCTION_ORDER_STATUS, type IProductionLaborEntry, type IProductionOrder } from '#services/production/production-orders.ts'
  import { ProductionLaborEntriesService, postProductionLaborEntry } from '#services/production/production-orders.svelte.ts'
  import { summarizeOrderLabor } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // The "Mano de obra" tab (docs/MANUFACTURING_PLAN.md §9.2): the hours each worker spent on the order,
  // of one labor type per entry. The server values every entry at the worker's real cost per hour
  // (capped by their monthly payroll cost), and the order's next receipt absorbs it.
  let { order, catalogs }: { order: IProductionOrder, catalogs: ProductionCatalogs } = $props()

  // The layer mounts this tab per order ({#key order.ID}), so the order's ID is fixed for its life.
  const entries = untrack(() => new ProductionLaborEntriesService(order.ID))
  const employees = new ProductionWorkersService(true)
  // Hours are logged on released and in-progress orders only: a finished order's are all absorbed.
  const canLog = $derived(order.ss === PRODUCTION_ORDER_STATUS.RELEASED || order.ss === PRODUCTION_ORDER_STATUS.IN_PROGRESS)
  // The row being typed, before it is saved.
  let newEntry = $state<IProductionLaborEntry>(makeNewEntry())

  function makeNewEntry(): IProductionLaborEntry {
    return { ID: 0, OrderID: order.ID, EmployeeID: 0, LaborTypeID: 0, Date: todayUnixDay(), Period: 0,
      Minutes: 0, CostPerHour: 0, Amount: 0, ss: 1, upd: 0 }
  }

  const rows = $derived(canLog ? [newEntry, ...entries.records] : entries.records)
  const employeeOptions = $derived((employees.records || []).map((employee) => ({
    ID: employee.ID, Name: `${employee.FirstName} ${employee.LastName}`.trim(),
  })))
  const employeeNameOf = (employeeID: number) => employeeOptions.find((option) => option.ID === employeeID)?.Name || `#${employeeID}`
  const formatCents = (cents: number) => formatN((cents || 0) / 100, 2)
  // Hours are logged for the last month: the grid has no date cell, so the day is picked from a list.
  const dayOptions = Array.from({ length: 31 }, (_, daysBack) => {
    const unixDay = todayUnixDay() - daysBack
    return { ID: unixDay, Name: String(formatTime(unixDay, 'd-M-Y')) }
  })

  const saveEntry = async (entry: IProductionLaborEntry, isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete these hours?|¿Eliminar estas horas?') }))) return
    if (!isDelete && (!entry.EmployeeID || !entry.LaborTypeID || !(entry.Minutes > 0) || !entry.Date)) {
      notifyFailure(tr('Pick the worker, the labor type, the date and the hours.|Elija el trabajador, el tipo de trabajo, la fecha y las horas.'))
      return
    }
    if (!catalogs.orders) return
    showLoading(tr('Saving hours...|Guardando horas...'))
    try {
      await postProductionLaborEntry(catalogs.orders, entries, { ...$state.snapshot(entry), OrderID: order.ID, ss: isDelete ? 0 : 1 })
      if (entry === newEntry) newEntry = makeNewEntry()
    } catch (saveError) {
      console.error('[production-orders] labor entry error', { entry: $state.snapshot(entry), saveError })
      notifyFailure(saveError)
    }
    hideLoading()
  }

  const columns = $derived<ITableColumn<IProductionLaborEntry>[]>([
    {
      id: 'employee', header: 'Worker|Trabajador', css: 'px-4', field: 'EmployeeID',
      getCellOptions: () => employeeOptions, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      getValue: (entry) => entry.EmployeeID ? employeeNameOf(entry.EmployeeID) : '',
      disableCellInteractions: () => !canLog,
      onCellSelect: (entry, value) => { entry.EmployeeID = Number(value) },
    },
    {
      id: 'laborType', header: 'Labor type|Tipo de trabajo', width: '170px', css: 'px-4', field: 'LaborTypeID',
      getCellOptions: () => catalogs.laborTypes.records, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      disableCellInteractions: () => !canLog,
      onCellSelect: (entry, value) => { entry.LaborTypeID = Number(value) },
    },
    {
      id: 'date', header: 'Date|Fecha', width: '130px', css: 'px-4', field: 'Date',
      getCellOptions: () => dayOptions, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      getValue: (entry) => entry.Date ? String(formatTime(entry.Date, 'd-M-Y')) : '',
      disableCellInteractions: () => !canLog,
      onCellSelect: (entry, value) => { entry.Date = Number(value) },
    },
    {
      id: 'hours', header: 'Hours|Horas', width: '90px', align: 'right', cellInputType: 'number',
      getValue: (entry) => entry.Minutes ? formatN(entry.Minutes / 60, 2) : '',
      disableCellInteractions: () => !canLog,
      onCellEdit: (entry, value) => { entry.Minutes = Math.round((parseFloat(String(value || '0')) || 0) * 60) },
    },
    { header: 'Cost / hour|Costo / hora', width: '110px', align: 'right', getValue: (entry) => entry.ID ? formatCents(entry.CostPerHour) : '' },
    { header: 'Amount|Monto', width: '110px', align: 'right', getValue: (entry) => entry.ID ? formatCents(entry.Amount) : '' },
    ...(canLog ? [{ id: 'actions', header: '', width: '84px', getValue: () => '' }] : []),
  ])

  const laborColumns: ITableColumn<ReturnType<typeof summarizeOrderLabor>[number]>[] = [
    { header: 'Labor type|Tipo de trabajo', css: 'px-6', getValue: (labor) => catalogs.laborTypes.recordsMap.get(labor.LaborTypeID)?.Name || `#${labor.LaborTypeID}` },
    { header: 'Planned h|Horas plan.', width: '110px', align: 'right', getValue: (labor) => formatN(labor.PlannedMinutes / 60, 1) },
    { header: 'Logged h|Horas reg.', width: '110px', align: 'right', getValue: (labor) => formatN((labor.RecordedMinutes || 0) / 60, 1) },
    { header: 'Planned cost|Costo plan.', width: '120px', align: 'right', getValue: (labor) => formatCents(labor.PlannedCost) },
    { header: 'Booked cost|Costo cargado', width: '120px', align: 'right', getValue: (labor) => formatCents(labor.RecordedCost || 0) },
  ]
</script>

<div class="mt-8" aria-label="Planned vs logged labor per labor type">
  <VTable columns={laborColumns} data={summarizeOrderLabor(order.Labor || [])} disableVirtualizer={true} estimateSize={36}
    emptyMessage="The recipes plan no labor.|Las recetas no planifican mano de obra." />
</div>

<div class="flex items-center justify-between mt-16 mb-6">
  <h3 class="font-bold">{tr('Hours logged|Horas registradas')}</h3>
  {#if !canLog}
    <span class="text-fg-muted">{tr('Hours are logged on released or in-progress orders.|Las horas se registran en órdenes liberadas o en proceso.')}</span>
  {/if}
</div>
<div aria-label="Labor entries of the order">
  <VTable {columns} data={rows} disableVirtualizer={true} estimateSize={42} emptyMessage="No hours yet.|Aún no hay horas.">
    {#snippet cellRenderer(entry, column, defaultContent)}
      {#if column.id === 'actions'}
        <div class="flex justify-center gap-4">
          <Button color="green" mode="subtle" useCircle height="tiny" icon="icon-[fa--check]" label="Saves this row of hours."
            onClick={() => saveEntry(entry)} />
          {#if entry.ID > 0}
            <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Deletes this row of hours."
              onClick={() => saveEntry(entry, true)} />
          {/if}
        </div>
      {:else}
        {defaultContent}
      {/if}
    {/snippet}
  </VTable>
</div>
