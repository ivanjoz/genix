<script lang="ts">
  import Layer from "../../../components/Layer.svelte";
  import Page from "../../../components/Page.svelte";
  import type { ITableColumn } from "../../../components/VTable";
  import VTable from "../../../components/VTable/vTable.svelte";
    import { throttle } from "../../../core/helpers";
  import { Core } from "../../../core/store.svelte";
  import { ProductosService, type IProducto } from "./productos.svelte";

  let filterText = $state("")
  const productos = new ProductosService()

  let productoColumns: ITableColumn<IProducto>[] = [
    { header: "ID", css: "c-blue text-center", headerCss: "w-48",
      getValue: e => e.ID
    },
    { header: "Producto", highlight: true,
      getValue: e => e.Nombre
    },
  ]

</script>

<Page sideLayerSize={770}>
  <div class="flex items-center mb-8">
    
    <div>hola</div>
    <div class="i-search w-200">
      <div><i class="icon-search"></i></div>
      <input type="text" onkeyup={ev => {
        const value = String(ev.target?.value||"")
        throttle(() => { filterText = value },150)
      }}>
    </div>

    <button class="bx-green ml-auto" onclick={ev => {
      ev.stopPropagation()
      Core.showSideLayer = 1
    }}>
      <i class="icon-plus"></i>Nuevo
    </button>
  </div>

  <Layer type="content">
    <VTable columns={productoColumns}
      data={productos.productos}
      filterText={filterText}
      getFilterContent={e => {
        return e.Nombre
      }}
      onRowClick={() => {
        Core.showSideLayer = 1
      }}
    />
  </Layer>
  <Layer css="p-8" title="nueva capa" type="side">
    <div><h1>hola mundo 2</h1></div>
  </Layer>
</Page>