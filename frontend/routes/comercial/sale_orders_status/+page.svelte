<script lang="ts">
  import Layer from '$components/Layer.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Core } from '$core/store.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify, formatN, formatTime } from '$libs/helpers';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { untrack } from 'svelte';
  import { SaleOrderGroup, SaleOrdersService, type ISaleOrder } from './sale_order_status.svelte';

  interface ISaleOrderDetailLine {
    detailPosition: number;
    productID: number;
    productName: string;
    sku: string;
    quantity: number;
    unitPrice: number;
    subtotalAmount: number;
  }

  let selectedGroup = $state(SaleOrderGroup.PENDIENTE_DE_PAGO);
  let saleOrdersService = $state<SaleOrdersService | null>(null);
  let saleOrderDetailsView = $state(1);
  let selectedSaleOrder = $state<ISaleOrder | null>(null);

  const productosService = new ProductosService();

  // Render tabs mapped to backend status filters.
  const options = [
    [SaleOrderGroup.PENDIENTE_DE_PAGO, 'Pend. Pago'],
    [SaleOrderGroup.PENDIENTE_DE_ENTREGA, 'Pend. Entrega'],
    [SaleOrderGroup.FINALIZADO, 'Finalizadas']
  ];

  function getSaleOrderStatusName(saleOrder: ISaleOrder): string {
    switch(saleOrder.ss) {
      case 1: return 'Generado';
      case 2: return 'Pagado';
      case 3: return 'Entregado';
      case 4: return 'Finalizado';
      default: return 'Desconocido';
    }
  }

  function renderTopProductsSummary(saleOrder: ISaleOrder): string {
    if (!saleOrder.TopPaidProducts || saleOrder.TopPaidProducts.length === 0) {
      return '-';
    }

    // Keep product-name lookup in the view as requested; service only provides ranked IDs/amounts.
    return saleOrder.TopPaidProducts
      .map((topProduct) => {
        const productName = productosService.recordsMap.get(topProduct.ProductID)?.Nombre || `Producto #${topProduct.ProductID}`;
        return `${productName} (${formatN(topProduct.LineAmount / 100, 2)})`;
      })
      .join(', ');
  }

  function getSaleOrderDetailLines(saleOrder: ISaleOrder): ISaleOrderDetailLine[] {
    // Normalize arrays because old/partial records may omit one or more detail fields.
    const detailProductIDs = saleOrder.DetailProductsIDs || [];
    const detailPrices = saleOrder.DetailPrices || [];
    const detailQuantities = saleOrder.DetailQuantities || [];
    const detailProductSkus = saleOrder.DetailProductSkus || [];

    // Build row-safe detail lines by using the shortest common detail length.
    const detailCount = Math.min(
      detailProductIDs.length,
      detailPrices.length,
      detailQuantities.length
    );

    const saleOrderDetailLines: ISaleOrderDetailLine[] = [];
    for (let detailPosition = 0; detailPosition < detailCount; detailPosition += 1) {
      const productID = detailProductIDs[detailPosition] || 0;
      const unitPrice = detailPrices[detailPosition] || 0;
      const quantity = detailQuantities[detailPosition] || 0;
      const sku = detailProductSkus[detailPosition] || '';

      saleOrderDetailLines.push({
        detailPosition,
        productID,
        productName: productosService.recordsMap.get(productID)?.Nombre || `Producto #${productID}`,
        sku,
        quantity,
        unitPrice,
        subtotalAmount: unitPrice * quantity,
      });
    }

    return saleOrderDetailLines;
  }

  function canPaySaleOrder(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 1 || saleOrder.ss === 3;
  }

  function canDeliverSaleOrder(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 1 || saleOrder.ss === 2;
  }

  const selectedSaleOrderDetailLines = $derived.by(() => {
    if (!selectedSaleOrder) { return []; }
    return getSaleOrderDetailLines(selectedSaleOrder);
  });

  function openSaleOrderDetailsLayer(saleOrder: ISaleOrder) {
    selectedSaleOrder = saleOrder;
    saleOrderDetailsView = 1;
    Core.openSideLayer(10);
  }

  function onClickPagar() {
    if (!selectedSaleOrder) {
      Notify.failure('No hay una orden seleccionada.');
      return;
    }

    // Action button is ready in UI; backend transition route is pending definition.
    Notify.failure('Acción de pago pendiente de implementación en backend.');
  }

  function onClickEntregar() {
    if (!selectedSaleOrder) {
      Notify.failure('No hay una orden seleccionada.');
      return;
    }

    // Action button is ready in UI; backend transition route is pending definition.
    Notify.failure('Acción de entrega pendiente de implementación en backend.');
  }

  // Instantiate and fetch when the selected group changes.
  $effect(() => {
    selectedGroup;
    untrack(() => {
      // Rebuild the service when group changes so cache keys stay isolated by route query.
      const nextSaleOrdersService = new SaleOrdersService(selectedGroup);
      saleOrdersService = nextSaleOrdersService;
      nextSaleOrdersService.fetch();
      selectedSaleOrder = null;
    });
  });

  const columns: ITableColumn<ISaleOrder>[] = [
    {
      header: 'ID',
      getValue: saleOrder => saleOrder.ID,
      css: 'ff-mono fs15 text-right',
      headerCss: 'w-52',
      mobile: { order: 1, css: 'col-span-6', icon: 'tag' }
    },
    {
      header: 'Fecha Hora',
      getValue: saleOrder => formatTime(saleOrder.Created, 'd-M h:n') as string,
      css: 'text-right',
      headerCss: 'w-52',
      mobile: { order: 2, css: 'col-span-6', icon: 'clock' }
    },
    {
      header: 'Total',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN(saleOrder.TotalAmount / 100, 2),
      mobile: { order: 3, css: 'col-span-12', labelLeft: 'Total:' }
    },
    {
      header: 'Deuda',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN(saleOrder.DebtAmount / 100, 2),
      mobile: { order: 4, css: 'col-span-12', labelLeft: 'Deuda:' }
    },
    {
      header: 'Top Productos',
      css: 'fs15',
      id: 'top-products',
      getValue: saleOrder => renderTopProductsSummary(saleOrder),
      mobile: { order: 5, css: 'col-span-24', labelTop: 'Top Productos', render: saleOrder => renderTopProductsSummary(saleOrder) }
    },
    {
      header: 'Estado',
      id: 'status',
      getValue: saleOrder => getSaleOrderStatusName(saleOrder),
      mobile: { order: 6, css: 'col-span-24' }
    }
  ];
