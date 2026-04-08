<script lang="ts">
  import DateInput from '$components/DateInput.svelte';
  import SearchSelect from '$components/SearchSelect.svelte';
  import VTable from '$components/vTable/VTable.svelte';
  import type { ITableColumn } from '$components/vTable/types';
  import Page from '$domain/Page.svelte';
  import { Loading, Notify, formatN, formatTime, throttle } from '$libs/helpers';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte';
  import { querySaleOrderReport, saleOrderStatusOptions, type ISaleOrder } from './sale_order_report.svelte';

  const productosService = new ProductosService();
  const clientesService = new ClientProviderService(ClientProviderType.CLIENT);

  const getCurrentUnixDay = (): number => Math.floor(Date.now() / (1000 * 60 * 60 * 24));

  const fechaFin = getCurrentUnixDay();
  const fechaInicio = fechaFin - 7;

  let reportForm = $state({
    fechaInicio,
    fechaFin,
    status: 0,
    productID: 0,
    clientID: 0,
  });
  let saleOrders = $state([] as ISaleOrder[]);
  let filterText = $state('');

  const clientOptions = $derived.by(() => {
    // Match by customer name and registry number in the same selector.
    return clientesService.records.map((clientRecord) => ({
      ...clientRecord,
      DisplayName: clientRecord.RegistryNumber
        ? `${clientRecord.Name} ${clientRecord.RegistryNumber}`
        : clientRecord.Name,
    }));
  });

  function isSaleOrderPaid(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 2 || saleOrder.ss === 4;
  }

  function isSaleOrderDelivered(saleOrder: ISaleOrder): boolean {
    return saleOrder.ss === 3 || saleOrder.ss === 4;
  }

  function renderSaleOrderStateSquares(saleOrder: ISaleOrder): string {
    const paidBadgeCss = isSaleOrderPaid(saleOrder)
      ? 'bg-green-100 text-green-800 border border-green-300'
      : 'bg-red-100 text-red-800 border border-red-300';
    const deliveredBadgeCss = isSaleOrderDelivered(saleOrder)
      ? 'bg-green-100 text-green-800 border border-green-300'
      : 'bg-red-100 text-red-800 border border-red-300';

    return `<div class="flex items-center justify-center gap-4">
      <span class="inline-flex h-22 min-w-22 px-4 rounded-[3px] text-10 ff-bold items-center justify-center ${paidBadgeCss}">PAG</span>
      <span class="inline-flex h-22 min-w-22 px-4 rounded-[3px] text-10 ff-bold items-center justify-center ${deliveredBadgeCss}">ENT</span>
    </div>`;
  }

  function renderTopProductsSummary(saleOrder: ISaleOrder): string {
    const detailCount = Math.min(
      saleOrder.DetailProductsIDs?.length || 0,
      saleOrder.DetailPrices?.length || 0,
      saleOrder.DetailQuantities?.length || 0,
    );
    if (detailCount <= 0) {
      return '-';
    }

    const productAmountByID = new Map<number, number>();
    for (let detailIndex = 0; detailIndex < detailCount; detailIndex += 1) {
      const productID = saleOrder.DetailProductsIDs[detailIndex] || 0;
      const linePrice = saleOrder.DetailPrices[detailIndex] || 0;
      const lineQuantity = saleOrder.DetailQuantities[detailIndex] || 0;
      const lineAmount = linePrice * lineQuantity;
      if (!productID || lineAmount <= 0) { continue; }

      productAmountByID.set(productID, (productAmountByID.get(productID) || 0) + lineAmount);
    }

    return Array.from(productAmountByID.entries())
      .sort((leftProduct, rightProduct) => {
        if (rightProduct[1] !== leftProduct[1]) {
          return rightProduct[1] - leftProduct[1];
        }
        return leftProduct[0] - rightProduct[0];
      })
      .slice(0, 3)
      .map(([productID, lineAmount]) => {
        const productName = productosService.recordsMap.get(productID)?.Nombre || `Producto #${productID}`;
        return `${productName} (${formatN(lineAmount / 100, 2)})`;
      })
      .join(', ');
  }

  async function consultarReporteVentas() {
    console.debug('[reporte_ventas] querying report with filters', $state.snapshot(reportForm));

    Loading.standard('Consultando ventas...');
    try {
      saleOrders = await querySaleOrderReport(reportForm);
    } catch (error) {
      console.error('[reporte_ventas] query error', error);
    } finally {
      Loading.remove();
    }
  }

  const columns: ITableColumn<ISaleOrder>[] = [
    {
      header: 'ID',
      getValue: saleOrder => saleOrder.ID,
      css: 'ff-mono fs15 text-right',
      headerCss: 'w-52',
    },
    {
      header: 'Fecha Hora',
      getValue: saleOrder => formatTime(saleOrder.Created, 'd-M h:n') as string,
      css: 'text-right',
      headerCss: 'w-100',
      cellCss: 'whitespace-nowrap',
    },
    {
      header: 'Estado',
      headerCss: 'w-82',
      css: 'text-center',
      render: saleOrder => renderSaleOrderStateSquares(saleOrder),
    },
    {
      header: 'Total',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN((saleOrder.TotalAmount || 0) / 100, 2),
    },
    {
      header: 'Deuda',
      css: 'ff-mono text-right',
      getValue: saleOrder => formatN((saleOrder.DebtAmount || 0) / 100, 2),
    },
    {
      header: 'Top Productos',
      css: 'fs15 leading-[1.15]',
      getValue: saleOrder => renderTopProductsSummary(saleOrder),
    },
  ];
