<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import { onMount, untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import LabelCell from '#components/form/LabelCell.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import LoadingBar from '#components/misc/LoadingBar.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, quantityAmount, quantityDivisorOf, unpackQuantityLine } from '#core/quantity.ts'
  import Page from '#domain/Page.svelte'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import type { ISaleOrderLine } from '#routes/sales/sale_order_create/sale_order.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import { useCitiesAPI } from '#services/services/cities.svelte.ts'
  import {
    DELIVERY_TYPE, FULFILLMENT_STEP, ONLINE_ORDER_GROUP, ONLINE_ORDER_GROUP_TABS, ONLINE_ORDER_STATUS, SETTLE_ACTION,
    buildAnnulRequest, buildSettleRequest, canMoveToFulfillmentStep, canSettle, customerName, defaultDeliveryWarehouseID, defaultPaymentCashBankID, deliveryTypeLabel,
    deliveryWarehouseOptions, isOnlineOrderDelivered, isOnlineOrderPaid, needsCulqiReconciliation, onlineOrderStatusLabel,
    paymentCashBankOptions, paymentMethodLabel, paymentStatusLabel, topOrderProducts, type IOnlineSaleOrder,
  } from './online-orders.ts'
  import {
    OnlineSaleOrdersService, postLinkedSaleOrderDelivery, postOnlineSaleOrderAnnul, postOnlineSaleOrderFulfillment,
    postOnlineSaleOrderSettle,
  } from './online-orders.svelte.ts'
  import OnlineOrderStatusSteps from './OnlineOrderStatusSteps.svelte'

  const ui = useUI()
  const DETAIL_LAYER_ID = 1

  const cashBanksService = new CajasService()
  const warehousesService = new WarehousesService()
  const productsService = new ProductsService()
  const cities = useCitiesAPI()

  let selectedGroup = $state<number>(ONLINE_ORDER_GROUP.PENDING)
  let ordersService = $state<OnlineSaleOrdersService>()
  let isLoadingOrders = $state(false)
  let selectedOrder = $state<IOnlineSaleOrder | null>(null)
  let settleForm = $state({ CashBankID: 0, WarehouseID: 0 })
  let annulForm = $state({ Reason: '' })
  let isAnnulFormOpen = $state(false)
  let isPostingAction = $state(false)
  let ordersLoadRequestID = 0

  const paymentCashBanks = $derived(paymentCashBankOptions(cashBanksService.Cajas || []))
  const deliveryWarehouses = $derived(deliveryWarehouseOptions(warehousesService.Almacenes || []))

  // Each tab is its own delta-cached route; a late answer of a tab left behind is dropped.
  async function loadOrders(statusGroup: number) {
    const requestID = ++ordersLoadRequestID
    isLoadingOrders = true
    const nextOrdersService = new OnlineSaleOrdersService(statusGroup)
    await nextOrdersService.fetchOnline()
    if (requestID !== ordersLoadRequestID) return

    // Product names for the Top Products column and the detail lines: synced before the rows show,
    // since the table does not re-render when the names arrive later.
    const productIDs = new Set<number>()
    for (const order of nextOrdersService.records) {
      for (const line of order.Lines || []) { if (line.ProductID > 0) productIDs.add(line.ProductID) }
    }
    try {
      await productsService.syncIDs([...productIDs])
    } catch (syncError) {
      console.error('[online-orders] failed to sync products by IDs', syncError)
    }
    if (requestID !== ordersLoadRequestID) return
    ordersService = nextOrdersService
    isLoadingOrders = false
  }

  const formatAmount = (cents: number) => `S/ ${formatN((cents || 0) / 100, 2)}`

  // Home delivery names the province and district; pickup names the store.
  function deliveryPlaceName(order: IOnlineSaleOrder): string {
    if (order.DeliveryType === DELIVERY_TYPE.PICKUP) {
      return warehousesService.SedesMap.get(order.PickupSiteID)?.Name || `Sede #${order.PickupSiteID}`
    }
    const districtID = order.Shipping?.CityID || 0
    if (!districtID) return ''
    const district = cities.citiesMap.get(districtID)
    if (!district) return `Distrito #${districtID}`
    return [cities.citiesMap.get(district.ParentID)?.Name, district.Name].filter(Boolean).join(', ')
  }

  function lineProductName(line: ISaleOrderLine): string {
    const product = productsService.recordsMap.get(line.ProductID)
    const productName = product?.Name || `Producto #${line.ProductID}`
    if (!line.PresentationID) return productName
    const presentation = product?.Presentations?.find((option) => option.id === line.PresentationID)
    return `${productName} (${presentation?.nm || `Presentación ${line.PresentationID}`})`
  }

  // Zero-valued line fields are omitted on the wire, so every read defaults them.
  function lineQuantityLabel(line: ISaleOrderLine): string {
    const product = productsService.recordsMap.get(line.ProductID)
    return formatQuantity(unpackQuantityLine(line.Quantity || 0), quantityDivisorOf(line.SubDivisor), product?.SbuUnit)
  }

  const lineSubtotal = (line: ISaleOrderLine) =>
    quantityAmount(unpackQuantityLine(line.Quantity || 0), line.Price || 0, line.SubPrice || 0)

  // The whole row is tinted when Culqi charged the card but the sale was never registered.
  const culqiRowCss = (order: IOnlineSaleOrder) => needsCulqiReconciliation(order) ? 'bg-amber-bg' : undefined

  const orderColumns: ITableColumn<IOnlineSaleOrder>[] = [
    { header: 'No.|N°', headerCss: 'w-70', css: 'text-center ff-bold', setCellCss: culqiRowCss,
      getValue: (order) => order.ID },
    { header: 'Date|Fecha', headerCss: 'w-100', css: 'px-6', setCellCss: culqiRowCss,
      getValue: (order) => String(formatTime(order.Date, 'd-M-Y')) },
    { header: 'Customer|Cliente', css: 'px-6', setCellCss: culqiRowCss, getValue: (order) => customerName(order) },
    // Two lines: the delivery type, then where it goes (province and district, or the pickup store).
    { id: 'destination', header: 'Destination|Destino', headerCss: 'w-220', css: 'px-6', setCellCss: culqiRowCss,
      getValue: (order) => `${tr(deliveryTypeLabel(order.DeliveryType))} ${deliveryPlaceName(order)}` },
    { header: 'Total', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', setCellCss: culqiRowCss,
      getValue: (order) => formatAmount(order.TotalAmount) },
    { id: 'payment', header: 'Payment|Pago', headerCss: 'w-150', css: 'px-6 text-center', setCellCss: culqiRowCss,
      getValue: (order) => tr(paymentMethodLabel(order.PaymentMethod)) },
    { id: 'delivery', header: 'Delivery|Entrega', headerCss: 'w-110', css: 'px-6 text-center', setCellCss: culqiRowCss,
      getValue: (order) => tr(isOnlineOrderDelivered(order) ? 'Delivered|Entregado' : 'Pending|Pendiente') },
    { id: 'top-products', header: 'Top Products|Top Productos', headerCss: 'w-[40%]', css: 'px-0 py-0', setCellCss: culqiRowCss,
      getValue: (order) => topOrderProducts(order, 3)
        .map((topProduct) => `${topProduct.Quantity}x ${productsService.recordsMap.get(topProduct.ProductID)?.Name || ''}`).join(', ') },
  ]

  const lineColumns: ITableColumn<ISaleOrderLine>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (line) => lineProductName(line) },
    { header: 'Qty.|Cant.', headerCss: 'w-90', css: 'px-6', align: 'right', getValue: (line) => lineQuantityLabel(line) },
    { header: 'Price|Precio', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', getValue: (line) => formatAmount(line.Price) },
    { header: 'Subtotal', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right', getValue: (line) => formatAmount(lineSubtotal(line)) },
  ]

  function openOrderDetail(order: IOnlineSaleOrder) {
    selectedOrder = order
    settleForm.CashBankID = defaultPaymentCashBankID(order, cashBanksService.Cajas || [])
    // The warehouse is kept across orders, so the operator picks it once per session.
    if (!settleForm.WarehouseID) settleForm.WarehouseID = defaultDeliveryWarehouseID(order, warehousesService.Almacenes || [])
    annulForm.Reason = ''
    isAnnulFormOpen = false
    ui.openSideLayer(DETAIL_LAYER_ID)
  }

  // The cash banks and warehouses may arrive after the detail was opened: fill the defaults then.
  $effect(() => {
    paymentCashBanks; deliveryWarehouses
    untrack(() => {
      if (!selectedOrder) return
      if (!settleForm.CashBankID) settleForm.CashBankID = defaultPaymentCashBankID(selectedOrder, cashBanksService.Cajas || [])
      if (!settleForm.WarehouseID) settleForm.WarehouseID = defaultDeliveryWarehouseID(selectedOrder, warehousesService.Almacenes || [])
    })
  })

  async function settleOrder(actionsIncluded: number[]) {
    if (!selectedOrder || isPostingAction) return
    const { request, error } = buildSettleRequest(selectedOrder, actionsIncluded, settleForm.CashBankID, settleForm.WarehouseID)
    if (!request) { notifyFailure(tr(error || '')); return }

    isPostingAction = true
    try {
      const settleResult = await postOnlineSaleOrderSettle(request)
      selectedOrder = settleResult.Order
      notifySuccess(tr(`Sale ${settleResult.Order.SaleOrderID} created.|Venta ${settleResult.Order.SaleOrderID} creada.`))
      // The order left the Pendientes tab: re-sync it.
      await ordersService?.fetchOnline()
    } catch (settleError) {
      console.error('[online-orders] settle error', { request, settleError })
      notifyFailure(String(settleError) || tr('Could not register the order.|No se pudo registrar el pedido.'))
    } finally {
      isPostingAction = false
    }
  }

  // A status step. Preparation and on-the-way only relabel the order. Delivered moves the stock: it
  // creates the sale when there is none yet, or delivers the existing one.
  async function moveOrderToStep(step: number) {
    if (!selectedOrder || isPostingAction || !canMoveToFulfillmentStep(selectedOrder, step)) return
    const order = selectedOrder

    if (step === FULFILLMENT_STEP.DELIVERED) {
      if (!(settleForm.WarehouseID > 0)) {
        notifyFailure(tr('Select the warehouse the goods leave from.|Seleccione el almacén del que salen los productos.'))
        return
      }
      const warehouseName = deliveryWarehouses.find((warehouse) => warehouse.ID === settleForm.WarehouseID)?.Name || ''
      const confirmed = await confirmWarn({
        title: tr('Register Delivery|Registrar Entrega'),
        message: tr(`Deliver order No. ${order.ID}? The products leave warehouse "${warehouseName}".`
          + `|¿Entregar el pedido N° ${order.ID}? Los productos salen del almacén "${warehouseName}".`),
      })
      if (!confirmed) return
      if (!order.SaleOrderID) { await settleOrder([SETTLE_ACTION.DELIVERY]); return }
    }

    isPostingAction = true
    try {
      if (step === FULFILLMENT_STEP.DELIVERED) {
        const deliveredSale = await postLinkedSaleOrderDelivery(order.SaleOrderID, settleForm.WarehouseID)
        // The backend copied the sale's status onto the online order in the same transaction.
        selectedOrder = { ...order, ss: deliveredSale.ss, DebtAmount: deliveredSale.DebtAmount }
        notifySuccess(tr('Order delivered.|Pedido entregado.'))
      } else {
        selectedOrder = await postOnlineSaleOrderFulfillment({ ID: order.ID, FulfillmentStatus: step })
        notifySuccess(tr('Status updated.|Estado actualizado.'))
      }
      await ordersService?.fetchOnline()
    } catch (stepError) {
      console.error('[online-orders] status step error', { orderID: order.ID, step, stepError })
      notifyFailure(String(stepError) || tr('Could not update the status.|No se pudo actualizar el estado.'))
    } finally {
      isPostingAction = false
    }
  }

  async function annulOrder() {
    if (!selectedOrder || isPostingAction) return
    const { request, error } = buildAnnulRequest(selectedOrder, annulForm.Reason)
    if (!request) { notifyFailure(tr(error || '')); return }

    const confirmed = await confirmWarn({
      title: tr('Annul Order|Anular Pedido'),
      message: tr(`Annul web order No. ${request.ID}? No sale exists yet, so nothing else is undone.`
        + `|¿Anular el pedido web N° ${request.ID}? Aún no existe una venta, así que no se revierte nada más.`),
    })
    if (!confirmed) return

    isPostingAction = true
    try {
      selectedOrder = await postOnlineSaleOrderAnnul(request)
      isAnnulFormOpen = false
      notifySuccess(tr('Order annulled.|Pedido anulado.'))
      await ordersService?.fetchOnline()
    } catch (annulError) {
      console.error('[online-orders] annul error', { request, annulError })
      notifyFailure(String(annulError) || tr('Could not annul the order.|No se pudo anular el pedido.'))
    } finally {
      isPostingAction = false
    }
  }

  onMount(() => { void loadOrders(selectedGroup) })
</script>

<Page title="Online Orders|Pedidos Web">
  <div class="h-46 flex items-center mb-10">
    <OptionsStrip options={ONLINE_ORDER_GROUP_TABS} selected={selectedGroup} useMobileGrid
      onSelect={(selectedOption) => {
        const nextGroup = selectedOption[0] as number
        if (nextGroup === selectedGroup) return
        selectedGroup = nextGroup
        void loadOrders(nextGroup)
      }}
    />
  </div>

  <Layer type="content">
    {#if isLoadingOrders}
      <div class="p-8 min-h-240 w-full fx-c rounded-md bg-surface-soft">
        <LoadingBar label={tr('Loading orders...|Cargando pedidos...')} />
      </div>
    {:else}
      <VTable columns={orderColumns} data={ordersService?.records || []}
        maxHeight="calc(100vh - var(--header-height) - 84px)"
        emptyMessage="No orders in this group.|No hay pedidos en este grupo."
        selected={selectedOrder?.ID || 0}
        isSelected={(order, selectedID) => order.ID === selectedID}
        onRowClick={(order) => openOrderDetail(order)}
      >
        {#snippet cellRenderer(order: IOnlineSaleOrder, column: ITableColumn<IOnlineSaleOrder>)}
          {#if column.id === 'destination'}
            <div class="leading-[1.15]">
              <div class="text-[15px] text-fg-muted">{tr(deliveryTypeLabel(order.DeliveryType))}</div>
              <div class="truncate">{deliveryPlaceName(order) || '-'}</div>
            </div>
          {:else if column.id === 'payment'}
            <!-- Status on top (a check once paid), the method the shopper chose below. -->
            <div class="leading-[1.15]">
              {#if isOnlineOrderPaid(order)}
                <i class="icon-[fa--check] text-green-fg" aria-label="Paid"></i>
              {:else if needsCulqiReconciliation(order)}
                <span class="ff-bold text-amber-fg" title={tr('Paid in Culqi, sale pending|Pagado en Culqi, venta pendiente')}>
                  <i class="icon-[fa--exclamation-triangle] mr-4"></i>Culqi
                </span>
              {:else}
                <div class="text-fg-muted">{tr('Pending|Pendiente')}</div>
              {/if}
              <div class="truncate text-sm">({tr(paymentMethodLabel(order.PaymentMethod))})</div>
            </div>
          {:else if column.id === 'delivery'}
            {#if isOnlineOrderDelivered(order)}
              <span class="text-green-fg"><i class="icon-[fa--check] mr-4"></i>{tr('Delivered|Entregado')}</span>
            {:else}
              <span class="text-fg-muted">{tr('Pending|Pendiente')}</span>
            {/if}
          {:else if column.id === 'top-products'}
            {@const topProducts = topOrderProducts(order, 3)}
            <div class="flex w-full min-w-0 h-full">
              {#each topProducts as topProduct, productIndex (topProduct.ProductID)}
                <!-- Quantity beside a clamped name, so the row keeps a stable height. -->
                <div class={`min-w-0 flex items-center basis-1/3 px-10 ${productIndex < topProducts.length - 1 ? 'border-r border-line' : ''}`}>
                  <div class="mr-4 text-sm ff-mono shrink-0">{topProduct.Quantity}x</div>
                  <div class="line-clamp-2 text-sm leading-[1.15]">
                    {productsService.recordsMap.get(topProduct.ProductID)?.Name || `Producto #${topProduct.ProductID}`}
                  </div>
                </div>
              {:else}
                <div class="px-8 text-fg-subtle">-</div>
              {/each}
            </div>
          {/if}
        {/snippet}
      </VTable>
    {/if}
  </Layer>

  <Layer type="side" id={DETAIL_LAYER_ID} sideLayerSize={760}
    title={selectedOrder ? tr(`Order No. ${selectedOrder.ID}|Pedido N° ${selectedOrder.ID}`) : ''}
    titleCss="h2 mb-6"
    onClose={() => { selectedOrder = null; isAnnulFormOpen = false }}
  >
    {#if selectedOrder}
      {@const order = selectedOrder}
      <div class="flex flex-col gap-12 mt-8" aria-label="Online order detail: contact, delivery, lines, amounts and actions">
        {#if needsCulqiReconciliation(order)}
          <div class="p-10 rounded-md bg-amber-bg text-amber-fg">
            <i class="icon-[fa--exclamation-triangle] mr-6"></i>
            <span class="ff-bold">{tr('Paid in Culqi, sale pending.|Pagado en Culqi, venta pendiente.')}</span>
            {tr(' The card was charged but the sale was not registered. Use Register payment with the Culqi account.| Se cobró la tarjeta pero la venta no se registró. Use Registrar pago con la cuenta de Culqi.')}
          </div>
        {/if}

        <div class="grid grid-cols-24 gap-8" aria-label="Order summary: date, status, payment method and amounts">
          <LabelCell css="col-span-8" valueCss="" label="Date|Fecha" value={formatTime(order.Date, 'd-M-Y') as string} />
          <LabelCell css="col-span-8" valueCss="" label="Status|Estado" value={tr(onlineOrderStatusLabel(order.ss))} />
          <LabelCell css="col-span-8" valueCss="" label="Payment|Pago" value={tr(paymentStatusLabel(order))} />
          <LabelCell css="col-span-8" valueCss="ff-mono" label="Shipping|Envío" value={formatAmount(order.ShippingAmount)} />
          <LabelCell css="col-span-8" valueCss="ff-mono ff-bold" label="Total|Total" value={formatAmount(order.TotalAmount)} />
          <LabelCell css="col-span-8" valueCss="ff-mono" label="Culqi charge|Cargo Culqi" value={order.CulqiChargeID || '-'} />
        </div>

        {#if order.ss !== ONLINE_ORDER_STATUS.ANNULLED}
          <OnlineOrderStatusSteps {order} onSelectStep={moveOrderToStep} />
          <!-- Always at hand: Entregado and the settle actions take the goods out of this warehouse. -->
          <SearchSelect saveOn={settleForm} save="WarehouseID" css="w-full md:w-1/2"
            options={deliveryWarehouses} keyId="ID" keyName="Name"
            label="Warehouse for delivery|Almacén para entrega" placeholder=":: seleccione ::" />
        {/if}

        <div class="grid grid-cols-24 gap-8 p-10 rounded-md bg-surface-muted" aria-label="Customer contact and delivery">
          <div class="col-span-24 md:col-span-12">
            <div class="text-fg-muted">{tr('Customer|Cliente')}</div>
            <div class="ff-bold">{customerName(order)}</div>
            <div>{order.Shipping?.Email || '-'}</div>
            <div>{tr('Phone|Teléfono')}: {order.Shipping?.Phone || '-'}</div>
            <div>DNI: {order.Shipping?.DocumentNumber || '-'}</div>
          </div>
          <div class="col-span-24 md:col-span-12">
            <div class="text-fg-muted">{tr(deliveryTypeLabel(order.DeliveryType))}</div>
            {#if order.DeliveryType === DELIVERY_TYPE.PICKUP}
              <div class="ff-bold">{deliveryPlaceName(order)}</div>
            {:else}
              <div>{order.Shipping?.Address || '-'}</div>
              {#if order.Shipping?.Reference}<div class="text-fg-muted">{order.Shipping.Reference}</div>{/if}
              <div class="ff-bold">{deliveryPlaceName(order)}</div>
            {/if}
          </div>
        </div>

        <VTable columns={lineColumns} data={order.Lines || []} disableVirtualizer={true} estimateSize={40}
          emptyMessage="No products in this order.|No hay productos en el pedido." />

        {#if isPostingAction}
          <div class="p-12 min-h-64 w-full rounded-md fx-c bg-surface-muted">
            <LoadingBar label={tr('Processing...|Procesando...')} />
          </div>
        {:else if order.SaleOrderID > 0}
          <div class="p-10 rounded-md bg-blue-bg" aria-label="Linked sale: payments and annulments happen in Sales Management">
            <span class="ff-bold">{tr(`Sale ${order.SaleOrderID}.|Venta ${order.SaleOrderID}.`)}</span>
            {tr(' Later payments and annulments are done in Sales Management.| Los pagos y anulaciones posteriores se hacen en Gestión Ventas.')}
          </div>
        {:else if order.ss === 0}
          <div class="p-10 rounded-md bg-red-bg" aria-label="Annulment reason">
            <span class="ff-bold mr-6">{tr('Annul reason:|Motivo de anulación:')}</span>{order.AnnulReason || '-'}
          </div>
        {:else if canSettle(order) && isAnnulFormOpen}
          <div class="p-10 rounded-md bg-red-bg" aria-label="Annul order form with reason">
            <Input saveOn={annulForm} save="Reason" label="Reason|Motivo" useTextArea rows={2} css="mb-8"
              placeholder="Why is this order annulled?|¿Por qué se anula este pedido?" />
            <div class="flex items-center gap-8">
              <Button color="red" icon="icon-[fa--trash]" name="Annul|Anular" label="Confirms annulling this online order."
                onClick={annulOrder} />
              <Button name="Cancel|Cancelar" label="Closes the annul form."
                onClick={() => { isAnnulFormOpen = false }} />
            </div>
          </div>
        {:else if canSettle(order)}
          <div class="grid grid-cols-24 gap-10 p-10 rounded-md bg-surface-muted" aria-label="Settle actions: payment, payment and delivery, or annulment">
            <SearchSelect saveOn={settleForm} save="CashBankID" css="col-span-24 md:col-span-12"
              options={paymentCashBanks} keyId="ID" keyName="Name"
              label={needsCulqiReconciliation(order) ? 'Account (choose Culqi)|Cuenta (elija la de Culqi)' : 'Cash/bank for payment|Caja/banco para pago'}
              placeholder=":: seleccione ::" />
            <div class="col-span-24 flex flex-wrap items-center gap-8">
              <Button color="green" icon="icon-[fa--check]" name="Register payment|Registrar pago"
                label="Creates the sale as paid into the selected account." onClick={() => settleOrder([SETTLE_ACTION.PAYMENT])} />
              <Button color="purple" icon="icon-[fa--check-square-o]" name="Payment and delivery|Pago y entrega"
                label="Creates the sale paid and delivered." onClick={() => settleOrder([SETTLE_ACTION.PAYMENT, SETTLE_ACTION.DELIVERY])} />
              <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Annul|Anular" css="ml-auto"
                label="Opens the form to annul this online order." onClick={() => { annulForm.Reason = ''; isAnnulFormOpen = true }} />
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </Layer>
</Page>
