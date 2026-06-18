<script lang="ts">
import DateInput from '$components/form/DateInput.svelte'
import Input from '$components/form/Input.svelte'
import Layer from '$components/layers/Layer.svelte'
import LayerStatic from '$components/layers/LayerStatic.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import Button from '$components/buttons/Button.svelte'
import TableGrid from '$components/vTable/TableGrid.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core, tr } from '$core/store.svelte'
import T from '$components/misc/T.svelte'
import { formatN, formatTime, Notify } from '$libs/helpers'
import { ClientProviderService, ClientProviderType } from '../../negocio/clientes/clientes-proveedores.svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import { AlmacenesService } from '../../negocio/sedes-almacenes/sedes-almacenes.svelte'
import {
  postPurchaseOrderEntry,
  PurchaseOrdersService,
  PurchaseOrderStatus,
  type IPurchaseOrder,
  type IPurchaseOrderEntryItem,
} from '../purchase-orders/purchase_order.svelte'

// Flat row mixing two shapes; the TableGrid uses `useRowRenderer` to switch render mode per row.
type EntryRow = {
  kind: 'entry'
  productID: number
  // PresentationID is required by the backend so it can compare received vs. ordered
  // by (ProductID, PresentationID); 0 = "sin presentación" line.
  presentationID: number
  lotCode: string          // '' = SIN LOTE bucket
  expirationDate: number   // UnixDay (project convention); 0 = unset
  quantity: number
  serialNumbers: { serial: string, quantity: number }[]
}

// Composite key for matching entry rows / cards against OC detail lines.
const cardKey = (productID: number, presentationID: number) =>
  `${productID}_${presentationID}`

type LotHeaderRow = {
  kind: 'lot-header'
  lotCode: string          // '' renders as "SIN LOTE"
}

type Row = LotHeaderRow | EntryRow

// Confirmed purchase orders are the only ones that can be received; pending/fulfilled/canceled are excluded.
const purchaseOrders = new PurchaseOrdersService(PurchaseOrderStatus.CONFIRMED, true)
const productos = new ProductosService(true)
const almacenes = new AlmacenesService()
const providersService = new ClientProviderService(ClientProviderType.PROVIDER, true)

let selectedOrderID = $state(0)
// Wrapped in an object so the project's `Input` component can bind via `saveOn`/`save`.
let lotState = $state({ lotCode: '' })
let warehouseID = $state(0)
let cardFilterText = $state('')      // FilterInput already lower-cases + trims
let entries = $state<EntryRow[]>([])
// Bumped whenever an entry's quantity changes via the TableGrid editable cell so derived totals refresh.
let entriesVersion = $state(0)

// Modal state — Layer id=4 hosts the serial-number editor.
let editingSerialEntry = $state<EntryRow | null>(null)
let serialDraft = $state<{ serial: string, quantity: number }[]>([])

const selectedOrder = $derived<IPurchaseOrder | undefined>(
  purchaseOrders.records.find(o => o.ID === selectedOrderID),
)

// Map a purchase-order line to a card; one card per (productID, presentationID) pair from the order detail.
// Lines with the same key are merged so the badge counts the total ordered for that pair.
const orderProductCards = $derived.by(() => {
  type Card = { key: string, productID: number, presentationID: number, ordered: number, name: string, sku: string, haystack: string }
  if (!selectedOrder) return [] as Card[]
  const productIDs = selectedOrder.DetailProductIDs || []
  const quantities = selectedOrder.DetailProductQuantity || []
  const presentations = selectedOrder.DetailProductPresentationIDs || []
  const byKey = new Map<string, Card>()
  for (let i = 0; i < productIDs.length; i++) {
    const pid = productIDs[i]
    const presID = presentations[i] || 0
    const key = cardKey(pid, presID)
    const existing = byKey.get(key)
    if (existing) {
      existing.ordered += quantities[i] || 0
      continue
    }
    const product = productos.recordsMap.get(pid)
    const name = product?.Name || `Producto-${pid}`
    const sku = product?.SKU || ''
    // Pre-build a lower-cased haystack so substring filtering is allocation-free per keystroke.
    const presentationSkus = (product?.Presentations || [])
      .map(p => p.sk || '')
      .filter(Boolean)
      .join(' ')
    const haystack = `${name} ${sku} ${presentationSkus}`.toLowerCase()
    byKey.set(key, { key, productID: pid, presentationID: presID, ordered: quantities[i] || 0, name, sku, haystack })
  }
  return Array.from(byKey.values())
})

