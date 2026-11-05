<script lang="ts">
  import Portal from '#components/misc/Portal.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import { employeeFullName } from '#routes/hr/employees/employees.ts'
  import type { IEmployee } from '#services/hr/employees.svelte.ts'
  import { PENSION_SYSTEM_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import type { IConcept, IPayrollRun, IPayslip, IPayslipLine } from '#services/hr/payroll.svelte.ts'
  import { formatCents, formatPeriod, formatRate, payslipLineName, payslipSections } from './payroll.ts'

  // The boleta de pago, rendered as HTML and printed by the browser (plan §6.2). The same markup is
  // the on-screen preview and, through a body-level copy, the printed page.
  let { payslip, run, employee, concepts, printing = false }: {
    payslip: IPayslip, run: IPayrollRun, employee: IEmployee | undefined, concepts: Map<number, IConcept>, printing?: boolean
  } = $props()

  const sections = $derived(payslipSections(payslip))
  const pensionName = $derived(tr(PENSION_SYSTEM_OPTIONS.find((option) => option.id === payslip.Parameters?.PensionSystem)?.name || ''))

  const lineDetail = (line: IPayslipLine): string => {
    if (line.Rate > 0) return formatRate(line.Rate)
    if (line.Quantity > 0) return String(line.Quantity)
    return ''
  }
</script>

{#snippet boleta()}
  <div class="_boleta" aria-label="Payslip">
    <div class="flex justify-between mb-8">
      <div>
        <div class="ff-bold text-[16px]">{tr('PAYSLIP|BOLETA DE PAGO')}</div>
        <div>{tr('Period|Periodo')}: {formatPeriod(payslip.Period)} · {tr('Pay date|Fecha de pago')}: {formatTime(run.PayDate, 'd-M-Y')}</div>
      </div>
      <div class="text-right">
        <div class="ff-bold">{employee ? employeeFullName(employee) : `#${payslip.EmployeeID}`}</div>
        <div>DNI {employee?.DocumentNumber || ''}</div>
        <div>{pensionName} · {tr('Days worked|Días laborados')}: {payslip.WorkedDays}</div>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-12">
      {#each [
        { title: 'Earnings|Ingresos', lines: sections.earnings, total: payslip.TotalEarnings },
        { title: 'Deductions|Descuentos', lines: sections.deductions, total: payslip.TotalDeductions },
      ] as section}
        <table class="w-full">
          <thead><tr><th class="text-left" colspan="3">{tr(section.title)}</th></tr></thead>
          <tbody>
            {#each section.lines as line}
              <tr>
                <td>
                  {tr(payslipLineName(line, concepts))}
                  {#if line.OutsideNetPay}
                    <span class="text-gray-500">({tr('deposited in the bank|depositada en el banco')})</span>
                  {/if}
                </td>
                <td class="text-right text-gray-500">{lineDetail(line)}</td>
                <td class="text-right">{formatCents(line.Amount)}</td>
              </tr>
            {/each}
            <tr class="ff-bold"><td colspan="2">{tr('Total|Total')}</td><td class="text-right">{formatCents(section.total)}</td></tr>
          </tbody>
        </table>
      {/each}
    </div>

    <div class="flex justify-between items-end mt-12">
      <div>
        <div class="ff-bold">{tr('Employer contributions|Aportes del empleador')}</div>
        {#each sections.employer as line}
          <div>{tr(payslipLineName(line, concepts))} ({formatRate(line.Rate)}): {formatCents(line.Amount)}</div>
        {/each}
      </div>
      <div class="ff-bold text-[16px]">{tr('Net pay|Neto a pagar')}: S/ {formatCents(payslip.NetPay)}</div>
    </div>
    <div class="flex justify-around mt-40">
      <div class="_signature">{tr('Employer|Empleador')}</div>
      <div class="_signature">{tr('Employee|Trabajador')}</div>
    </div>
  </div>
{/snippet}

{@render boleta()}

{#if printing}
  <!-- Body level: the side layer has a transform that would capture position: fixed. -->
  <Portal><div class="_printCopy">{@render boleta()}</div></Portal>
{/if}

<style>
  ._boleta {
    background-color: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
    padding: 16px;
  }
  ._boleta th {
    border-bottom: 1px solid #d1d5db;
  }
  ._boleta td {
    padding: 2px 4px;
  }
  ._signature {
    border-top: 1px solid #6b7280;
    width: 180px;
    text-align: center;
  }
  ._printCopy {
    display: none;
  }
  @media print {
    :global(body *) {
      visibility: hidden !important;
    }
    ._printCopy {
      display: block;
      visibility: visible !important;
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      color: #000000;
    }
    ._printCopy :global(*) {
      visibility: visible !important;
    }
  }
</style>
