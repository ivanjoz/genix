<script lang="ts">
import DateInput from '$components/form/DateInput.svelte'
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import type { IClientProvider } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import type { IWarehouse } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import type { IPurchaseOrder } from './purchase_order.svelte'

// Reusable header form for a purchase order. Used by both the creation flow (Información tab)
// and the report-mode "Editar" modal. The cart/products list is intentionally outside this form
// because edits never alter products — only metadata fields here.
// Partial<IPurchaseOrder> lets callers pass either the full record (report) or the create-form shape.
let {
  form = $bindable(),
  providers,
  almacenes,
  disableProvider = false,
}: {
  form: Partial<IPurchaseOrder>
  providers: IClientProvider[]
  almacenes: IWarehouse[]
  // Edit flow forbids changing the provider; create flow keeps it editable.
  disableProvider?: boolean
} = $props()
</script>

<div class="grid grid-cols-2 gap-8" aria-label="Purchase order header form with provider, warehouse, dates, notes, and invoice number">
  <SearchSelect
    label="Proveedor"
    keyId="ID"
    keyName="Name"
    options={providers}
    selected={form.ProviderID}
    disabled={disableProvider}
    onChange={(provider) => { if (!disableProvider) { form.ProviderID = provider?.ID || 0 } }}
  />
  <SearchSelect
    label="Almacén"
    keyId="ID"
    keyName="Name"
    options={almacenes}
    selected={form.WarehouseID}
    onChange={(almacen: IWarehouse) => { form.WarehouseID = almacen?.ID || 0 }}
  />
  <DateInput
    bind:saveOn={form}
    save="DeliveryDate"
    type="unix"
    label="Date Entrega"
  />
  <DateInput
    bind:saveOn={form}
    save="PaymentDate"
    type="unix"
    label="Date Pago"
  />
  <Input css="col-span-2"
    bind:saveOn={form}
    save="Notes"
    label="Notas"
  />
  <Input
    bind:saveOn={form}
    save="InvoiceNumber"
    label="Factura"
  />
</div>