// Sum already-added units per (productID, presentationID) across all lots — used by the card progress badge.
const addedQuantityByCardKey = $derived.by(() => {
  entriesVersion // re-run on edits
  const totals = new Map<string, number>()
  for (const entry of entries) {
    const key = cardKey(entry.productID, entry.presentationID)
    totals.set(key, (totals.get(key) || 0) + (entry.quantity || 0))
  }
  return totals
})

// Drop cards whose ordered quantity has been fully received — list shows only pending products.
// `ordered <= 0` cards stay (no target to satisfy, e.g. ad-hoc lines).
const filteredOrderProductCards = $derived.by(() => {
  const totals = addedQuantityByCardKey
  return orderProductCards.filter(card => {
    if (card.ordered > 0 && (totals.get(card.key) || 0) >= card.ordered) return false
    if (cardFilterText && !card.haystack.includes(cardFilterText)) return false
    return true
  })
})

// Group entries by lotCode preserving first-seen order; emit a LotHeaderRow before each group.
const tableRows = $derived.by((): Row[] => {
  entriesVersion
  const groups = new Map<string, EntryRow[]>()
  for (const entry of entries) {
    const bucket = groups.get(entry.lotCode)
    if (bucket) { bucket.push(entry) } else { groups.set(entry.lotCode, [entry]) }
  }
  const rows: Row[] = []
  for (const [lotCode, groupEntries] of groups) {
    rows.push({ kind: 'lot-header', lotCode })
    for (const entry of groupEntries) { rows.push(entry) }
  }
  return rows
})

const handleProductCardClick = (productID: number, presentationID: number) => {
  // Match = same (product, presentation) + same current lot input value. Match → +1; no match → push new row with quantity=1.
  const currentLot = lotState.lotCode
  const matching = entries.find(e =>
    e.productID === productID && e.presentationID === presentationID && e.lotCode === currentLot,
  )
  if (matching) {
    matching.quantity += 1
  } else {
    entries.push({
      kind: 'entry',
      productID,
      presentationID,
      lotCode: currentLot,
      expirationDate: 0,
      quantity: 1,
      serialNumbers: [],
    })
  }
  entriesVersion++
}

const removeEntry = (entry: EntryRow) => {
  const idx = entries.indexOf(entry)
  if (idx >= 0) {
    entries.splice(idx, 1)
    entriesVersion++
  }
}

const openSerialModal = (entry: EntryRow) => {
  editingSerialEntry = entry
  // Clone so cancelling (just closing) leaves the row's serialNumbers untouched until apply.
  serialDraft = entry.serialNumbers.map(s => ({ ...s }))
  // Always keep one trailing blank row so the user can append without an explicit "+" click.
  if (serialDraft.length === 0 || serialDraft[serialDraft.length - 1].serial) {
    serialDraft.push({ serial: '', quantity: 0 })
  }
  Core.openSideLayer(4)
}

const applySerialDraft = () => {
  if (!editingSerialEntry) return
  // Cleanup: drop empties + dedupe by serial (keep first occurrence — matches the user's typing order).
  const seen = new Set<string>()
  const cleaned: { serial: string, quantity: number }[] = []
  for (const s of serialDraft) {
    const serial = s.serial.trim()
    if (!serial || seen.has(serial)) continue
    seen.add(serial)
    cleaned.push({ serial, quantity: s.quantity || 0 })
  }
  editingSerialEntry.serialNumbers = cleaned
  // If serial total exceeds the entry quantity, raise quantity to match — sum of serials is authoritative.
  const serialTotal = cleaned.reduce((acc, s) => acc + (s.quantity || 0), 0)
  if (serialTotal > editingSerialEntry.quantity) {
    editingSerialEntry.quantity = serialTotal
  }
  editingSerialEntry = null
  serialDraft = []
  entriesVersion++
}

// Reset entry state and default the warehouse selector when the user picks a new OC.
const onChangePurchaseOrder = () => {
  entries = []
  entriesVersion++
  warehouseID = selectedOrder?.WarehouseID || 0
  cardFilterText = ''
}

let isSaving = $state(false)

