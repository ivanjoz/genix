<script lang="ts">
import Layer from '$components/layers/Layer.svelte'
import VTable from '$components/vTable/VTable.svelte'
import RecordByIDText from '$components/misc/RecordByIDText.svelte'
import type { ITableColumn } from '$components/vTable/types'
import Button from '$components/buttons/Button.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import DateInput from '$components/form/DateInput.svelte'
import Checkbox from '$components/form/Checkbox.svelte'
import OptionsStrip from '$components/navigation/OptionsStrip.svelte'
import LoadingBar from '$components/misc/LoadingBar.svelte'
import { onMount, untrack } from 'svelte'
import { Core, tr } from '$core/store.svelte'
import { Loading, Notify, formatN, formatTime } from '$libs/helpers'
import { CajasService, getCashBankMovementByID, type ICashBankMovement } from '../cajas/cajas.svelte'
import {
  ExpensesService,
  postExpense,
  postExpensePayment,
  expenseCategories,
  currencyTypes,
  expenseStatusTabs,
  ExpenseStatus,
  localizeOptions,
  type IExpense,
  type IExpensePayment,
} from './expenses.svelte'

const cajas = new CajasService()	

// Localized option lists (re-resolve when the language switches).
const categoryOptions = $derived(localizeOptions(expenseCategories))
const currencyOptions = $derived(localizeOptions(currencyTypes))

const categoriesMap = new Map(expenseCategories.map(c => [c.id, c]))

// Default today as UnixDay (local-offset-adjusted days since epoch).
const todayUnixDay = Math.floor((Date.now() - new Date().getTimezoneOffset() * 60000) / 86400000)

let filterText = $state("")
let statusFilter = $state(ExpenseStatus.TODOS)
let records = $state([] as IExpense[])
let isLoading = $state(false)
let form = $state({} as IExpense)
let layerView = $state(1)
let paymentForm = $state({} as IExpensePayment)

// A fully-paid expense (ss=2) is locked: its detail fields can't be edited and no further
// payments are allowed (the backend enforces both rules authoritatively).
const isPaid = $derived(form.ss === 2)
// Outstanding balance the next payment must not exceed.
const pendingAmount = $derived(Math.max(0, (form.Amount || 0) - (form.PaidAmount || 0)))

// Default source register = the first cash register whose currency matches the expense
// (the backend requires the register and expense currencies to match), else the first one.
const defaultCashBankID = $derived(
  (cajas.Cajas.find(c => c.CurrencyType === form.CurrencyType) || cajas.Cajas[0])?.ID
)

// Preselect the default register once the cajas have loaded and the user hasn't picked one.
$effect(() => {
  const id = defaultCashBankID
  untrack(() => {
    if (id && form.ID && !paymentForm.CashBankID) paymentForm = { ...paymentForm, CashBankID: id }
  })
})

// For a fully-paid expense the payment form is replaced by the list of settling payments
// (CashBankMovements with DocumentID = expense ID), loaded when the Payment tab is shown.
let payments = $state([] as ICashBankMovement[])
let paymentsLoading = $state(false)

const loadPayments = async (expenseID: number) => {
  paymentsLoading = true
  try {
    // Cache keyed by the expense's `upd` watermark: registering a payment advances `upd`,
    // so the next load misses the cache and re-fetches the settling movements.
    payments = await getCashBankMovementByID({ documentID: expenseID, updated: form.upd })
  } finally {
    paymentsLoading = false
  }
}

$effect(() => {
  const expenseID = form.ID
  // Load the settling movements whenever anything has been paid (partial or full), and
  // re-run when `upd` advances (a new payment) so the route cache re-fetches.
  const hasPayments = (form.PaidAmount || 0) > 0
  void form.upd
  const showPayments = hasPayments && layerView === 2 && !!expenseID
  untrack(() => {
    if (showPayments) void loadPayments(expenseID)
    else payments = []
  })
})