</script>

<Page title="Reporte Ventas">
  <div class="flex items-center justify-between mb-12 gap-12">
    <div class="flex items-center w-full gap-12">
	    <DateInput
	      label="Fecha Inicio"
	      css="w-140"
	      save="fechaInicio"
	      bind:saveOn={reportForm}
	    />
	    <DateInput
	      label="Fecha Fin"
	      css="w-140"
	      save="fechaFin"
	      bind:saveOn={reportForm}
	    />
      <SearchSelect
        bind:saveOn={reportForm}
        save="clientID"
        css="w-260"
        label="Cliente"
        keyId="ID"
        keyName="DisplayName"
        options={clientOptions}
        placeholder=""
      />
      <SearchSelect
        bind:saveOn={reportForm}
        save="productID"
        css="w-260"
        label="Producto"
        keyId="ID"
        keyName="Nombre"
        options={productosService.records}
        placeholder=""
      />
      <SearchSelect
        bind:saveOn={reportForm}
        save="status"
        css="w-180"
        label="Estado"
        keyId="ID"
        keyName="Nombre"
        options={saleOrderStatusOptions}
        placeholder=""
      />
      <button class="px-16 py-8 bx-purple mt-8 h-44"
        aria-label="Consultar reporte"
        onclick={(event) => {
          event.stopPropagation();
          consultarReporteVentas();
        }}
      >
        <i class="icon-search"></i>
      </button>
    </div>

    <div class="flex items-center mr-16 w-224 ml-auto relative">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-search"></i>
      </div>
      <input
        class="w-full pl-36 pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder="Buscar..."
        onkeyup={(event) => {
          event.stopPropagation();
          throttle(() => {
            filterText = ((event.target as HTMLInputElement).value || '').toLowerCase().trim();
          }, 150);
        }}
      />
    </div>
  </div>

  <VTable
    data={saleOrders}
    columns={columns}
    css="w-full"
    tableCss="w-full"
    maxHeight="calc(100vh - 8rem - 12px)"
    filterText={filterText}
    getFilterContent={(saleOrder) => {
      const clientName = clientesService.recordsMap.get(saleOrder.ClientID)?.Name || '';
      const topProducts = renderTopProductsSummary(saleOrder);
      return [saleOrder.ID, clientName, topProducts].join(' ').toLowerCase();
    }}
  />
</Page>