// Send the entry to the backend: it computes diferences, applies stock movements, and marks the OC as Cumplida.
const handleSave = async () => {
  if (isSaving) return
  if (!selectedOrderID) {
    Notify.failure(tr('Please select a Purchase Order.|Seleccione una Órden de Compra.'))
    return
  }
  if (!warehouseID) {
    Notify.failure(tr('Please select the destination Warehouse.|Seleccione el Almacén destino.'))
    return
  }
  if (entries.length === 0) {
    Notify.failure(tr('No products to enter.|No hay productos para ingresar.'))
    return
  }

  // Cada entry de cantidad N se serializa como un único item con esa cantidad, salvo
  // que tenga seriales: cada serial se envía como un item independiente con su propia
  // cantidad y SerialNumber para que el backend los individualice en el detalle de stock.
  const items: IPurchaseOrderEntryItem[] = []
  for (const entry of entries) {
    if (!entry.productID) continue
    const baseFields = {
      ProductID: entry.productID,
      PresentationID: entry.presentationID,
      LotCode: entry.lotCode || undefined,
    }
    if (entry.serialNumbers.length > 0) {
      for (const sn of entry.serialNumbers) {
        if (!sn.serial || sn.quantity <= 0) continue
        items.push({ ...baseFields, Quantity: sn.quantity, SerialNumber: sn.serial })
      }
      continue
    }
    if ((entry.quantity || 0) <= 0) continue
    items.push({ ...baseFields, Quantity: entry.quantity })
  }

  if (items.length === 0) {
    Notify.failure(tr('No valid items to enter.|No hay items válidos para ingresar.'))
    return
  }

  isSaving = true
  try {
    await postPurchaseOrderEntry({
      PurchaseOrderID: selectedOrderID,
      WarehouseID: warehouseID,
      Items: items,
    })
    Notify.success(tr('Purchase order entered successfully.|Órden de compra ingresada correctamente.'))
    // Reset local state — the saved OC is no longer Confirmada, so it disappears from the picker.
    selectedOrderID = 0
    entries = []
    entriesVersion++
    warehouseID = 0
    cardFilterText = ''
  } catch (error) {
    Notify.failure(String(error || 'Error al guardar el ingreso.'))
  } finally {
    isSaving = false
  }
}

const formatOrderLabel = (order: IPurchaseOrder): string => {
  // formatTime layout uses single-letter tokens: Y=year, m=month, d=day.
  const dateLabel = order.Date ? (formatTime(order.Date, 'Y-m-d') as string) || '' : ''
  return `OC #${order.ID}${dateLabel ? ` — ${dateLabel}` : ''}`
}

// Columns operate on Row but lot-header rows skip column rendering via useRowRenderer.
const entryColumns: ITableColumn<Row>[] = [
  {
    id: 'producto',
    header: 'Product|Producto',
    width: 'minmax(0, 1fr)',
    useLineClamp: true,
    getValue: (row) => {
      if (row.kind !== 'entry') return ''
      return productos.recordsMap.get(row.productID)?.Name || `Producto-${row.productID}`
    },
  },
  {
    id: 'vencimiento',
    header: 'Expiry|Vencimiento',
    width: '130px',
    css: "px-0",
    useCellRenderer: true,
    showHoverEffect: true,
  },
  {
    id: 'cantidad',
    header: 'Qty.|Cant.',
    width: '90px',
    align: 'right',
    cellInputType: 'number',
    inputCss: 'text-right pr-6',    
    getValue: (row) => row.kind === 'entry' ? row.quantity : '',
    onCellEdit: (row, value) => {
      if (row.kind !== 'entry') return
      row.quantity = parseInt(String(value || '0')) || 0
      entriesVersion++
    },
  },
  {
    id: 'serial',
    header: 'Serial #|S/N',
    width: '110px',
    align: 'right',
    useCellRenderer: true,
  },
  {
    header: '...',
    width: '34px',
    css: "px-2",
    // Lot-header rows are not column-rendered (useRowRenderer short-circuits),
    // so the handler only ever sees EntryRow despite the union type on Row.
    buttonDeleteHandler: (row) => {
      if (row.kind !== 'entry') return
      removeEntry(row)
    },
  },
]

