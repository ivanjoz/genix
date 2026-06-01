<script lang="ts">
import Layer from '$components/layers/Layer.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import Button from '$components/buttons/Button.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import DateInput from '$components/form/DateInput.svelte'
import { Core, tr } from '$core/store.svelte'
import { Loading, Notify, formatN, formatTime } from '$libs/helpers'
import {
  ExpensesScheduledService,
  postExpenseScheduled,
  getSchedulePeriods,
  expenseCategories,
  expenseCadences,
  currencyTypes,
  localizeOptions,
  packFrequency,
  unpackFrequency,
  frequencySummary,
  type IExpenseScheduled,
  type IExpense,
} from './expenses.svelte'

const schedules = new ExpensesScheduledService(true)

const categoryOptions = $derived(localizeOptions(expenseCategories))
const currencyOptions = $derived(localizeOptions(currencyTypes))
const cadenceOptions = $derived(localizeOptions(expenseCadences))
const categoriesMap = new Map(expenseCategories.map(c => [c.id, c]))

// Weekday options used when the cadence is "Weekly" (1=Mon .. 7=Sun).
const weekdayOptions = $derived([
  { id: 1, label: "Monday|Lunes" }, { id: 2, label: "Tuesday|Martes" }, { id: 3, label: "Wednesday|Miércoles" },
  { id: 4, label: "Thursday|Jueves" }, { id: 5, label: "Friday|Viernes" }, { id: 6, label: "Saturday|Sábado" },
  { id: 7, label: "Sunday|Domingo" },
].map(o => ({ id: o.id, name: tr(o.label) })))

let filterText = $state("")
let form = $state({} as IExpenseScheduled)
let layerView = $state(1)
// Cadence is edited as (cadence, day) and packed into form.Frequency on save.
let cadenceForm = $state({ cadence: 2, day: 1 })
let periods = $state([] as IExpense[])

const newSchedule = () => {
  form = { ss: 1, CurrencyType: 1, CategoryID: 1 } as IExpenseScheduled
  cadenceForm = { cadence: 2, day: 1 }
  periods = []
  layerView = 1
  Core.openSideLayer(1)
}

const openSchedule = async (schedule: IExpenseScheduled) => {
  form = { ...schedule }
  const { cadence, day } = unpackFrequency(schedule.Frequency)
  cadenceForm = { cadence: cadence || 2, day: day || 1 }
  periods = []
  layerView = 1
  Core.openSideLayer(1)
  // Lazily materialize and load this schedule's periods (see EXPENSES.md §5).
  try {
    periods = await getSchedulePeriods(schedule.ID)
  } catch { /* error already surfaced by the service */ }
}

const saveSchedule = async () => {
  if ((form.Amount || 0) <= 0) {
    Notify.failure(tr("The amount must be greater than 0.|El monto debe ser mayor a 0."))
    return
  }
  if (!form.StartDate) {
    Notify.failure(tr("Select a start date.|Seleccione una fecha de inicio."))
    return
  }
  // Pack the cadence dropdown + day field into the CDD Frequency code.
  form.Frequency = packFrequency(cadenceForm.cadence, cadenceForm.day)

  Loading.standard(tr("Saving schedule...|Guardando programación..."))
  try {
    const saved = await postExpenseScheduled(form)
    if (form.ID) {
      const current = schedules.recordsMap.get(form.ID)
      if (current) Object.assign(current, form)
    } else {
      form.ID = saved.ID
      schedules.addSavedRecords({ ...form })
    }
    Core.openSideLayer(0)
    // Clear the form so the list row deselects; defer on mobile so the slide-out isn't shown empty.
    if (Core.deviceType === 3) setTimeout(() => { form = {} as IExpenseScheduled; periods = [] }, 300)
    else { form = {} as IExpenseScheduled; periods = [] }
  } catch (error) {
    Notify.failure(error as string)
  } finally {
    Loading.remove()
  }
}

const filteredSchedules = $derived.by(() => {
  const text = filterText.toLowerCase()
  if (!text) return schedules.records
  return schedules.records.filter(e => e.Name?.toLowerCase().includes(text))
})

const columns: ITableColumn<IExpenseScheduled>[] = [
  { header: "ID", headerCss: "w-32", css: "text-center text-purple-600 px-6", getValue: e => e.ID },
  { header: "Name|Nombre", css: "px-6", getValue: e => e.Name },
  {
    header: "Category|Categoría", css: "px-6",
    getValue: e => tr(categoriesMap.get(e.CategoryID)?.label || ""),
  },
  { header: "Cadence|Cadencia", css: "px-6", getValue: e => frequencySummary(e.Frequency) },
  {
    header: "Amount|Monto", headerCss: "w-120", css: "text-right ff-mono px-6",
    getValue: e => `${formatN((e.Amount || 0) / 100, 2)} ${e.CurrencyType === 2 ? "USD" : "PEN"}`,
  },
  {
    header: "Start|Inicio", headerCss: "w-120", css: "whitespace-nowrap px-6",
    getValue: e => (e.StartDate ? formatTime(e.StartDate, "Y-m-d") : "") as string,
  },
]

