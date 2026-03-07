<script lang="ts">
  import SearchSelect from '$components/SearchSelect.svelte';
  import Layer from '$components/Layer.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Core } from '$core/store.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify, formatN, formatTime } from '$libs/helpers';
  import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { untrack } from 'svelte';
  import {
    SaleOrderGroup,
    SaleOrdersService,
    postSaleOrderUpdate,
    type ISaleOrder
  } from './sale_order_status.svelte';

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
  let isPostingSaleOrderAction = $state(false);
  let saleOrderPaymentForm = $state({ CajaID_: 0 });

  const productosService = new ProductosService();
  const cajasService = new CajasService();

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
    if (isPostingSaleOrderAction) {
      Notify.failure('Ya se está procesando una acción para esta orden.');
      return;
    }
    // Payment caja priority: order caja first, otherwise use the user-selected caja.
    const paymentCajaID = selectedSaleOrder.CajaID_ || saleOrderPaymentForm.CajaID_;
    if (!paymentCajaID) {
      Notify.failure('La orden no posee Caja ID para registrar el pago.');
      return;
    }

    // Payment transition: mark as paid and set debt to zero for a full payment flow.
    void processSaleOrderAction('pago', {
      ID: selectedSaleOrder.ID,
      ActionsIncluded: [2],
      CajaID_: paymentCajaID,
      DebtAmount: 0,
    });
  }

  function onClickEntregar() {
    if (!selectedSaleOrder) {
      Notify.failure('No hay una orden seleccionada.');
      return;
    }
    if (isPostingSaleOrderAction) {
      Notify.failure('Ya se está procesando una acción para esta orden.');
      return;
    }
    if (!selectedSaleOrder.AlmacenID) {
      Notify.failure('La orden no posee Almacén ID para registrar la entrega.');
      return;
    }

    // Delivery transition: trigger stock movement using the existing order details.
    void processSaleOrderAction('entrega', {
      ID: selectedSaleOrder.ID,
      ActionsIncluded: [3],
      AlmacenID: selectedSaleOrder.AlmacenID,
    });
  }

  async function processSaleOrderAction(actionLabel: 'pago' | 'entrega', updatePayload: {
    ID: number;
    ActionsIncluded: number[];
    CajaID_?: number;
    AlmacenID?: number;
    DebtAmount?: number;
  }) {
    if (!selectedSaleOrder) { return; }
    isPostingSaleOrderAction = true;
    console.debug('[sale_orders_status] starting action', {
      actionLabel,
      saleOrderID: selectedSaleOrder.ID,
      updatePayload,
    });

    try {
      const updateResult = await postSaleOrderUpdate(updatePayload);
      const updatedSaleOrder = updateResult as ISaleOrder;
      selectedSaleOrder = updatedSaleOrder;
      // Force immediate refresh to sync table group filters after status transitions.
      saleOrdersService?.fetch();
      Notify.success(`Pedido actualizado (${actionLabel}).`);
      console.debug('[sale_orders_status] action success', {
        actionLabel,
        saleOrderID: updatedSaleOrder.ID,
        nextStatus: updatedSaleOrder.ss,
      });
    } catch (error) {
      console.error('[sale_orders_status] action error', {
        actionLabel,
        updatePayload,
        error,
      });
      Notify.failure(`No se pudo procesar la ${actionLabel}.`);
    } finally {
      isPostingSaleOrderAction = false;
    }
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
      getValue: saleOrder => formatN((saleOrder.DebtAmount || 0) / 100, 2),
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

<Page title="Gestión de Pedidos" sideLayerSize={780}>
  <div class="">
	 	<div class="h-46 flex items-center">
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
	  </div>
     <Layer type="content">
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
     </Layer>
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
            <div class="text-gray-500">Total</div>
            <div class="ff-mono">S/ {formatN(selectedSaleOrder.TotalAmount / 100, 2)}</div>
          </div>
          <div class="col-span-12 md:col-span-8">
            <div class="text-gray-500">Deuda</div>
            <div class="ff-mono">S/ {formatN((selectedSaleOrder.DebtAmount || 0) / 100, 2)}</div>
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
	       	<div class="p-10 bg-gray-100 min-h-64 w-full rounded-md">
          	<SearchSelect
              bind:saveOn={saleOrderPaymentForm}
              save="CajaID_" css="mb-8"
              options={(cajasService.Cajas || []).filter((cajaRecord) => (cajaRecord?.ss || 0) > 0)}
              keyId="ID"
              keyName="Nombre"
              label="Caja para Pago"
              placeholder=":: seleccione ::"
            />
		        <button class={`bx-green justify-center w-[50%] h-36 ${!canPaySaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
		          disabled={!canPaySaleOrder(selectedSaleOrder) || isPostingSaleOrderAction}
		          onclick={() => {
		            onClickPagar();
		          }}
		        >
		          <i class="icon-ok"></i>
		          <span class="mr-12">Pagar</span>
		        </button>
	        </div>
					<div class="p-10 bg-gray-100 min-h-64 w-full rounded-md flex items-end">
	      		<button class={`w-[50%] justify-center bx-blue h-36 ${!canDeliverSaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
	            disabled={!canDeliverSaleOrder(selectedSaleOrder) || isPostingSaleOrderAction}
	            onclick={() => {
	              onClickEntregar();
	            }}
	          >
	            <i class="icon-truck"></i>
	            <span>Entregar</span>
	          </button>
					</div>
        </div>
      </div>
    {/if}
  </Layer>
</Page>
