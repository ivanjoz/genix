<script lang="ts">
import Input from '$components/Input.svelte';
import LayerStatic from '$components/LayerStatic.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import Page from '$domain/Page.svelte';
import { Loading, formatN, include } from '$libs/helpers';

import CheckboxOptions from '$components/CheckboxOptions.svelte';
import { Core } from '$core/store.svelte';
import SystemParametersEditor from '$domain/SystemParametersEditor.svelte';
import { CajasService } from '$routes/finanzas/cajas/cajas.svelte';
import { getProductosStock, type IProductoStock } from '$routes/logistica/products-stock/stock-movement';
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

  const chunkedProductos = $derived.by(() => {
    const result = [];
    for (let i = 0; i < productosParsed.length; i += 2) {
      result.push(productosParsed.slice(i, i + 2));
    }
    return result;
  });

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

    	const key = [e.ProductID, e.PresentationID||0, e.SKU ? 1 : 0].join("_")
     	if(!productStockGroups.has(key)){
        const presentationID = e.PresentationID || 0
        const presentationName = presentationID
          ? producto.Presentaciones?.find((presentationOption) => presentationOption.id === presentationID)?.nm || `Presentación ${presentationID}`
          : ""
        const brandName = listasService.recordsMap.get(producto.MarcaID)?.Nombre || ""
        const displayName = presentationName ? `${producto.Nombre} (${presentationName})` : producto.Nombre

        // Build the final sale row once and only mutate the aggregated fields while iterating stock.
      	productStockGroups.set(key, {
      	  key,
      	  cant: 0,
      	  presentationID,
      	  presentationName,
      	  displayName,
      	  searchText: `${producto.Nombre} ${presentationName} ${brandName}`.toLowerCase(),
      	  producto,
       	  skus: [] as IProductoStock[],
       } as ProductoVenta)
      }
      
     	if(e.SKU){
      	productStockGroups.get(key)?.skus?.push(e)
      }

      const productStockGroup = productStockGroups.get(key)
      if(productStockGroup){
        productStockGroup.cant += e.Quantity
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

  function parseProductosDeprecated() {
    if (!productosService.records.length || !productosStock.length) return;

    const stockGroupsByProductID = new Map<number, Map<number, {
      presentationID: number
      genericQuantity: number
      skuQuantity: number
      skus: IProductoStock[]
    }>>();

    // Group stock once so we avoid scanning the full stock list for every product.
    for (const stockRecord of productosStock) {
      const presentationID = stockRecord.PresentationID || 0;
      let stockGroupsByPresentationID = stockGroupsByProductID.get(stockRecord.ProductID);

      if (!stockGroupsByPresentationID) {
        stockGroupsByPresentationID = new Map();
        stockGroupsByProductID.set(stockRecord.ProductID, stockGroupsByPresentationID);
      }

      let stockGroup = stockGroupsByPresentationID.get(presentationID);
      if (!stockGroup) {
        stockGroup = {
          presentationID,
          genericQuantity: 0,
          skuQuantity: 0,
          skus: [],
        };
        stockGroupsByPresentationID.set(presentationID, stockGroup);
      }

      if (stockRecord.SKU) {
        stockGroup.skus.push(stockRecord);
        stockGroup.skuQuantity += stockRecord.Quantity;
      } else {
        stockGroup.genericQuantity += stockRecord.Quantity;
      }
    }

    const newProductos: ProductoVenta[] = [];

    for (const producto of productosService.records) {
      const stockGroupsByPresentationID = stockGroupsByProductID.get(producto.ID);
      if (!stockGroupsByPresentationID) continue;

      const brandName = listasService.recordsMap.get(producto.MarcaID)?.Nombre || "";
      const baseSearchText = `${producto.Nombre} ${brandName}`.toLowerCase();
      const presentationNameByID = new Map(
        (producto.Presentaciones || []).map((presentationOption) => [presentationOption.id, presentationOption.nm]),
      );

      for (const stockGroup of stockGroupsByPresentationID.values()) {
        const presentationName = stockGroup.presentationID
          ? presentationNameByID.get(stockGroup.presentationID) || `Presentación ${stockGroup.presentationID}`
          : "";
        const displayName = presentationName ? `${producto.Nombre} (${presentationName})` : producto.Nombre;
        const searchText = presentationName
          ? `${baseSearchText} ${presentationName}`.toLowerCase()
          : baseSearchText;

        if (stockGroup.skuQuantity > 0) {
          // Reuse the aggregated stock group instead of creating intermediate rows first.
          newProductos.push({
            producto,
            cant: stockGroup.skuQuantity,
            key: `K${producto.ID}-${stockGroup.presentationID}`,
            presentationID: stockGroup.presentationID,
            presentationName,
            displayName,
            searchText,
            skus: stockGroup.skus,
          });
        }

        if (stockGroup.genericQuantity > 0) {
          newProductos.push({
            producto,
            cant: stockGroup.genericQuantity,
            key: `P${producto.ID}-${stockGroup.presentationID}`,
            presentationID: stockGroup.presentationID,
            presentationName,
            displayName,
            searchText,
          });

          if ((producto.SbnCantidad || 0) > 1) {
            // Preserve the sub-unit sale flow only when the base presentation has stock.
            newProductos.push({
              producto,
              cant: producto.SbnCantidad!,
              key: `S${producto.ID}-${stockGroup.presentationID}`,
              presentationID: stockGroup.presentationID,
              presentationName,
              displayName,
              searchText,
              isSubUnidad: true,
            });
          }
        }
      }
    }

    productosParsedAll = newProductos;
    filterProductos(ventasState.filterText);
  }

  function applyFilters() {
    const text = ventasState.filterText.toLowerCase();
    const skuText = ventasState.filterSku.toLowerCase();

    if (!text && !skuText) {
      productosParsed = productosParsedAll;
      return;
    }

    const terms = text ? text.split(" ") : [];

    productosParsed = productosParsedAll.filter((e) => {
        // Filter by Name
        const matchName = terms.length === 0 || include(e.searchText, terms);

        // Filter by SKU
        let matchSku = true;
        if(skuText) {
            matchSku = false;
            if(e.skus && e.skus.length > 0) {
                // Check if any stock SKU matches
                matchSku = e.skus.some(s => s.SKU && s.SKU.toLowerCase().includes(skuText));
            }
        }

        return matchName && matchSku;
    });

    productoSelected = -1;
  }

  function filterProductos(text: string) {
    ventasState.filterText = text;
    applyFilters();
  }

  function filterSkus(text: string) {
    ventasState.filterSku = text;
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
      if (prod.skus?.length) {
        ventasState.ventaErrorMessage = "Seleccione un SKU específico.";
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
    <div class="flex h-full gap-16">
      <!-- Main Content -->
      <div class="flex-1 flex flex-col min-w-0">
        <!-- Toolbar -->
        <div class="flex mb-12">
          <div class="w-250 mr-12">
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

          <div class="flex-1 relative flex gap-4">
            <div class="relative flex-1">
               <i
                 class="icon-search absolute left-12 top-1/2 -translate-y-1/2 text-gray-400"
               ></i>
               <input
                 bind:this={searchInput}
                 type="text"
                 class="w-full pl-36 pr-16 py-8 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                 placeholder="Producto..."
                 value={ventasState.filterText}
                 oninput={(e) => filterProductos(e.currentTarget.value)}
               />
            </div>
            <div class="relative w-200">
               <i
                 class="icon-barcode absolute left-12 top-1/2 -translate-y-1/2 text-gray-400"
               ></i>
               <input
                 type="text"
                 class="w-full pl-36 pr-16 py-8 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                 placeholder="SKU..."
                 value={ventasState.filterSku}
                 oninput={(e) => filterSkus(e.currentTarget.value)}
               />
            </div>
          </div>
        </div>

        <!-- Grid/List -->
        <div class="flex-1 overflow-y-auto">
          {#each chunkedProductos as group, groupIdx (group[0].key)}
            <div class="flex gap-8 mb-6">
              {#each group as item, itemIdx (item.key)}
                <div class="flex-1 min-w-0">
                  <ProductoVentaCard
                    idx={groupIdx * 2 + itemIdx}
                    productoStock={item}
                    isSelected={(groupIdx * 2 + itemIdx) === productoSelected}
                    ventaProducto={ventasState.ventaProductosMap.get(item.key)}
                    filterText={ventasState.filterText}
                    onselect={(i) => (productoSelected = i)}
                    onmouseover={() => (productoSelected = -1)}
                    onadd={(n, sku) => {
                      ventasState.addProducto(item, n, sku);
                      filterProductos("");
                    }}
                  />
                </div>
              {/each}
              {#if group.length === 1}
                <div class="flex-1 min-w-0"></div>
              {/if}
            </div>
          {/each}

          {#if productosParsed.length === 0}
            <div class="text-center py-48 text-gray-400">
              No se encontraron productos
            </div>
          {/if}
        </div>
      </div>

      <!-- Side Cart -->
      <div class="w-[38%] min-w-350 ml-24"></div>
      <LayerStatic css="w-[38%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-60px)] shadow-lg">
        <!-- Error Message -->
        {#if ventasState.ventaErrorMessage}
          <div class="bg-red-50 m-8 text-red-600 p-12 text-sm font-medium border-b border-red-100 animate-in slide-in-from-top-2"
          >
            {ventasState.ventaErrorMessage}
          </div>
        {/if}
        <!-- Header -->
        <div class="px-16 py-12 border-b border-gray-100 flex items-center justify-between bg-gray-50/50"
        >
       	<div class="grow mr-16">
          <div class="font-bold font-xl mb-2 -mt-2 text-gray-800 mb-4 flex items-center justify-between">
            <span>Detalle de Venta</span>
          </div>
          <div class="grid grid-cols-3 gap-12">
            <div class="bg-gray-50 flex p-6 rounded-md items-center border border-gray-100 shadow-sm">
                <div class="text-[10px] leading-[1.1] text-gray-500 mb-4 font-bold tracking-wider">SUB <br> TOTAL</div>
                <div class="leading-[1] text-gray-800 text-[17px] ml-auto">
                    {formatMo(ventasState.form.TotalAmount - ventasState.form.TaxAmount)}
                </div>
            </div>

            <div class="bg-blue-50 items-center flex p-6 rounded-md border border-blue-100 shadow-sm">
                <div class="text-[10px] text-blue-600 mb-4 uppercase font-bold tracking-wider">Total</div>
                <div class="leading-[1] text-blue-700 font-bold text-xl ml-auto">
                    {formatMo(ventasState.form.TotalAmount)}
                </div>
            </div>
          </div>
        </div>
          <button class="bx-blue"
            onclick={handlePostSaleOrder}
            title="Guardar venta"
          >
          	Generar
            <i class="icon-floppy"></i>
          </button>
        </div>
        <div class="flex w-full px-12 py-6">
	        <SearchSelect css="mr-16"
	          label="" save="LastPaymentCajaID"
	          keyId="ID"
	          keyName="Nombre" saveOn={ventasState.form}
	          options={cajas?.Cajas||[]}
	          placeholder="CAJA"
	        />
        	<CheckboxOptions type="multiple"
       			options={[ { id: 2, name: "Pagado" }, { id: 3, name: "Recibido" } ]}
         		keyId="id" keyName="name" save="ActionsIncluded"
         		saveOn={ventasState.form}
         	/>
	        <SearchSelect useStyle={1}
	           label=""
	           keyId="ID" css="ml-auto max-w-168 text-sm"
	           keyName="Nombre"
	           options={clientModeOptions}
	           selected={clientModeSelected}
	           onChange={handleClientModeChange}
	           placeholder="SIN CLIENTE"
	         />
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
                {#if item.skus && item.skus.size > 0}
                  <div class="flex flex-wrap gap-4 mt-2">
                    {#each item.skus.entries() as [sku, qty]}
                      <span
                        class="text-[10px] bg-white border border-gray-200 px-6 rounded text-gray-600"
                      >
                        {sku} <span class="font-bold text-gray-800">x{qty}</span>
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
                  class="text-red-400 hover:text-red-600 p-4 opacity-0 group-hover:opacity-100 transition-opacity"
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
