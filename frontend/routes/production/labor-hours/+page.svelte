<script lang="ts">
  import { onMount } from 'svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import LoadingBar from '#components/misc/LoadingBar.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { formatN } from '#libs/helpers.ts'
  import { employeeFullName } from '#routes/hr/employees/employees.ts'
  import { formatPeriod, periodOfDay, periodOptionsAround, previousPeriod } from '#routes/hr/payroll/payroll.ts'
  import type { ILaborTypeMinutes, IOffPayrollWorkerRow, IPayrollWorkerRow } from './labor-hours.ts'
  import { LaborHoursReader } from './labor-hours.svelte.ts'

  // Shell of "Horas vs planilla" (docs/MANUFACTURING_PLAN.md §9.3): one month, the Planilla workers
  // compared with their payslips, and the workers off payroll apart (§9.5).
  const reader = new LaborHoursReader()
  const currentPeriod = periodOfDay(Math.floor(Date.now() / 86400000))
  // A report of a finished month: default the previous one, never a future one.
  const periodOptions = periodOptionsAround(currentPeriod).filter((option) => option.id <= currentPeriod)
  const selection = $state({ period: previousPeriod(currentPeriod) })

  onMount(() => { void reader.read(selection.period) })

  const formatAmount = (cents: number | null) => cents === null ? '' : `S/ ${formatN(cents / 100, 2)}`
  const formatHours = (hours: number | null) => hours === null ? '' : formatN(hours, 1)
  const employeeName = (employeeID: number) => {
    if (!employeeID) return tr('Total|Total')
    const employee = reader.employeesService.recordsMap.get(employeeID)
    return employee ? employeeFullName(employee) : `#${employeeID}`
  }
  const laborTypeSplit = (minutesByLaborType: ILaborTypeMinutes[]) => minutesByLaborType
    .map(({ LaborTypeID, Minutes }) =>
      `${reader.laborTypesService.recordsMap.get(LaborTypeID)?.Name || `#${LaborTypeID}`} ${formatN(Minutes / 60, 1)}`)
    .join(' · ')
  const totalsRowCss = (row: { EmployeeID: number }) => row.EmployeeID === 0 ? 'ff-bold bg-surface-soft' : undefined

  const payrollColumns: ITableColumn<IPayrollWorkerRow>[] = $derived.by(() => {
    const isPayrollPending = !!reader.report?.isPayrollPending
    return [
      { header: 'Employee|Trabajador', css: 'px-6', setCellCss: totalsRowCss, getValue: (row) => employeeName(row.EmployeeID) },
      { header: 'Hours logged|Horas registradas', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', setCellCss: totalsRowCss,
        getValue: (row) => formatHours(row.LoggedMinutes / 60) },
      { header: 'By labor type (h)|Por tipo de labor (h)', css: 'px-6', setCellCss: totalsRowCss,
        getValue: (row) => laborTypeSplit(row.MinutesByLaborType) },
      { header: 'Hours paid|Horas pagadas', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', hidden: isPayrollPending,
        setCellCss: totalsRowCss, getValue: (row) => row.IsMissingPayslip ? tr('No payslip|Sin boleta') : formatHours(row.PaidHours) },
      { header: 'Utilization|Utilización', headerCss: 'w-100', css: 'px-6 ff-mono', align: 'right', hidden: isPayrollPending,
        setCellCss: totalsRowCss, getValue: (row) => row.Utilization === null ? '—' : `${formatN(row.Utilization * 100, 0)}%` },
      { header: 'Idle hours|Horas ociosas', headerCss: 'w-100', css: 'px-6 ff-mono', align: 'right', hidden: isPayrollPending,
        setCellCss: totalsRowCss, getValue: (row) => formatHours(row.IdleHours) },
      { header: 'Labor booked|M.O. cargada', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: totalsRowCss,
        getValue: (row) => formatAmount(row.BookedAmount) },
      { header: 'Payroll cost|Costo planilla', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', hidden: isPayrollPending,
        setCellCss: totalsRowCss, getValue: (row) => formatAmount(row.ActualCost) },
      { header: 'Adjustment|Ajuste', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right', hidden: isPayrollPending,
        setCellCss: (row) => totalsRowCss(row) || (row.Adjustment ? 'text-red-fg' : undefined),
        getValue: (row) => formatAmount(row.Adjustment) },
    ]
  })

  const offPayrollColumns: ITableColumn<IOffPayrollWorkerRow>[] = [
    { header: 'Worker|Trabajador', css: 'px-6', setCellCss: totalsRowCss, getValue: (row) => employeeName(row.EmployeeID) },
    { header: 'Hours|Horas', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', setCellCss: totalsRowCss,
      getValue: (row) => formatHours(row.LoggedMinutes / 60) },
    { header: 'By labor type (h)|Por tipo de labor (h)', css: 'px-6', setCellCss: totalsRowCss,
      getValue: (row) => laborTypeSplit(row.MinutesByLaborType) },
    { header: 'Rate per hour|Tarifa por hora', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: totalsRowCss,
      getValue: (row) => row.EmployeeID ? formatAmount(row.CostPerHour) : '' },
    { header: 'Labor booked|M.O. cargada', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: totalsRowCss,
      getValue: (row) => formatAmount(row.BookedAmount) },
  ]
</script>

<Page title="Hours vs Payroll|Horas vs Planilla">
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-12">
      <SearchSelect label="Month|Mes" css="w-190" keyId="id" keyName="name" options={periodOptions}
        saveOn={selection} save="period" onChange={() => { void reader.read(selection.period) }} />
      {#if reader.report && !reader.report.isPayrollPending}
        <div class="ml-auto px-12 py-8 rounded-md bg-surface-soft" role="status"
          aria-label="Total labor adjustment of the month">
          <span class="text-fg-muted">{tr('Adjustment of the month (Dr 7111 · Cr 2111)|Ajuste del mes (Debe 7111 · Haber 2111)')}:</span>
          <span class="ml-6 ff-bold ff-mono">{formatAmount(reader.report.totalAdjustment)}</span>
        </div>
      {/if}
    </div>

    {#if !reader.report}
      <div class="p-8 min-h-240 w-full fx-c rounded-md bg-surface-soft">
        <LoadingBar label={tr('Loading...|Cargando...')} />
      </div>
    {:else}
      {@const report = reader.report}
      {#if report.isPayrollPending}
        <div class="mb-8 text-yellow-fg" role="status">
          <i class="icon-[fa--exclamation-triangle]"></i>
          {tr(`Payroll pending: ${formatPeriod(reader.period)} has no closed payroll yet, so only the hours and the labor booked are shown.`
            + `|Planilla pendiente: ${formatPeriod(reader.period)} aún no tiene planilla cerrada; solo se muestran las horas y la mano de obra cargada.`)}
        </div>
      {/if}

      <div class="h3 mb-6">{tr('Payroll workers|Trabajadores en planilla')}</div>
      <VTable columns={payrollColumns} disableVirtualizer={true} estimateSize={36}
        data={report.payrollRows.length ? [...report.payrollRows, report.payrollTotals] : []}
        emptyMessage="No payroll worker logged hours or works in production.|Ningún trabajador en planilla registró horas ni trabaja en producción." />

      {#if report.offPayrollRows.length}
        <div class="h3 mt-20 mb-6">{tr('Workers off payroll|Trabajadores sin planilla')}</div>
        <VTable columns={offPayrollColumns} disableVirtualizer={true} estimateSize={36}
          data={[...report.offPayrollRows, report.offPayrollTotals]} />
      {/if}
    {/if}
  </Layer>
</Page>