</script>

<Page title="Gestión de Pedidos">
  <div class="p-10">
    <OptionsStrip
      {options}
      selected={selectedGroup}
      onSelect={(selectedOption) => {
        // Skip reactive work when user clicks the already selected tab.
        const nextSelectedGroup = selectedOption[0] as number;
        if (nextSelectedGroup === selectedGroup) { return; }
        selectedGroup = nextSelectedGroup;
      }}
      useMobileGrid
      css="mb-10"
    />

    <VTable
      {columns}
      data={saleOrdersService?.records || []}
      selected={selectedSaleOrder?.ID || 0}
      isSelected={(saleOrder, selectedID) => saleOrder.ID === selectedID}
      onRowClick={(saleOrder) => {
      	console.log("saleOrder", $state.snapshot(saleOrder))
        openSaleOrderDetailsLayer(saleOrder);
      }}
      mobileCardCss="mb-10"
    />
  </div>

  <Layer
    type="side"
    id={10}
    bind:selected={saleOrderDetailsView}
    title={selectedSaleOrder ? `Pedido #${selectedSaleOrder.ID}` : 'Detalle de Pedido'}
    titleCss="h2"
    css="px-8 py-8 md:px-16 md:py-10"
    contentCss="px-0"
    onClose={() => {
      selectedSaleOrder = null;
    }}
  >
    {#if selectedSaleOrder}
      <div class="flex flex-col gap-10 mt-8">
        <div class="grid grid-cols-24 gap-x-8 gap-y-8 text-13 md:text-14">
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Fecha</div>
            <div>{formatTime(selectedSaleOrder.Created, 'd-M-Y h:n')}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Estado</div>
            <div>{getSaleOrderStatusName(selectedSaleOrder)}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Almacén ID</div>
            <div class="ff-mono">{selectedSaleOrder.AlmacenID || '-'}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Caja ID</div>
            <div class="ff-mono">{selectedSaleOrder.CajaID_ || '-'}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Total</div>
            <div class="ff-mono">S/ {formatN(selectedSaleOrder.TotalAmount / 100, 2)}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Deuda</div>
            <div class="ff-mono">S/ {formatN(selectedSaleOrder.DebtAmount / 100, 2)}</div>
          </div>
        </div>

        <div class="border border-gray-200 rounded-md overflow-hidden">
          <div class="grid grid-cols-24 gap-6 px-8 py-6 bg-gray-100 ff-bold text-12 md:text-13">
            <div class="col-span-8">Producto</div>
            <div class="col-span-4 text-center">SKU</div>
            <div class="col-span-3 text-right">Cant.</div>
            <div class="col-span-4 text-right">Precio</div>
            <div class="col-span-5 text-right">Subtotal</div>
          </div>

          {#if selectedSaleOrderDetailLines.length === 0}
            <div class="px-8 py-10 text-gray-500 text-center">No hay productos en el detalle.</div>
          {:else}
            {#each selectedSaleOrderDetailLines as saleOrderDetailLine (saleOrderDetailLine.detailPosition)}
              <div class="grid grid-cols-24 gap-6 px-8 py-6 border-t border-gray-100 text-12 md:text-13">
                <div class="col-span-8 leading-16">{saleOrderDetailLine.productName}</div>
                <div class="col-span-4 text-center ff-mono">{saleOrderDetailLine.sku || '-'}</div>
                <div class="col-span-3 text-right ff-mono">{saleOrderDetailLine.quantity}</div>
                <div class="col-span-4 text-right ff-mono">S/ {formatN(saleOrderDetailLine.unitPrice / 100, 2)}</div>
                <div class="col-span-5 text-right ff-mono">S/ {formatN(saleOrderDetailLine.subtotalAmount / 100, 2)}</div>
              </div>
            {/each}
          {/if}
        </div>

        <div class="grid grid-cols-2 gap-8 mt-4">
          <button
            class={`bx-green h-36 ${!canPaySaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
            disabled={!canPaySaleOrder(selectedSaleOrder)}
            onclick={() => {
              onClickPagar();
            }}
          >
            <i class="icon-check"></i>
            <span>Pagar</span>
          </button>
          <button
            class={`bx-blue h-36 ${!canDeliverSaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
            disabled={!canDeliverSaleOrder(selectedSaleOrder)}
            onclick={() => {
              onClickEntregar();
            }}
          >
            <i class="icon-truck"></i>
            <span>Entregar</span>
          </button>
        </div>
      </div>
    {/if}
  </Layer>
</Page>
