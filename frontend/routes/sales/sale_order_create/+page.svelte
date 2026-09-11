<script lang="ts">
import Input from '$components/form/Input.svelte';
import LayerStatic from '$components/layers/LayerStatic.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import VirtualCards from '$components/misc/VirtualCards.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import Page from '$domain/Page.svelte';
import { Loading, formatN, wordInclude } from '$libs/helpers';
import Button from '$components/buttons/Button.svelte';

import CheckboxOptions from '$components/form/CheckboxOptions.svelte';
import SystemParametersEditor from '$domain/SystemParametersEditor.svelte';
import { CajasService } from '$routes/finance/cash-banks/cajas.svelte';
import { getWarehouseProductStock, type IProductStock, type IProductStockDetail } from '$routes/logistics/products-stock/stock-movement';
import { ClientProviderService, ClientProviderType, type IClientProvider } from '$services/crm/client-provider.svelte';
import { ProductsService } from '$services/production/products.svelte';
import { SharedListsService } from "$services/business/shared-lists.svelte";
import { SystemParametersService } from '$services/services/system-parameters.svelte';
import { untrack } from 'svelte';
import { EmpresaParametrosService } from '../../company/configuration/empresas.svelte';
import { DOC_TYPE_BOLETA, DOC_TYPE_FACTURA, docTypeName } from '../../company/configuration/invoice-series';
import { tr } from '$core/store.svelte';
import type { IWarehouse } from "../../business/branches-warehouses/branches-warehouses.svelte";
import { WarehousesService } from "../../business/branches-warehouses/branches-warehouses.svelte";
import ProductoVentaCard from './SaleProductCard.svelte';
import { type Quantity, addQuantity, formatQuantity, quantityAmount, quantityDivisorOf, totalSubUnits } from '$core/quantity';
import type { ProductoVenta, VentaProducto } from "./sale_order.svelte";
import { useUI } from '@genix/ui';
import { SaleOrderState, SALE_ACTION_PAYMENT, SALE_ACTION_DELIVERY } from "./sale_order.svelte";
    import DateInput from '$components/form/DateInput.svelte';

  // Helpers
  const formatMo = (n: number) => formatN(n / 100, 2);

  // Services
  const almacenesService = new WarehousesService();
  const clientesService = new ClientProviderService(ClientProviderType.CLIENT, true);
  const productosService = new ProductsService(true);
  const listasService = new SharedListsService([2], true); // 2: Marcas
  const parametrosService = new EmpresaParametrosService();
  const systemParamsService = new SystemParametersService();
  const cajas = new CajasService()

  // State
  const ventasState = new SaleOrderState();
  const ui = useUI();

  let almacenSelected = $state(-1);
  let productoSelected = $state(-1);
  let searchInput = $state<HTMLInputElement>();
  let clientModeSelected = $state(0);

  // Computed
  const separarProcesoVenta = $derived(systemParamsService.recordsMap.get(1)?.ValueInts || []);
  const isSeparadoProceso = $derived(separarProcesoVenta.includes(2));
  const clientModeOptions = [
    { ID: 1, Name: "Selecionar Cliente" },
    { ID: 2, Name: "Registrar Cliente" },
  ];
  // Payments book a cash-bank movement, so without a registered caja the "Pagado" action must not be offered.
  const hasCashBankRegistered = $derived(cajas.isReady > 0 && cajas.Cajas.length > 0);
  const missingCashBankWarning = $derived(cajas.isReady > 0 && cajas.Cajas.length === 0);
  const saleActionOptions = $derived([
    { id: SALE_ACTION_DELIVERY, name: "Recibido" },
    ...(hasCashBankRegistered ? [{ id: SALE_ACTION_PAYMENT, name: "Pagado" }] : []),
  ]);
  // Paid now: the money lands in a caja. Not paid: only a due date makes sense, so the two selectors are exclusive.
  const isPaidNow = $derived(ventasState.form.ActionsIncluded.includes(SALE_ACTION_PAYMENT));
  // A sale is only ever issued as a factura or a boleta; note series exist to correct a document
  // that already went out, so offering them here would mint a sale id no document can use.
  const invoiceSeriesOptions = $derived(
    (parametrosService.empresa.InvoiceSeries || [])
      .filter((series) => series.ss === 1
        && (series.DocType === DOC_TYPE_FACTURA || series.DocType === DOC_TYPE_BOLETA))
      .map((series) => ({
        ID: series.SeriesID,
        Name: `${tr(docTypeName(series.DocType)).toUpperCase()} · ${series.SeriesCode}`,
      })),
  );
  const clientOptions = $derived.by(() => {
    // Build a combined label so the selector matches by name and registry number with the shared SearchSelect component.
    return clientesService.records.map((clientRecord) => ({
      ...clientRecord,
      DisplayName: clientRecord.RegistryNumber
        ? `${clientRecord.Name} ${clientRecord.RegistryNumber}`
        : clientRecord.Name,
    }));
  });

  // Data
  let productosStock = $state([] as IProductStock[]);
  let productosParsed = $state([] as ProductoVenta[]);
  let productosParsedAll = $state([] as ProductoVenta[]); // cache for filtering

  // Effects
  $effect(() => {
  	almacenesService.Almacenes;
   	if(!almacenesService.Almacenes.length || almacenSelected > 0){ return }
   
		untrack(() => {
		   almacenSelected = almacenesService.Almacenes[0].ID;
		   ventasState.form.WarehouseID = almacenSelected;
		   loadStock(almacenSelected);	
		})
  });

  $effect(() => {
	  if(!cajas.isReady){ return }
	  const firstCajaID = cajas.Cajas[0]?.ID || 0

	  // Untracked: the form field written here is also read here, so tracking it would re-trigger this effect forever.
	  // A missing caja is NOT resolved here: the first (cached) response can arrive empty, and dropping the payment
	  // action on it would silently untick "Pagado". postSaleOrder drops it at submit time instead.
	  untrack(() => { ventasState.form.LastPaymentCajaID = firstCajaID })
  });
  
  $effect(() => {
  	productosService.records;
	   untrack(() => {
	      	if(productosStock.length > 0){ parseProductos() }
	   });
  });

  $effect(() => {
    // Keep the outgoing payload in sync with the chosen client mode to avoid stale values.
    if (clientModeSelected === 1) {
      ventasState.form.ClientInfo = undefined;
    } else if (clientModeSelected === 2) {
      ventasState.form.ClientID = 0;
      ventasState.form.ClientInfo = ventasState.form.ClientInfo || {
        Name: "",
        RegistryNumber: "",
      };
    } else {
      ventasState.form.ClientID = 0;
      ventasState.form.ClientInfo = undefined;
    }
  });

  async function loadStock(almacenID: number) {
    Loading.standard("Cargando stock...");
    productosStock = await getWarehouseProductStock(almacenID);
    console.log("productosStock:", productosStock)
    parseProductos();
    Loading.remove();
  }
  
  function parseProductos() {
    if (!productosService.records.length || !productosStock.length) return;

  	const productStockGroups: Map<string,ProductoVenta> = new Map()
   
   	for(const e of productosStock){
      const producto = productosService.recordsMap.get(e.ProductID)
      if(!producto){ continue }

      // Generic stock is the base row quantity plus non-serialized details.
      const serialNumbers = (e.StockDetails || []).filter((stockDetail) => !!stockDetail.SerialNumber?.trim())
      const genericDetailQuantity = (e.StockDetails || []).reduce((detailQuantity, stockDetail) => {
        return stockDetail.SerialNumber?.trim() ? detailQuantity : detailQuantity + stockDetail.Quantity
      }, 0)
      const genericDetailSubQuantity = (e.StockDetails || []).reduce((detailSub, stockDetail) => {
        return stockDetail.SerialNumber?.trim() ? detailSub : detailSub + (stockDetail.SubQuantity || 0)
      }, 0)
      const stockDivisor = quantityDivisorOf(producto.SbuQuantity)

      const stockBuckets: Array<{
        key: string
        quantity: Quantity
        serialNumbers?: IProductStockDetail[]
      }> = [
        {
          key: [e.ProductID, e.PresentationID || 0, 0].join("_"),
          quantity: {
            units: e.Quantity + genericDetailQuantity,
            sub: (e.SubQuantity || 0) + genericDetailSubQuantity,
          },
        },
      ]

      if (serialNumbers.length > 0) {
        stockBuckets.push({
          key: [e.ProductID, e.PresentationID || 0, 1].join("_"),
          // A serial number is one physical item, so serialized stock has no sub-unit half.
          quantity: {
            units: serialNumbers.reduce((detailQuantity, stockDetail) => detailQuantity + stockDetail.Quantity, 0),
            sub: 0,
          },
          serialNumbers,
        })
      }

      for (const stockBucket of stockBuckets) {
        if (totalSubUnits(stockBucket.quantity, stockDivisor) <= 0) { continue }

        if(!productStockGroups.has(stockBucket.key)){
          const presentationID = e.PresentationID || 0
          const presentationName = presentationID
            ? producto.Presentations?.find((presentationOption) => presentationOption.id === presentationID)?.nm || `Presentación ${presentationID}`
            : ""
          const brandName = listasService.recordsMap.get(producto.BrandID)?.Name || ""
          const displayName = presentationName ? `${producto.Name} (${presentationName})` : producto.Name

          // Build the final sale row once and only mutate the aggregated fields while iterating stock.
          productStockGroups.set(stockBucket.key, {
            key: stockBucket.key,
            available: { units: 0, sub: 0 },
            subDivisor: quantityDivisorOf(producto.SbuQuantity),
            presentationID,
            presentationName,
            displayName,
            searchText: `${producto.Name} ${presentationName} ${brandName}`.toLowerCase(),
            producto,
            serialNumbers: [] as IProductStockDetail[],
          } as ProductoVenta)
        }

        const productStockGroup = productStockGroups.get(stockBucket.key)
        if(productStockGroup){
          productStockGroup.available = addQuantity(productStockGroup.available, stockBucket.quantity)
          if (stockBucket.serialNumbers?.length) {
            // Keep serial numbers attached to the serialized row for filtering and selection.
            productStockGroup.serialNumbers = [
              ...(productStockGroup.serialNumbers || []),
              ...stockBucket.serialNumbers,
            ]
            productStockGroup.searchText = `${productStockGroup.searchText} ${stockBucket.serialNumbers.map((stockDetail) => stockDetail.SerialNumber || "").join(" ")}`.trim()
          }
        }
      }
    }

    // No synthesized sub-unit row: a product with a sub-unit is one row whose card offers
    // both whole units and sub-units, so the two can never disagree about available stock.
    productosParsedAll = [...productStockGroups.values()]
    filterProductos(ventasState.filterText)
  }

  function applyFilters() {
    const text = ventasState.filterText.toLowerCase();
    const serialNumberText = ventasState.filterSerialNumber.toLowerCase();

    if (!text && !serialNumberText) {
      productosParsed = productosParsedAll;
      return;
    }

    const terms = text ? text.split(" ") : [];

    productosParsed = productosParsedAll.filter((e) => {
        // Filter by Name
        const matchName = terms.length === 0 || wordInclude(e.searchText, terms);

        // Filter by serial number
        let matchSerialNumber = true;
        if(serialNumberText) {
            matchSerialNumber = false;
            if(e.serialNumbers && e.serialNumbers.length > 0) {
                // Serialized rows stay searchable by partial serial number.
                matchSerialNumber = e.serialNumbers.some((stockDetail) =>
                  stockDetail.SerialNumber && stockDetail.SerialNumber.toLowerCase().includes(serialNumberText),
                );
            }
        }

        return matchName && matchSerialNumber;
    });

    productoSelected = -1;
  }

  function filterProductos(text: string) {
    ventasState.filterText = text;
    applyFilters();
  }

  function filterSerialNumbers(text: string) {
    ventasState.filterSerialNumber = text;
    applyFilters();
  }

  function handleKeydown(ev: KeyboardEvent) {
    if (ventasState.ventaErrorMessage) ventasState.ventaErrorMessage = "";

    if (ev.key === "ArrowUp") {
      ev.preventDefault();
      const newIdx = productoSelected - 1;
      if (newIdx >= -1) productoSelected = newIdx;
    } else if (ev.key === "ArrowDown") {
      ev.preventDefault();
      const newIdx = productoSelected + 1;
      if (newIdx < productosParsed.length) productoSelected = newIdx;
    } else if (ev.key === "Enter" && productoSelected >= 0) {
      ev.preventDefault();
      // Add 1 unit of selected
      const prod = productosParsed[productoSelected];
      if (prod.serialNumbers?.length) {
        ventasState.ventaErrorMessage = "Seleccione una serie específica.";
        return;
      }
      ventasState.addProducto(prod, { units: 1, sub: 0 });
      filterProductos("");
    } else if (ev.key === "Escape") {
      productoSelected = -1;
      searchInput?.focus();
    }
  }

  function handleClientModeChange(option?: { ID: number }) {
    clientModeSelected = option?.ID || 0;
  }

  function handleClientSelected(clientRecord?: IClientProvider) {
    ventasState.form.ClientID = clientRecord?.ID || 0;
  }

  async function handlePostSaleOrder() {
    const selectedClient = clientesService.records.find(
      (clientRecord) => clientRecord.ID === ventasState.form.ClientID);
    const wasSaved = await ventasState.postSaleOrder(
      parametrosService.empresa.InvoiceSeries || [], selectedClient);
    if (wasSaved) {
      clientModeSelected = 0;
    }
  }

  // The table runs with disableHeaderPadding, so the header's whole box comes from here.
  const cartHeaderCss = 'text-[14px] pt-6 pb-4 px-8';

  const cartColumns: ITableColumn<VentaProducto>[] = [
    {
      id: 'quantity',
      header: 'CANT.',
      align: 'right',
      headerStyle: { width: '55px' },
      headerInnerCss: cartHeaderCss,
      css: 'text-blue-600 font-bold text-sm',
      getValue: (item) => formatQuantity(item.cantidad, item.subDivisor, item.producto?.SbuUnit),
    },
    {
      id: 'product',
      header: 'PRODUCTO',
      align: 'left',
      headerInnerCss: cartHeaderCss,
      css: 'text-sm text-gray-800 py-4',
      getValue: (item) => item.displayName,
      // Serialized items carry their picked series as chips under the name.
      render: (item) => {
        const serialChips = [...(item.serialNumbers?.entries() || [])].map(([serialNumber, qty]) =>
          `<span class="text-[10px] bg-white border border-gray-200 px-6 rounded text-gray-600">Serie ${serialNumber} <span class="font-bold text-gray-800">x${qty}</span></span>`);
        if (serialChips.length === 0) { return item.displayName }
        return `<div>${item.displayName}</div><div class="flex flex-wrap gap-4 mt-2">${serialChips.join("")}</div>`;
      },
    },
    {
      id: 'price',
      header: 'PRECIO',
      align: 'right',
      headerStyle: { width: '80px' },
      headerInnerCss: cartHeaderCss,
      css: 'font-mono text-sm font-bold text-gray-700',
      getValue: (item) => formatMo(quantityAmount(
        item.cantidad, item.producto?.FinalPrice || 0, item.producto?.SbuFinalPrice || 0)),
    },
  ];