// Column set for the Serial Number modal table.
const serialColumns: ITableColumn<{ serial: string, quantity: number }>[] = [
  {
    id: 'serial',
    header: 'Serial',  // same in both languages
    width: 'minmax(0, 1fr)',
    css: 'ff-mono',
    getValue: (row) => row.serial,
    onCellEdit: (row, value) => {
      const next = String(value || '').trim()
      const wasEmpty = !row.serial
      // Reject duplicates: another row in the draft already owns this serial. Clear and warn.
      if (next && serialDraft.some(s => s !== row && s.serial.trim() === next)) {
        Notify.failure(`El serial "${next}" ya fue ingresado.`)
        row.serial = ''
        serialDraft = [...serialDraft]
        return
      }
      row.serial = next
      // Auto-default qty=1 the first time the user names a serial so they don't have to dual-edit.
      if (wasEmpty && next && (!row.quantity || row.quantity <= 0)) {
        row.quantity = 1
      }
      // Auto-append a fresh empty row so the user can keep typing without manually adding rows.
      if (next && serialDraft[serialDraft.length - 1].serial) {
        serialDraft = [...serialDraft, { serial: '', quantity: 0 }]
      } else {
        serialDraft = [...serialDraft]
      }
    },
  },
  {
    id: 'quantity',
    header: 'Qty.|Cant.',
    width: '110px',
    align: 'right',
    cellInputType: 'number',
    inputCss: 'text-right pr-6',
    getValue: (row) => row.quantity || '',
    onCellEdit: (row, value) => {
      row.quantity = parseInt(String(value || '0')) || 0
      serialDraft = [...serialDraft]
    },
  },
]

const purchaseOrderOptions = $derived(
  purchaseOrders.records.map(o => ({
    ID: o.ID,
    Name: formatOrderLabel(o),
    ProviderName: providersService.recordsMap.get(o.ProviderID)?.Name || `Proveedor #${o.ProviderID}`,
    Date: o.Date,
    TotalAmount: o.TotalAmount,
  })),
)

const totalSerialCountForEntry = (entry: EntryRow): number => {
  return entry.serialNumbers.reduce((acc, s) => acc + (s.quantity || 0), 0)
}
</script>

