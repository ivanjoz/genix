<script lang="ts">
  import SearchSelect from '$components/SearchSelect.svelte';
  import LoadingBar from '$components/micro/LoadingBar.svelte';
  import RecordByIDText from '$components/micro/RecordByIDText.svelte';
  import Layer from '$components/Layer.svelte';
  import OptionsStrip from '$components/OptionsStrip.svelte';
  import TableGrid from '$components/vTable/TableGrid.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { TableGridColumn } from '$components/vTable/tableGridTypes';
  import type { ITableColumn } from '$components/vTable/types';
  import { Core } from '$core/store.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify, formatN, formatTime } from '$libs/helpers';
  import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
  import { AlmacenesService } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte';
  import {
    ClientProviderService,
    ClientProviderType,
    type IClientProvider,
  } from '$routes/negocio/clientes/clientes-proveedores.svelte';
  import {
    ProductosService,
    type IProducto,
    type IProductoPresentacion,
  } from '$routes/negocio/productos/productos.svelte';
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

  type ISaleOrderDetailColumn = ITableColumn<ISaleOrderDetailLine> & TableGridColumn<ISaleOrderDetailLine>;

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
  let productOptions = $state<IProducto[]>([]);
  let saleOrdersQueryRequestID = 0;

  const cajasService = new CajasService();
  const almacenesService = new AlmacenesService();
  const productosService = new ProductosService();
  const clientesService = new ClientProviderService();

  async function querySaleOrders(orderStatus: number): Promise<void> {
    const currentRequestID = ++saleOrdersQueryRequestID;
    selectedSaleOrder = null;
    isQueryingSaleOrders = true;
    saleOrderRecords = [];

    const nextSaleOrdersService = new SaleOrdersService(orderStatus);
    const queryRoute = nextSaleOrdersService.route;
    console.debug('[sale_orders_status] querying sale orders', {
      orderStatus,
      queryRoute,
    });

    await nextSaleOrdersService.fetchOnline();
    const fetchedSaleOrders = nextSaleOrdersService.records;
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
      await Promise.all([
        productosService.syncIDs(productIDs),
        clientesService.syncIDs(clientIDs),
      ]);
      if (currentRequestID !== saleOrdersQueryRequestID) return;

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
        .filter((productRecord): productRecord is IProducto => Boolean(productRecord))
        .sort((leftProduct, rightProduct) => leftProduct.Nombre.localeCompare(rightProduct.Nombre));

      console.debug('[sale_orders_status] sale orders ready', {
        saleOrdersCount: fetchedSaleOrders.length,
        resolvedProducts: productIDs.filter((productID) => productosService.recordsMap.has(productID)).length,
        resolvedClients: clientIDs.filter((clientID) => clientesService.recordsMap.has(clientID)).length,
      });
    } catch (queryError) {
      if (currentRequestID !== saleOrdersQueryRequestID) return;
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
      }
    }
  }

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

  function saleOrderHasSelectedProduct(saleOrder: ISaleOrder, selectedProductID: number): boolean {
    if (!selectedProductID) { return true; }
    return (saleOrder.DetailProductsIDs || []).some((productID) => Number(productID || 0) === selectedProductID);
  }

  function getProductPresentationName(productID: number, presentationID: number): string {
    if (!presentationID) { return ''; }

    const productRecord = productosService.recordsMap.get(productID);
    const presentationRecord = productRecord?.Presentaciones?.find((presentationOption: IProductoPresentacion) => presentationOption.id === presentationID);
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
    resetSaleOrderActionForms(saleOrder);
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
      const updatedSaleOrder = await postSaleOrderUpdate(updatePayload) as ISaleOrder;
      applyUpdatedSaleOrderLocally(updatedSaleOrder);
      
      Notify.success(`Pedido actualizado (${actionLabel}).`);
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
      Notify.failure(`No se pudo procesar la ${actionLabel}.`);
    } finally {
      isPostingSaleOrderAction = false;
      saleOrderActionInProgress = null;
    }
  }

  onMount(() => {
    void querySaleOrders(selectedGroup);
  });

</script>

<Page title="Gestión de Pedidos">
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

      <div class="grid grid-cols-24 gap-10 ml-16 grow-1">
        <SearchSelect
          bind:saveOn={saleOrderFilterForm}
          save="clientID"
          css="col-span-24 md:col-span-6"
          label=""
          keyId="ID"
          keyName="DisplayName"
          options={clientOptions}
          placeholder="CLIENTE ::"
        />
        <SearchSelect
          bind:saveOn={saleOrderFilterForm}
          save="productID"
          css="col-span-24 md:col-span-9"
          label=""
          keyId="ID"
          keyName="Nombre" placeholder="PRODUCTO ::"
          options={productOptions}
        />
      </div>
    </div>
     <Layer type="content">
       {#if isQueryingSaleOrders}
         <div class="p-8 min-h-240 w-full fx-c rounded-md bg-gray-50">
           <LoadingBar label="Cargando pedidos..." />
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