// Each tab is its own server-side delta query (?status=<code>); switching tabs spins up
// a fresh service so the cache key (and its IDsToRemove eviction) stays per-status.
let queryRequestID = 0
const queryExpenses = async (status: number) => {
  const requestID = ++queryRequestID
  isLoading = true
  try {
    const service = new ExpensesService(status)
    // fetchCached honors the 30s TTL — re-clicking a tab within the window skips the server.
    await service.fetchCached()
    if (requestID !== queryRequestID) return // a newer tab switch superseded this query
    records = service.records
  } finally {
    if (requestID === queryRequestID) isLoading = false
  }
}

onMount(() => { void queryExpenses(statusFilter) })

// Does an expense with lifecycle `ss` belong to the given status tab?
const belongsToTab = (ss: number, tab: number) => {
  if (tab === ExpenseStatus.PEND_PAGO) return ss === 1
  if (tab === ExpenseStatus.PAGADOS) return ss === 2
  return ss === 1 || ss === 2 // Todos = all live
}

// Patch the visible table from a single updated expense (e.g. after a payment) instead of
// refetching: update the row in place, insert it if newly relevant, or drop it if it left the tab.
const applyExpenseToList = (updated: IExpense) => {
  const index = records.findIndex(e => e.ID === updated.ID)
  if (belongsToTab(updated.ss, statusFilter)) {
    if (index >= 0) {
      const next = [...records]
      next[index] = { ...next[index], ...updated }
      records = next
    } else {
      records = [updated, ...records]
    }
  } else if (index >= 0) {
    records = records.filter(e => e.ID !== updated.ID)
  }
}

const newExpense = () => {
  form = { ss: 1, ExpenseScheduledID: 0, CurrencyType: 1, CategoryID: 10, Date: todayUnixDay, DueDate: todayUnixDay } as IExpense
  paymentForm = {} as IExpensePayment
  layerView = 1
  Core.openSideLayer(1)
}

const openExpense = (expense: IExpense) => {
  form = { ...expense }
  // Default the payment date to today; the source register is preselected by $effect.
  paymentForm = { ExpenseID: expense.ID, IsFullyPaid: false, Date: todayUnixDay } as IExpensePayment
  // Keep the currently-selected tab (Detalle/Pago) when navigating between rows; both tabs
  // exist for any saved expense, so there's no invalid-tab case here.
  Core.openSideLayer(1)
}

// Save the expense detail fields (create or edit). Payment state (PaidAmount, ss) is server-maintained.
const saveExpense = async () => {
  if ((form.Amount || 0) <= 0) {
    Notify.failure(tr("The amount must be greater than 0.|El monto debe ser mayor a 0."))
    return
  }
  Loading.standard(tr("Saving expense...|Guardando gasto..."))
  try {
    const saved = await postExpense(form)
    // Merge the server-set fields (ID, ss, Created) onto the edited form for the row.
    const savedRow = { ...form, ...saved }
    Core.openSideLayer(0)
    // Clear the form so the list row deselects; defer on mobile so the slide-out isn't shown empty.
    if (Core.deviceType === 3) setTimeout(() => { form = {} as IExpense }, 300)
    else form = {} as IExpense
    // Patch the table locally from the result instead of refetching: update/insert the row
    // when it belongs to the active tab, or drop it if it doesn't.
    applyExpenseToList(savedRow)
  } catch (error) {
    Notify.failure(error as string)
  } finally {
    Loading.remove()
  }
}