const periodColumns: ITableColumn<IExpense>[] = [
  {
    header: "Period|Periodo", css: "px-6 whitespace-nowrap",
    getValue: e => (e.PeriodDate ? formatTime(e.PeriodDate, "Y-m-d") : "") as string,
  },
  {
    header: "Amount|Monto", headerCss: "w-110", css: "text-right ff-mono px-6",
    getValue: e => formatN((e.Amount || 0) / 100, 2) as string,
  },
  {
    header: "Paid|Pagado", headerCss: "w-110", css: "text-right ff-mono px-6",
    getValue: e => formatN((e.PaidAmount || 0) / 100, 2) as string,
  },
  {
    header: "Status|Estado", headerCss: "w-110", css: "px-6",
    // Derived from the lifecycle (ss) + paid amount, since PaymentStatus was removed.
    getValue: e => tr(e.ss === 2 ? "Paid|Pagado" : (e.PaidAmount || 0) > 0 ? "Partial|Parcial" : "Unpaid|Sin pagar"),
  },
]
</script>

<div class="flex items-center justify-between mb-6">
  <FilterInput bind:value={filterText} css="w-256" />
  <Button color="green" icon="icon-plus" name="New|Nuevo" onClick={newSchedule} />
</div>

<Layer type="content">
  <VTable css="w-full" maxHeight="calc(80vh - 13rem)"
    columns={columns} data={filteredSchedules}
    selected={form?.ID} isSelected={(e, id) => e.ID === id}
    onRowClick={(e) => openSchedule(e)}
  />
</Layer>

<Layer type="side" id={1} sideLayerSize={680}
  css="px-8 py-8 md:px-14 md:py-10"
  title={form?.ID ? (form?.Name || tr("Schedule|Programación")) : tr("New Schedule|Nueva Programación")}
  titleCss="h2 ff-bold"
  bind:selected={layerView}
  options={form?.ID ? [[1, tr("Details|Detalle")], [2, tr("Periods|Periodos")]] : [[1, tr("Details|Detalle")]]}
  onSave={layerView === 1 ? saveSchedule : undefined}
  onClose={() => { form = {} as IExpenseScheduled; periods = [] }}
>
  {#if layerView === 1}
    <div class="grid grid-cols-24 gap-10 mt-12">
      <Input bind:saveOn={form} save="Name" css="col-span-24 md:col-span-14" label="Name|Nombre" required={true} />
      <SearchSelect bind:saveOn={form} save="CategoryID" css="col-span-24 md:col-span-10"
        label="Category|Categoría" keyId="id" keyName="name" options={categoryOptions} required={true} />
      <Input bind:saveOn={form} save="Description" css="col-span-24" label="Description|Descripción" />
      <SearchSelect bind:saveOn={form} save="CurrencyType" css="col-span-24 md:col-span-8"
        label="Currency|Moneda" keyId="id" keyName="name" options={currencyOptions} required={true} />
      <Input bind:saveOn={form} save="Amount" type="number" baseDecimals={2}
        inputCss="ff-mono text-right" css="col-span-24 md:col-span-8" label="Amount|Monto" required={true} />
      <div class="col-span-24 md:col-span-8"></div>

      <!-- Cadence dropdown + day combine into the packed Frequency code on save. -->
      <SearchSelect bind:saveOn={cadenceForm} save="cadence" css="col-span-24 md:col-span-12"
        label="Cadence|Cadencia" keyId="id" keyName="name" options={cadenceOptions} required={true}
        onChange={() => { cadenceForm = { ...cadenceForm, day: 1 } }} />
      {#if cadenceForm.cadence === 1}
        <SearchSelect bind:saveOn={cadenceForm} save="day" css="col-span-24 md:col-span-12"
          label="Weekday|Día de la semana" keyId="id" keyName="name" options={weekdayOptions} required={true} />
      {:else}
        <Input bind:saveOn={cadenceForm} save="day" type="number" css="col-span-24 md:col-span-12"
          label="Day of month|Día del mes" required={true} />
      {/if}

      <DateInput bind:saveOn={form} save="StartDate" css="col-span-24 md:col-span-12" label="Start Date|Fecha Inicio" />
      <DateInput bind:saveOn={form} save="EndDate" css="col-span-24 md:col-span-12" label="End Date (optional)|Fecha Fin (opcional)" />
    </div>
  {/if}

  {#if layerView === 2}
    <div class="mt-12">
      {#if periods.length === 0}
        <div class="text-slate-500 py-16 text-center">{tr("No periods generated yet.|Aún no hay periodos generados.")}</div>
      {:else}
        <VTable css="w-full" maxHeight="calc(80vh - 16rem)" columns={periodColumns} data={periods} />
      {/if}
    </div>
  {/if}
</Layer>
