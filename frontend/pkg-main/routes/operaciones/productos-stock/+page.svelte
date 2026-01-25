<script lang="ts">
import Page from '$ui/Page.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import type { ITableColumn } from '$components/vTable/types';
import VTable from '$components/vTable/VTable.svelte';
import { Notify, throttle } from '$core/helpers';
    import pkg from 'notiflix'
const { Loading } = pkg;
    import { ProductosService } from "../productos/productos.svelte";
    import { AlmacenesService } from "../sedes-almacenes/sedes-almacenes.svelte";
    import { getProductosStock, postProductosStock, type IProductoStock } from "./productos-stock.svelte";
import { Core } from '$core/store.svelte';
import Layer from '$components/Layer.svelte';
import Input from '$components/Input.svelte';
import Checkbox from '$components/Checkbox.svelte';
    import { untrack } from "svelte";

  const almacenes = new AlmacenesService()
  const productos = new ProductosService()

  let filters = $state({ almacenID: 0, showTodosProductos: false })
  let filterText = $state("")
  let almacenStock = $state([] as IProductoStock[])
  let almacenStockGetted = [] as IProductoStock[]
  let form = $state({} as IProductoStock)

  let formProducto = $derived(productos?.productosMap?.get(form.ProductoID||0))

  $effect(() => {
    if(!filters.almacenID){ return }
  })

  let columns: ITableColumn<IProductoStock>[] = [
    { header: "Producto", highlight: true,
      getValue: e => {
        const producto = productos.productosMap.get(e.ProductoID)?.Nombre
        return producto || ""
      }
    },
    { header: "Lote",
      getValue: e => e.Lote || ""
    },
    { header: "SKU",
      getValue: e => e.SKU || ""
    },
    { header: "Presentación",
      getValue: e => {
        if(!e.PresentacionID){ return "" }
        const producto = productos.productosMap.get(e.ProductoID)
        const pr = producto?.Presentaciones?.find(x => x.id === e.PresentacionID)
        return pr?.nm || ""
      }
    },
    { header: "Stock", css: "justify-end", inputCss: "text-right pr-6",
      getValue: e => e.Cantidad,
      onCellEdit: (e, value) => {
        agregarStock(e, parseInt(value as string||"0"))
      },
      render: e => {
        if(e._cantidadPrev && e._cantidadPrev !== e.Cantidad){
          return {
            css: "flex items-center",
            children: [
              { text: String(e._cantidadPrev > 0 ?  e._cantidadPrev : 0) },
              { text: "→", css: "ml-2 mr-2"  },
              { text: e.Cantidad, css: "text-red-500"  }
            ]
          }
        } else {
          return { text: e.Cantidad || ""  }
        }
      }
    },
    { header: "Costo Un.",
      onCellEdit: (e, value) => {
        e._hasUpdated = true
        e.CostoUn = parseInt(value as string||"0")
      },
    },
  ]

  const onChangeAlmacen = async () => {
    if(!filters.almacenID){ return }
    Loading.standard()
    try {
      var result = await getProductosStock(filters.almacenID)
    } catch (error) {
      Loading.remove()
      return
    }
    Loading.remove()
    almacenStock = result || []
    almacenStockGetted = result || []
  }

  const guardarRegistros = async () => {
    const recordsForUpdate = almacenStock.filter(e => e._hasUpdated)
    if(recordsForUpdate.length === 0){
      Notify.failure("No hay registros a actualizar."); return
    }

    Loading.standard("Enviando registros...")
    let result
    try {
      result = await postProductosStock(recordsForUpdate)
    } catch (error) {
      Loading.remove(); return
    }

    for(const e of almacenStock){
      e._cantidadPrev = 0
    }
    console.log("resultado obtenido::", result)
    // setProductosStock([...productosStock()])
    Loading.remove()
  }

  const fillAllProductos = () => {
    console.log("almacenStockGetted", $state.snapshot(almacenStockGetted))

    const productosStockMap = new Map(almacenStockGetted?.map(x => [x.ID, x]))
    for(const pr of productos.productos){
      const presentacionesIDs = pr.Presentaciones?.length > 0 ? pr.Presentaciones.map(x => x.id) : [0]
      for(const presentacionID of presentacionesIDs){
        const stockID = [filters.almacenID, pr.ID, presentacionID || "", "", ""].join("_")
        if(productosStockMap.has(stockID)){
          const e = productosStockMap.get(stockID) as IProductoStock
          e.PresentacionID = presentacionID
          continue
        }

        const stock = {
          ID: stockID,
          AlmacenID: filters.almacenID,
          ProductoID: pr.ID,
          PresentacionID: presentacionID,
          Cantidad: 0
        } as IProductoStock
        productosStockMap.set(stockID, stock)
      }
    }
    almacenStock = [...productosStockMap.values()]
  }

  const agregarStock = (e: IProductoStock, cantidad: number) => {
    if(!e.ID){
      e.ID = [
        e.AlmacenID, e.ProductoID, e.PresentacionID||0, e.SKU||"", e.Lote||""
      ].join("_")
    }

    const base = almacenStock.find(x => x.ID === e.ID) || almacenStockGetted.find(x => x.ID === e.ID)

    if(base){
      base._hasUpdated = true
      base._cantidadPrev = base._cantidadPrev || base.Cantidad || -1
      base.Cantidad = cantidad
    } else {
      e._cantidadPrev = -1
      e._hasUpdated = true
      almacenStockGetted.unshift(e)
      almacenStock.unshift(e)
      almacenStock = [...almacenStock]
    }
  }

  $effect(() => {
    if(filters.showTodosProductos){
      untrack(() => { fillAllProductos() })
    } else {
      untrack(() => { almacenStock = almacenStockGetted })
    }
  })