// Record a payment against the open expense via POST.expense-payment.
const registerPayment = async () => {
  if ((paymentForm.Amount || 0) <= 0) {
    Notify.failure(tr("Enter a payment amount greater than 0.|Ingrese un monto de pago mayor a 0."))
    return
  }
  if (!paymentForm.CashBankID) {
    Notify.failure(tr("Select a source cash register.|Seleccione una caja de origen."))
    return
  }
  // The payment cannot exceed the outstanding balance (also enforced on the backend).
  if (paymentForm.Amount > pendingAmount) {
    Notify.failure(tr("The payment amount cannot exceed the pending amount.|El monto del pago no puede ser mayor al monto pendiente."))
    return
  }
  // The backend authoritatively enforces the register currency match and computes the
  // resulting balance (rejecting the payment only if it would go negative).
  const payload: IExpensePayment = {
    ExpenseID: form.ID,
    CashBankID: paymentForm.CashBankID,
    Amount: paymentForm.Amount,
    Date: paymentForm.Date || todayUnixDay,
    IsFullyPaid: !!paymentForm.IsFullyPaid,
  }

  Loading.standard(tr("Recording payment...|Registrando pago..."))
  try {
    const result = await postExpensePayment(payload)
    // Reflect the server-updated paid amount / lifecycle status in the open form. `upd` must
    // advance too so the payments cache (keyed by it) invalidates and re-fetches below.
    form = { ...form, PaidAmount: result.PaidAmount, ss: result.ss, upd: result.upd }
    paymentForm = { ExpenseID: form.ID, IsFullyPaid: false } as IExpensePayment
    Notify.success(tr("Payment recorded.|Pago registrado."))
    // Patch the table locally from the result (ID + new status): a fully-paid row leaves
    // the Pend. Pago tab, joins Pagados, etc. — no extra server fetch.
    applyExpenseToList({ ...form, ID: result.ID, ss: result.ss, PaidAmount: result.PaidAmount })
  } catch (error) {
    Notify.failure(error as string)
  } finally {
    Loading.remove()
  }
}

// Status chip derived from the lifecycle (ss) + paid amount: ss=2 paid, partial when
// something has been paid, otherwise unpaid.
const statusChip = (e: IExpense) => {
  if (e.ss === 2) return { label: tr("Paid|Pagado"), color: "bg-green-100 text-green-700" }
  if ((e.PaidAmount || 0) > 0) return { label: tr("Partial|Parcial"), color: "bg-amber-100 text-amber-700" }
  return { label: tr("Unpaid|Sin pagar"), color: "bg-red-100 text-red-700" }
}

const filteredExpenses = $derived.by(() => {
  const text = filterText.toLowerCase()
  if (!text) return records
  return records.filter(e => e.Name?.toLowerCase().includes(text))
})

const columns: ITableColumn<IExpense>[] = [
  { header: "ID", headerCss: "w-32", css: "text-center text-purple-600 px-6", getValue: e => e.ID },
  {
    header: "Registered|Registro", headerCss: "w-106", css: "whitespace-nowrap px-6 ff-mono text-sm",
    getValue: e => (e.Created ? formatTime(e.Created, "Y-m-d") : "") as string,
  },
  { header: "Status|Estado", id: "status", headerCss: "w-110", css: "px-6", getValue: e => e.ss },
  { header: "Name|Nombre", css: "px-6", getValue: e => e.Name },
  {
    header: "Category|Categoría", css: "px-6",
    getValue: e => tr(categoriesMap.get(e.CategoryID)?.label || ""),
  },
  {
    header: "Due Date|Vencimiento", headerCss: "w-120", css: "whitespace-nowrap px-6",
    getValue: e => (e.DueDate ? formatTime(e.DueDate, "Y-m-d") : "") as string,
  },
  {
    header: "Amount|Monto", headerCss: "w-120", css: "text-right ff-mono px-6",
    getValue: e => `${formatN((e.Amount || 0) / 100, 2)} ${e.CurrencyType === 2 ? "USD" : "PEN"}`,
  },
  {
    header: "Paid|Pagado", headerCss: "w-110", css: "text-right ff-mono px-6",
    getValue: e => formatN((e.PaidAmount || 0) / 100, 2) as string,
  },
]

