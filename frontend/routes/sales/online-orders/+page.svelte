<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import { onMount, untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
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
    DELIVERY_TYPE, ONLINE_ORDER_GROUP, ONLINE_ORDER_GROUP_TABS, SETTLE_ACTION, buildAnnulRequest, buildSettleRequest,
    canSettle, customerName, defaultDeliveryWarehouseID, defaultPaymentCashBankID, deliveryTypeLabel,
    deliveryWarehouseOptions, needsCulqiReconciliation, onlineOrderStatusLabel, paymentCashBankOptions,
    paymentMethodLabel, type IOnlineSaleOrder,
  } from './online-orders.ts'
  import { OnlineSaleOrdersService, postOnlineSaleOrderAnnul, postOnlineSaleOrderSettle } from './online-orders.svelte.ts'

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
    ordersService = nextOrdersService
    isLoadingOrders = false

    // Product names for the detail lines; the table does not need them.
    const productIDs = new Set<number>()
    for (const order of nextOrdersService.records) {
      for (const line of order.Lines || []) { if (line.ProductID > 0) productIDs.add(line.ProductID) }
    }
    try {
      await productsService.syncIDs([...productIDs])
    } catch (syncError) {
      console.error('[online-orders] failed to sync products by IDs', syncError)
    }
  }

  const formatAmount = (cents: number) => `S/ ${formatN((cents || 0) / 100, 2)}`

  // Home delivery names the district; pickup names the store.
  function deliveryPlaceName(order: IOnlineSaleOrder): string {
    if (order.DeliveryType === DELIVERY_TYPE.PICKUP) {
      return warehousesService.SedesMap.get(order.PickupSiteID)?.Name || `Sede #${order.PickupSiteID}`
    }
    const districtID = order.Shipping?.CityID || 0
    if (!districtID) return ''
    return cities.citiesMap.get(districtID)?.Name || `Distrito #${districtID}`
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
    { header: 'Delivery|Entrega', css: 'px-6', setCellCss: culqiRowCss,
      getValue: (order) => `${tr(deliveryTypeLabel(order.DeliveryType))} · ${deliveryPlaceName(order)}` },
    { header: 'Payment|Pago', headerCss: 'w-150', css: 'px-6', setCellCss: culqiRowCss,
      getValue: (order) => tr(paymentMethodLabel(order.PaymentMethod)) },
    { header: 'Total', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', setCellCss: culqiRowCss,
      getValue: (order) => formatAmount(order.TotalAmount) },
    { header: 'Status|Estado', headerCss: 'w-200', css: 'px-6', setCellCss: culqiRowCss,
      getValue: (order) => needsCulqiReconciliation(order)
        ? tr('Paid in Culqi, sale pending|Pagado en Culqi, venta pendiente')
        : tr(onlineOrderStatusLabel(order.ss)),
      render: (order) => needsCulqiReconciliation(order)
        ? `<span class="ff-bold text-amber-fg"><i class="icon-[fa--exclamation-triangle] mr-4"></i>${tr('Paid in Culqi, sale pending|Pagado en Culqi, venta pendiente')}</span>`
        : tr(onlineOrderStatusLabel(order.ss)) },
    { header: 'Sale|Venta', headerCss: 'w-110', css: 'px-6 text-center', setCellCss: culqiRowCss,
      getValue: (order) => order.SaleOrderID || '—' },
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
    settleForm.WarehouseID = defaultDeliveryWarehouseID(order, warehousesService.Almacenes || [])
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
      />
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
          <div class="col-span-8"><div class="text-fg-muted">{tr('Date|Fecha')}</div>{formatTime(order.Date, 'd-M-Y')}</div>
          <div class="col-span-8"><div class="text-fg-muted">{tr('Status|Estado')}</div>{tr(onlineOrderStatusLabel(order.ss))}</div>
          <div class="col-span-8"><div class="text-fg-muted">{tr('Payment|Pago')}</div>{tr(paymentMethodLabel(order.PaymentMethod))}</div>
          <div class="col-span-8"><div class="text-fg-muted">{tr('Shipping|Envío')}</div><span class="ff-mono">{formatAmount(order.ShippingAmount)}</span></div>
          <div class="col-span-8"><div class="text-fg-muted">Total</div><span class="ff-mono ff-bold">{formatAmount(order.TotalAmount)}</span></div>
          <div class="col-span-8"><div class="text-fg-muted">{tr('Debt|Deuda')}</div><span class="ff-mono">{formatAmount(order.DebtAmount)}</span></div>
          {#if order.CulqiChargeID}
            <div class="col-span-24"><span class="text-fg-muted mr-6">{tr('Culqi charge:|Cargo Culqi:')}</span><span class="ff-mono">{order.CulqiChargeID}</span></div>
          {/if}
        </div>

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
          <div class="p-10 rounded-md bg-blue-bg" aria-label="Linked sale: further actions happen in Sales Management">
            <span class="ff-bold">{tr(`Sale ${order.SaleOrderID}.|Venta ${order.SaleOrderID}.`)}</span>
            {tr(' Later payments, deliveries and annulments are done in Sales Management.| Los pagos, entregas y anulaciones posteriores se hacen en Gestión Ventas.')}
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
          <div class="grid grid-cols-24 gap-10 p-10 rounded-md bg-surface-muted" aria-label="Settle actions: payment, delivery or annulment">
            <SearchSelect saveOn={settleForm} save="CashBankID" css="col-span-24 md:col-span-12"
              options={paymentCashBanks} keyId="ID" keyName="Name"
              label={needsCulqiReconciliation(order) ? 'Account (choose Culqi)|Cuenta (elija la de Culqi)' : 'Cash/bank for payment|Caja/banco para pago'}
              placeholder=":: seleccione ::" />
            <SearchSelect saveOn={settleForm} save="WarehouseID" css="col-span-24 md:col-span-12"
              options={deliveryWarehouses} keyId="ID" keyName="Name"
              label="Warehouse for delivery|Almacén para entrega" placeholder=":: seleccione ::" />
            <div class="col-span-24 flex flex-wrap items-center gap-8">
              <Button color="green" icon="icon-[fa--check]" name="Register payment|Registrar pago"
                label="Creates the sale as paid into the selected account." onClick={() => settleOrder([SETTLE_ACTION.PAYMENT])} />
              <Button color="blue" icon="icon-[fa--truck]" name="Register delivery|Registrar entrega"
                label="Creates the sale as delivered from the selected warehouse." onClick={() => settleOrder([SETTLE_ACTION.DELIVERY])} />
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
