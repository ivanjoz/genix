<script lang="ts">
    import Page from "$components/Page.svelte";
    import SearchSelect from "$components/SearchSelect.svelte";
    import type { ITableColumn } from "$components/VTable";
    import VTable from "$components/VTable/vTable.svelte";
    import { throttle } from "$lib/helpers";
    import { Loading } from "notiflix";
    import { ProductosService } from "../productos/productos.svelte";
    import { AlmacenesService } from "../sedes-almacenes/sedes-almacenes.svelte";
    import { getProductosStock, type IProductoStock } from "./productos-stock.svelte";
    import { Core } from "$core/store.svelte";
    import Layer from "$components/Layer.svelte";
    import Input from "$components/Input.svelte";

  const almacenes = new AlmacenesService()
  const productos = new ProductosService()

  let filters = $state({ almacenID: 0 })
  let filterText = $state("")
  let almacenStock = $state([] as IProductoStock[])
  let form = $state({} as IProductoStock)

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
    { header: "Stock",
      getValue: e => e.Cantidad,
      onEditChange: (e, value) => {
        e._hasUpdated = true
        e._cantidadPrev = e._cantidadPrev || e.Cantidad || -1
        e.Cantidad = parseInt(value as string||"0")
      },
      render: e => {
        if(e._cantidadPrev && e._cantidadPrev !== e.Cantidad){
          return {
            css: "flex items-center",
            children: [
              { text: String(e._cantidadPrev > 0 ?  e._cantidadPrev : 0) },
              { text: "→", css: "ml-2 mr-2"  },
              { text: e.Cantidad, css: "c-red"  }
            ]
          }          
        } else {
          return { text: e.Cantidad || ""  }
        }
      }
    },
    { header: "Costo Un.",
      onEditChange: (e, value) => {
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
    almacenStock = result
  }

</script>

<Page sideLayerSize={640} title="productos-stock">
  <div class="flex items-center mb-8">
    <SearchSelect options={almacenes?.Almacenes||[]} keyId="ID" keyName="Nombre" 
      saveOn={filters} save="almacenID" placeholder="ALMACÉN ::"
      css="w-270"
      onChange={() => {
        onChangeAlmacen()
      }}
    />
    {#if !filters.almacenID}
      <div class="ml-12 c-red"><i class="icon-attention"></i>Debe seleccionar un almacén.</div>
    {:else}
      <div class="i-search w-full md:w-200 md:ml-12 col-span-5">
        <div><i class="icon-search"></i></div>
        <input type="text" onkeyup={ev => {
          const value = String((ev.target as any).value||"")
          throttle(() => { filterText = value },150)
        }}>
      </div>
    {/if}
    <div class="ml-auto">
      <button class="bx-green" onclick={() => {
        Core.openSideLayer(1)
      }}>
        <i class="icon-plus"></i>Agregar
      </button>
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
  <Layer id={1} type="side" css="p-16" title="Agregar Stock" titleCss="h2">
    <div class="grid-cols-24 mt-6">
      <SearchSelect label="Producto" css="col-span-24"
        saveOn={form} save="ProductoID" options={productos.productos||[]}
        keyName="Nombre" keyId="ID"
      />
    </div>
  </Layer>
</Page>