// Columns for the read-only payments table shown when a paid expense is opened (see Payment tab).
const paymentColumns: ITableColumn<ICashBankMovement>[] = [
  {
    header: "Date|Fecha", headerCss: "w-110", css: "ff-mono px-6",
    getValue: e => formatTime(e.Date, "d-m-Y") as string,
  },
  {
    header: "Source Register|Caja de Origen", css: "px-6",
    getValue: e => cajas.CajasMap.get(e.CashBankID)?.Name || "-",
  },
  {
    // id triggers the cellRenderer snippet so we can mount RecordByIDText per row.
    id: "paymentUsuario", header: "User|Usuario", headerCss: "w-120", css: "px-6",
    getValue: e => e.CreatedBy,
  },
  {
    // Outflow amounts are stored negative; show the positive paid value.
    header: "Amount|Monto", headerCss: "w-110", css: "text-right ff-mono px-6",
    getValue: e => formatN(Math.abs(e.Amount || 0) / 100, 2) as string,
  },
]
</script>

<!-- Read-only list of settling movements; shared by the fully-paid and partial Payment views.
     Declared at markup top level (not inside <Layer>) so it isn't read as a component prop. -->
{#snippet paymentsTable()}
  {#if paymentsLoading}
    <div class="py-8 fx-c"><LoadingBar label={tr("Loading payments...|Cargando pagos...")} /></div>
  {:else}
    <VTable css="w-full" maxHeight="50vh"
      columns={paymentColumns} data={payments}
      emptyMessage="No payments found.|No se encontraron pagos."
    >
      {#snippet cellRenderer(record: ICashBankMovement, col: ITableColumn<ICashBankMovement>)}
        {#if col.id === "paymentUsuario"}
          <RecordByIDText apiRoute="usuarios-ids" recordID={record.CreatedBy} placeholder="-" />
        {/if}
      {/snippet}
    </VTable>
  {/if}
{/snippet}

<div class="flex items-center justify-between mb-6">
  <div class="flex items-center gap-12">
    <FilterInput bind:value={filterText} css="w-256" />
    <OptionsStrip selected={statusFilter}
      options={expenseStatusTabs.map(o => ({ id: o.id, name: tr(o.label) }))}
      keyId="id" keyName="name"
      onSelect={(e) => {
        const next = e.id as number
        if (next === statusFilter) return
        statusFilter = next
        void queryExpenses(next)
      }}
    />
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" onClick={newExpense} />
</div>

<Layer type="content">
  {#if isLoading}
    <div class="p-8 min-h-240 w-full fx-c rounded-md bg-gray-50">
      <LoadingBar label={tr("Loading expenses...|Cargando gastos...")} />
    </div>
  {:else}
    <VTable css="w-full" maxHeight="calc(80vh - 13rem)"
      columns={columns} data={filteredExpenses}
      selected={form?.ID} isSelected={(e, id) => e.ID === id}
      onRowClick={(e) => openExpense(e)}
    >
      {#snippet cellRenderer(record: IExpense, col: ITableColumn<IExpense>)}
        {#if col.id === "status"}
          {@const chip = statusChip(record)}
          <span class="px-8 py-2 rounded-full {chip.color}">{chip.label}</span>
        {/if}
      {/snippet}
    </VTable>
  {/if}
</Layer>

<Layer type="side" id={1} sideLayerSize={640}
  css="px-8 py-8 md:px-14 md:py-10"
  title={form?.ID ? (form?.Name || tr("Expense|Gasto")) : tr("New Expense|Nuevo Gasto")}
  titleCss="h2 ff-bold"
  bind:selected={layerView}
  options={form?.ID ? [[1, tr("Details|Detalle")], [2, tr("Payment|Pago")]] : [[1, tr("Details|Detalle")]]}
  onSave={layerView === 1 && !isPaid ? saveExpense : undefined}
  onClose={() => { form = {} as IExpense }}
>
  {#if layerView === 1}
    {#if isPaid}
      <!-- A fully-paid expense is locked from edits; surface why the fields are read-only. -->
      <div class="col-span-24 mt-12 mb-2 px-10 py-6 rounded bg-green-50 text-green-700 text-sm">
        {tr("This expense is fully paid and can no longer be edited.|Este gasto está pagado y ya no puede editarse.")}
      </div>
    {/if}
    <div class="grid grid-cols-24 gap-10 mt-12">
      <Input bind:saveOn={form} save="Name" css="col-span-24 md:col-span-14" label="Name|Nombre" required={true} disabled={isPaid} />
      <SearchSelect bind:saveOn={form} save="CategoryID" css="col-span-24 md:col-span-10"
        label="Category|Categoría" keyId="id" keyName="name" options={categoryOptions} required={true} disabled={isPaid} />
      <Input bind:saveOn={form} save="Description" css="col-span-24" label="Description|Descripción" disabled={isPaid} />
      <SearchSelect bind:saveOn={form} save="CurrencyType" css="col-span-24 md:col-span-8"
        label="Currency|Moneda" keyId="id" keyName="name" options={currencyOptions} required={true} disabled={isPaid} />
      <Input bind:saveOn={form} save="Amount" type="number" baseDecimals={2}
        inputCss="ff-mono text-center pr-8" css="col-span-24 md:col-span-16" label="Amount|Monto" required={true} disabled={isPaid} />
      <DateInput bind:saveOn={form} save="Date" css="col-span-24 md:col-span-12" label="Date|Fecha" disabled={isPaid} />
      <DateInput bind:saveOn={form} save="DueDate" css="col-span-24 md:col-span-12" label="Due Date|Vencimiento" disabled={isPaid} />
    </div>
  {/if}

  {#if layerView === 2}
    <div class="grid grid-cols-24 gap-10 mt-12">
      <!-- Current payment state summary -->
      <div class="col-span-24 grid grid-cols-3 gap-10 mb-4">
        <div class="bg-slate-100 rounded py-8 text-center">
          <div class="text-sm text-slate-600">{tr("Amount|Monto")}</div>
          <div class="ff-mono ff-bold">{formatN((form.Amount || 0) / 100, 2)}</div>
        </div>
        <div class="bg-slate-100 rounded py-8 text-center">
          <div class="text-sm text-slate-600">{tr("Paid|Pagado")}</div>
          <div class="ff-mono ff-bold">{formatN((form.PaidAmount || 0) / 100, 2)}</div>
        </div>
        <div class="bg-slate-100 rounded py-8 text-center">
          <div class="text-sm text-slate-600">{tr("Pending|Pendiente")}</div>
          <div class="ff-mono ff-bold">{formatN(Math.max(0, (form.Amount || 0) - (form.PaidAmount || 0)) / 100, 2)}</div>
        </div>
      </div>

      {#if isPaid}
        <!-- Paid expense: the form is replaced by the list of payments that settled it. -->
        <div class="col-span-24">
          {@render paymentsTable()}
        </div>
      {:else}
        <SearchSelect bind:saveOn={paymentForm} save="CashBankID" css="col-span-24 md:col-span-12"
          label="Source Register|Caja de Origen" keyId="ID" keyName="Name" options={cajas.Cajas} required={true} />
        <Input bind:saveOn={paymentForm} save="Amount" type="number" baseDecimals={2}
          inputCss="ff-mono text-right" css="col-span-24 md:col-span-12" label="Payment Amount|Monto del Pago" required={true} />
        <DateInput bind:saveOn={paymentForm} save="Date" css="col-span-24 md:col-span-12" label="Payment Date|Fecha del Pago" />
        <div class="col-span-24 md:col-span-12 flex items-end">
          <Checkbox bind:saveOn={paymentForm} save="IsFullyPaid" label="Is Fully Paid|Pagado Completo" />
        </div>
        <div class="col-span-24 mt-8">
          <Button color="blue" icon="icon-[fa--check]" name="Register Payment|Registrar Pago" onClick={registerPayment} />
        </div>

        {#if (form.PaidAmount || 0) > 0}
          <!-- Partially-paid expense: list the payments made so far below the form. -->
          <div class="col-span-24 mt-10">
            <div class="text-sm text-slate-600 ff-bold mb-4">{tr("Payments made|Pagos realizados")}</div>
            {@render paymentsTable()}
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</Layer>
