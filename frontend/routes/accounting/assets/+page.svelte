<script lang="ts">
import { useUI } from '@genix/ui'
import Button from '$components/buttons/Button.svelte'
import Checkbox from '$components/form/Checkbox.svelte'
import DateInput from '$components/form/DateInput.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import Layer from '$components/layers/Layer.svelte'
import Modal from '$components/layers/Modal.svelte'
import T from '$components/misc/T.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ExcelTableColumn } from '@genix/ui/excel'
import Page from '$domain/Page.svelte'
import { tr } from '$core/store.svelte'
import { ConfirmWarn, formatN, formatTime, Loading, Notify } from '$libs/helpers'
import { WarehousesService } from '../../business/branches-warehouses/branches-warehouses.svelte'
import { ClientProviderService, ClientProviderType } from '../../business/customers/customers.svelte'
import { SupplyMaterialService } from '../../logistics/supplies-materials/supply-material.svelte'
import {
  assetBookValue, assetDepreciatedPercent, assetPaymentLabels, assetPendingAmount,
  assetRemainingMonths, assetStatusLabels, assetUnitLabel, canDisposeAsset, canPayAsset,
  depreciationSchedule, type IAsset,
} from './assets'
import {
  AssetsService, getAssetDepreciation, postAssetAcquisition, postAssetPayment,
  putAssetDisposal, runAssetDepreciation, type IAssetAcquisition, type IAssetPayment,
  type IDepreciationEntry,
} from './assets.svelte'
import { CajasService } from '../../finance/cash-banks/cajas.svelte'

const ui = useUI()

// Side layers are 1 (acquisition) and 2 (asset detail); the payment dialog is its own handle.
const PAYMENT_MODAL_ID = 11

const assets = new AssetsService(true)
const warehouses = new WarehousesService()
const providers = new ClientProviderService(ClientProviderType.PROVIDER, true)
// Only supplies with a depreciation term can be acquired as assets — the rest are stock.
const supplies = new SupplyMaterialService(true)
// An asset is paid from a cash register, on its own account rather than through an expense.
const cajas = new CajasService()

let filterText = $state("")
let acquisitionForm = $state({} as IAssetAcquisition)
// Serial numbers are typed as one per line: entering any makes the acquisition per-unit.
// Held in an object because Input writes through saveOn/save rather than bind:value.
let serialsInput = $state({ text: "" })
let selectedAsset = $state(null as IAsset | null)
let depreciationEntries = $state([] as IDepreciationEntry[])
let paymentForm = $state({} as IAssetPayment)

const depreciableSupplies = $derived(
  (supplies.records || []).filter(supply => (supply.DepreciationMonths || 0) > 0),
)
const supplyNameByID = $derived(
  new Map((supplies.records || []).map(supply => [supply.ID, supply.Name])),
)
const warehouseOptions = $derived(warehouses.Almacenes || [])
const currencyOptions = $derived([
  { id: 1, name: "PEN" },
  { id: 2, name: "USD" },
])

// Depreciation is generated lazily, so the register is only up to date once a run has
// happened. Doing it on load keeps the page honest without a cron to maintain.
$effect(() => {
  runAssetDepreciation()
    .then(result => {
      if (result?.PostedEntries > 0) {
        Notify.success(tr(
          `Posted ${result.PostedEntries} depreciation entries.|Se registraron ${result.PostedEntries} asientos de depreciación.`,
        ))
        assets.fetch()
      }
    })
    .catch(error => { console.error("asset depreciation run failed", error) })
})

