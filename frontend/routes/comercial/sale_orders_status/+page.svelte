<script lang="ts">
  import Layer from '$components/layers/Layer.svelte';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import LoadingBar from '$components/misc/LoadingBar.svelte';
  import RecordByIDText from '$components/misc/RecordByIDText.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import { Core, tr } from '$core/store.svelte';
  import T from '$components/misc/T.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify, formatN, formatTime } from '$libs/helpers';
  import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
  import {
      ClientProviderService,
      type IClientProvider
  } from '$routes/negocio/clientes/clientes-proveedores.svelte';
  import {
      ProductosService,
      type IProduct,
      type IProductPresentation,
  } from '$routes/negocio/productos/productos.svelte';
  import { AlmacenesService } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte';
  import { onMount, untrack } from 'svelte';
  import SaleOrdersTable from '../SaleOrdersTable.svelte';
  import {
      SaleOrderGroup,
      SaleOrdersService,
      postSaleOrderUpdate,
      type ISaleOrder
  } from './sale_order_status.svelte';

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

  interface ISaleOrderFilterForm {
    clientID: number;
    productID: number;
  }

  interface ISaleOrderClientOption extends IClientProvider {
    DisplayName: string;
  }

  type ISaleOrderDetailColumn = ITableColumn<ISaleOrderDetailLine>;

  let selectedGroup = $state(SaleOrderGroup.PENDIENTE_DE_PAGO);
  let saleOrderRecords = $state<ISaleOrder[]>([]);
  let saleOrderDetailsView = $state(1);
  let selectedSaleOrder = $state<ISaleOrder | null>(null);
  let isQueryingSaleOrders = $state(false);
  let isPostingSaleOrderAction = $state(false);
  let saleOrderActionInProgress = $state<'pago' | 'entrega' | null>(null);
  let saleOrderPaymentForm = $state({ LastPaymentCajaID: 0 });
  let saleOrderDeliveryForm = $state({ WarehouseID: 0 });
  let saleOrderFilterForm = $state<ISaleOrderFilterForm>({ clientID: 0, productID: 0 });
  let clientOptions = $state<ISaleOrderClientOption[]>([]);
  let productOptions = $state<IProduct[]>([]);
  let saleOrdersQueryRequestID = 0;

  const cajasService = new CajasService();
  const almacenesService = new AlmacenesService();
  const productosService = new ProductosService();
  const clientesService = new ClientProviderService();

  function withTimeout<T>(promise: Promise<T>, timeoutLabel: string, timeoutMs: number = 8000): Promise<T> {
    // Prevent auxiliary lookups from leaving the whole page in a perpetual loading state.
    return Promise.race([
      promise,
      new Promise<T>((_, reject) => {
        setTimeout(() => reject(new Error(`Timeout while waiting for ${timeoutLabel}`)), timeoutMs);
      }),
    ]);
  }

  async function querySaleOrders(orderStatus: number): Promise<void> {
    const currentRequestID = ++saleOrdersQueryRequestID;
    selectedSaleOrder = null;
    isQueryingSaleOrders = true;
    saleOrderRecords = [];
    console.debug(`[sale_orders_status] query:start req=${currentRequestID} status=${orderStatus}`);

    const nextSaleOrdersService = new SaleOrdersService(orderStatus);
    const queryRoute = nextSaleOrdersService.route;
    console.debug('[sale_orders_status] querying sale orders', {
      orderStatus,
      queryRoute,
    });

    await nextSaleOrdersService.fetchOnline();
    const fetchedSaleOrders = nextSaleOrdersService.records;
    console.debug(`[sale_orders_status] query:fetched req=${currentRequestID} rows=${fetchedSaleOrders.length}`);
    if (currentRequestID !== saleOrdersQueryRequestID) return;

    // Commit table rows first so the page does not stay blank while related lookups resolve.
    saleOrderRecords = fetchedSaleOrders;

    const uniqueProductIDs = new Set<number>();
    const uniqueClientIDs = new Set<number>();
    for (const saleOrder of fetchedSaleOrders) {
      const clientID = Number(saleOrder.ClientID || 0);
      if (Number.isFinite(clientID) && clientID > 0) {
        uniqueClientIDs.add(clientID);
      }

      const detailProductIDs = saleOrder.DetailProductsIDs || [];
      for (const rawProductID of detailProductIDs) {
        const productID = Number(rawProductID || 0);
        if (!Number.isFinite(productID) || productID <= 0) continue;
        uniqueProductIDs.add(productID);
      }
    }
    const productIDs = Array.from(uniqueProductIDs);
    const clientIDs = Array.from(uniqueClientIDs);

    console.debug('[sale_orders_status] querying related records', {
      saleOrderCount: fetchedSaleOrders.length,
      productIDsCount: productIDs.length,
      clientIDsCount: clientIDs.length,
    });

    try {
      const [productosSyncResult, clientesSyncResult] = await Promise.allSettled([
        withTimeout(productosService.syncIDs(productIDs), 'productosService.syncIDs'),
        withTimeout(clientesService.syncIDs(clientIDs), 'clientesService.syncIDs'),
      ]);
      console.debug(
        `[sale_orders_status] sync:settled req=${currentRequestID} productos=${productosSyncResult.status} clientes=${clientesSyncResult.status}`
      );
      if (currentRequestID !== saleOrdersQueryRequestID) return;

      if (productosSyncResult.status === 'rejected') {
        console.error('[sale_orders_status] failed to sync products by IDs', {
          queryRoute,
          productIDs,
          productosSyncError: productosSyncResult.reason,
        });
      }
      if (clientesSyncResult.status === 'rejected') {
        console.error('[sale_orders_status] failed to sync clients by IDs', {
          queryRoute,
          clientIDs,
          clientesSyncError: clientesSyncResult.reason,
        });
      }

      // Keep selectors scoped to the current query result so they never show unrelated cached records.
      clientOptions = clientIDs
        .map((clientID) => clientesService.recordsMap.get(clientID))
        .filter((clientRecord): clientRecord is IClientProvider => Boolean(clientRecord))
        .map((clientRecord) => ({
          ...clientRecord,
          DisplayName: clientRecord.Name
            ? `${clientRecord.Name} · ${clientRecord.ID}`
            : `Cliente #${clientRecord.ID}`,
        }))
        .sort((leftClient, rightClient) => leftClient.DisplayName.localeCompare(rightClient.DisplayName));

      // Reuse the synced records map but expose only the products present in the loaded sale orders.
      productOptions = productIDs
        .map((productID) => productosService.recordsMap.get(productID))
        .filter((productRecord): productRecord is IProduct => Boolean(productRecord))
        .sort((leftProduct, rightProduct) => leftProduct.Name.localeCompare(rightProduct.Name));

      console.debug('[sale_orders_status] sale orders ready', {
        saleOrdersCount: fetchedSaleOrders.length,
        resolvedProducts: productIDs.filter((productID) => productosService.recordsMap.has(productID)).length,
        resolvedClients: clientIDs.filter((clientID) => clientesService.recordsMap.has(clientID)).length,
      });
      console.debug(
        `[sale_orders_status] query:ready req=${currentRequestID} rows=${fetchedSaleOrders.length} products=${productOptions.length} clients=${clientOptions.length}`
      );
    } catch (queryError) {
      if (currentRequestID !== saleOrdersQueryRequestID) return;
      console.error(`[sale_orders_status] query:error req=${currentRequestID} route=${queryRoute}`);
      console.error('[sale_orders_status] failed to query sale orders', {
        queryError,
        queryRoute,
        productIDs,
        clientIDs,
      });
      // Keep the already loaded sale orders visible even if auxiliary lookups fail.
    } finally {
      if (currentRequestID === saleOrdersQueryRequestID) {
        isQueryingSaleOrders = false;
        console.debug(`[sale_orders_status] query:end req=${currentRequestID} loading=${isQueryingSaleOrders}`);
      }
    }
  }

  // Render tabs mapped to backend status filters.
  const options = [
    [SaleOrderGroup.PENDIENTE_DE_PAGO, 'Pend. Payment|Pend. Pago'],
    [SaleOrderGroup.PENDIENTE_DE_ENTREGA, 'Pend. Delivery|Pend. Entrega'],
    [SaleOrderGroup.FINALIZADO, 'Completed|Finalizadas']
  ];

  function getSaleOrderStatusName(saleOrder: ISaleOrder): string {
    switch(saleOrder.ss) {
      case 1: return tr('Generated|Generado');
      case 2: return tr('Paid|Pagado');
      case 3: return tr('Delivered|Entregado');
      case 4: return tr('Completed|Finalizado');
      default: return tr('Unknown|Desconocido');
    }
  }

  function saleOrderHasSelectedProduct(saleOrder: ISaleOrder, selectedProductID: number): boolean {
    if (!selectedProductID) { return true; }
    return (saleOrder.DetailProductsIDs || []).some((productID) => Number(productID || 0) === selectedProductID);
  }

  function getProductPresentationName(productID: number, presentationID: number): string {
    if (!presentationID) { return ''; }

    const productRecord = productosService.recordsMap.get(productID);
    const presentationRecord = productRecord?.Presentations?.find((presentationOption: IProductPresentation) => presentationOption.id === presentationID);
    return presentationRecord?.nm || `Presentación ${presentationID}`;
  }

  function getSaleOrderDetailProductName(productID: number, presentationID: number): string {
    const productRecord = productosService.recordsMap.get(productID);
    const productName = productRecord?.Name || `Producto #${productID}`;
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
        productBaseName: productosService.recordsMap.get(productID)?.Name || `Producto #${productID}`,
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
    return cajasService.CajasMap.get(cajaID)?.Name || `Caja #${cajaID}`;
  }

  function getAlmacenName(almacenID: number): string {
    if (!almacenID) { return '-'; }
    return almacenesService.AlmacenesMap.get(almacenID)?.Name || `Almacén #${almacenID}`;
  }

  function formatActionTime(unixTime: number): string {
    if (!unixTime) { return '-'; }
    return String(formatTime(unixTime, 'd-M-Y h:n'));
  }

  function getActionInProgressLabel(actionInProgress: 'pago' | 'entrega' | null): string {
    // Keep message explicit so operators know the exact transition being processed.
    if (actionInProgress === 'pago') { return tr('Processing Payment...|Realizando Pago...'); }
    if (actionInProgress === 'entrega') { return tr('Processing Delivery...|Realizando Entrega...'); }
    return tr('Processing...|Procesando...');
  }

  function resetSaleOrderActionForms(saleOrder: ISaleOrder): void {
    // Rehydrate mutable selectors from the latest backend state after each action.
    saleOrderPaymentForm.LastPaymentCajaID = saleOrder.LastPaymentCajaID || 0;
    saleOrderDeliveryForm.WarehouseID = saleOrder.WarehouseID || 0;
  }

  function applyUpdatedSaleOrderLocally(updatedSaleOrder: ISaleOrder): void {
    // Keep side panel and current table row in sync with the write response.
    // Do not remove the row from the current table; let the next full query/page reload do that.
    selectedSaleOrder = updatedSaleOrder;
    resetSaleOrderActionForms(updatedSaleOrder);

    const saleOrderIndex = saleOrderRecords.findIndex((saleOrderRecord) => saleOrderRecord.ID === updatedSaleOrder.ID);
    if (saleOrderIndex < 0) { return; }

    const nextSaleOrderRecords = [...saleOrderRecords];
    nextSaleOrderRecords[saleOrderIndex] = updatedSaleOrder;
    saleOrderRecords = nextSaleOrderRecords;
  }

  const selectedSaleOrderDetailLines = $derived.by(() => {
    if (!selectedSaleOrder) { return []; }
    return getSaleOrderDetailLines(selectedSaleOrder);
  });

  const filteredSaleOrderRecords = $derived.by(() => {
    const selectedClientID = Number(saleOrderFilterForm.clientID || 0);
    const selectedProductID = Number(saleOrderFilterForm.productID || 0);

    return saleOrderRecords.filter((saleOrder) => {
      if (selectedClientID > 0 && Number(saleOrder.ClientID || 0) !== selectedClientID) {
        return false;
      }
      return saleOrderHasSelectedProduct(saleOrder, selectedProductID);
    });
  });

  const detailTableColumns: ISaleOrderDetailColumn[] = [
    {
      id: 'productName',
      header: 'Product|Producto',
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
      header: 'Qty.|Cant.',
      width: '90px',
      align: 'right',
      getValue: (detailLineRecord) => detailLineRecord.quantity,
    },
    {
      id: 'unitPrice',
      header: 'Price|Precio',
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
    resetSaleOrderActionForms(saleOrder);
    saleOrderDetailsView = 1;
    Core.openSideLayer(10);
  }

  function onClickPagar() {
    if (!selectedSaleOrder) {
      Notify.failure(tr('No order selected.|No hay una orden seleccionada.'));
      return;
    }
    if (isPostingSaleOrderAction) {
      Notify.failure(tr('An action is already in progress for this order.|Ya se está procesando una acción para esta orden.'));
      return;
    }
    // Payment caja priority: order caja first, otherwise use the user-selected caja.
    const paymentCajaID = selectedSaleOrder.LastPaymentCajaID || saleOrderPaymentForm.LastPaymentCajaID;
    if (!paymentCajaID) {
      Notify.failure(tr('Order has no Cash Register ID for payment.|La orden no posee Caja ID para registrar el pago.'));
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
      Notify.failure(tr('No order selected.|No hay una orden seleccionada.'));
      return;
    }
    if (isPostingSaleOrderAction) {
      Notify.failure(tr('An action is already in progress for this order.|Ya se está procesando una acción para esta orden.'));
      return;
    }
    // Delivery uses the selector value so the operator can choose a different almacén.
    const selectedAlmacenID = saleOrderDeliveryForm.WarehouseID || selectedSaleOrder.WarehouseID;
    if (!selectedAlmacenID) {
      Notify.failure(tr('Order has no Warehouse ID for delivery.|La orden no posee Almacén ID para registrar la entrega.'));
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
    Notify.failure(tr('Cancellation coming soon.|Anulación disponible próximamente.'));
  }

  function getSaleOrderLayerTitle(saleOrder: ISaleOrder | null): string {
    if (!saleOrder) { return tr('Order Detail|Detalle de Pedido'); }
    // Keep ID and date in the same title line as requested.
    return tr(`Order #${saleOrder.ID} · ${formatTime(saleOrder.Created, 'd-M-Y h:n')}|Pedido #${saleOrder.ID} · ${formatTime(saleOrder.Created, 'd-M-Y h:n')}`);
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
      const updatedSaleOrder = await postSaleOrderUpdate(updatePayload) as ISaleOrder;
      applyUpdatedSaleOrderLocally(updatedSaleOrder);
      
      Notify.success(tr(`Order updated (${actionLabel}).|Pedido actualizado (${actionLabel}).`));
      console.debug('[sale_orders_status] action success', {
        actionLabel,
        saleOrderID: updatePayload.ID,
      });
    } catch (error) {
      console.error('[sale_orders_status] action error', {
        actionLabel,
        updatePayload,
        error,
      });
      Notify.failure(tr(`Could not process: ${actionLabel}.|No se pudo procesar la ${actionLabel}.`));
    } finally {
      isPostingSaleOrderAction = false;
      saleOrderActionInProgress = null;
    }
  }

  onMount(() => {
    void querySaleOrders(selectedGroup);
  });

</script>

<Page title="Order Management|Gestión de Pedidos">
  <div class="">
    <div class="flex flex-col gap-10 mb-10 xl:flex-row xl:items-center">
      <div class="h-46 flex items-center">
        <OptionsStrip
          {options}
          selected={selectedGroup}
          onSelect={(selectedOption) => {
            const nextSelectedGroup = selectedOption[0] as number;
            if (nextSelectedGroup === selectedGroup) { return; }
            selectedGroup = nextSelectedGroup;
            void querySaleOrders(nextSelectedGroup);
          }}
          useMobileGrid
        />
      </div>

      <div class="grid grid-cols-24 gap-10 md:ml-16 grow-1">
        <SearchSelect
          bind:saveOn={saleOrderFilterForm}
          save="clientID"
          css="col-span-10 md:col-span-6"
          label=""
          keyId="ID"
          keyName="DisplayName"
          options={clientOptions}
          placeholder="CLIENT|CLIENTE ::"
        />
        <SearchSelect
          bind:saveOn={saleOrderFilterForm}
          save="productID"
          css="col-span-14 md:col-span-9"
          label=""
          keyId="ID"
          keyName="Name" placeholder="PRODUCT|PRODUCTO ::"
          options={productOptions}
        />
      </div>
    </div>
     <Layer type="content">
       {#if isQueryingSaleOrders}
         <div class="p-8 min-h-240 w-full fx-c rounded-md bg-gray-50">
           <LoadingBar label={tr("Loading orders...|Cargando pedidos...")} />
         </div>
       {:else}
	       <SaleOrdersTable maxHeight="calc(100vh - var(--header-height) - 84px)"
	         data={filteredSaleOrderRecords}
	         productsMap={productosService.recordsMap}
	         clientsMap={clientesService.recordsMap}
	         selected={selectedSaleOrder?.ID || 0}
	         isSelected={(saleOrder, selectedID) => saleOrder.ID === selectedID}
	         onRowClick={(saleOrder) => {
	         	console.log("saleOrder", $state.snapshot(saleOrder))
	           openSaleOrderDetailsLayer(saleOrder);
	         }}
	       />
       {/if}
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
	       <button
	         class="w-30 h-30 text-sm rounded-full bg-red-100 text-red-700 fx-c"
	         type="button"
	         title={tr("Cancel order|Anular pedido")}
	         aria-label="Cancel this sale order"
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
      <div class="flex flex-col gap-10 mt-8" aria-label="Sale order detail panel with status, amounts, products, and action buttons">
        <div class="grid grid-cols-24 gap-x-8 gap-y-8 text-13 md:text-14" aria-label="Sale order summary: status, total, and debt">
          <div class="col-span-8">
            <div class="text-gray-500"><T text="Status|Estado" /></div>
            <div>{getSaleOrderStatusName(selectedSaleOrder)}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500">Total</div>
            <div class="ff-mono">S/ {formatN(selectedSaleOrder.TotalAmount / 100, 2)}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500"><T text="Debt|Deuda" /></div>
            <div class="ff-mono">S/ {formatN((selectedSaleOrder.DebtAmount || 0) / 100, 2)}</div>
          </div>
        </div>

        <VTable
          columns={detailTableColumns}
          data={selectedSaleOrderDetailLines}
          emptyMessage="No products in this order.|No hay productos en el detalle."
        />

        <div class="grid grid-cols-2 gap-8 mt-4" aria-label="Sale order actions: payment and delivery">
          {#if isPostingSaleOrderAction}
            <div class="col-span-2 p-12 bg-gray-100 min-h-64 w-full rounded-md fx-c">
              <LoadingBar label={getActionInProgressLabel(saleOrderActionInProgress)} />
            </div>
          {:else}
            <div class="p-10 bg-gray-100 min-h-64 w-full rounded-md" aria-label="Payment action panel with cash register selector">
              {#if canPaySaleOrder(selectedSaleOrder)}
                <SearchSelect
                  bind:saveOn={saleOrderPaymentForm}
                  save="LastPaymentCajaID"
                  css="mb-8"
                  options={(cajasService.Cajas || []).filter((cajaRecord) => (cajaRecord?.ss || 0) > 0)}
                  keyId="ID"
                  keyName="Name"
                  label="Cash Register for Payment|Caja para Pago"
                  placeholder=":: seleccione ::"
                />
                <button class={`bx-green justify-center w-[50%] h-36 ${!canPaySaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
                  disabled={!canPaySaleOrder(selectedSaleOrder) || isPostingSaleOrderAction}
                  aria-label="Process payment for this sale order"
                  onclick={() => {
                    onClickPagar();
                  }}
                >
                  <i class="icon-ok"></i>
                  <span class="mr-12"><T text="Pay|Pagar" /></span>
                </button>
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label mr-2"><T text="Paid on:|Pagado el:" /></span> {formatActionTime(selectedSaleOrder.LastPaymentTime)}.<br>
                  <span class="ff-bold text-xs color-label mr-2"><T text="At:|En:" /></span> {getCajaName(selectedSaleOrder.LastPaymentCajaID)}.<br>
                  <span class="ff-bold text-xs color-label mr-2"><T text="User:|Usuario:" /></span>
                  <RecordByIDText apiRoute="usuarios-ids" recordID={selectedSaleOrder.LastPaymentUser} placeholder="-" />.
                </div>
              {/if}
            </div>
            <div class="p-10 bg-gray-100 min-h-64 w-full rounded-md flex flex-col justify-between" aria-label="Delivery action panel with warehouse selector">
              {#if canDeliverSaleOrder(selectedSaleOrder)}
                <SearchSelect
                  bind:saveOn={saleOrderDeliveryForm}
                  save="WarehouseID"
                  css="mb-8 w-full"
                  inputCss="w-full"
                  options={(almacenesService.Almacenes || []).filter((almacenRecord) => (almacenRecord?.ss || 0) > 0)}
                  keyId="ID"
                  keyName="Name"
                  label="Warehouse for Delivery|Almacén para Entrega"
                  placeholder=":: seleccione ::"
                />
                <div class="flex items-center">
                  <button class={`w-[50%] justify-center bx-blue h-36 ${!canDeliverSaleOrder(selectedSaleOrder) ? 'opacity-60' : ''}`}
                    disabled={!canDeliverSaleOrder(selectedSaleOrder) || isPostingSaleOrderAction}
                    aria-label="Mark this sale order as delivered from the selected warehouse"
                    onclick={() => {
                      onClickEntregar();
                    }}
                  >
                    <i class="icon-truck"></i>
                    <span><T text="Deliver|Entregar" /></span>
                  </button>
                </div>
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label"><T text="Delivered on|Entregado el" /></span> {formatActionTime(selectedSaleOrder.DeliveryTime)}.<br>
                  <span class="ff-bold text-xs color-label"><T text="At|En" /></span> {getAlmacenName(selectedSaleOrder.WarehouseID)}.<br>
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
