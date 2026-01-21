<script lang="ts">
  import Input from "$components/Input.svelte";
    import LayerStatic from "$components/micro/LayerStatic.svelte";
  import Page from "$components/Page.svelte";
  import SearchSelect from "$components/SearchSelect.svelte";
  import { Loading, include } from "$lib/helpers";
  import { formatN } from "$shared/main";
  import type { IProductoStock } from "../productos-stock/productos-stock.svelte";
  import { getProductosStock } from "../productos-stock/productos-stock.svelte";
  import { ProductosService } from "../productos/productos.svelte";
  import type { IAlmacen } from "../sedes-almacenes/sedes-almacenes.svelte";
  import { AlmacenesService } from "../sedes-almacenes/sedes-almacenes.svelte";
  import { ListasCompartidasService } from "../productos/productos.svelte"; 
  import ProductoVentaCard from "./ProductoVentaCard.svelte";
  import type { ProductoVenta } from "./ventas.svelte";
  import { VentasState } from "./ventas.svelte";

  // Helpers
  const formatMo = (n: number) => formatN(n, 2);

  // Services
  const almacenesService = new AlmacenesService();
  const productosService = new ProductosService();
  const listasService = new ListasCompartidasService([2]); // 2: Marcas

  // State
  const ventasState = new VentasState();

  let almacenSelected = $state(-1);
  let productoSelected = $state(-1);
  let searchInput: HTMLInputElement;

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
    // Auto-select first almacen
    if (almacenSelected === -1 && almacenesService.Almacenes.length > 0) {
      almacenSelected = almacenesService.Almacenes[0].ID;
      loadStock(almacenSelected);
    }
  });

  async function loadStock(almacenID: number) {
    Loading.standard("Cargando stock...");
    productosStock = await getProductosStock(almacenID);
    parseProductos();
    Loading.remove();
  }

  function parseProductos() {
    if (!productosService.productos.length || !productosStock.length) return;

    const productoToStockMap = new Map<number, IProductoStock[]>();
    for (const s of productosStock) {
      const list = productoToStockMap.get(s.ProductoID) || [];
      list.push(s);
      productoToStockMap.set(s.ProductoID, list);
    }

    const newProductos: ProductoVenta[] = [];

    for (const producto of productosService.productos) {
      const stocks = productoToStockMap.get(producto.ID) || [];
      if (stocks.length === 0) continue;

      const brandName = listasService.RecordsMap.get(producto.MarcaID)?.Nombre || "";
      
      const base: ProductoVenta = {
        producto: producto,
        cant: 0,
        key: `P${producto.ID}`,
        searchText: (producto.Nombre + " " + brandName).toLowerCase(),
      };

      const skusStock: IProductoStock[] = [];
      const mainStock: IProductoStock[] = [];

      for (const e of stocks) {
        e.SKU ? skusStock.push(e) : mainStock.push(e);
      }

      // SKU entries
      if (skusStock.length > 0) {
        const clone = { ...base, skus: skusStock, key: `K${producto.ID}` };
        for (const e of skusStock) clone.cant += e.Cantidad;
        newProductos.push(clone);
      }

      // Main entries & Sub-units
      if (mainStock.length > 0) {
        let clone: ProductoVenta | undefined;
        if ((producto.SbnCantidad || 0) > 1) {
          clone = {
            ...base,
            key: `S${producto.ID}`,
            isSubUnidad: true,
            cant: producto.SbnCantidad!,
          };
          // Recalculate generic quantity for sub-units?
          // Legacy: `clone.cant = producto.SbnCantidad` (Logic seems to be per unit or just flagging?)
          // Legacy logic sets cant to `SbnCantidad`. Let's stick to it.
        }

        for (const e of mainStock) base.cant += e.Cantidad;
        newProductos.push(base);

        if (clone) newProductos.push(clone);
      }
    }

    productosParsedAll = newProductos;
    filterProductos(ventasState.filterText);
  }

  let filterTimeout: any;
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
    } else if (ev.key === "Escape") {
      productoSelected = -1;
      searchInput?.focus();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<Page title="Ventas" sideLayerSize={400}
  options={[{ id: 1, name: "Ventas" }, { id: 2, name: "Configuración" }]}
>
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
          <div class="flex gap-6 mb-6">
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
                  onadd={(n, sku) => ventasState.addProducto(item, n, sku)}
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
        <h3 class="font-bold text-gray-800">DETALLE DE VENTA</h3>
        <button class="bx-blue"
          title="Guardar venta"
        >
          Guardar
          <i class="icon-floppy"></i>
        </button>
      </div>

      <!-- Summary -->
      <div
        class="p-16 grid grid-cols-2 gap-16 border-b border-gray-100 bg-white"
      >
        <div class="col-span-2">
          <Input
            label="Cliente"
            css="w-full"
            saveOn={ventasState.form}
            save="clienteID"
          />
        </div>

        <div class="bg-gray-50 p-8 rounded border border-gray-100">
          <div class="text-xs text-gray-500 mb-4">Sub Total</div>
          <div class="font-mono text-gray-800 font-medium">
            {formatMo(ventasState.form.subtotal)}
          </div>
        </div>

        <div class="bg-blue-50 p-8 rounded border border-blue-100">
          <div class="text-xs text-blue-600 mb-4 font-bold">Total</div>
          <div class="font-mono text-blue-700 font-bold text-lg">
            {formatMo(ventasState.form.total)}
          </div>
        </div>
      </div>

      <!-- List -->
      <div class="flex-1 overflow-y-auto p-8 space-y-8">
        {#each ventasState.ventaProductos as item (item.key)}
          <div
            class="flex items-start gap-8 p-8 rounded-lg bg-gray-50 hover:bg-white hover:shadow-sm border border-transparent hover:border-gray-200 transition-all group"
          >
            <div class="text-xs font-bold text-gray-400 pt-4 w-16">
              {ventasState.ventaProductos.indexOf(item) + 1}
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-gray-800 truncate">
                {item.producto?.Nombre}
                {#if item.isSubUnidad}
                  <span class="text-purple-600 text-xs ml-4"
                    >({item.producto?.SbnUnidad})</span
                  >
                {/if}
              </div>
              {#if item.skus && item.skus.size > 0}
                <div class="flex flex-wrap gap-4 mt-4">
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

            <div class="text-right">
              <div class="font-mono text-sm font-bold text-gray-700">
                {item.cantidad}
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
</Page>