</script>

<Page sideLayerSize={640} title="productos-stock">
  <div class="flex items-center mb-8">
    <SearchSelect options={almacenes?.Almacenes||[]} keyId="ID" keyName="Nombre"
      bind:saveOn={filters} save="almacenID" placeholder="ALMACÉN ::"
      css="w-270"
      onChange={() => {
        onChangeAlmacen()
      }}
    />
    {#if !filters.almacenID}
      <div class="ml-12 text-red-500"><i class="icon-attention"></i>Debe seleccionar un almacén.</div>
    {:else}
      <div class="i-search w-full md:w-200 md:ml-12 col-span-5">
        <div><i class="icon-search"></i></div>
        <input type="text" onkeyup={ev => {
          const value = String((ev.target as any).value||"")
          throttle(() => { filterText = value },150)
        }}>
      </div>
      <Checkbox label="Todos los Productos" bind:saveOn={filters} save="showTodosProductos"
        css="ml-16" />
    {/if}
    <div class="ml-auto">
      {#if filters.almacenID > 0}
        <button class="bx-blue mr-8" onclick={() => {
          guardarRegistros()
        }}>
          <i class="icon-floppy"></i>Guardar
        </button>
        <button class="bx-green" aria-label="agregar" onclick={() => {
          form = { AlmacenID: filters.almacenID } as IProductoStock
          Core.openSideLayer(1)
        }}>
          <i class="icon-plus"></i>
        </button>
      {/if}
    </div>
  </div>
  <VTable columns={columns} data={almacenStock}
    filterText={filterText}
    useFilterCache={true}
    getFilterContent={e => {
      const producto = productos.productosMap.get(e.ProductoID)
      return [producto?.Nombre, e.SKU, e.Lote].filter(x => x).join(" ").toLowerCase()
    }}
  >

  </VTable>
  <Layer id={1} type="bottom" css="p-12 min-h-360" title="Agregar Stock" titleCss="h2"
    saveButtonName="Agregar" saveButtonIcon="icon-ok" contentOverflow={true}
    onSave={() => {
      if(!form.ProductoID || !form.Cantidad){
        Notify.failure("Debe seleccionar un producto y una cantidad.")
      }
      agregarStock(form, form.Cantidad)
      // almacenStock = [...almacenStock]
      Core.hideSideLayer()
    }}
  >
    <div class="grid grid-cols-24 gap-10 mt-6 p-4">
      <SearchSelect label="Producto" css="col-span-24" required={true}
        bind:saveOn={form} save="ProductoID" options={productos.productos||[]}
        keyName="Nombre" keyId="ID"
      />
      {#if (formProducto?.Presentaciones?.filter(x => x.ss)||[]).length > 0}
        <SearchSelect label="Presentación" css="col-span-24"
          bind:saveOn={form} save="PresentacionID" options={formProducto?.Presentaciones||[]}
          keyName="nm" keyId="id"
        />
      {/if}
      <Input label="SKU" css="col-span-16"
        bind:saveOn={form} save="SKU"
      />
      <Input label="Cantidad" required={true} css="col-span-8"
        bind:saveOn={form} save="Cantidad" type="number"
      />
      <Input label="Lote" css="col-span-16"
        bind:saveOn={form} save="Lote"
      />
      <Input label="Costo x Unidad" css="col-span-8"
        bind:saveOn={form} save="CostoUn" type="number"
      />
    </div>
  </Layer>
</Page>
