<script lang="ts">
import Input from '$components/Input.svelte';
import LayerStatic from '$components/LayerStatic.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import VirtualCards from '$components/VirtualCards.svelte';
import Page from '$domain/Page.svelte';
import { Loading, formatN, wordInclude } from '$libs/helpers';

import CheckboxOptions from '$components/CheckboxOptions.svelte';
import { Core } from '$core/store.svelte';
import SystemParametersEditor from '$domain/SystemParametersEditor.svelte';
import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
import { getProductosStock, type IProductoStock, type IProductStockDetail } from '$routes/logistica/products-stock/stock-movement';
import { ClientProviderService, ClientProviderType, type IClientProvider } from '$routes/negocio/clientes/clientes-proveedores.svelte';
import { ProductosService } from '$routes/negocio/productos/productos.svelte';
import { ListasCompartidasService } from "$services/negocio/listas-compartidas.svelte";
import { SystemParametersService } from '$services/services/system-parameters.svelte';
import { untrack } from 'svelte';
import { EmpresaParametrosService } from '../../configuracion/parametros/empresas.svelte';
import type { IAlmacen } from "../../negocio/sedes-almacenes/sedes-almacenes.svelte";
import { AlmacenesService } from "../../negocio/sedes-almacenes/sedes-almacenes.svelte";
import ProductoVentaCard from './ProductoVentaCard.svelte';
import type { ProductoVenta } from "./sale_order.svelte";
import { SaleOrderState } from "./sale_order.svelte";

  // Helpers
  const formatMo = (n: number) => formatN(n / 100, 2);

  // Services
  const almacenesService = new AlmacenesService();
  const clientesService = new ClientProviderService(ClientProviderType.CLIENT, true);
  const productosService = new ProductosService(true);
  const listasService = new ListasCompartidasService([2], true); // 2: Marcas
  const parametrosService = new EmpresaParametrosService();
  const systemParamsService = new SystemParametersService();
  const cajas = new CajasService()

  // State
  const ventasState = new SaleOrderState();

  let almacenSelected = $state(-1);
  let productoSelected = $state(-1);
  let searchInput = $state<HTMLInputElement>();
  let clientModeSelected = $state(0);

  // Computed
  const separarProcesoVenta = $derived(systemParamsService.recordsMap.get(1)?.ValueInts || []);
  const isSeparadoProceso = $derived(separarProcesoVenta.includes(2));
  const clientModeOptions = [
    { ID: 1, Nombre: "Selecionar Cliente" },
    { ID: 2, Nombre: "Registrar Cliente" },
  ];
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
  let productosStock = $state([] as IProductoStock[]);
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
	  if(cajas.isReady && cajas.Cajas.length > 0){
	  	ventasState.form.LastPaymentCajaID = cajas.Cajas[0].ID
	  }
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
    productosStock = await getProductosStock(almacenID);
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
      const stockBuckets: Array<{
        key: string
        quantity: number
        serialNumbers?: IProductStockDetail[]
      }> = [
        {
          key: [e.ProductID, e.PresentationID || 0, 0].join("_"),
          quantity: e.Quantity + genericDetailQuantity,
        },
      ]

      if (serialNumbers.length > 0) {
        stockBuckets.push({
          key: [e.ProductID, e.PresentationID || 0, 1].join("_"),
          quantity: serialNumbers.reduce((detailQuantity, stockDetail) => detailQuantity + stockDetail.Quantity, 0),
          serialNumbers,
        })
      }

      for (const stockBucket of stockBuckets) {
        if (stockBucket.quantity <= 0) { continue }

        if(!productStockGroups.has(stockBucket.key)){
          const presentationID = e.PresentationID || 0
          const presentationName = presentationID
            ? producto.Presentaciones?.find((presentationOption) => presentationOption.id === presentationID)?.nm || `Presentación ${presentationID}`
            : ""
          const brandName = listasService.recordsMap.get(producto.MarcaID)?.Nombre || ""
          const displayName = presentationName ? `${producto.Nombre} (${presentationName})` : producto.Nombre

          // Build the final sale row once and only mutate the aggregated fields while iterating stock.
          productStockGroups.set(stockBucket.key, {
            key: stockBucket.key,
            cant: 0,
            presentationID,
            presentationName,
            displayName,
            searchText: `${producto.Nombre} ${presentationName} ${brandName}`.toLowerCase(),
            producto,
            serialNumbers: [] as IProductStockDetail[],
          } as ProductoVenta)
        }

        const productStockGroup = productStockGroups.get(stockBucket.key)
        if(productStockGroup){
          productStockGroup.cant += stockBucket.quantity
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

    const productosParsedNew = [...productStockGroups.values()]

    for (const productoVenta of [...productStockGroups.values()]) {
      if (productoVenta.key.endsWith("_1") || (productoVenta.producto.SbnCantidad || 0) <= 1) { continue }

      // Only create sub-unit rows for generic stock groups so the UI preserves the existing sale flow.
      productosParsedNew.push({
        key: `${productoVenta.key}_s`,
        cant: productoVenta.producto.SbnCantidad!,
        producto: productoVenta.producto,
        presentationID: productoVenta.presentationID,
        presentationName: productoVenta.presentationName,
        displayName: productoVenta.displayName,
        searchText: productoVenta.searchText,
        isSubUnidad: true,
      })
    }

    productosParsedAll = productosParsedNew
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
      ventasState.addProducto(prod, 1);
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
    const wasSaved = await ventasState.postSaleOrder();
    if (wasSaved) {
      clientModeSelected = 0;
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<Page title="Ventas"
  options={[{ id: 1, name: "Ventas" }, { id: 2, name: "Configuración" }]}
>
  {#if Core.pageOptionSelected === 1}
    <div class="flex h-full gap-20">
      <!-- Main Content -->
      <div class="flex-1 flex flex-col min-w-0 relative">
        <!-- Toolbar -->
        <div class="mb-12 flex gap-6 md:gap-12">
          <div class="w-[40%] md:w-250 md:mr-12">
            <SearchSelect
              label=""
              keyId="ID"
              keyName="Nombre"
              options={almacenesService.Almacenes}
              placeholder="ALMACÉN"
              selected={almacenSelected}
              onChange={(e: IAlmacen) => {
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
                onmouseover={() => (productoSelected = -1)}
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
        >
       	<div class="grow mr-16">
          <div class="hidden font-bold font-xl mb-2 -mt-2 text-gray-800 mb-4 items-center justify-between md:flex">
            <span>Detalle de Venta</span>
          </div>
          <div class="flex items-center gap-6">
            <div class="bg-gray-50 flex flex-1 p-6 rounded-md items-center gap-6 border border-gray-100 shadow-sm">
                <div class="text-[10px] leading-[1] text-gray-500 font-bold tracking-wider uppercase">
                  <div>Sub</div>
                  <div>Total</div>
                </div>
                <div class="leading-[1] text-gray-800 text-[16px] ml-auto">
                    {formatMo(ventasState.form.TotalAmount - ventasState.form.TaxAmount)}
                </div>
            </div>

            <div class="bg-blue-50 items-center flex flex-1 p-6 rounded-md gap-6 border border-blue-100 shadow-sm">
                <div class="text-[10px] text-blue-600 uppercase font-bold tracking-wider">Total</div>
                <div class="leading-[1] text-blue-700 font-bold text-[22px] ml-auto">
                    {formatMo(ventasState.form.TotalAmount)}
                </div>
            </div>
          </div>
        </div>
          <button class="bx-blue shrink-0"
            onclick={handlePostSaleOrder}
            title="Guardar venta"
          >
            <span class="hidden md:inline">Generar</span>
            <i class="icon-floppy"></i>
          </button>
        </div>
        <div class="grid w-full grid-cols-2 gap-8 px-12 py-6 md:flex md:items-center">
          <div class="col-span-2 md:col-span-1 md:order-2">
        	  <CheckboxOptions type="multiple"
       			  options={[ { id: 2, name: "Pagado" }, { id: 3, name: "Recibido" } ]}
         		  keyId="id" keyName="name" save="ActionsIncluded"
         		  saveOn={ventasState.form}
         	  />
          </div>
          <div class="md:order-1 md:mr-16">
	          <SearchSelect
              css="w-full"
	            label="" save="LastPaymentCajaID"
	            keyId="ID"
	            keyName="Nombre" saveOn={ventasState.form}
	            options={cajas?.Cajas||[]}
	            placeholder="CAJA"
	          />
          </div>
          <div class="md:order-3 md:ml-auto md:max-w-168">
	          <SearchSelect useStyle={1}
	             label=""
	             keyId="ID" css="w-full text-sm"
	             keyName="Nombre"
	             options={clientModeOptions}
	             selected={clientModeSelected}
	             onChange={handleClientModeChange}
	             placeholder="SIN CLIENTE"
	           />
          </div>
        </div>
        <div class="px-12 pb-10 mt-6">
          <div class="grid grid-cols-[170px,1fr] gap-8">
            {#if clientModeSelected === 1}
              <SearchSelect
                label=""
                keyId="ID"
                keyName="DisplayName"
                options={clientOptions}
                selected={ventasState.form.ClientID}
                onChange={handleClientSelected}
                placeholder="Buscar cliente por nombre o documento"
              />
            {:else if clientModeSelected === 2 && ventasState.form.ClientInfo}
              <div class="grid grid-cols-12 gap-8">
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
              </div>
            {/if}
          </div>
        </div>
        <!-- List -->
        <div class="flex-1 overflow-y-auto px-8 py-4 space-y-4">
          {#each ventasState.ventaProductos as item (item.key)}
            <div
              class="flex items-center gap-8 py-4 px-8 rounded-lg bg-gray-50 hover:bg-white hover:shadow-sm border border-transparent hover:border-gray-200 transition-all group"
            >
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-800 truncate">
                  <span class="text-blue-600 font-bold mr-4">{item.cantidad} X</span>
                  {item.displayName}
                  {#if item.isSubUnidad}
                    <span class="text-purple-600 text-[10px] ml-4 font-normal"
                      >({item.producto?.SbnUnidad})</span
                  >
                  {/if}
                </div>
                {#if item.serialNumbers && item.serialNumbers.size > 0}
                  <div class="flex flex-wrap gap-4 mt-2">
                    {#each item.serialNumbers.entries() as [serialNumber, qty]}
                      <span
                        class="text-[10px] bg-white border border-gray-200 px-6 rounded text-gray-600"
                      >
                        Serie {serialNumber} <span class="font-bold text-gray-800">x{qty}</span>
                      </span>
                    {/each}
                  </div>
                {/if}
              </div>

              <div class="flex items-center gap-8">
                <div class="font-mono text-sm font-bold text-gray-700">
                  {formatMo((item.isSubUnidad && item.producto?.SbnPreciFinal ? item.producto.SbnPreciFinal : (item.producto?.PrecioFinal || 0)) * item.cantidad)}
                </div>
                <button
                  class="p-4 text-red-400 transition-opacity hover:text-red-600 md:opacity-0 md:group-hover:opacity-100"
                  onclick={() => ventasState.removeProducto(item.key)}
                  title="Eliminar"
                >
                  <i class="icon-trash"></i>
                </button>
              </div>
            </div>
          {/each}

          {#if ventasState.ventaProductos.length === 0}
            <div
              class="flex flex-col items-center justify-center h-192 text-gray-300 gap-8"
            >
              <i class="icon-cart text-4xl"></i>
              <span class="text-sm">Carrito vacío</span>
            </div>
          {/if}
        </div>
      </LayerStatic>
    </div>
  {:else if Core.pageOptionSelected === 2}
    <div class="flex justify-center py-24">
      <SystemParametersEditor />
    </div>
  {/if}
</Page>
