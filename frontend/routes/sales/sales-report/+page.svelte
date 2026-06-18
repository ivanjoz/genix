<script lang="ts">
  import DateInput from '$components/form/DateInput.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import Page from '$domain/Page.svelte';
  import { formatTime, Loading } from '$libs/helpers';
  import { ProductsService } from '$routes/business/products/products.svelte';
  import { ClientProviderService, ClientProviderType } from '$routes/business/customers/customers.svelte';
  import SaleOrdersTable from '../SaleOrdersTable.svelte';
  import { querySaleOrderReport, saleOrderStatusOptions, type ISaleOrder } from './sale_order_report.svelte';
    import ButtonLayer from '$components/buttons/ButtonLayer.svelte';
  import KeyValueStrip from '$components/misc/KeyValueStrip.svelte';
  import { tr } from '$core/store.svelte';
  import T from '$components/misc/T.svelte';

  const productosService = new ProductsService(true);
  const clientesService = new ClientProviderService(ClientProviderType.CLIENT, true);

  const getCurrentUnixDay = (): number => Math.floor(Date.now() / (1000 * 60 * 60 * 24));

  const dateFin = getCurrentUnixDay();
  const dateInicio = dateFin - 7;

  let reportForm = $state({
    dateInicio,
    dateFin,
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

    Loading.standard(tr('Querying sales...|Consultando ventas...'));
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

<Page title="Sales Report|Reporte Ventas">
  <div class="grid grid-cols-[auto_minmax(0,1fr)] items-start mb-12 gap-12 md:flex md:items-center md:justify-between">
    <div class="contents md:flex md:items-center md:w-full md:gap-12">
  		<ButtonLayer buttonClass="bx-purple" bind:isOpen={isSearchOpen}
				horizontalOffset={0} useOutline={true}
				edgeMargin={0} buttonClassOnShow="bx-red"
				layerClass="w-600"
				icon="icon-[fa--search]" iconOnShow="icon-[fa--close]"
				label="Opens the search filter for the sales report."
			>
				<div class="w-full grid grid-cols-24 gap-12 p-12" aria-label="Sales report filter with date range, client, product, and status">
			    <DateInput
			      label="Start Date|Fecha Inicio"
			      css="col-span-12"
			      save="dateInicio"
			      bind:saveOn={reportForm}
			    />
			    <DateInput
			      label="End Date|Fecha Fin"
			      css="col-span-12"
			      save="dateFin"
			      bind:saveOn={reportForm}
			    />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="clientID"
		        css="col-span-12"
		        label="Client|Cliente"
		        keyId="ID"
		        keyName="DisplayName"
		        options={clientOptions}
		        placeholder=""
		      />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="productID"
		        css="col-span-12"
		        label="Product|Producto" optionsCss="w-480"
		        keyId="ID"
		        keyName="Name"
		        options={productosService.records}
		        placeholder=""
		      />
		      <SearchSelect
		        bind:saveOn={reportForm}
		        save="status"
		        css="col-span-12"
		        label="Status|Estado"
		        keyId="ID"
		        keyName="Name"
		        options={saleOrderStatusOptions}
		        placeholder=""
		      />
					<div class="col-span-12 flex items-center justify-center">
			      <button class="px-16 py-8 bx-purple mt-8 h-44"
			        aria-label="Search report"
			        onclick={(event) => {
			          event.stopPropagation();
			          consultarReporteVentas();
							isSearchOpen = false
			        }}
			      >
			        <T text="Search|Buscar" /> <i class="icon-[fa--search]"></i>
			      </button>
					</div>
				</div>
			</ButtonLayer>
			<KeyValueStrip
				css="col-span-2 row-start-2 w-full md:w-auto"
				label1="Start Date|Fec. Inicio"
				value1={reportForm.dateInicio}
				getContent1={v => formatTime(v,"d-m-Y") as string}
				label2="End Date|Fec. Fin"
				value2={reportForm.dateFin}
				getContent2={v => formatTime(v,"d-m-Y") as string}
				label3="Client|Cliente"
				value3={reportForm.clientID}
				getContent3={(clientID) => {
					// Resolve the selected client id to the visible customer name in the filter summary.
					return clientID
						? (clientesService.recordsMap.get(Number(clientID))?.Name || `Cliente #${clientID}`)
						: tr('All|Todos');
				}}
				label4="Product|Producto"
				value4={reportForm.productID}
				getContent4={(productID) => {
					// Resolve the selected product id to the visible product name in the filter summary.
					return productID
						? (productosService.recordsMap.get(Number(productID))?.Name || `Producto #${productID}`)
						: tr('All|Todos');
				}}
				label5="Status|Estado"
				value5={reportForm.status}
				getContent5={(statusID) => {
					// Keep the summary aligned with the same option labels used by the status selector.
					return saleOrderStatusOptions.find((statusOption) => statusOption.ID === Number(statusID))?.Name || tr('All|Todos');
				}}
			/>
    </div>

    <div class="relative col-start-2 row-start-1 flex items-start self-start w-full max-w-224 ml-auto md:mr-16 md:w-224">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-[fa--search]"></i>
      </div>
      <input
        class="w-full pl-36 bg-white pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder={tr("Search...|Buscar...")}
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