const assetColumns: ExcelTableColumn<IAsset>[] = [
  {
    header: "ID",
    field: "ID",
    css: "c-blue text-center",
    headerCss: "w-40", headerInnerCss: "min-w-36",
    getValue: (asset) => asset.ID || "",
    mobile: { order: 1, css: "col-span-6 ff-bold", icon: "[fa--cube]" },
  },
  {
    header: "Asset|Activo",
    highlight: true,
    getValue: (asset) => supplyNameByID.get(asset.ProductID) || `#${asset.ProductID}`,
    mobile: { order: 2, css: "col-span-18", render: (asset) => `<strong>${supplyNameByID.get(asset.ProductID) || ""}</strong>` },
  },
  {
    header: "Serial / Qty|Serie / Cant.",
    getValue: (asset) => assetUnitLabel(asset),
    mobile: { order: 3, css: "col-span-12", labelLeft: "Serie:" },
  },
  {
    header: "Warehouse|Almacén",
    getValue: (asset) => warehouses.AlmacenesMap.get(asset.WarehouseID)?.Name || "",
    mobile: { order: 4, css: "col-span-12", labelLeft: "Almacén:" },
  },
  {
    header: "Acquired|Adquirido",
    getValue: (asset) => formatTime(asset.AcquisitionDate, "d-m-Y") as string,
    mobile: { order: 5, css: "col-span-12", labelLeft: "Adquirido:" },
  },
  {
    header: "Value|Valor",
    css: "text-right",
    getValue: (asset) => formatN(asset.AcquisitionValue / 100, 2),
    mobile: { order: 6, css: "col-span-12", labelLeft: "Valor:" },
  },
  {
    header: "Book Value|Valor en Libros",
    css: "text-right",
    getValue: (asset) => formatN(assetBookValue(asset) / 100, 2),
    mobile: { order: 7, css: "col-span-12", labelLeft: "En libros:" },
  },
  {
    header: "Depreciated|Depreciado",
    css: "text-right",
    getValue: (asset) => `${assetDepreciatedPercent(asset)}%`,
    mobile: { order: 8, css: "col-span-12", labelLeft: "Depreciado:" },
  },
  {
    header: "Owed|Adeudado",
    css: "text-right",
    getValue: (asset) => assetPendingAmount(asset) > 0 ? formatN(assetPendingAmount(asset) / 100, 2) : "—",
    mobile: { order: 9, css: "col-span-12", labelLeft: "Adeudado:" },
  },
  {
    header: "Payment|Pago",
    getValue: (asset) => tr(assetPaymentLabels[asset.PaymentStatus] || ""),
    mobile: { order: 10, css: "col-span-12", labelLeft: "Pago:" },
  },
  {
    header: "Status|Estado",
    getValue: (asset) => tr(assetStatusLabels[asset.ss] || ""),
    mobile: { order: 11, css: "col-span-12", labelLeft: "Estado:" },
  },
]

const openAcquisitionLayer = () => {
  acquisitionForm = {
    ProductID: 0, SerialNumbers: [], Quantity: 1, WarehouseID: 0,
    AcquisitionDate: 0, DueDate: 0, Name: "", Description: "", SupplierID: 0,
    CurrencyType: 1, AcquisitionValue: 0, PurchaseAmount: 0, DepreciationMonths: 0,
  }
  serialsInput = { text: "" }
  ui.openSideLayer(1)
}

const onAcquire = async () => {
  if (!acquisitionForm.ProductID) {
    Notify.failure(tr("Select the supply this asset is.|Seleccione el insumo que corresponde al activo."))
    return
  }
  if (!acquisitionForm.WarehouseID) {
    Notify.failure(tr("Select a warehouse.|Seleccione un almacén."))
    return
  }
  if (!acquisitionForm.AcquisitionValue) {
    Notify.failure(tr("The asset value must be greater than 0.|El valor del activo debe ser mayor a 0."))
    return
  }

  const serialNumbers = (serialsInput.text || "")
    .split("\n").map(serial => serial.trim()).filter(Boolean)

  Loading.standard(tr("Registering asset...|Registrando activo..."))
  try {
    await postAssetAcquisition({ ...acquisitionForm, SerialNumbers: serialNumbers })
  } catch (error) {
    Notify.failure(error as string)
    Loading.remove()
    return
  }
  Loading.remove()
  await assets.fetch()
  ui.openSideLayer(0)
}

const openDepreciationLayer = async (asset: IAsset) => {
  selectedAsset = asset
  depreciationEntries = []
  ui.openSideLayer(2)
  try {
    depreciationEntries = await getAssetDepreciation(asset.ID)
  } catch (error) {
    Notify.failure(error as string)
  }
}

// The dialog opens pre-filled with the outstanding balance, which is the payment the user
// makes in almost every case.
const openPaymentModal = () => {
  if (!selectedAsset) return
  paymentForm = {
    AssetID: selectedAsset.ID, CashBankID: 0,
    Amount: assetPendingAmount(selectedAsset), Date: 0, IsFullyPaid: false,
  }
  ui.openModal(PAYMENT_MODAL_ID)
}

