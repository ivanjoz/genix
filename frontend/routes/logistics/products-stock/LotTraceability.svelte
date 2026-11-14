<script lang="ts">
  import Modal from '#components/layers/Modal.svelte'
  import LoadingBar from '#components/misc/LoadingBar.svelte'
  import T from '#components/misc/T.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { StockMovementType } from '#core/stock-movement-type.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { untrack } from 'svelte'
  import type { ILotTraceRow } from './lot-traceability'
  import { loadLotTrace, type ILotTrace } from './lot-traceability.svelte'
  import type { IProductStockLot } from './stock-movement'

  // "Trazabilidad" of a lot (docs/MANUFACTURING_PLAN.md §10): backward, what its production consumed;
  // forward, what the productions that consumed it made.
  let { id, lot, productName }: {
    id: number
    lot: IProductStockLot | undefined
    productName: (productID: number) => string
  } = $props()

  let lotTrace = $state<ILotTrace | undefined>(undefined)
  let isLoadingTrace = $state(false)
  let traceRequestVersion = 0

  $effect(() => {
    const tracedLot = lot
    untrack(() => {
      lotTrace = undefined
      if (!tracedLot) { return }
      // A newer lot opened meanwhile wins over a slower answer for the previous one.
      const requestVersion = ++traceRequestVersion
      isLoadingTrace = true
      loadLotTrace(tracedLot)
        .then((loadedTrace) => { if (requestVersion === traceRequestVersion) { lotTrace = loadedTrace } })
        .finally(() => { if (requestVersion === traceRequestVersion) { isLoadingTrace = false } })
    })
  })

  const lotName = (lotID: number) => lotID ? (lotTrace?.lotNamesByID.get(lotID) || `LOT-${lotID}`) : tr('No batch|Sin lote')
  const quantityText = (row: ILotTraceRow) => row.SubDivisor > 1
    ? formatN(row.SubUnits / row.SubDivisor, 3) : formatN(row.SubUnits)
</script>

{#snippet traceTable(rows: ILotTraceRow[], productHeader: string)}
  <table class="w-full text-sm mb-12" aria-label="Lot trace rows">
    <thead>
      <tr class="border-b border-line text-fg-muted">
        <th class="text-left py-4"><T text="Order|Orden" /></th>
        <th class="text-left py-4"><T text="Receipt|Ingreso" /></th>
        <th class="text-left py-4"><T text={productHeader} /></th>
        <th class="text-left py-4"><T text="Batch|Lote" /></th>
        <th class="text-right py-4"><T text="Quantity|Cantidad" /></th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row}
        <tr class="border-b border-line-soft">
          <td class="py-4">OP-{row.OrderID}</td>
          <td class="py-4">#{row.ReceiptN} · {formatTime(row.ReceiptDate, 'Y-m-d')}</td>
          <td class="py-4">{productName(row.ProductID)}</td>
          <td class="py-4 {row.LotID ? '' : 'text-fg-muted'}">{lotName(row.LotID)}</td>
          <td class="py-4 text-right ff-mono">{quantityText(row)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/snippet}

<Modal {id} size={5} bodyCss="px-16 py-12" title={`${tr('Traceability|Trazabilidad')} · ${lot?.Name || ''}`}>
  {#if isLoadingTrace}
    <LoadingBar label={tr('Loading traceability...|Cargando trazabilidad...')} />
  {:else if lotTrace && lot}
    <div class="h3 ff-bold mb-6"><T text="Origin|Origen" /></div>
    {#if lot.OriginMovementType === StockMovementType.PURCHASE_RECEPTION}
      <div class="mb-12"><T text="Purchase order reception|Ingreso por orden de compra" /> N° {lot.OriginDocumentID}</div>
    {:else if lotTrace.backward.length > 0}
      <div class="mb-6"><T text="Consumed by its production|Consumido por su producción" /></div>
      {@render traceTable(lotTrace.backward, 'Supply|Insumo')}
    {:else}
      <div class="mb-12 text-fg-muted"><T text="No recorded origin.|Sin origen registrado." /></div>
    {/if}
    <div class="h3 ff-bold mb-6"><T text="Destination|Destino" /></div>
    {#if lotTrace.forward.length > 0}
      <div class="mb-6"><T text="Produced with this batch|Producido con este lote" /></div>
      {@render traceTable(lotTrace.forward, 'Product|Producto')}
    {:else}
      <div class="text-fg-muted"><T text="No production consumed this batch.|Ninguna producción consumió este lote." /></div>
    {/if}
  {/if}
</Modal>
