<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Page from '#domain/Page.svelte'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import { ClientProviderService } from '#services/crm/client-provider.svelte.ts'
  import {
    LoansService, postLoan, postLoanPayment, type ILoan, type ILoanInstallment, type ILoanPayment,
  } from '#services/finance/loans.svelte.ts'
  import LoanForm from './LoanForm.svelte'
  import LoanInstallmentsTable from './LoanInstallmentsTable.svelte'
  import LoansCalendar from './LoansCalendar.svelte'
  import {
    installmentTotal, LOAN_METHOD_FRENCH, loanFormProblem, nextPendingInstallment, outstandingPrincipal,
  } from './loans.ts'

  // Loans the company owes, with the lender's installment schedule and its payments. A payment is one
  // cash movement split into fees (671), interest (673) and principal (451).
  const ui = useUI()
  const LOAN_LAYER_ID = 1
  const PAYMENT_MODAL_ID = 21
  const PROVIDER_TYPE = 2
  const LOAN_STATUS_PAID_OFF = 2

  const loansService = new LoansService(true)
  const lendersService = new ClientProviderService(PROVIDER_TYPE, true)
  const cashBanksService = new CajasService()

  let filterText = $state('')
  let form = $state({ Installments: [] } as unknown as ILoan)
  let layerView = $state(1)
  let payment = $state({} as ILoanPayment)
  let paymentInstallment = $state(null as ILoanInstallment | null)

  const formatAmount = (cents: number) => formatN(cents / 100, 2)
  const lenderName = (lenderID: number) => lendersService.recordsMap.get(lenderID)?.Name || ''

  const openLoan = (loan?: ILoan) => {
    form = loan
      ? { ...loan, Installments: loan.Installments.map((installment) => ({ ...installment })) }
      : { ss: 1, Method: LOAN_METHOD_FRENCH, GraceMonths: 0, Installments: [], RegisterDisbursement: true } as unknown as ILoan
    layerView = 1
    ui.openSideLayer(LOAN_LAYER_ID)
  }

  const saveLoan = async (isDelete = false) => {
    if (isDelete) {
      if (!(await confirmWarn({ title: tr('Delete loan|Eliminar préstamo'), message: tr(`Delete "${form.Name}"?|¿Eliminar "${form.Name}"?`) }))) return
    } else {
      const problem = loanFormProblem(form)
      if (problem) { notifyFailure(tr(problem)); return }
    }
    showLoading(tr('Saving...|Guardando...'))
    try {
      const saved = await postLoan(loansService, { ...form, ss: isDelete ? 0 : form.ss || 1 })
      if (isDelete) ui.openSideLayer(0)
      else form = { ...saved, Installments: saved.Installments || [] }
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // The payment defaults to what the installment still owes, from the loan's own account, today.
  const openPayment = (installment: ILoanInstallment) => {
    paymentInstallment = installment
    payment = {
      LoanID: form.ID, InstallmentNumber: installment.Number, CashBankID: form.CashBankID,
      Amount: installmentTotal(installment) - installment.PaidAmount, Date: todayUnixDay(),
    }
    ui.openModal(PAYMENT_MODAL_ID)
  }

  const savePayment = async () => {
    if (!(payment.Amount > 0) || !payment.CashBankID) {
      notifyFailure(tr('Enter the amount and the account.|Ingrese el monto y la caja o banco.'))
      return
    }
    showLoading(tr('Paying...|Pagando...'))
    try {
      const paidLoan = await postLoanPayment(loansService, payment)
      form = { ...paidLoan, Installments: paidLoan.Installments || [] }
      ui.closeModal(PAYMENT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<ILoan>[] = [
    { header: 'Loan|Préstamo', css: 'px-6', highlight: true, getValue: (loan) => loan.Name },
    { header: 'Lender|Entidad', css: 'px-6', getValue: (loan) => lenderName(loan.LenderID) },
    { header: 'Amount|Monto', css: 'px-6 text-right', getValue: (loan) => formatAmount(loan.Principal) },
    { header: 'Outstanding|Saldo capital', css: 'px-6 text-right ff-bold', getValue: (loan) => formatAmount(outstandingPrincipal(loan)) },
    {
      header: 'Next installment|Próxima cuota', css: 'px-6',
      getValue: (loan) => {
        const next = nextPendingInstallment(loan)
        return next ? `${formatTime(next.DueDate, 'd-m-Y')} · ${formatAmount(installmentTotal(next) - next.PaidAmount)}` : ''
      },
    },
    {
      header: 'Status|Estado', css: 'px-6',
      getValue: (loan) => tr(loan.ss === LOAN_STATUS_PAID_OFF ? 'Paid off|Cancelado' : 'Active|Vigente'),
    },
  ]
</script>

<Page title="Loans|Préstamos" options={[{ id: 1, name: 'Loans|Préstamos' }, { id: 2, name: 'Calendar|Calendario' }]}>
  {#if ui.state.pageOptionSelected === 2}
    <LoansCalendar loans={loansService.loans.filter((loan) => loan.ss !== LOAN_STATUS_PAID_OFF)} />
  {:else}
    <div class="flex items-center justify-between mb-8">
      <FilterInput bind:value={filterText} css="w-256" />
      <Button color="green" icon="icon-[fa--plus]" name="New loan|Nuevo préstamo"
        label="Opens the form to register a loan and its installments.|Abrir el formulario para registrar un préstamo y sus cuotas." onClick={() => openLoan()} />
    </div>
    <Layer type="content">
      <VTable css="w-full" maxHeight="calc(100vh - 13rem)" {columns} data={loansService.loans}
        {filterText} getFilterContent={(loan) => `${loan.Name} ${lenderName(loan.LenderID)}`}
        selected={form?.ID} isSelected={(loan, loanID) => loan.ID === loanID}
        onRowClick={(loan) => openLoan(loan)}
      />
    </Layer>
  {/if}

  <Layer type="side" id={LOAN_LAYER_ID} width={900}
    title={form?.ID ? form.Name : tr('New loan|Nuevo préstamo')} titleCss="h2 ff-bold"
    bind:selected={layerView}
    options={[[1, 'Loan|Préstamo'], [2, `${tr('Installments|Cuotas')} (${form.Installments?.length || 0})`]]}
    onSave={() => saveLoan()}
    onDelete={form?.ID ? () => saveLoan(true) : undefined}
    onClose={() => { form = { Installments: [] } as unknown as ILoan }}
  >
    {#if layerView === 1}
      <LoanForm bind:loan={form} lenders={lendersService.records} cashBanks={cashBanksService.Cajas} />
    {:else}
      <LoanInstallmentsTable bind:loan={form} onPay={openPayment} />
    {/if}
  </Layer>
</Page>

<Modal id={PAYMENT_MODAL_ID} size={4} bodyCss="px-16 py-14"
  title={`${tr('Pay installment|Pagar cuota')} ${paymentInstallment?.Number || ''}`}
  onSave={savePayment} saveButtonLabel="Pay|Pagar"
>
  <div class="grid grid-cols-24 gap-10" aria-label="Loan installment payment form">
    {#if paymentInstallment}
      <div class="col-span-24 text-fg-muted">
        {tr('Pending|Pendiente')}: {formatAmount(installmentTotal(paymentInstallment) - paymentInstallment.PaidAmount)}
        · {tr('applied to fees, then interest, then principal|se aplica a comisiones, luego interés, luego capital')}
      </div>
    {/if}
    <SearchSelect bind:saveOn={payment} save="CashBankID" label="Pay from|Pagar desde" required={true}
      options={cashBanksService.Cajas} keyId="ID" keyName="Name" css="col-span-24" />
    <Input bind:saveOn={payment} save="Amount" type="number" baseDecimals={2} label="Amount|Monto" required={true}
      css="col-span-12" />
    <DateInput bind:saveOn={payment} save="Date" label="Date|Fecha" required={true} css="col-span-12" />
  </div>
</Modal>