const registerAssetPayment = async () => {
  if (!selectedAsset) return
  if ((paymentForm.Amount || 0) <= 0) {
    Notify.failure(tr("Enter a payment amount greater than 0.|Ingrese un monto de pago mayor a 0."))
    return
  }
  if (!paymentForm.CashBankID) {
    Notify.failure(tr("Select the source register.|Seleccione la caja de origen."))
    return
  }

  Loading.standard(tr("Registering payment...|Registrando pago..."))
  try {
    const updated = await postAssetPayment({ ...paymentForm, AssetID: selectedAsset.ID })
    selectedAsset = { ...selectedAsset, ...updated }
    await assets.fetch()
    // Only on success: a failed post keeps the dialog open with the input intact.
    ui.closeModal(PAYMENT_MODAL_ID)
  } catch (error) {
    Notify.failure(error as string)
  } finally {
    Loading.remove()
  }
}

const onDispose = () => {
  if (!selectedAsset) return
  const asset = selectedAsset
  ConfirmWarn(
    tr("Dispose Asset|Dar de Baja"),
    tr(
      `Dispose "${assetUnitLabel(asset)}"? It leaves the warehouse and stops depreciating.`
      + `|¿Dar de baja "${assetUnitLabel(asset)}"? Sale del almacén y deja de depreciarse.`,
    ),
    "SI", "NO",
    async () => {
      Loading.standard(tr("Disposing...|Dando de baja..."))
      try {
        await putAssetDisposal({ AssetID: asset.ID, DisposalDate: 0 })
      } catch (error) {
        Notify.failure(error as string)
        Loading.remove()
        return
      }
      Loading.remove()
      await assets.fetch()
      ui.openSideLayer(0)
    },
  )
}
</script>

