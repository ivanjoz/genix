<script lang="ts">
  import SearchSelect from '$components/SearchSelect.svelte';
  import LoadingBar from '$components/micro/LoadingBar.svelte';
  import RecordByIDText from '$components/micro/RecordByIDText.svelte';
  import Layer from '$components/Layer.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import TableGrid from '$components/vTable/TableGrid.svelte';
  import type { TableGridColumn } from '$components/vTable/tableGridTypes';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Core } from '$core/store.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify, formatN, formatTime } from '$libs/helpers';
  import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
  import { AlmacenesService } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { untrack } from 'svelte';
  import {
    SaleOrderGroup,
    SaleOrdersService,
    type ISaleOrderTopProduct,
    postSaleOrderUpdate,
    type ISaleOrder
  } from './sale_order_status.svelte';
    import { getRecordByIDUpdated } from '$libs/cache/cache-by-ids.svelte';

  interface ISaleOrderDetailLine {
    detailPosition: number;
    productID: number;
    productBaseName: string;
    productName: string;
    presentationName: string;
    sku: string;
    quantity: number;
    unitPrice: number;
    subtotalAmount: number;
  }

  type ISaleOrderDetailColumn = ITableColumn<ISaleOrderDetailLine> & TableGridColumn<ISaleOrderDetailLine>;

  let selectedGroup = $state(SaleOrderGroup.PENDIENTE_DE_PAGO);
  let saleOrdersService = $state<SaleOrdersService | null>(null);
  let saleOrderDetailsView = $state(1);
  let selectedSaleOrder = $state<ISaleOrder | null>(null);
  let isPostingSaleOrderAction = $state(false);
  let saleOrderActionInProgress = $state<'pago' | 'entrega' | null>(null);
  let saleOrderPaymentForm = $state({ LastPaymentCajaID: 0 });
  let saleOrderDeliveryForm = $state({ WarehouseID: 0 });

  const productosService = new ProductosService();
  const cajasService = new CajasService();
  const almacenesService = new AlmacenesService();

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

  function isSaleOrderPaid(saleOrder: ISaleOrder): boolean {
    // Paid is true for paid and finalized orders.
    return saleOrder.ss === 2 || saleOrder.ss === 4;
  }

  function isSaleOrderDelivered(saleOrder: ISaleOrder): boolean {
    // Delivered is true for delivered and finalized orders.
    return saleOrder.ss === 3 || saleOrder.ss === 4;
  }

  function renderSaleOrderStateSquares(saleOrder: ISaleOrder): string {
    const paidBadgeCss = isSaleOrderPaid(saleOrder)
      ? 'bg-green-100 text-green-800 border border-green-300'
      : 'bg-red-100 text-red-800 border border-red-300';
    const deliveredBadgeCss = isSaleOrderDelivered(saleOrder)
      ? 'bg-green-100 text-green-800 border border-green-300'
      : 'bg-red-100 text-red-800 border border-red-300';

    // Render compact square-like badges so operators can read status at a glance.
    return `<div class="flex items-center justify-center gap-4">
      <span class="inline-flex h-22 min-w-22 px-4 rounded-[3px] text-10 ff-bold items-center justify-center ${paidBadgeCss}">PAG</span>
      <span class="inline-flex h-22 min-w-22 px-4 rounded-[3px] text-10 ff-bold items-center justify-center ${deliveredBadgeCss}">ENT</span>
    </div>`;
  }

  function getTopPaidProductsByAmount(saleOrder: ISaleOrder): ISaleOrderTopProduct[] {
    const detailCount = Math.min(
      saleOrder.DetailProductsIDs?.length || 0,
      saleOrder.DetailPrices?.length || 0,
      saleOrder.DetailQuantities?.length || 0
    );
    const productAmountByID = new Map<number, ISaleOrderTopProduct>();

    // Rebuild top products from detail arrays so hydrated rows and list rows behave the same.
    for (let detailIndex = 0; detailIndex < detailCount; detailIndex += 1) {
      const productID = saleOrder.DetailProductsIDs[detailIndex] || 0;
      const linePrice = saleOrder.DetailPrices[detailIndex] || 0;
      const lineQuantity = saleOrder.DetailQuantities[detailIndex] || 0;
      const lineAmount = linePrice * lineQuantity;
      if (!productID || lineAmount <= 0) { continue; }

      const previousProductData = productAmountByID.get(productID);
      if (previousProductData) {
        previousProductData.LineAmount += lineAmount;
        continue;
      }

      productAmountByID.set(productID, {
        ProductID: productID,
        LineAmount: lineAmount,
      });
    }

    const sortedProducts = Array.from(productAmountByID.values()).sort((leftProduct, rightProduct) => {
      if (rightProduct.LineAmount !== leftProduct.LineAmount) {
        return rightProduct.LineAmount - leftProduct.LineAmount;
      }
      return leftProduct.ProductID - rightProduct.ProductID;
    });
    if (sortedProducts.length <= 3) { return sortedProducts; }

    const tieAwareCutoffAmount = sortedProducts[2].LineAmount;
    return sortedProducts.filter((product) => product.LineAmount >= tieAwareCutoffAmount);
  }

  function renderTopProductsSummary(saleOrder: ISaleOrder): string {
    const topPaidProducts = getTopPaidProductsByAmount(saleOrder);
    if (topPaidProducts.length === 0) {
      return '-';
    }

    // Keep product-name lookup in the view and derive amounts from the raw detail arrays.
    return topPaidProducts
      .map((topProduct) => {
        const productName = productosService.recordsMap.get(topProduct.ProductID)?.Nombre || `Producto #${topProduct.ProductID}`;
        return `${productName} (${formatN(topProduct.LineAmount / 100, 2)})`;
      })
      .join(', ');
  }

  function getProductPresentationName(productID: number, presentationID: number): string {
    if (!presentationID) { return ''; }

    const productRecord = productosService.recordsMap.get(productID);
    const presentationRecord = productRecord?.Presentaciones?.find((presentationOption) => presentationOption.id === presentationID);
    return presentationRecord?.nm || `Presentación ${presentationID}`;
  }

  function getSaleOrderDetailProductName(productID: number, presentationID: number): string {
    const productRecord = productosService.recordsMap.get(productID);
    const productName = productRecord?.Nombre || `Producto #${productID}`;
    const presentationName = getProductPresentationName(productID, presentationID);

    if (!presentationName) { return productName; }
    return `${productName} (${presentationName})`;
  }

  function getSaleOrderDetailLines(saleOrder: ISaleOrder): ISaleOrderDetailLine[] {
    // Normalize arrays because old/partial records may omit one or more detail fields.
    const detailProductIDs = saleOrder.DetailProductsIDs || [];
    const detailPrices = saleOrder.DetailPrices || [];
    const detailQuantities = saleOrder.DetailQuantities || [];
    const detailProductSkus = saleOrder.DetailProductSkus || [];
    const detailProductPresentations = saleOrder.DetailProductPresentations || [];

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
      const presentationID = detailProductPresentations[detailPosition] || 0;

      saleOrderDetailLines.push({
        detailPosition,
        productID,
        productBaseName: productosService.recordsMap.get(productID)?.Nombre || `Producto #${productID}`,
        productName: getSaleOrderDetailProductName(productID, presentationID),
        presentationName: getProductPresentationName(productID, presentationID),
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

  function getCajaName(cajaID: number): string {
    if (!cajaID) { return '-'; }
    return cajasService.CajasMap.get(cajaID)?.Nombre || `Caja #${cajaID}`;
  }

  function getAlmacenName(almacenID: number): string {
    if (!almacenID) { return '-'; }
    return almacenesService.AlmacenesMap.get(almacenID)?.Nombre || `Almacén #${almacenID}`;
  }

  function formatActionTime(unixTime: number): string {
    if (!unixTime) { return '-'; }
    return String(formatTime(unixTime, 'd-M-Y h:n'));
  }

  function getActionInProgressLabel(actionInProgress: 'pago' | 'entrega' | null): string {
    // Keep message explicit so operators know the exact transition being processed.
    if (actionInProgress === 'pago') { return 'Realizando Pago...'; }
    if (actionInProgress === 'entrega') { return 'Realizando Entrega...'; }
    return 'Procesando';
  }

  const selectedSaleOrderDetailLines = $derived.by(() => {
    if (!selectedSaleOrder) { return []; }
    return getSaleOrderDetailLines(selectedSaleOrder);
  });

  const filteredSaleOrders = $derived.by(() => {
    const fetchedSaleOrders = saleOrdersService?.records || [];

    // Enforce UI-specific tab filters after service fetch to keep badge semantics strict.
    if (selectedGroup === SaleOrderGroup.PENDIENTE_DE_PAGO) {
      return fetchedSaleOrders.filter((saleOrderRecord) => !isSaleOrderPaid(saleOrderRecord));
    }
    if (selectedGroup === SaleOrderGroup.PENDIENTE_DE_ENTREGA) {
      return fetchedSaleOrders.filter((saleOrderRecord) => !isSaleOrderDelivered(saleOrderRecord));
    }
    return fetchedSaleOrders;
  });

  const detailTableColumns: ISaleOrderDetailColumn[] = [
    {
      id: 'productName',
      header: 'Producto',
      width: 'minmax(280px, 1.7fr)',
      getValue: (detailLineRecord) => detailLineRecord.productName,
      render: (detailLineRecord) => {
        if (!detailLineRecord.presentationName) {
          return detailLineRecord.productBaseName;
        }

        return `${detailLineRecord.productBaseName}<span class="text-blue-600 text-sm ff-bold"> (${detailLineRecord.presentationName})</span>`;
      },
    },
    {
      id: 'sku',
      header: 'SKU',
      width: 'minmax(120px, 0.7fr)',
      align: 'center',
      getValue: (detailLineRecord) => detailLineRecord.sku || '-',
    },
    {
      id: 'quantity',
      header: 'Cant.',
      width: '90px',
      align: 'right',
      getValue: (detailLineRecord) => detailLineRecord.quantity,
    },
    {
      id: 'unitPrice',
      header: 'Precio',
      width: '120px',
      align: 'right',
      getValue: (detailLineRecord) => `S/ ${formatN(detailLineRecord.unitPrice / 100, 2)}`,
    },
    {
      id: 'subtotalAmount',
      header: 'Subtotal',
      width: '130px',
      align: 'right',
      getValue: (detailLineRecord) => `S/ ${formatN(detailLineRecord.subtotalAmount / 100, 2)}`,
    },
  ];

  $effect(() => {
    cajasService.Cajas;
    saleOrderPaymentForm.LastPaymentCajaID;
    untrack(() => {
      // Default caja selector to the first active caja so payment can be processed in one click.
      if (saleOrderPaymentForm.LastPaymentCajaID > 0) { return; }
      const firstActiveCajaRecord = (cajasService.Cajas || []).find((cajaRecord) => (cajaRecord?.ss || 0) > 0);
      if (firstActiveCajaRecord?.ID) {
        saleOrderPaymentForm.LastPaymentCajaID = firstActiveCajaRecord.ID;
      }
    });
  });

  function openSaleOrderDetailsLayer(saleOrder: ISaleOrder) {
    selectedSaleOrder = saleOrder;
    // Keep selectors prefilled with the order values to reduce manual clicks.
    saleOrderPaymentForm.LastPaymentCajaID = saleOrder.LastPaymentCajaID || 0;
    saleOrderDeliveryForm.WarehouseID = saleOrder.WarehouseID || 0;
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
    const paymentCajaID = selectedSaleOrder.LastPaymentCajaID || saleOrderPaymentForm.LastPaymentCajaID;
    if (!paymentCajaID) {
      Notify.failure('La orden no posee Caja ID para registrar el pago.');
      return;
    }

    // Payment transition: mark as paid and set debt to zero for a full payment flow.
    void processSaleOrderAction('pago', {
      ID: selectedSaleOrder.ID,
      ActionsIncluded: [2],
      LastPaymentCajaID: paymentCajaID,
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
    // Delivery uses the selector value so the operator can choose a different almacén.
    const selectedAlmacenID = saleOrderDeliveryForm.WarehouseID || selectedSaleOrder.WarehouseID;
    if (!selectedAlmacenID) {
      Notify.failure('La orden no posee Almacén ID para registrar la entrega.');
      return;
    }

    // Delivery transition: trigger stock movement using the existing order details.
    void processSaleOrderAction('entrega', {
      ID: selectedSaleOrder.ID,
      ActionsIncluded: [3],
      WarehouseID: selectedAlmacenID,
    });
  }

  function onClickAnularSoloUI() {
    // This button is intentionally UI-only until backend supports cancel action.
    Notify.failure('Anulación disponible próximamente.');
  }

  function getSaleOrderLayerTitle(saleOrder: ISaleOrder | null): string {
    if (!saleOrder) { return 'Detalle de Pedido'; }
    // Keep ID and date in the same title line as requested.
    return `Pedido #${saleOrder.ID} · ${formatTime(saleOrder.Created, 'd-M-Y h:n')}`;
  }

  async function processSaleOrderAction(actionLabel: 'pago' | 'entrega', updatePayload: {
    ID: number;
    ActionsIncluded: number[];
    LastPaymentCajaID?: number;
    WarehouseID?: number;
    DebtAmount?: number;
  }) {
    if (!selectedSaleOrder) { return; }
    // Lock the action area to a single progress card while this transition is running.
    saleOrderActionInProgress = actionLabel;
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
      saleOrderActionInProgress = null;
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
      headerCss: 'w-100', cellCss: "whitespace-nowrap",
      mobile: { order: 2, css: 'col-span-6', icon: 'clock' }
    },
    {
      header: 'Estado',
      id: 'status',
      headerCss: 'w-82',
      css: 'text-center',
      render: saleOrder => renderSaleOrderStateSquares(saleOrder),
      mobile: {
        order: 3,
        css: 'col-span-24',
        render: saleOrder => renderSaleOrderStateSquares(saleOrder)
      }
    },
    {
      header: 'Total',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN(saleOrder.TotalAmount / 100, 2),
      mobile: { order: 4, css: 'col-span-12', labelLeft: 'Total:' }
    },
    {
      header: 'Deuda',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN((saleOrder.DebtAmount || 0) / 100, 2),
      mobile: { order: 5, css: 'col-span-12', labelLeft: 'Deuda:' }
    },
    {
      header: 'Top Productos',
      css: 'fs15 leading-[1.15]',
      id: 'top-products',
      getValue: saleOrder => renderTopProductsSummary(saleOrder),
      mobile: { order: 6, css: 'col-span-24', labelTop: 'Top Productos', render: saleOrder => renderTopProductsSummary(saleOrder) }
    }
  ];
</script>

<Page title="Gestión de Pedidos">
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
	       data={filteredSaleOrders}
	       selected={selectedSaleOrder?.ID || 0}
	       isSelected={(saleOrder, selectedID) => saleOrder.ID === selectedID}
	       onRowClick={(saleOrder) => {
	       	console.log("saleOrder", $state.snapshot(saleOrder))
	         openSaleOrderDetailsLayer(saleOrder);
	       }}
				 estimateSize={38}
	       mobileCardCss="mb-10"
				 getRowObject={(e) => getRecordByIDUpdated("sale-order-by-ids", e.ID||0, e.upd||0) as Promise<ISaleOrder>}
	     />
     </Layer>
  </div>

  <Layer
    type="side"
    id={10}
    sideLayerSize={780}
    bind:selected={saleOrderDetailsView}
    title={getSaleOrderLayerTitle(selectedSaleOrder)}
    titleCss="h2"
    css="px-8 py-8 md:px-16 md:py-10"
    contentCss="px-0"
    onClose={() => {
      selectedSaleOrder = null;
      saleOrderActionInProgress = null;
    }}
  >
    {#snippet titleSide()}
      {#if selectedSaleOrder}
     	<div class="flex items-center">
    		<div class="mr-8 ml-6 ff-semibold">{formatTime(selectedSaleOrder.Created, 'd-M-Y h:n')}</div>
	       <button
	         class="w-30 h-30 text-sm rounded-full bg-red-100 text-red-700 fx-c"
	         type="button"
	         title="Anular pedido"
	         onclick={() => {
	           onClickAnularSoloUI();
	         }}
	       >
	         <i class="icon-trash"></i>
	       </button>
      </div>
      {/if}
    {/snippet}
    {#if selectedSaleOrder}
      <div class="flex flex-col gap-10 mt-8">
        <div class="grid grid-cols-24 gap-x-8 gap-y-8 text-13 md:text-14">
          <div class="col-span-8">
            <div class="text-gray-500">Estado</div>
            <div>{getSaleOrderStatusName(selectedSaleOrder)}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500">Total</div>
            <div class="ff-mono">S/ {formatN(selectedSaleOrder.TotalAmount / 100, 2)}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500">Deuda</div>
            <div class="ff-mono">S/ {formatN((selectedSaleOrder.DebtAmount || 0) / 100, 2)}</div>
          </div>
        </div>

        <VTable
          columns={detailTableColumns}
          data={selectedSaleOrderDetailLines}
          emptyMessage="No hay productos en el detalle."
        />

        <div class="grid grid-cols-2 gap-8 mt-4">
          {#if isPostingSaleOrderAction}
            <div class="col-span-2 p-12 bg-gray-100 min-h-64 w-full rounded-md fx-c">
              <LoadingBar label={getActionInProgressLabel(saleOrderActionInProgress)} />
            </div>
          {:else}
            <div class="p-10 bg-gray-100 min-h-64 w-full rounded-md">
              {#if canPaySaleOrder(selectedSaleOrder)}
                <SearchSelect
                  bind:saveOn={saleOrderPaymentForm}
                  save="LastPaymentCajaID"
                  css="mb-8"
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
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label mr-2">Pagado el:</span> {formatActionTime(selectedSaleOrder.LastPaymentTime)}.<br>
                  <span class="ff-bold text-xs color-label mr-2">En:</span> {getCajaName(selectedSaleOrder.LastPaymentCajaID)}.<br>
                  <span class="ff-bold text-xs color-label mr-2">Usuario:</span>
                  <RecordByIDText apiRoute="usuarios-ids" recordID={selectedSaleOrder.LastPaymentUser} placeholder="-" />.
                </div>
              {/if}
            </div>
            <div class="p-10 bg-gray-100 min-h-64 w-full rounded-md flex flex-col justify-between">
              {#if canDeliverSaleOrder(selectedSaleOrder)}
                <SearchSelect
                  bind:saveOn={saleOrderDeliveryForm}
                  save="WarehouseID"
                  css="mb-8 w-full"
                  inputCss="w-full"
                  options={(almacenesService.Almacenes || []).filter((almacenRecord) => (almacenRecord?.ss || 0) > 0)}
                  keyId="ID"
                  keyName="Nombre"
                  label="Almacén para Entrega"
                  placeholder=":: seleccione ::"
                />
                <div class="flex items-center">
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
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label">Entregado el</span> {formatActionTime(selectedSaleOrder.DeliveryTime)}.<br>
                  <span class="ff-bold text-xs color-label">En</span> {getAlmacenName(selectedSaleOrder.WarehouseID)}.<br>
                  <RecordByIDText apiRoute="usuarios-ids" recordID={selectedSaleOrder.DeliveryUser} placeholder="-" />.
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </Layer>
</Page>