<div class="flex h-full gap-20">
  <!-- Left column: OC picker + lot input + filter, then product cards. -->
  <div class="flex-1 flex flex-col min-w-0 relative">
    <div class="flex items-center gap-8 mb-12" aria-label="Purchase order and lot selection toolbar">
      <div class="w-260">
        <SearchSelect options={purchaseOrderOptions} keyId="ID" keyName="Name"
          selected={selectedOrderID}
          placeholder="ÓRDEN DE COMPRA ::"
          optionsCss="w-380"
          useDividingLine
          onChange={(option) => {
            selectedOrderID = Number(option?.ID || 0)
            onChangePurchaseOrder()
          }}
          getSearchText={(o) => `OC ${o.ID} ${o.ProviderName}`}
          {optionRenderer}
        />
        {#snippet optionRenderer(o: typeof purchaseOrderOptions[number], _words: string[])}
          <div class="flex flex-col min-w-0 w-full">
            <div class="truncate"><b>OC #{o.ID}</b> — {o.ProviderName}</div>
            <div class="text-xs text-gray-500 truncate">
              {o.Date ? formatTime(o.Date, 'Y-m-d') : ''} — S/. {formatN((o.TotalAmount || 0) / 100, 2)}
            </div>
          </div>
        {/snippet}
      </div>
      <div class="w-160">
        <Input bind:saveOn={lotState} save="lotCode" placeholder="Lote (opcional)" />
      </div>
      <div class="ml-auto w-220">
        <FilterInput placeholder="Filtrar Nombre o SKU…" bind:value={cardFilterText} />
      </div>
    </div>

    {#if !selectedOrderID}
      <!-- Empty state: nothing selected yet. -->
      <div class="bg-red-50 text-red-600 text-center font-medium py-16 px-12 rounded-md border border-red-100">
        Seleccione una Órden de Compra
      </div>
    {:else}
      <!-- Product cards from the selected purchase order, filtered by name/SKU. -->
      <div class="flex flex-col gap-6 overflow-y-auto pr-4" style="max-height: calc(100vh - var(--header-height) - 160px);" aria-label="Product cards from the selected purchase order">
        {#each filteredOrderProductCards as card (card.key)}
          {@const added = addedQuantityByCardKey.get(card.key) || 0}
          <button type="button" class="text-left bg-gray-50 hover:bg-blue-50 border border-gray-200 rounded-md px-12 py-8 flex items-center gap-12 transition-colors"
            onclick={() => handleProductCardClick(card.productID, card.presentationID)}
          >
            <div class="flex-1 min-w-0">
              <div class="text-sm font-semibold text-gray-800 truncate">{card.name}</div>
              {#if card.sku}
                <div class="text-xs text-gray-500 mt-2">SKU: {card.sku}</div>
              {/if}
            </div>
            <div class="text-xs text-gray-500 shrink-0 text-right leading-[1.2]">
              <div>Ingresado:</div>
              <div class="text-gray-800 font-semibold">{added} / {card.ordered}</div>
            </div>
          </button>
        {:else}
          <div class="text-sm text-gray-400 text-center py-12">
            {orderProductCards.length === 0 ? 'La órden no tiene productos.' : 'Sin coincidencias.'}
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Right column: order header + warehouse selector + grouped entries table. -->
  <LayerStatic
    css="w-[50%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
    mobileLayerTitle="Ingreso de OC"
    useMobileLayerVertical={124}
  >
    <div class="px-12 py-10 border-b border-gray-100 flex items-center justify-between bg-gray-50/50 gap-12" aria-label="Purchase order entry header with warehouse selector and save button">
      <div class="font-bold text-gray-800 text-lg">
        {selectedOrder ? `Orden #${selectedOrder.ID}` : '—'}
      </div>
      <div class="w-200">
        <SearchSelect options={almacenes?.Almacenes || []} keyId="ID" keyName="Name"
          selected={warehouseID}
          placeholder="ALMACÉN ::"
          id={12}
          onChange={(option) => {
            warehouseID = Number(option?.ID || 0)
          }}
        />
      </div>
      <Button color="blue" icon="icon-[fa--floppy-o]" css="shrink-0"
        disabled={isSaving || !selectedOrderID || entries.length === 0}
        label="Submits the received items from the purchase order into the selected warehouse stock." onClick={handleSave}
        name={isSaving ? tr('Saving…|Guardando…') : tr('Save|Guardar')} hideNameOnMobile />
    </div>

    <div class="flex-1 min-h-0 px-12 py-8">
      {#if !selectedOrderID}
        <div class="text-sm text-gray-400 text-center py-24">Seleccione una órden para iniciar el ingreso.</div>
      {:else}
        <TableGrid columns={entryColumns} data={tableRows}
          height="calc(100vh - var(--header-height) - 160px)"
          rowHeight={42}
          headerCss="px-6 flex items-center min-h-32"
          cellCss="px-6"
          getRowId={(row, idx) => row.kind === 'lot-header' ? `lot:${row.lotCode}` : `entry:${idx}`}
          useRowRenderer={(row) => row.kind === 'lot-header'}
          getRowHeight={e => e.kind === 'lot-header' ? 30 : 0}
        >
          {#snippet rowRenderer(row, _rowIndex)}
            {#if row.kind === 'lot-header'}
              <div class="px-12 w-full h-full flex items-center text-blue-600 bg-blue-50 font-semibold text-sm border-b-1 pt-2 border-b-blue-200">
                {row.lotCode ? `LOTE ${row.lotCode}` : 'SIN LOTE'}
              </div>
            {/if}
          {/snippet}

          {#snippet cellRenderer(row, colDef, _rowIndex)}
            {#if row.kind === 'entry'}
              {#if colDef.id === 'vencimiento'}
                <!-- usePopover: calendar escapes table-cell overflow. useInlineStyle: the cell already owns the visuals. -->
                <DateInput saveOn={row} save="expirationDate" type="unix" usePopover useInlineStyle
                  onChange={() => { entriesVersion++ }}
                />
              {:else if colDef.id === 'serial'}
                {@const total = totalSerialCountForEntry(row)}
                <div class="w-full flex items-center justify-end gap-6">
                  <span class={total === 0 ? 'text-gray-400' : 'text-gray-800 font-semibold'}>
                    {total === 0 ? '-' : total}
                  </span>
                  <button type="button"
                    class="w-26 h-26 rounded-full b-color-green flex items-center justify-center text-xs"
                    onclick={() => openSerialModal(row)}
                    title="Editar seriales"
                    aria-label="Edit serial numbers for this entry"
                  >
                    <i class={total === 0 ? 'icon-[fa--plus]' : 'icon-[fa--pencil]'}></i>
                  </button>
                </div>
              {/if}
            {/if}
          {/snippet}
        </TableGrid>
      {/if}
    </div>
  </LayerStatic>
</div>

<!-- Serial Number editor modal — mirrors the table-only pattern from ProductStockMovement.svelte. -->
<Layer id={4} type="side" sideLayerSize={520}
  title={editingSerialEntry ? `Seriales — ${productos.recordsMap.get(editingSerialEntry.productID)?.Name || ''}` : 'Seriales'}
  titleCss="h2"
  css="px-12 py-10"
  onClose={() => {
    applySerialDraft()
  }}
>
  {#if editingSerialEntry}
    <TableGrid columns={serialColumns} data={serialDraft}
      height="calc(100vh - 14rem)"
      cellCss="px-6"
      headerCss="px-6 flex items-center min-h-32"
      css="mt-6"
      getRowId={(row, idx) => `${row.serial || 'pending'}_${idx}`}
    />
  {/if}
</Layer>
