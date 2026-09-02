<script lang="ts">
import DateInput from '$components/form/DateInput.svelte'
import Info from '$components/misc/Info.svelte'
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import T from '$components/misc/T.svelte'
import type { IAssetForm } from './assets'

// One form, two shells: the create Layer and the edit Modal in +page.svelte both render this.
// `mode` decides only what an existing asset makes meaningless — which fields are locked, which
// hints disappear, and whether serials are a list or a single value.
interface Props {
  form: IAssetForm
  mode: "create" | "edit"
  // Create only. Serials are typed one per line, and their presence is what makes the
  // acquisition per-unit. Held in an object because Input writes through saveOn/save.
  serialsInput?: { text: string }
  supplyOptions: { ID: number, Name: string }[]
  warehouseOptions: { ID: number, Name: string }[]
  providerOptions: { ID: number, Name: string }[]
  currencyOptions: { id: number, name: string }[]
}

const {
  form, mode, serialsInput, supplyOptions, warehouseOptions, providerOptions, currencyOptions,
}: Props = $props()

const isEdit = $derived(mode === "edit")

// The create form asks for per-unit money and multiplies by Quantity; the edit form writes the
// asset row's totals, because that is what the column holds and what the detail panel shows.
const purchaseLabel = $derived(isEdit
  ? "Purchase Amount|Monto de Compra"
  : "Purchase Amount (per unit)|Monto de Compra (por unidad)")
const bookValueLabel = $derived(isEdit
  ? "Book Value|Valor en Libros"
  : "Book Value (per unit)|Valor en Libros (por unidad)")
</script>

<div class="grid grid-cols-24 items-start gap-x-10 gap-y-10" aria-label="Asset Acquisition Form">
  <SearchSelect label="Material|Material"
    saveOn={form}
    css="col-span-24 md:col-span-14"
    save="ProductID"
    keyId="ID"
    keyName="Name"
    disabled={isEdit}
    options={supplyOptions}
  />
  <SearchSelect label="Warehouse|Almacén"
    saveOn={form}
    css="col-span-24 md:col-span-10"
    save="WarehouseID"
    keyId="ID"
    keyName="Name"
    disabled={isEdit}
    options={warehouseOptions}
  />
  {#if !isEdit}
    <Info css="col-span-24 md:col-span-14">
      <T text="The asset must be registered as a material before it can be acquired.|El activo debe registrarse como material para ser adquirido." />
      <a href="/production/supplies-materials" class="c-purple underline">
        <T text="Register it here.|Regístrelo aqui." />
      </a>
    </Info>
  {/if}
  <DateInput label="Acquisition Date|Fecha de Adquisición"
    saveOn={form}
    css="col-span-24 md:col-span-10"
    save="AcquisitionDate"
  />
  <SearchSelect label="Supplier|Proveedor"
    saveOn={form}
    css="col-span-24 md:col-span-14"
    save="SupplierID"
    keyId="ID"
    keyName="Name"
    disabled={isEdit}
    options={providerOptions}
  />
  <Input label={purchaseLabel}
    saveOn={form}
    css="col-span-12 md:col-span-10"
    save="PurchaseAmount"
    type="number"
    baseDecimals={2}
  />
  <!-- The Modal's focus target in edit mode. Without it the dialog lands on Fecha de
       Adquisición — the first field not locked — and DateInput opens its calendar over the rest
       of the form. It is also the field an edit is usually about. -->
  <Input label={bookValueLabel}
    saveOn={form}
    css="col-span-12 md:col-span-8"
    save="AcquisitionValue"
    type="number"
    baseDecimals={2}
    focusOnOpen={isEdit}
  />
  <SearchSelect label="Currency|Moneda"
    saveOn={form}
    css="col-span-24 md:col-span-6"
    save="CurrencyType"
    keyId="id"
    keyName="name"
    disabled={isEdit}
    options={currencyOptions}
  />
  <DateInput label="Payment Due|Vencimiento del Pago"
    saveOn={form}
    css="col-span-24 md:col-span-10"
    save="DueDate"
  />
</div>

<!-- Purchase 0 with a value above 0 is the donated case: nothing is owed, but it is still an
     asset on the books and still depreciates. Only worth saying while the choice is still open. -->
{#if !isEdit}
  <Info css="mt-8"
    text="Leave the purchase amount at 0 for a donated or contributed asset — nothing is owed, but it still has a book value and still depreciates.|Deje el monto de compra en 0 para un activo donado o aportado."
  />
{/if}

<div class="mt-16" aria-label="Asset units">
  <div class="h4 ff-bold mb-6">
    <T text="Units|Unidades" />
  </div>
  <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10">
    <Input label="Quantity|Cantidad"
      saveOn={form}
      css="col-span-12 md:col-span-6"
      save="Quantity"
      type="number"
      disabled={isEdit}
    />
    {#if isEdit}
      <!-- One asset row is one serial. Changing it moves the units between stock buckets,
           since the serial is part of the ProductStockDetail key. -->
      <Input label="Serial Number|Número de Serie"
        saveOn={form}
        save="SerialNumber"
        css="col-span-24 md:col-span-18"
      />
    {:else if serialsInput}
      <Input label="Serial Numbers (one per line)|Números de Serie (uno por línea)"
        saveOn={serialsInput}
        save="text"
        css="col-span-24 md:col-span-18"
      />
    {/if}
  </div>
  {#if !isEdit}
    <Info css="mt-8"
      text="Enter serials to track each unit as its own asset; leave empty to register the whole lot as one.|Ingrese series para registrar cada unidad como un activo independiente; déjelo vacío para registrar todo el lote como uno solo."
    />
  {/if}
</div>