</script>

<svelte:window onkeydown={handleKeydown} />

{#snippet cartRowRemoveButton(item: VentaProducto)}
  <Button icon="icon-[fa--trash]"
    css="mr-6 flex h-24 w-24 items-center justify-center rounded-full bg-red-500 text-[12px] text-white shadow-sm hover:bg-red-600"
    onClick={() => ventasState.removeProducto(item.key)}
    label="Removes this product from the current sale order cart."
  />
{/snippet}

<Page title="Ventas"
  options={[{ id: 1, name: "Ventas" }, { id: 2, name: "Configuración" }]}
>
  {#if ui.state.pageOptionSelected === 1}
    <div class="flex h-full gap-20">
      <!-- Main Content -->
      <div class="flex-1 flex flex-col min-w-0 relative">
        <!-- Toolbar -->
        <div class="mb-12 flex gap-6 md:gap-12" aria-label="Product search toolbar with warehouse selector and product filter">
          <div class="w-[40%] md:w-250 md:mr-12">
            <SearchSelect
              label=""
              keyId="ID"
              keyName="Name"
              options={almacenesService.Almacenes}
              placeholder="ALMACÉN"
              selected={almacenSelected}
              onChange={(e: IWarehouse) => {
                if (e) {
                  almacenSelected = e.ID;
                  ventasState.form.WarehouseID = e.ID;
                  loadStock(e.ID);
                }
              }}
            />
          </div>

          <div class="flex w-[60%] gap-6 md:flex-1 md:gap-4">
            <div class="w-1/2 md:flex-1">
               <input
                 bind:this={searchInput}
                 type="text"
                 class="w-full px-12 py-8 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                 placeholder="PRODUCTO..."
                 value={ventasState.filterText}
                 oninput={(e) => filterProductos(e.currentTarget.value)}
               />
            </div>
            <div class="w-1/2 md:w-200">
               <input
                 type="text"
                 class="w-full px-12 py-8 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                 placeholder="Serie..."
                 value={ventasState.filterSerialNumber}
                 oninput={(e) => filterSerialNumbers(e.currentTarget.value)}
               />
            </div>
          </div>
        </div>

        <!-- Grid/List -->
        <div class="flex-1 min-h-0">
          <VirtualCards
            items={productosParsed}
            height="calc(100vh - 76px - var(--header-height))"
            maxColumns={2}
            mobileBreakpointPx={920}
            estimatedRowHeight={98}
            bufferSize={6}
            columnGapPx={8}
            rowGapPx={6}
            containerCss="h-full"
            emptyMessage="No se encontraron productos"
            useInnerPadding
          >
            {#snippet children(item, itemIndex)}
              <ProductoVentaCard
                idx={itemIndex}
                productoStock={item}
                isSelected={itemIndex === productoSelected}
                ventaProducto={ventasState.ventaProductosMap.get(item.key)}
                filterText={ventasState.filterText}
                onselect={(i) => (productoSelected = i)}
                onadd={(n, serialNumber) => {
                  ventasState.addProducto(item, n, serialNumber);
                  filterProductos("");
                }}
              />
            {/snippet}
          </VirtualCards>
        </div>
      </div>

      <LayerStatic
        css="w-[40%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
        mobileLayerTitle="Detalle de Venta"
        useMobileLayerVertical={124}
      >
        <!-- Error Message -->
        {#if ventasState.ventaErrorMessage}
          <div class="bg-red-50 m-8 text-red-600 p-12 text-sm font-medium border-b border-red-100 animate-in slide-in-from-top-2"
          >
            {ventasState.ventaErrorMessage}
          </div>
        {/if}
        <!-- Header -->
        <div class="px-10 py-8 border-b border-gray-100 flex items-center justify-between bg-gray-50/50"
          aria-label="Sale order totals and save action bar"
        >
       	<div class="mr-16">
          <div class="hidden font-bold font-xl mb-2 -mt-2 text-gray-800 mb-4 items-center justify-between md:flex">
            <span>Detalle de Venta</span>
          </div>
          <div class="flex items-center gap-6">
            <div class="bg-gray-100 flex flex-1 p-6 rounded-md items-center gap-6 min-h-36 md:min-w-170">
                <div class="text-[10px] leading-[1] text-gray-500 font-bold tracking-wider uppercase mr-8">
                  <div>Sub</div>
                  <div>Total</div>
                </div>
                <div class="leading-[1] text-gray-800 text-[16px] ml-auto">
                    {formatMo(ventasState.form.TotalAmount - ventasState.form.TaxAmount)}
                </div>
            </div>

            <div class="bg-blue-50 items-center flex flex-1 p-6 rounded-md gap-6 min-h-36 md:min-w-170">
                <div class="text-[10px] text-blue-600 uppercase font-bold tracking-wider mr-8">Total</div>
                <div class="leading-[1] text-blue-700 font-bold text-[22px] ml-auto">
                    {formatMo(ventasState.form.TotalAmount)}
                </div>
            </div>
          </div>
        </div>
          <Button color="blue" icon="icon-[fa--floppy-o]" name="Generar" hideNameOnMobile
            css="shrink-0" label="Saves the current sale order and generates it in the system." onClick={handlePostSaleOrder} />
        </div>
        <!-- Every row below shares the same 12-column grid so the left column (actions, client mode,
             document) and the right column (caja/due date, client name) line up across rows. -->
        <div class="w-full px-12 mt-6 mb-6">
	        <div class="grid grid-cols-12 gap-8 items-center" aria-label="Sale order actions and payment">
	      	  <CheckboxOptions type="multiple" css="col-span-5"
	     			  options={saleActionOptions}
	       		  keyId="id" keyName="name" save="ActionsIncluded"
	       		  saveOn={ventasState.form}
	       	  />
	        	{#if isPaidNow && hasCashBankRegistered}
		        	<SearchSelect
		             css="col-span-7"
			            label="" save="LastPaymentCajaID"
			            keyId="ID"
			            keyName="Name" saveOn={ventasState.form}
			            options={cajas.Cajas}
			            placeholder="CAJA"
			          />
	        	{:else}
		          <DateInput
		            css="col-span-7"
		            label="" save="PaymentDueDate"
		            saveOn={ventasState.form}
		            placeholder="Date Pago"
		          />
	        	{/if}
	        </div>
	        {#if missingCashBankWarning}
	          <div class="mt-6 flex items-center gap-6 rounded-md border border-amber-200 bg-amber-50 px-8 py-6 text-sm text-amber-700">
	            <i class="icon-[fa--exclamation-triangle] shrink-0"></i>
	            <span>Necesitas registrar una caja para aceptar pagos.</span>
	          </div>
	        {/if}
        </div>
        <div class="px-12 grid grid-cols-12 gap-8" aria-label="Client mode and invoice series">
          <SearchSelect useStyle={1}
             label=""
             keyId="ID" css="col-span-5 text-sm"
             keyName="Name"
             options={clientModeOptions}
             selected={clientModeSelected}
             onChange={handleClientModeChange}
             placeholder="SIN CLIENTE"
           />
          <SearchSelect useStyle={1}
             label="" save="IssueSeriesID"
             keyId="ID" css="col-span-7 text-sm"
             keyName="Name" saveOn={ventasState.form}
             options={invoiceSeriesOptions}
             placeholder="SIN COMPROBANTE"
           />
        </div>
        <div class="px-12 pb-10 mt-8 grid grid-cols-12 gap-8" aria-label="Client selection or registration form">
          {#if clientModeSelected === 1}
            <SearchSelect
              css="col-span-12"
              label=""
              keyId="ID"
              keyName="DisplayName"
              options={clientOptions}
              selected={ventasState.form.ClientID}
              onChange={handleClientSelected}
              placeholder="Buscar cliente por nombre o documento"
            />
          {:else if clientModeSelected === 2 && ventasState.form.ClientInfo}
            <Input
              label="" css="col-span-5"
              saveOn={ventasState.form.ClientInfo}
              save="RegistryNumber"
              placeholder="Documento / RUC"
            />
            <Input
              label="" css="col-span-7"
              saveOn={ventasState.form.ClientInfo}
              save="Name"
              placeholder="Nombre del cliente"
            />
          {/if}
        </div>
        <!-- List -->
        <div class="flex-1 min-h-0 px-8 pb-8" aria-label="Sale order cart items list">
          <VTable
            columns={cartColumns}
            data={ventasState.ventaProductos}
            maxHeight="100%"
            disableVirtualizer
            emptyMessage="Empty cart|Carrito vacío"
            onRowHover={cartRowRemoveButton}
            disableHeaderPadding
          />
        </div>
      </LayerStatic>
    </div>
  {:else if ui.state.pageOptionSelected === 2}
    <div class="flex justify-center py-24">
      <SystemParametersEditor />
    </div>
  {/if}
</Page>
