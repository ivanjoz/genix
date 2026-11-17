<script lang="ts">
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import { untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import TableGrid from '#components/vTable/TableGrid.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import type { IProductPack } from '#core/product-packs.ts'
  import { DOC_TYPE_BOLETA, DOC_TYPE_FACTURA, docTypeName } from '#core/sunat-doc-type.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { packGroupRows, packGroupRowsTotal, type IPackGroupRow } from '#core/sale-packs.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { EmpresaParametrosService } from '#routes/company/configuration/empresas.svelte.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import { ClientProviderService, ClientProviderType } from '#services/crm/client-provider.svelte.ts'
  import { TILL_IDENTITY_DOC_OPTIONS, deriveIdentityDocType } from '#services/crm/identity-doc.ts'
  import type { ITableOrder } from '../dining.ts'
  import { postTableOrderAnnul, postTableOrderCharge } from './floor-service.svelte.ts'
  import { billLines, type IBillLine } from './floor-service.ts'
  import PackHeaderRow, { PACK_HEADER_ROW_HEIGHT } from './PackHeaderRow.svelte'
  import {
    CLIENT_MODE, buildChargeRequest, buildTableAnnulRequest, chargeBlocker, chargeCashBankOptions, newTableChargeForm,
  } from './table-charge.ts'

  // The "Pagar" tab: the bill listed PoS style, charged into a sale ("Cobrar mesa"), or annulled. The
  // packs the whole bill fills show as a header row over their products, priced as the charge will.
  let { order, productName, productCategoryIDs, packs, onDone }: {
    order: ITableOrder
    productName: (productID: number) => string
    productCategoryIDs: (productID: number) => number[]
    packs: IProductPack[]
    // The order as the backend left it: charged (ss 2) or annulled (ss 0).
    onDone: (order: ITableOrder) => void
  } = $props()

  let mode = $state<'charge' | 'annul'>('charge')

  const cashBanksService = new CajasService()
  const companyService = new EmpresaParametrosService()
  const clientsService = new ClientProviderService(ClientProviderType.CLIENT, true)

  let chargeForm = $state(newTableChargeForm())
  let annulForm = $state({ Reason: '' })
  let isPosting = $state(false)
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  const blocker = $derived(chargeBlocker(order))
  const billRows = $derived(packGroupRows(billLines(order), productCategoryIDs, packs))
  const billTotal = $derived(packGroupRowsTotal(billRows))
  const cashBanks = $derived(chargeCashBankOptions(cashBanksService.Cajas || []))
  // A sale is issued as a factura or a boleta, never under a note's series.
  const seriesOptions = $derived((companyService.empresa.InvoiceSeries || [])
    .filter((series) => series.ss === 1 && (series.DocType === DOC_TYPE_FACTURA || series.DocType === DOC_TYPE_BOLETA))
    .map((series) => ({ ID: series.SeriesID, Name: `${tr(docTypeName(series.DocType)).toUpperCase()} · ${series.SeriesCode}` })))
  const clientModeOptions = [
    { ID: CLIENT_MODE.SELECT, Name: tr('Select client|Seleccionar cliente') },
    { ID: CLIENT_MODE.REGISTER, Name: tr('Register client|Registrar cliente') },
  ]
  const clientOptions = $derived(clientsService.records.map((client) => ({
    ID: client.ID, Name: client.RegistryNumber ? `${client.Name} ${client.RegistryNumber}` : client.Name,
  })))

  // Most bills are paid on the spot: the first cash-bank in soles is preselected once it loads.
  $effect(() => {
    const firstCashBankID = cashBanks[0]?.ID || 0
    untrack(() => { if (!chargeForm.CashBankID) chargeForm.CashBankID = firstCashBankID })
  })

  async function charge() {
    if (isPosting) return
    const { request, error } = buildChargeRequest(order, chargeForm)
    if (!request) { notifyFailure(tr(error || '')); return }
    const cashBankName = cashBanks.find((cashBank) => cashBank.ID === request.CashBankID)?.Name || ''
    const confirmed = await confirmWarn({
      title: tr('Charge Table|Cobrar Mesa'),
      message: request.CashBankID
        ? tr(`Charge ${formatAmount(billTotal)} into "${cashBankName}"?|¿Cobrar ${formatAmount(billTotal)} en "${cashBankName}"?`)
        : tr(`Close the bill of ${formatAmount(billTotal)} unpaid? The client will owe it.`
          + `|¿Cerrar la cuenta de ${formatAmount(billTotal)} sin pago? Quedará como deuda del cliente.`),
    })
    if (!confirmed) return

    isPosting = true
    try {
      const chargeResult = await postTableOrderCharge(request)
      notifySuccess(tr(`Sale ${chargeResult.SaleOrder.ID} created.|Venta ${chargeResult.SaleOrder.ID} creada.`))
      onDone(chargeResult.Order)
    } catch (chargeError) {
      console.error('[floor-service] charge error', { request, chargeError })
    } finally {
      isPosting = false
    }
  }

  async function annul() {
    if (isPosting) return
    const { request, error } = buildTableAnnulRequest(order, annulForm.Reason)
    if (!request) { notifyFailure(tr(error || '')); return }
    const confirmed = await confirmWarn({
      title: tr('Annul Order|Anular Pedido'),
      message: tr('Annul this table order? Its products are released and nothing is charged.'
        + '|¿Anular el pedido de la mesa? Sus productos se liberan y no se cobra nada.'),
    })
    if (!confirmed) return

    isPosting = true
    try {
      const annulledOrder = await postTableOrderAnnul(request)
      notifySuccess(tr('Order annulled.|Pedido anulado.'))
      onDone(annulledOrder)
    } catch (annulError) {
      console.error('[floor-service] annul error', { request, annulError })
    } finally {
      isPosting = false
    }
  }

  const billColumns: ITableColumn<IPackGroupRow<IBillLine>>[] = [
    { header: 'Qty.|Cant.', width: '64px', css: 'px-6 text-right ff-bold',
      getValue: (billRow) => formatQuantity(unpackQuantityLine(billRow.Quantity), billRow.line?.SubDivisor || 1) },
    { header: 'Product|Producto', css: 'px-6', getValue: (billRow) => productName(billRow.line?.ProductID || 0) },
    { header: 'Amount|Importe', width: '110px', css: 'px-6 text-right ff-mono whitespace-nowrap', getValue: (billRow) => formatAmount(billRow.Amount) },
  ]
</script>

<TableGrid columns={billColumns} data={billRows} rowHeight={34} getRowId={(billRow) => billRow.key}
  getRowHeight={(billRow) => billRow.pack ? PACK_HEADER_ROW_HEIGHT : undefined}
  useRowRenderer={(billRow) => !!billRow.pack} rowRenderer={packHeaderRow}
  emptyMessage="Nothing to charge.|Nada que cobrar." />

{#snippet packHeaderRow(billRow: IPackGroupRow<IBillLine>)}
  <PackHeaderRow packName={billRow.pack?.Name || ''} packAmount={formatAmount(billRow.Amount)} />
{/snippet}
<div class="flex justify-end gap-8 my-10 ff-bold" aria-label="The bill's total">
  <span>{tr('Total|Total')}</span><span class="ff-mono">{formatAmount(billTotal)}</span>
</div>


{#if mode === 'annul'}
  <div class="p-10 rounded-md bg-red-bg flex flex-col gap-8" aria-label="Annul the table order with a reason">
    <Input saveOn={annulForm} save="Reason" label="Reason|Motivo" useTextArea rows={2}
      placeholder="Why is this order annulled?|¿Por qué se anula este pedido?" />
    <div class="flex items-center gap-8">
      <Button color="red" icon="icon-[fa--trash]" name="Annul|Anular" label="Confirms annulling this table order."
        disabled={isPosting} onClick={annul} />
      <Button name="Back|Volver" label="Back to the charge form." onClick={() => { mode = 'charge' }} />
    </div>
  </div>
{:else}
  <div class="p-10 rounded-md bg-surface-muted grid grid-cols-24 gap-10 items-end" aria-label="Charge the table: payment, comprobante and client">
    {#if blocker}
      <div class="col-span-24 px-8 py-6 rounded-md border border-amber-border bg-amber-bg text-amber-fg">{tr(blocker)}</div>
    {/if}
    <SearchSelect saveOn={chargeForm} save="CashBankID" css="col-span-24 md:col-span-12"
      options={cashBanks} keyId="ID" keyName="Name" label="Cash/bank for payment|Caja/banco del pago"
      placeholder="UNPAID|SIN PAGAR" />
    <SearchSelect saveOn={chargeForm} save="IssueSeriesID" css="col-span-24 md:col-span-12"
      options={seriesOptions} keyId="ID" keyName="Name" label="Comprobante" placeholder="NO COMPROBANTE|SIN COMPROBANTE" />
    <SearchSelect saveOn={chargeForm} save="ClientMode" css="col-span-24 md:col-span-9"
      options={clientModeOptions} keyId="ID" keyName="Name" label="Client|Cliente" placeholder="NO CLIENT|SIN CLIENTE" />
    {#if chargeForm.ClientMode === CLIENT_MODE.SELECT}
      <SearchSelect saveOn={chargeForm} save="ClientID" css="col-span-24 md:col-span-15" label=""
        options={clientOptions} keyId="ID" keyName="Name" placeholder="Search by name or document|Buscar por nombre o documento" />
    {:else if chargeForm.ClientMode === CLIENT_MODE.REGISTER}
      <Input saveOn={chargeForm.ClientInfo} save="RegistryNumber" css="col-span-24 md:col-span-6" label=""
        placeholder="Document No.|Nº Documento" leftOptions={TILL_IDENTITY_DOC_OPTIONS} saveLeft="IdentityDocType"
        deriveLeftOption={(value) => deriveIdentityDocType(String(value ?? ''))} />
      <Input saveOn={chargeForm.ClientInfo} save="Name" css="col-span-24 md:col-span-9" label=""
        placeholder="Client name|Nombre del cliente" />
    {/if}
    <div class="col-span-24 flex items-center gap-8">
      <Button color="green" icon="icon-[fa--check]" name="Charge {formatAmount(billTotal)}|Cobrar {formatAmount(billTotal)}"
        label="Closes the bill into a sale, paid into the selected cash-bank or owed when none is selected."
        disabled={isPosting || !!blocker} onClick={charge} />
      <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Annul order|Anular pedido" css="ml-auto"
        label="Opens the form to annul this table order." onClick={() => { mode = 'annul' }} />
    </div>
  </div>
{/if}
