<script lang="ts">
  import DateInput from '$components/DateInput.svelte';
  import SearchSelect from '$components/SearchSelect.svelte';
  import Page from '$domain/Page.svelte';
  import { formatTime, Loading } from '$libs/helpers';
  import { ProductosService } from '$routes/negocio/productos/productos.svelte';
  import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte';
  import SaleOrdersTable from '../SaleOrdersTable.svelte';
  import { querySaleOrderReport, saleOrderStatusOptions, type ISaleOrder } from './sale_order_report.svelte';
    import ButtonLayer from '$components/ButtonLayer.svelte';
  import KeyValueStrip from '$components/micro/KeyValueStrip.svelte'

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
  let searchText = $state('');
  let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
  let isSearchOpen = $state(false)

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

  function onSearchInput(event: Event): void {
    event.stopPropagation();
    const nextSearchText = ((event.target as HTMLInputElement).value || '').toLowerCase().trim();

    // Debounce table filtering so large result sets do not recompute on every keystroke.
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }

    searchDebounceTimer = setTimeout(() => {
      searchText = nextSearchText;
      searchDebounceTimer = null;
    }, 150);
  }
</script>

<Page title="Reporte Ventas">
  <div class="flex items-center justify-between mb-12 gap-12">
    <div class="flex items-center w-full gap-12">
  		<ButtonLayer buttonClass="bx-purple" bind:isOpen={isSearchOpen}
				horizontalOffset={0} useOutline={true}
				edgeMargin={0} buttonClassOnShow="bx-red"
				layerClass="w-600"
				icon="icon-search" iconOnShow="icon-cancel"
			>
				<div class="w-full grid grid-cols-24 gap-12 p-12">
			    <DateInput
			      label="Fecha Inicio"
			      css="col-span-12"
			      save="fechaInicio"
			      bind:saveOn={reportForm}
			    />
			    <DateInput
			      label="Fecha Fin"
			      css="col-span-12"
			      save="fechaFin"
			      bind:saveOn={reportForm}
			    />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="clientID"
		        css="col-span-12"
		        label="Cliente"
		        keyId="ID"
		        keyName="DisplayName"
		        options={clientOptions}
		        placeholder=""
		      />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="productID"
		        css="col-span-12"
		        label="Producto" optionsCss="w-480"
		        keyId="ID"
		        keyName="Nombre"
		        options={productosService.records}
		        placeholder=""
		      />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="status"
		        css="col-span-12"
		        label="Estado"
		        keyId="ID"
		        keyName="Nombre"
		        options={saleOrderStatusOptions}
		        placeholder=""
		      />
					<div class="col-span-12 flex items-center justify-center">
			      <button class="px-16 py-8 bx-purple mt-8 h-44"
			        aria-label="Consultar reporte"
			        onclick={(event) => {
			          event.stopPropagation();
			          consultarReporteVentas();
								isSearchOpen = false
			        }}
			      >
			        Buscar <i class="icon-search"></i>
			      </button>
					</div>
				</div>
			</ButtonLayer>
			<KeyValueStrip
				label1="Fec. Inicio"
				value1={reportForm.fechaInicio}
				getContent1={v => formatTime(v,"d-m-Y") as string}
				label2="Fec. Fin"
				value2={reportForm.fechaFin}
				getContent2={v => formatTime(v,"d-m-Y") as string}
				label3="Cliente"
				value3={reportForm.clientID}
				getContent3={(clientID) => {
					// Resolve the selected client id to the visible customer name in the filter summary.
					return clientID
						? (clientesService.recordsMap.get(Number(clientID))?.Name || `Cliente #${clientID}`)
						: 'Todos';
				}}
				label4="Producto"
				value4={reportForm.productID}
				getContent4={(productID) => {
					// Resolve the selected product id to the visible product name in the filter summary.
					return productID
						? (productosService.recordsMap.get(Number(productID))?.Nombre || `Producto #${productID}`)
						: 'Todos';
				}}
				label5="Estado"
				value5={reportForm.status}
				getContent5={(statusID) => {
					// Keep the summary aligned with the same option labels used by the status selector.
					return saleOrderStatusOptions.find((statusOption) => statusOption.ID === Number(statusID))?.Nombre || 'Todos';
				}}
			/>
    </div>

    <div class="flex items-center mr-16 w-224 ml-auto relative">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-search"></i>
      </div>
      <input
        class="w-full pl-36 bg-white pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder="Buscar..."
        oninput={onSearchInput}
      />
    </div>
  </div>

  <SaleOrdersTable
    data={saleOrders}
    productsMap={productosService.recordsMap}
    clientsMap={clientesService.recordsMap}
    css="w-full"
    maxHeight="calc(100vh - 8rem - 12px)"
    searchText={searchText}
  />
</Page>
