<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, todayUnixDay } from '#libs/helpers.ts'
  import type { ILoan, ILoanInstallment } from '#services/finance/loans.svelte.ts'
  import {
    addMonthsToUnixDay, installmentState, installmentTotal, LOAN_INSTALLMENT_OVERDUE, LOAN_INSTALLMENT_PAID,
  } from './loans.ts'
  import { dmyToUnixDay, unixDayToDMY } from './loans.utils.ts'

  // The schedule, edited to match the lender's. An installment that received a payment keeps its
  // place (it can't be removed); a saved loan's pending installments are paid from here.
  let { loan = $bindable(), onPay }: {
    loan: ILoan, onPay: (installment: ILoanInstallment) => void,
  } = $props()

  const today = todayUnixDay()
  const stateLabels: Record<number, string> = {
    [LOAN_INSTALLMENT_PAID]: 'Paid|Pagada', [LOAN_INSTALLMENT_OVERDUE]: 'Overdue|Vencida',
  }
  const principalsTotal = $derived(loan.Installments.reduce((total, installment) => total + installment.Principal, 0))

  const toCents = (value: string | number) => Math.round((parseFloat(String(value || '0')) || 0) * 100)
  const renumber = () => { loan.Installments = loan.Installments.map((installment, index) => ({ ...installment, Number: index + 1 })) }

  const addInstallment = () => {
    const last = loan.Installments[loan.Installments.length - 1]
    const dueDate = addMonthsToUnixDay(last?.DueDate || loan.DisbursementDate || today, 1)
    loan.Installments = [...loan.Installments,
      { Number: loan.Installments.length + 1, DueDate: dueDate, Principal: 0, Interest: 0, Fees: 0, PaidAmount: 0 }]
  }

  const amountColumn = (header: string, key: 'Principal' | 'Interest' | 'Fees'): ITableColumn<ILoanInstallment> => ({
    id: key, header, css: 'px-6 text-right', headerCss: 'w-110', cellInputType: 'number',
    // A paid installment is locked and VTable prints a locked cell's value as is: give it formatted.
    getValue: (installment) => installment.PaidAmount > 0 ? formatN(installment[key] / 100, 2) : installment[key] / 100,
    render: (installment) => formatN(installment[key] / 100, 2),
    onCellEdit: (installment, value) => { installment[key] = toCents(value) },
    disableCellInteractions: (installment) => installment.PaidAmount > 0,
  })

  const columns: ITableColumn<ILoanInstallment>[] = [
    { id: 'number', header: '#', headerCss: 'w-44', css: 'px-6 text-center', getValue: (installment) => installment.Number },
    {
      id: 'dueDate', header: 'Due (dd-mm-yyyy)|Vence (dd-mm-aaaa)', headerCss: 'w-130', css: 'px-6',
      getValue: (installment) => unixDayToDMY(installment.DueDate),
      onCellEdit: (installment, value) => { installment.DueDate = dmyToUnixDay(String(value)) || installment.DueDate },
      disableCellInteractions: (installment) => installment.PaidAmount > 0,
    },
    amountColumn('Principal|Capital', 'Principal'),
    amountColumn('Interest|Interés', 'Interest'),
    amountColumn('Fees|Comisiones', 'Fees'),
    {
      id: 'total', header: 'Total|Total', headerCss: 'w-110', css: 'px-6 text-right ff-bold',
      getValue: (installment) => formatN(installmentTotal(installment) / 100, 2),
    },
    {
      id: 'paid', header: 'Paid|Pagado', headerCss: 'w-100', css: 'px-6 text-right',
      getValue: (installment) => installment.PaidAmount ? formatN(installment.PaidAmount / 100, 2) : '',
    },
    { id: 'actions', header: '', headerCss: 'w-96', css: 'px-4', getValue: () => '' },
  ]
</script>

<div class="flex items-center justify-between mt-8 mb-6">
  <div class={principalsTotal === loan.Principal ? 'text-fg-muted' : 'text-red-fg'}>
    {tr('Principal of the installments|Capital de las cuotas')}: {formatN(principalsTotal / 100, 2)}
    / {formatN((loan.Principal || 0) / 100, 2)}
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="Installment|Cuota" label="Adds an empty installment a month after the last one."
    onClick={addInstallment} />
</div>
<VTable {columns} data={loan.Installments} disableVirtualizer={true} estimateSize={42} css="w-full"
  emptyMessage="Calculate or add the installments.|Calcule o agregue las cuotas.">
  {#snippet cellRenderer(installment, column, defaultContent)}
    {#if column.id === 'actions'}
      <div class="flex justify-center gap-6">
        {#if installmentState(installment, today) === LOAN_INSTALLMENT_PAID}
          <span class="text-green-fg">{tr(stateLabels[LOAN_INSTALLMENT_PAID])}</span>
        {:else if loan.ID}
          <Button color="blue" mode="subtle" height="tiny" name="Pay|Pagar"
            label={`Pays installment ${installment.Number}.`} onClick={() => onPay(installment)} />
        {/if}
        {#if installment.PaidAmount === 0}
          <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the installment."
            onClick={() => { loan.Installments = loan.Installments.filter((item) => item !== installment); renumber() }} />
        {/if}
      </div>
    {:else if column.id === 'number' && installmentState(installment, today) === LOAN_INSTALLMENT_OVERDUE}
      <span class="text-red-fg ff-bold" title={tr(stateLabels[LOAN_INSTALLMENT_OVERDUE])}>{installment.Number}</span>    {:else}
      {defaultContent}
    {/if}
  {/snippet}
</VTable>
