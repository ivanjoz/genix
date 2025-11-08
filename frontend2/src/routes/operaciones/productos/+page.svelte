<script lang="ts">
  import Input from "../../../components/Input.svelte";
import Layer from "../../../components/Layer.svelte";
    import OptionsStrip from "../../../components/micro/OptionsStrip.svelte";
  import Page from "../../../components/Page.svelte";
    import SearchSelect from "../../../components/SearchSelect.svelte";
  import type { ITableColumn } from "../../../components/VTable";
  import VTable from "../../../components/VTable/vTable.svelte";
    import { throttle } from "../../../core/helpers";
  import { Core } from "../../../core/store.svelte";
  import { ProductosService, type IProducto } from "./productos.svelte";

  let filterText = $state("")
  const productos = new ProductosService()
  let productoSelected: IProducto | null = $state(null)
  let view = $state(1)
  let layerView = $state(1)
  let productoForm = $state({} as IProducto)

  let productoColumns: ITableColumn<IProducto>[] = [
    { header: "ID", css: "c-blue text-center", headerCss: "w-48",
      getValue: e => e.ID
    },
    { header: "Producto", highlight: true,
      getValue: e => e.Nombre
    },
  ]

</script>

<Page sideLayerSize={820}>
  <div class="flex items-center mb-8">
    <OptionsStrip selected={view}
      options={[[1,"Productos"],[2,"Categorías"],[3,"Marcas"]]} 
      onSelect={e => {
        Core.showSideLayer = 0  
        productoSelected = null
        view = e[0] as number
      }}
    />
    <div class="i-search w-200 ml-12">
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

  {#if view === 1}
    <Layer type="content">
      <VTable columns={productoColumns}
        data={productos.productos}
        filterText={filterText}
        selected={productoSelected?.ID}
        isSelected={(e,id) => e.ID === id}
        getFilterContent={e => {
          return e.Nombre
        }}
        onRowClick={e => {
          productoSelected = e
          Core.showSideLayer = 1
        }}
      />
    </Layer>
  {/if}
  <Layer css="p-12" title={productoSelected?.Nombre || ""} type="side"
    titleCss="fs18"
    options={[[1,"Información"],[2,"Ficha"],[3,"Fotos"]]}
    selected={layerView}
    onSelect={e => layerView = e[0]}
  >
    {#if layerView === 1}
      <div class="grid grid-cols-24 gap-10 mt-12">
        <Input label="Nombre" saveOn={productoForm} css="col-span-24 mb-2"
          required={true} save="Nombre"
        />
        <div class="col-span-6 row-span-3">
          <h1>hola?</h1>
        </div>
        <Input label="Precio Base" saveOn={productoForm} css="col-span-5 mb-2"
          save="Precio" type="number"
        />
        <Input label="Desc." saveOn={productoForm} css="col-span-4 mb-2"
          save="Descuento"
        />
        <Input label="Precio Final" saveOn={productoForm} css="col-span-5 mb-2"
          save="PrecioFinal"
        />
        <SearchSelect label="Moneda" saveOn={productoForm} css="col-span-4 mb-2"
          save="Moneda" keys="i.v"
          options={[
            {i:1, v:"PEN (S/.)"},{i:2, v:"g"},{i:3, v:"Libras"}
          ]}
        />
        <Input label="Peso" saveOn={productoForm} css="col-span-5 mb-2"
          save="Peso"
        />
        <Input label="Unidad" saveOn={productoForm} css="col-span-4 mb-2"
          save="Unidad"
        />
        <Input label="Volumen" saveOn={productoForm} css="col-span-5 mb-2"
          save="Volumen"
        />
        <SearchSelect label="Marca" saveOn={productoForm} css="col-span-9 mb-2"
          save="Moneda" keys="i.v"
          options={[
            {i:1, v:"PEN (S/.)"},{i:2, v:"g"},{i:3, v:"Libras"}
          ]}
        />
      </div>
    {/if}
  </Layer>
</Page>