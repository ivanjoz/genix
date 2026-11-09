<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { todayUnixDay } from '#libs/helpers.ts'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import { employeeFullName } from '#routes/hr/employees/employees.ts'
  import type { EmployeesService } from '#services/hr/employees.svelte.ts'
  import type { OrganizationService } from '#services/hr/organization.svelte.ts'
  import {
    PayslipsService, postPayrollContributions, postPayrollEnsure, postPayrollRun, postPayrollTransition,
    type IContributionsPayment, type IPayrollEnsureResult, type IPayrollRun, type IPayslip, type PayrollConfigService,
    type PayrollTransition,
  } from '#services/hr/payroll.svelte.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import PayslipBoleta from './PayslipBoleta.svelte'
  import {
    CONTRIBUTION_ACCOUNTS, PROFIT_SHARING_RATE_OPTIONS, RUN_STATE_LABELS, allowedTransitions, formatCents, formatPeriod,
    payrollCashBankOptions, payrollMonths, periodOfDay, runContributionTotals, runState, runTotals, type IPayrollMonth,
    type RunState,
  } from './payroll.ts'

  let { config, employees, organization }: {
    config: PayrollConfigService, employees: EmployeesService, organization: OrganizationService
  } = $props()

  const ui = useUI()
  const RUN_MODAL_ID = 32
  const PAY_MODAL_ID = 34
  const CONTRIBUTIONS_MODAL_ID = 35
  const PAYSLIP_LAYER_ID = 1
  /** backend/access.toml "Planilla" (41) and its sub-accesses. */
  const ACCESS_ID_PAYROLL = 41
  const canCalculate = security.checkSubAcceso(ACCESS_ID_PAYROLL, 2)
  const canClose = security.checkSubAcceso(ACCESS_ID_PAYROLL, 3)
  const canPay = security.checkSubAcceso(ACCESS_ID_PAYROLL, 4)

  // The list is one row per month of the window the server answers, never a manual "new payroll".
  let selectedPeriod = $state(0)
  let payrollWindow = $state<IPayrollEnsureResult>()
  let isUpdating = $state(false)
  let payslips = $state<PayslipsService>()
  let runForm = $state({} as IPayrollRun)
  let selectedPayslip = $state<IPayslip>()
  let isPrinting = $state(false)
  let payForm = $state({ CashBankID: 0 })
  let contributionForm = $state({} as IContributionsPayment)

  const cajas = new CajasService()
  const cashBankOptions = $derived(payrollCashBankOptions(cajas.Cajas))
  const contributionOptions = CONTRIBUTION_ACCOUNTS.map((account) => ({ ...account, name: `${account.code} ${tr(account.name)}` }))

  const months = $derived(payrollMonths(config.Runs, payrollWindow))
  const selectedRun = $derived(config.Runs.find((run) => run.Period === selectedPeriod))
  const selectedRunID = $derived(selectedRun?.ID || 0)
  const runPayslips = $derived((payslips?.records || []).filter((payslip) => payslip.ss > 0 && payslip.RunID === selectedRunID))
  const contributionTotals = $derived(runContributionTotals(runPayslips))
  const totals = $derived(runTotals(runPayslips))
  const transitions = $derived(selectedRun ? allowedTransitions(selectedRun) : undefined)
  const employeeName = (employeeID: number) => {
    const employee = employees.recordsMap.get(employeeID)
    return employee ? employeeFullName(employee) : `#${employeeID}`
  }

  // Each period's payslips are their own cached list, built when a month needs it.
  const selectMonth = (month: IPayrollMonth) => {
    if (payslips?.route !== `hr-payslips?period=${month.Period}`) {
      payslips = new PayslipsService(month.Period)
    }
    selectedPeriod = month.Period
  }

  // Opening the page leaves every month calculated; the first view is the current month.
  $effect(() => {
    if (selectedPeriod > 0 || months.length === 0) return
    const currentPeriod = periodOfDay(todayUnixDay())
    untrack(() => selectMonth(months.find((month) => month.Period === currentPeriod) || months[0]))
  })

  // updatePayroll asks the server to create the months with something to pay and to recalculate the
  // open ones whose inputs changed (records, loans, contracts... are edited on other pages, so
  // opening this page is what brings the runs up to date), then reloads what the page shows.
  const updatePayroll = async () => {
    if (!canCalculate) return
    isUpdating = true
    try {
      payrollWindow = await postPayrollEnsure(config)
      await payslips?.fetch()
      if (payrollWindow.Skipped.length > 0) {
        notifyFailure(payrollWindow.Skipped.map((skipped) =>
          `${formatPeriod(skipped.Period)} · ${employeeName(skipped.EmployeeID)}: ${skipped.Message}`).join('\n'))
      }
    } catch (error) {
      notifyFailure(error as string)
    }
    isUpdating = false
  }
  onMount(updatePayroll)

  const profitSharingRateOptions = PROFIT_SHARING_RATE_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))

  // The server recalculates the run with its new pay date and profit sharing.
  const saveRun = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      await postPayrollRun(config, runForm)
      await payslips?.fetch()
      ui.closeModal(RUN_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const TRANSITION_MESSAGES: Record<PayrollTransition, string> = {
    'hr-payroll-calculate': 'Calculating payroll...|Calculando planilla...',
    'hr-payroll-close': 'Closing payroll...|Cerrando planilla...',
    'hr-payroll-reopen': 'Reopening payroll...|Reabriendo planilla...',
    'hr-payroll-paid': 'Registering payment...|Registrando pago...',
  }

  const runTransition = async (transition: PayrollTransition) => {
    if (!selectedRun) return
    // Paying asks for the account in its own modal, which is the confirmation.
    if (transition !== 'hr-payroll-calculate' && transition !== 'hr-payroll-paid' && !(await confirmWarn({
      title: tr('Confirm|Confirmar'), message: tr(TRANSITION_MESSAGES[transition]) }))) return
    showLoading(tr(TRANSITION_MESSAGES[transition]))
    try {
      const result = await postPayrollTransition(config, transition, selectedRun, payForm.CashBankID) as
        { Skipped?: { EmployeeID: number, Message: string }[] }
      ui.closeModal(PAY_MODAL_ID)
      await payslips?.fetch()
      if (result?.Skipped?.length) {
        notifyFailure(result.Skipped.map((skipped) => `${employeeName(skipped.EmployeeID)}: ${skipped.Message}`).join('\n'))
      }
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const openContributions = () => {
    if (!selectedRun) return
    contributionForm = { RunID: selectedRun.ID, AccountCode: 4031, Amount: contributionTotals.get(4031) || 0,
      Date: todayUnixDay(), CashBankID: selectedRun.CashBankID } as IContributionsPayment
    ui.openModal(CONTRIBUTIONS_MODAL_ID)
  }

  const payContributions = async () => {
    showLoading(tr('Registering payment...|Registrando pago...'))
    try {
      const result = await postPayrollContributions(contributionForm)
      notifySuccess(tr(`Paid. Still pending on this account: S/ ${formatCents(result.Pending)}|Pagado. Pendiente en esta cuenta: S/ ${formatCents(result.Pending)}`))
      ui.closeModal(CONTRIBUTIONS_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const printBoleta = () => {
    isPrinting = true
    // Let the body-level copy render before the print dialog freezes the page.
    setTimeout(() => { window.print(); isPrinting = false }, 50)
  }

  /** Badge colors of each run state on the month cards. */
  const RUN_STATE_BADGE_CSS: Record<RunState, string> = {
    draft: 'bg-gray-100 text-gray-600',
    calculated: 'bg-blue-100 text-blue-700',
    closed: 'bg-green-100 text-green-700',
    paid: 'bg-purple-100 text-purple-700',
  }

  const payslipColumns: ITableColumn<IPayslip>[] = [
    { header: 'Employee|Trabajador', css: 'px-6', highlight: true, getValue: (e) => employeeName(e.EmployeeID) },
    { header: 'Department|Área', css: 'px-6', getValue: (e) => organization.DepartmentsMap.get(e.DepartmentID)?.Name || '' },
    { header: 'Days|Días', css: 'px-6 text-center', getValue: (e) => e.WorkedDays },
    { header: 'Earnings|Ingresos', css: 'px-6 text-right', getValue: (e) => formatCents(e.TotalEarnings) },
    { header: 'Deductions|Descuentos', css: 'px-6 text-right', getValue: (e) => formatCents(e.TotalDeductions) },
    { header: 'Net pay|Neto', css: 'px-6 text-right', getValue: (e) => formatCents(e.NetPay) },
    { header: 'Employer cost|Costo empleador', css: 'px-6 text-right', getValue: (e) => formatCents(e.EmployerCost) },
    { header: 'Observations|Observaciones', css: 'px-6',
      getValue: (e) => (e.Diagnostics || []).map((diagnostic) => diagnostic.Code).join(', ') },
  ]
</script>

<!-- Full height of the page: each column scrolls on its own on desktop. -->
<div class="grid grid-cols-1 xl:grid-cols-[340px_1fr] xl:h-full">
  <div class="bg-page p-12 xl:overflow-y-auto" aria-label="Payroll months">
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('Payrolls|Planillas')}</div>
      {#if isUpdating}
        <div class="text-gray-500">{tr('Updating...|Actualizando...')}</div>
      {/if}
    </div>
    <div class="flex flex-col gap-8">
      {#each months as month (month.Period)}
        {@const isSelectedMonth = month.Period === selectedPeriod}
        <button type="button" aria-pressed={isSelectedMonth} onclick={() => selectMonth(month)}
          class="flex items-center gap-10 rounded-[10px] border bg-surface px-14 py-10 text-left shadow-sm transition-colors hover:border-indigo-300 hover:bg-indigo-50
            {isSelectedMonth ? 'border-indigo-500 ring-2 ring-indigo-200' : 'border-line'}">
          <div class="mr-auto">
            <div class="ff-bold text-[16px]">{formatPeriod(month.Period)}</div>
            {#if month.Run}
              <span class="mt-4 inline-block rounded-full px-8 py-1 text-[14px] {RUN_STATE_BADGE_CSS[runState(month.Run)]}">
                {tr(RUN_STATE_LABELS[runState(month.Run)])}
              </span>
            {:else}
              <span class="text-gray-400">{tr('No payroll|Sin planilla')}</span>
            {/if}
          </div>
          {#if month.Run}
            <div class="text-right">
              <div class="text-gray-500">{tr('Net pay|Neto')}</div>
              <div class="ff-bold text-[16px]">{formatCents(month.Run.TotalNetPay)}</div>
            </div>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  {#if selectedPeriod > 0 && !selectedRun}
    <div class="p-16 text-gray-500" aria-label="Payroll detail">
      {tr(`${formatPeriod(selectedPeriod)} has no employees to pay.|${formatPeriod(selectedPeriod)} no tiene trabajadores a quienes pagar.`)}
    </div>
  {:else if selectedRun && transitions}
    <!-- A column at the page's height: the payslips table takes what is left and scrolls inside. -->
    <div class="min-w-0 p-16 flex flex-col xl:h-full xl:overflow-hidden" aria-label="Payroll detail">
      <div class="flex flex-wrap items-center gap-8 mb-10">
        <div class="h3 ff-bold">
          {formatPeriod(selectedRun.Period)} · {tr(RUN_STATE_LABELS[runState(selectedRun)])}
        </div>
        <div class="mr-auto text-gray-500">· {runPayslips.length} {tr('employees|trabajadores')}</div>
        {#if transitions.edit}
          <Button mode="subtle" color="blue" icon="icon-[fa--pencil]" label="Edits the pay date, note and profit sharing of the payroll."
            onClick={() => { runForm = { ...selectedRun }; ui.openModal(RUN_MODAL_ID) }} />
        {/if}
        {#if canCalculate && transitions.calculate}
          <Button mode="subtle" color="blue" icon="icon-[fa--calculator]" name="Recalculate|Recalcular"
            label="Recalculates every payslip of the payroll now; it is already recalculated when its data changes."
            onClick={() => runTransition('hr-payroll-calculate')} />
        {/if}
        {#if canClose && transitions.close}
          <Button color="green" icon="icon-[fa--lock]" name="Close|Cerrar" label="Closes the payroll: its payslips become final."
            disabled={totals.blocked > 0} onClick={() => runTransition('hr-payroll-close')} />
        {/if}
        {#if canClose && transitions.reopen}
          <Button color="orange" icon="icon-[fa--unlock]" name="Reopen|Reabrir" label="Reopens the closed payroll."
            onClick={() => runTransition('hr-payroll-reopen')} />
        {/if}
        {#if canPay && transitions.pay}
          <Button color="purple" icon="icon-[fa--money]" name="Pay|Pagar" label="Opens the modal to pay the payroll's net pay."
            onClick={() => { payForm = { CashBankID: 0 }; ui.openModal(PAY_MODAL_ID) }} />
        {/if}
        {#if canPay && (runState(selectedRun) === 'closed' || runState(selectedRun) === 'paid')}
          <Button mode="subtle" color="purple" icon="icon-[fa--university]" name="Contributions|Aportes"
            label="Opens the modal to pay EsSalud, ONP, 5ta, AFP or withholdings of the payroll." onClick={openContributions} />
        {/if}
      </div>

      {#if totals.blocked > 0}
        <div class="text-red-600 mb-8">
          {tr(`${totals.blocked} payslips have observations that block closing.|${totals.blocked} boletas tienen observaciones que impiden el cierre.`)}
        </div>
      {/if}

      <div class="flex flex-wrap items-end gap-x-24 gap-y-8">
        <div class="ml-auto flex flex-wrap gap-x-24 gap-y-4">
          {#each [
            ['Earnings|Ingresos', formatCents(totals.earnings)],
            ['Net pay|Neto a pagar', formatCents(totals.netPay)],
            ['Employer cost|Costo empleador', formatCents(totals.employerCost)],
          ] as [label, value]}
            <div class="text-right">
              <div class="text-gray-500 leading-tight">{tr(label)}</div>
              <div class="ff-bold text-[16px] leading-tight">{value}</div>
            </div>
          {/each}
        </div>
      </div>
      <!-- Takes the rest of the column; the table scrolls inside it (maxHeight 100%). The payslip side
           layer is fixed and covers the table, so the table is not wrapped in a content Layer: that
           one shrinks and clips the table while the side layer is open. -->
      <div class="mt-10 min-h-0 flex-1 overflow-y-auto">
        <VTable columns={payslipColumns} data={runPayslips} css="w-full" maxHeight="100%" selected={selectedPayslip?.ID}
          isSelected={(payslip, id) => payslip.ID === id}
          onRowClick={(payslip) => { selectedPayslip = payslip; ui.openSideLayer(PAYSLIP_LAYER_ID) }} />

      </div>
    </div>
  {/if}
</div>

<Layer type="side" id={PAYSLIP_LAYER_ID} sideLayerSize={820}
  title={selectedPayslip ? employeeName(selectedPayslip.EmployeeID) : ''} titleCss="h2 mb-6"
  saveButtonName="Print|Imprimir" saveButtonIcon="icon-[fa--print]" onSave={printBoleta}
  onClose={() => { selectedPayslip = undefined }}
>
  {#if selectedPayslip && selectedRun}
    {#each selectedPayslip.Diagnostics || [] as diagnostic}
      <div class={diagnostic.Blocking ? 'text-red-600 mb-4' : 'text-orange-600 mb-4'}>{diagnostic.Code}: {diagnostic.Message}</div>
    {/each}
    <PayslipBoleta payslip={selectedPayslip} run={selectedRun} employee={employees.recordsMap.get(selectedPayslip.EmployeeID)}
      concepts={config.ConceptsMap} printing={isPrinting} />
  {/if}
</Layer>

<Modal id={PAY_MODAL_ID} size={3} bodyCss="px-16 py-14" title={tr('Pay payroll|Pagar planilla')}
  saveButtonLabel="Pay|Pagar" onSave={() => runTransition('hr-payroll-paid')}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Payroll payment form">
    <div class="col-span-24">{tr('Net pay|Neto a pagar')}: <span class="ff-bold">S/ {formatCents(totals.netPay)}</span></div>
    <SearchSelect bind:saveOn={payForm} save="CashBankID" label="Paid from|Pagado desde" required={true}
      options={cashBankOptions} keyId="ID" keyName="Name" css="col-span-24" />
  </div>
</Modal>

<Modal id={CONTRIBUTIONS_MODAL_ID} size={4} bodyCss="px-16 py-14" title={tr('Pay contributions|Pagar aportes')}
  saveButtonLabel="Pay|Pagar" onSave={payContributions}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Contributions payment form">
    <SearchSelect bind:saveOn={contributionForm} save="AccountCode" label="Payable|Concepto a pagar" required={true}
      options={contributionOptions} keyId="code" keyName="name" css="col-span-24"
      onChange={(option) => { if (option) contributionForm.Amount = contributionTotals.get(option.code) || 0 }} />
    <Input bind:saveOn={contributionForm} save="Amount" type="number" baseDecimals={2} label="Amount (S/)|Monto (S/)"
      required={true} css="col-span-24 md:col-span-12" />
    <DateInput bind:saveOn={contributionForm} save="Date" label="Date|Fecha" required={true} css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={contributionForm} save="CashBankID" label="Paid from|Pagado desde" required={true}
      options={cashBankOptions} keyId="ID" keyName="Name" css="col-span-24" />
  </div>
</Modal>

<Modal id={RUN_MODAL_ID} size={4} bodyCss="px-16 py-14" title={tr('Edit payroll|Editar planilla')} onSave={saveRun}>
  <div class="grid grid-cols-24 gap-10" aria-label="Payroll form">
    <DateInput bind:saveOn={runForm} save="PayDate" label="Pay date|Fecha de pago" required={true} css="col-span-24" />
    <div class="col-span-24 text-gray-600">
      {tr('Profit sharing: fill it in the month it is paid. It distributes the previous fiscal year, from its annual tax return; leave it empty when there is none.|Utilidades: complételo en el mes en que se pagan. Reparte el ejercicio anterior, según su declaración anual; déjelo vacío si no hay.')}
    </div>
    <Input bind:saveOn={runForm} save="NetIncomeBeforeTax" type="number" baseDecimals={2}
      label="Net income before tax (S/)|Renta neta antes de impuestos (S/)" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={runForm} save="SharingRate" label="Sector rate|Porcentaje del sector"
      options={profitSharingRateOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={runForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>
