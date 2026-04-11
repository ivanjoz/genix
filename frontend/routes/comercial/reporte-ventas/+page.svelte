<script lang="ts">
  import DateInput from '$components/DateInput.svelte';
  import SearchSelect from '$components/SearchSelect.svelte';
  import Page from '$domain/Page.svelte';
  import { Loading, throttle } from '$libs/helpers';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte';
  import SaleOrdersTable from '../SaleOrdersTable.svelte';
  import { querySaleOrderReport, saleOrderStatusOptions, type ISaleOrder } from './sale_order_report.svelte';

  const productosService = new ProductosService(true);
  const clientesService = new ClientProviderService(ClientProviderType.CLIENT, true);

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

  function getSaleOrderClientName(saleOrder: ISaleOrder): string {
    if (!saleOrder.ClientID) {
      return '-';
    }

    return clientesService.recordsMap.get(saleOrder.ClientID)?.Name || `Cliente #${saleOrder.ClientID}`;
  }
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

  <SaleOrdersTable
    data={saleOrders}
    getProductName={(productID) => productosService.recordsMap.get(productID)?.Nombre || `Producto #${productID}`}
    getClientName={getSaleOrderClientName}
    css="w-full"
    maxHeight="calc(100vh - 8rem - 12px)"
    filterText={filterText}
  />
</Page>
