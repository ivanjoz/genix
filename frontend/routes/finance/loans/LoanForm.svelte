<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import { confirmWarn, notifyFailure } from '@genix/ui/notify'
  import { tr } from '#core/store.svelte.ts'
  import type { ICashBank } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import type { IClientProvider } from '#services/crm/client-provider.svelte.ts'
  import type { ILoan } from '#services/finance/loans.svelte.ts'
  import { addMonthsToUnixDay, buildLoanSchedule, LOAN_METHOD_OPTIONS } from './loans.ts'

  // The loan's data and the calculator that proposes its installments. The proposal replaces the
  // schedule; the user then adjusts it to the lender's in the Installments tab.
  let { loan = $bindable(), lenders, cashBanks }: {
    loan: ILoan, lenders: IClientProvider[], cashBanks: ICashBank[],
  } = $props()

  const methodOptions = LOAN_METHOD_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const hasPayments = $derived(loan.Installments.some((installment) => installment.PaidAmount > 0))
  // A registered disbursement already moved cash: its amount, account and date are fixed.
  const isDisbursementLocked = $derived(!!loan.IsDisbursementRegistered)

  let calculator = $state({ Months: 12, FirstDueDate: 0, Fee: 0 })

  const proposeSchedule = async () => {
    if (hasPayments) {
      notifyFailure(tr('The loan already has payments: edit its installments instead.|El préstamo ya tiene pagos: edite sus cuotas.'))
      return
    }
    if (!(loan.Principal > 0) || !loan.DisbursementDate || !(calculator.Months > 0)) {
      notifyFailure(tr('Enter the amount, the disbursement date and the months.|Ingrese el monto, la fecha del desembolso y los meses.'))
      return
    }
    if (loan.Installments.length && !(await confirmWarn({
      title: tr('Replace installments|Reemplazar cuotas'),
      message: tr('Replace the current installments with the calculated ones?|¿Reemplazar las cuotas actuales por las calculadas?'),
    }))) return
    loan.Installments = buildLoanSchedule({
      principal: loan.Principal, annualRate: loan.AnnualRate || 0, months: calculator.Months,
      disbursementDate: loan.DisbursementDate,
      firstDueDate: calculator.FirstDueDate || addMonthsToUnixDay(loan.DisbursementDate, 1),
      method: loan.Method || LOAN_METHOD_OPTIONS[0].id, graceMonths: loan.GraceMonths || 0, fee: calculator.Fee || 0,
    })
  }
</script>

<div class="grid grid-cols-24 gap-10 mt-12" aria-label="Loan form">
  <Input bind:saveOn={loan} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
  <SearchSelect bind:saveOn={loan} save="LenderID" label="Lender|Entidad que presta" required={true}
    options={lenders} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12" />
  <SearchSelect bind:saveOn={loan} save="CashBankID" label="Cash / bank account|Caja o banco" required={true}
    options={cashBanks} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12" disabled={isDisbursementLocked}
    onChange={(cashBank) => { loan.CurrencyType = cashBank?.CurrencyType || 0 }} />
  <Input bind:saveOn={loan} save="Principal" type="number" baseDecimals={2} label="Amount|Monto" required={true}
    css="col-span-12 md:col-span-8" disabled={isDisbursementLocked} />
  <DateInput bind:saveOn={loan} save="DisbursementDate" label="Disbursement|Desembolso" required={true}
    css="col-span-12 md:col-span-8" disabled={isDisbursementLocked} />
  <Input bind:saveOn={loan} save="AnnualRate" type="number" baseDecimals={4} label="TEA (%)|TEA (%)"
    css="col-span-12 md:col-span-8" />
  <SearchSelect bind:saveOn={loan} save="Method" label="Method|Método" options={methodOptions}
    keyId="id" keyName="name" css="col-span-12 md:col-span-16" />
  <Input bind:saveOn={loan} save="GraceMonths" type="number" label="Grace months|Meses de gracia"
    css="col-span-12 md:col-span-8" />
  <Input bind:saveOn={loan} save="Notes" label="Notes|Notas" css="col-span-24" />
  {#if !loan.ID}
    <Checkbox bind:saveOn={loan} save="RegisterDisbursement" css="col-span-24"
      label="Register the disbursement in the account|Registrar el desembolso en la caja o banco" />
  {/if}
</div>

<!-- The calculator only proposes: the lender's schedule always prevails. -->
<div class="mt-16 rounded-[10px] border border-line p-12" aria-label="Installment calculator">
  <div class="h4 ff-bold mb-8">{tr('Installment calculator|Calculadora de cuotas')}</div>
  <div class="grid grid-cols-24 gap-10">
    <Input bind:saveOn={calculator} save="Months" type="number" label="Months|Meses" css="col-span-12 md:col-span-6" />
    <DateInput bind:saveOn={calculator} save="FirstDueDate" label="First due date|Primera cuota"
      css="col-span-12 md:col-span-8" />
    <Input bind:saveOn={calculator} save="Fee" type="number" baseDecimals={2} label="Fee per installment|Comisión por cuota"
      css="col-span-12 md:col-span-6" />
    <div class="col-span-12 md:col-span-4 flex items-end">
      <Button color="blue" icon="icon-[fa--calculator]" name="Calculate|Calcular"
        label="Proposes the installments from the amount, TEA, method and grace.|Proponer las cuotas según monto, TEA, método y gracia." onClick={proposeSchedule} />
    </div>
  </div>
  <div class="text-fg-muted mt-8">
    {tr('Interest compounds the TEA by the actual days between installments over a 360-day year.|El interés capitaliza la TEA por los días reales entre cuotas sobre un año de 360 días.')}
  </div>
</div>