<Page title="Assets|Activos">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <FilterInput label="Filter assets|Filtrar activos"
      css="w-full md:w-200 col-span-9"
      icon="icon-[fa--search]"
      bind:value={filterText}
    />
    <Button name="Acquire|Adquirir"
      label="Shows the form to register a new fixed asset."
      color="green"
      icon="icon-[fa--plus]"
      hideNameOnMobile
      css="col-span-3 ml-auto"
      onClick={openAcquisitionLayer}
    />
  </div>

  <Layer type="content">
    <VTable
      columns={assetColumns}
      data={assets.records}
      {filterText}
      selected={selectedAsset?.ID}
      isSelected={(asset, selectedID) => asset.ID === selectedID}
      getFilterContent={(asset) =>
        `${supplyNameByID.get(asset.ProductID) || ""} ${asset.SerialNumber || ""}`}
      onRowClick={(asset) => { openDepreciationLayer(asset) }}
      mobileCardCss="mb-2"
    />
  </Layer>

  <!-- Acquisition. An asset is a supply with a depreciation term, so the form starts there. -->
  <Layer
    type="side"
    id={1}
    sideLayerSize={720}
    css="px-8 py-8 md:px-16 md:py-10"
    title="Nuevo Activo"
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    onClose={() => { acquisitionForm = {} as IAssetAcquisition; serialsInput = { text: "" } }}
    onSave={() => { onAcquire() }}
  >
    <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16" aria-label="Asset Acquisition Form">
      <SearchSelect label="Supply / Material|Insumo o Material"
        saveOn={acquisitionForm}
        css="col-span-24 md:col-span-14"
        save="ProductID"
        keyId="ID"
        keyName="Name"
        options={depreciableSupplies}
      />
      <SearchSelect label="Warehouse|Almacén"
        saveOn={acquisitionForm}
        css="col-span-24 md:col-span-10"
        save="WarehouseID"
        keyId="ID"
        keyName="Name"
        options={warehouseOptions}
      />
      <DateInput label="Acquisition Date|Fecha de Adquisición"
        bind:saveOn={acquisitionForm}
        css="col-span-24 md:col-span-12"
        save="AcquisitionDate"
      />
      <SearchSelect label="Supplier|Proveedor"
        saveOn={acquisitionForm}
        css="col-span-24 md:col-span-12"
        save="SupplierID"
        keyId="ID"
        keyName="Name"
        options={providers.records || []}
      />
      <Input label="Book Value (per unit)|Valor en Libros (por unidad)"
        saveOn={acquisitionForm}
        css="col-span-12 md:col-span-8"
        save="AcquisitionValue"
        type="number"
        baseDecimals={2}
      />
      <Input label="Purchase Amount (per unit)|Monto de Compra (por unidad)"
        saveOn={acquisitionForm}
        css="col-span-12 md:col-span-8"
        save="PurchaseAmount"
        type="number"
        baseDecimals={2}
      />
      <SearchSelect label="Currency|Moneda"
        saveOn={acquisitionForm}
        css="col-span-24 md:col-span-8"
        save="CurrencyType"
        keyId="id"
        keyName="name"
        options={currencyOptions}
      />
      <DateInput label="Payment Due|Vencimiento del Pago"
        bind:saveOn={acquisitionForm}
        css="col-span-24 md:col-span-12"
        save="DueDate"
      />
    </div>

    <!-- Purchase 0 with a value above 0 is the donated case: nothing is owed, but it is
         still an asset on the books and still depreciates. -->
    <div class="mt-8 text-sm c-gray">
      <T text="Leave the purchase amount at 0 for a donated or contributed asset — nothing is owed, but it still has a book value and still depreciates.|Deje el monto de compra en 0 para un activo donado o aportado: no se debe nada, pero igual tiene valor en libros y se deprecia." />
    </div>

    <div class="mt-16" aria-label="Asset units">
      <div class="h4 ff-bold mb-6">
        <T text="Units|Unidades" />
      </div>
      <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10">
        <Input label="Quantity|Cantidad"
          saveOn={acquisitionForm}
          css="col-span-12 md:col-span-6"
          save="Quantity"
          type="number"
        />
        <Input label="Serial Numbers (one per line)|Números de Serie (uno por línea)"
          saveOn={serialsInput}
          save="text"
          css="col-span-24 md:col-span-18"
        />
      </div>
      <div class="mt-6 text-sm c-gray">
        <T text="Enter serials to track each unit as its own asset; leave empty to register the whole lot as one.|Ingrese series para registrar cada unidad como un activo independiente; déjelo vacío para registrar todo el lote como uno solo." />
      </div>
    </div>
  </Layer>

  <!-- Depreciation schedule for the selected asset. -->
  <Layer
    type="side"
    id={2}
    sideLayerSize={620}
    css="px-8 py-8 md:px-16 md:py-10"
    title={selectedAsset ? (supplyNameByID.get(selectedAsset.ProductID) || "Activo") : "Activo"}
    titleCss="h2 mb-6"
    contentCss="px-0 md:px-0"
    onClose={() => { selectedAsset = null; depreciationEntries = [] }}
  >
    {#if selectedAsset}
      <div class="grid grid-cols-24 gap-10 mt-6 md:mt-16">
        <div class="col-span-12">
          <div class="text-[15px] leading-[16px] stat-label"><T text="Acquisition Value|Valor de Adquisición" /></div>
          <div class="h3 ff-bold">{formatN(selectedAsset.AcquisitionValue / 100, 2)}</div>
        </div>
        <div class="col-span-12">
          <div class="text-[15px] leading-[16px] stat-label"><T text="Book Value|Valor en Libros" /></div>
          <div class="h3 ff-bold">{formatN(assetBookValue(selectedAsset) / 100, 2)}</div>
        </div>
        <div class="col-span-12">
          <div class="text-[15px] leading-[16px] stat-label"><T text="Depreciated|Depreciado" /></div>
          <div class="h4">{assetDepreciatedPercent(selectedAsset)}%</div>
        </div>
        <div class="col-span-12">
          <div class="text-[15px] leading-[16px] stat-label"><T text="Months Remaining|Meses Restantes" /></div>
          <div class="h4">{assetRemainingMonths(selectedAsset)} / {selectedAsset.DepreciationMonths}</div>
        </div>
      </div>

      <!-- Purchase and payment. An asset is not an expense, so this is settled here rather
           than in the expense register; only depreciation reaches that table. -->
      <div class="mt-16" aria-label="Asset purchase and payment">
        {#if (selectedAsset.PurchaseAmount || 0) === 0}
          <div class="c-gray">
            <T text="Donated or contributed — nothing was owed for this asset.|Donado o aportado: no se adeuda nada por este activo." />
          </div>
        {:else}
          <div class="grid grid-cols-3 gap-10">
            <div class="bg-slate-100 rounded py-8 text-center">
              <div class="text-sm text-slate-600"><T text="Purchase|Compra" /></div>
              <div class="ff-mono ff-bold">{formatN(selectedAsset.PurchaseAmount / 100, 2)}</div>
            </div>
            <div class="bg-slate-100 rounded py-8 text-center">
              <div class="text-sm text-slate-600"><T text="Paid|Pagado" /></div>
              <div class="ff-mono ff-bold">{formatN((selectedAsset.PaidAmount || 0) / 100, 2)}</div>
            </div>
            <div class="bg-slate-100 rounded py-8 text-center">
              <div class="text-sm text-slate-600"><T text="Owed|Adeudado" /></div>
              <div class="ff-mono ff-bold">{formatN(assetPendingAmount(selectedAsset) / 100, 2)}</div>
            </div>
          </div>

          {#if !canPayAsset(selectedAsset)}
            <div class="c-gray mt-10"><T text="Fully paid.|Pagado completamente." /></div>
          {/if}
        {/if}
      </div>

      {#if canPayAsset(selectedAsset) || canDisposeAsset(selectedAsset)}
        <div class="flex gap-10 mt-16">
          {#if canPayAsset(selectedAsset)}
            <Button color="blue" icon="icon-[fa--check]"
              name="Register Payment|Registrar Pago"
              label="Opens the dialog to register a payment against this asset."
              onClick={openPaymentModal}
            />
          {/if}
          {#if canDisposeAsset(selectedAsset)}
            <Button name="Dispose|Dar de Baja"
              label="Disposes the asset: removes it from the warehouse and stops its depreciation."
              color="red"
              icon="icon-[fa--trash]"
              onClick={onDispose}
            />
          {/if}
        </div>
      {/if}

      <div class="mt-16" aria-label="Depreciation entries">
        <div class="h4 ff-bold mb-8"><T text="Depreciation Ledger|Asientos de Depreciación" /></div>
        {#if depreciationEntries.length === 0}
          <div class="c-gray"><T text="No entries posted yet.|Aún no hay asientos registrados." /></div>
        {:else}
          <div class="flex flex-col gap-4">
            {#each depreciationSchedule(depreciationEntries, selectedAsset.DepreciationMonths) as period (period.entry.ID)}
              <div class="flex justify-between border-b py-6">
                <div><T text={period.label} /></div>
                <div class="flex gap-16">
                  <div class="c-gray">{formatTime(period.entry.Date, "d-m-Y")}</div>
                  <div class="ff-bold">{formatN(period.entry.Amount / 100, 2)}</div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </Layer>

  <!-- Payment dialog. Kept out of the detail panel so the panel stays a read surface:
       the balances above it are what the payment changes. -->
  <Modal id={PAYMENT_MODAL_ID} size={3}
    css="min-h-0!"
    title="Register Payment|Registrar Pago"
    saveButtonLabel="Register Payment|Registrar Pago"
    saveIcon="icon-[fa--check]"
    onSave={registerAssetPayment}
  >
    <!-- Amount first on purpose: Modal focuses the dialog's first input on open, and a
         SearchSelect opens its dropdown on focus — leading with the register would cover
         the date field with the list. The amount is also the field users actually edit,
         since it arrives pre-filled with the outstanding balance. -->
    <div class="grid grid-cols-24 gap-10 py-8">
      <Input label="Payment Amount|Monto del Pago"
        bind:saveOn={paymentForm}
        css="col-span-24 md:col-span-12"
        save="Amount"
        type="number"
        baseDecimals={2}
        inputCss="ff-mono text-right"
      />
      <SearchSelect label="Source Register|Caja de Origen"
        bind:saveOn={paymentForm}
        css="col-span-24 md:col-span-12"
        save="CashBankID"
        keyId="ID"
        keyName="Name"
        options={cajas.Cajas}
      />
      <DateInput label="Payment Date|Fecha del Pago"
        bind:saveOn={paymentForm}
        css="col-span-24 md:col-span-12"
        save="Date"
      />
      <div class="col-span-24 md:col-span-12 flex items-end">
        <Checkbox bind:saveOn={paymentForm} save="IsFullyPaid" label="Is Fully Paid|Pagado Completo" />
      </div>
    </div>
  </Modal>
</Page>

<style>
  /* Stat labels read as field labels — same token FieldShell paints its <label> with,
     so a theme override on `body` moves both together. */
  .stat-label {
    color: var(--input-label-color, #6d5dad);
  }
</style>
