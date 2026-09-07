<script lang="ts">
  import { useUI } from '@genix/ui';
  const ui = useUI();
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
  import { security } from '$libs/ui-runtime.svelte';
  import { ConfirmWarn, Notify, formatN, formatTime } from '$libs/helpers';
  import { type Quantity, formatQuantity, quantityAmount, quantityDivisorOf, unpackQuantityLine } from '$core/quantity';
  import { CajasService } from '$routes/finance/cash-banks/cajas.svelte';
  import {
      ClientProviderService,
      type IClientProvider
  } from '$services/crm/client-provider.svelte';
  import {
      ProductsService,
      type IProduct,
      type IProductPresentation,
  } from '$services/production/products.svelte';
  import { WarehousesService } from '$routes/business/branches-warehouses/branches-warehouses.svelte';
  import { onMount, untrack } from 'svelte';
  import SaleOrdersTable from '../SaleOrdersTable.svelte';
  import {
      ANNUL_SALE_SUB_ACCESS_ID,
      SALES_MANAGEMENT_ACCESS_ID,
      SaleOrderGroup,
      SaleOrdersService,
      postSaleOrderAnnul,
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
    // Rendered from the packed line, so it reads "1 + 4 unidad" when a sub-unit was sold.
    quantityLabel: string;
    quantity: Quantity;
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
  let saleOrderActionInProgress = $state<'pago' | 'entrega' | 'anulacion' | null>(null);
  let saleOrderPaymentForm = $state({ LastPaymentCajaID: 0 });
  let saleOrderDeliveryForm = $state({ WarehouseID: 0 });
  // The annul form replaces the payment/delivery panels while it is open.
  let isAnnulFormOpen = $state(false);
  let saleOrderAnnulForm = $state({ RefundCashBankID: 0, Reason: '' });
  let saleOrderFilterForm = $state<ISaleOrderFilterForm>({ clientID: 0, productID: 0 });
  let clientOptions = $state<ISaleOrderClientOption[]>([]);
  let productOptions = $state<IProduct[]>([]);
  let saleOrdersQueryRequestID = 0;

  const cajasService = new CajasService();
  const almacenesService = new WarehousesService();
  const productosService = new ProductsService();
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
    [SaleOrderGroup.FINALIZADO, 'Completed|Finalizadas'],
    [SaleOrderGroup.ANULADO, 'Annulled|Anuladas']
  ];

  function getSaleOrderStatusName(saleOrder: ISaleOrder): string {
    switch(saleOrder.ss) {
      case 0: return tr('Annulled|Anulada');
      case 1: return tr('Generated|Generado');
      case 2: return tr('Paid|Pagado');
      case 3: return tr('Delivered|Entregado');
      case 4: return tr('Completed|Finalizado');
      default: return tr('Unknown|Desconocido');
    }
  }

  function isSaleOrderAnnulled(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 0;
  }

  // The handler enforces this too; hiding the button only keeps operators from being offered
  // an action that would be refused.
  const canAnnulSaleOrders = security.checkSubAcceso(SALES_MANAGEMENT_ACCESS_ID, ANNUL_SALE_SUB_ACCESS_ID);

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
    const detailSubPrices = saleOrder.DetailSubPrices || [];
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
      const quantity = unpackQuantityLine(detailQuantities[detailPosition] || 0);
      const subDivisor = quantityDivisorOf(productosService.recordsMap.get(productID)?.SbuQuantity);
      const subUnitName = productosService.recordsMap.get(productID)?.SbuUnit;
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
        quantityLabel: formatQuantity(quantity, subDivisor, subUnitName),
        unitPrice,
        subtotalAmount: quantityAmount(
          quantity, unitPrice, detailSubPrices[detailPosition] || 0),
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

  function getActionInProgressLabel(actionInProgress: 'pago' | 'entrega' | 'anulacion' | null): string {
    // Keep message explicit so operators know the exact transition being processed.
    if (actionInProgress === 'pago') { return tr('Processing Payment...|Realizando Pago...'); }
    if (actionInProgress === 'entrega') { return tr('Processing Delivery...|Realizando Entrega...'); }
    if (actionInProgress === 'anulacion') { return tr('Annulling...|Anulando...'); }
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
      mobile: { order: 1, css: 'col-span-24' },
    },
    {
      id: 'sku',
      header: 'SKU',
      width: 'minmax(120px, 0.7fr)',
      align: 'center',
      getValue: (detailLineRecord) => detailLineRecord.sku || '-',
      mobile: { order: 2, css: 'col-span-24', labelLeft: 'SKU:', if: (detailLineRecord) => !!detailLineRecord.sku },
    },
    {
      id: 'quantity',
      header: 'Qty.|Cant.',
      width: '90px',
      align: 'right',
      getValue: (detailLineRecord) => detailLineRecord.quantityLabel,
      mobile: { order: 3, css: 'col-span-12', labelLeft: 'Cant:' },
    },
    {
      id: 'unitPrice',
      header: 'Price|Precio',
      width: '120px',
      align: 'right',
      getValue: (detailLineRecord) => `S/ ${formatN(detailLineRecord.unitPrice / 100, 2)}`,
      mobile: { order: 4, css: 'col-span-12', labelLeft: 'Precio:', contentCss: 'ff-mono' },
    },
    {
      id: 'subtotalAmount',
      header: 'Subtotal',
      width: '130px',
      align: 'right',
      getValue: (detailLineRecord) => `S/ ${formatN(detailLineRecord.subtotalAmount / 100, 2)}`,
      mobile: { order: 5, css: 'col-span-12', labelLeft: 'Sub:', contentCss: 'ff-mono ff-bold' },
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
    isAnnulFormOpen = false;
    saleOrderDetailsView = 1;
    ui.openSideLayer(10);
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

  // A paid order needs somewhere to take the refund from; an unpaid one has nothing to give back
  // and the selector is not shown at all.
  function saleOrderWasPaid(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 2 || saleOrder.ss === 4;
  }

  function openAnnulForm() {
    if (!selectedSaleOrder) { return; }
    // Default to the cash bank that collected — the common case — while leaving it changeable.
    saleOrderAnnulForm.RefundCashBankID = selectedSaleOrder.LastPaymentCajaID || 0;
    saleOrderAnnulForm.Reason = '';
    isAnnulFormOpen = true;
  }

  function onClickAnular() {
    if (!selectedSaleOrder || isPostingSaleOrderAction) { return; }

    const annulReason = saleOrderAnnulForm.Reason.trim();
    if (!annulReason) {
      Notify.failure(tr('A reason is required to annul.|Se requiere un motivo para anular.'));
      return;
    }
    if (saleOrderWasPaid(selectedSaleOrder) && !saleOrderAnnulForm.RefundCashBankID) {
      Notify.failure(tr('Select the cash register the refund comes from.|Seleccione la caja de la cual se devolverá el dinero.'));
      return;
    }

    const saleOrderToAnnul = selectedSaleOrder;
    ConfirmWarn(
      tr('Annul Order|Anular Pedido'),
      tr(`Annul order #${saleOrderToAnnul.ID}? The payment is returned, delivered stock re-enters the warehouse, and the sale leaves the day totals.`
        + `|¿Anular el pedido #${saleOrderToAnnul.ID}? Se devuelve el pago, el stock entregado reingresa al almacén y la venta sale de los totales del día.`),
      'SI', 'NO',
      () => { void processSaleOrderAnnul(saleOrderToAnnul, annulReason); },
    );
  }

  async function processSaleOrderAnnul(saleOrderToAnnul: ISaleOrder, annulReason: string) {
    saleOrderActionInProgress = 'anulacion';
    isPostingSaleOrderAction = true;

    try {
      const annulledSaleOrder = await postSaleOrderAnnul({
        ID: saleOrderToAnnul.ID,
        RefundCashBankID: saleOrderAnnulForm.RefundCashBankID,
        Reason: annulReason,
      }) as ISaleOrder;

      applyUpdatedSaleOrderLocally(annulledSaleOrder);
      isAnnulFormOpen = false;
      Notify.success(tr('Order annulled.|Pedido anulado.'));
    } catch (error) {
      console.error('[sale_orders_status] annul error', { saleOrderID: saleOrderToAnnul.ID, error });
      Notify.failure(String(error) || tr('Could not annul the order.|No se pudo anular el pedido.'));
    } finally {
      isPostingSaleOrderAction = false;
      saleOrderActionInProgress = null;
    }
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
      isAnnulFormOpen = false;
    }}
  >
    {#snippet titleSide()}
      {#if selectedSaleOrder && canAnnulSaleOrders && !isSaleOrderAnnulled(selectedSaleOrder)}
     	<div class="flex items-center">
	       <button
	         class="w-30 h-30 text-sm rounded-full bg-red-100 text-red-700 fx-c"
	         type="button"
	         title={tr("Annul order|Anular pedido")}
	         aria-label="Annul this sale order"
	         onclick={() => {
	           openAnnulForm();
	         }}
	       >
	         <i class="icon-[fa--trash]"></i>
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
          {:else if isSaleOrderAnnulled(selectedSaleOrder)}
            <div class="col-span-2 p-10 bg-red-50 w-full rounded-md text-13 leading-20 text-gray-700"
              aria-label="Annulment details: date, user and reason">
              <span class="ff-bold text-xs color-label mr-2"><T text="Annulled on:|Anulada el:" /></span> {formatActionTime(selectedSaleOrder.upd)}.<br>
              <span class="ff-bold text-xs color-label mr-2"><T text="User:|Usuario:" /></span>
              <RecordByIDText apiRoute="users-ids" recordID={selectedSaleOrder.UpdatedBy} placeholder="-" />.<br>
              <span class="ff-bold text-xs color-label mr-2"><T text="Reason:|Motivo:" /></span> {selectedSaleOrder.AnnulReason || '-'}
            </div>
          {:else if isAnnulFormOpen}
            <div class="col-span-2 p-10 bg-red-50 w-full rounded-md" aria-label="Annul order form with refund cash register and reason">
              {#if saleOrderWasPaid(selectedSaleOrder)}
                <SearchSelect
                  bind:saveOn={saleOrderAnnulForm}
                  save="RefundCashBankID"
                  css="mb-8"
                  options={(cajasService.Cajas || []).filter((cajaRecord) => (cajaRecord?.ss || 0) > 0)}
                  keyId="ID"
                  keyName="Name"
                  label="Cash Register for Refund|Caja para la Devolución"
                  placeholder=":: seleccione ::"
                />
              {/if}
              <label class="block mb-8">
                <span class="text-xs color-label ff-bold"><T text="Reason|Motivo" /></span>
                <textarea
                  bind:value={saleOrderAnnulForm.Reason}
                  class="w-full h-56 px-8 py-6 text-13 rounded-md border border-gray-300"
                  maxlength="200"
                  aria-label="Reason for annulling this sale order"
                  placeholder={tr('Why is this order annulled?|¿Por qué se anula este pedido?')}
                ></textarea>
              </label>
              <div class="flex items-center gap-8">
                <button class={`bx-red justify-center h-36 px-16 ${!saleOrderAnnulForm.Reason.trim() ? 'opacity-60' : ''}`}
                  disabled={!saleOrderAnnulForm.Reason.trim()}
                  aria-label="Confirm annulling this sale order"
                  onclick={() => { onClickAnular(); }}
                >
                  <i class="icon-[fa--trash]"></i>
                  <span><T text="Annul|Anular" /></span>
                </button>
                <button class="bn-white justify-center h-36 px-16"
                  aria-label="Cancel the annulment and go back"
                  onclick={() => { isAnnulFormOpen = false; }}
                >
                  <T text="Cancel|Cancelar" />
                </button>
              </div>
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
                  <i class="icon-[fa--check]"></i>
                  <span class="mr-12"><T text="Pay|Pagar" /></span>
                </button>
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label mr-2"><T text="Paid on:|Pagado el:" /></span> {formatActionTime(selectedSaleOrder.LastPaymentTime)}.<br>
                  <span class="ff-bold text-xs color-label mr-2"><T text="At:|En:" /></span> {getCajaName(selectedSaleOrder.LastPaymentCajaID)}.<br>
                  <span class="ff-bold text-xs color-label mr-2"><T text="User:|Usuario:" /></span>
                  <RecordByIDText apiRoute="users-ids" recordID={selectedSaleOrder.LastPaymentUser} placeholder="-" />.
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
                    <i class="icon-[fa--truck]"></i>
                    <span><T text="Deliver|Entregar" /></span>
                  </button>
                </div>
              {:else}
                <div class="text-13 leading-20 text-gray-700">
                  <span class="ff-bold text-xs color-label"><T text="Delivered on|Entregado el" /></span> {formatActionTime(selectedSaleOrder.DeliveryTime)}.<br>
                  <span class="ff-bold text-xs color-label"><T text="At|En" /></span> {getAlmacenName(selectedSaleOrder.WarehouseID)}.<br>
                  <RecordByIDText apiRoute="users-ids" recordID={selectedSaleOrder.DeliveryUser} placeholder="-" />.
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </Layer>
</Page>
