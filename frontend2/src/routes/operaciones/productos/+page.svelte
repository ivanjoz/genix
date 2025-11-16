<script lang="ts">
    import CheckboxOptions from "$components/CheckboxOptions.svelte";
    import Input from "$components/Input.svelte";
    import { openModal } from "$core/store.svelte";
    import { Loading, Notify } from "$lib/helpers";
    import ImageUploader from "../../../components/ImageUploader.svelte";
    import Layer from "../../../components/Layer.svelte";
    import OptionsStrip from "../../../components/micro/OptionsStrip.svelte";
    import Page from "../../../components/Page.svelte";
    import SearchCard from "../../../components/SearchCard.svelte";
    import SearchSelect from "../../../components/SearchSelect.svelte";
    import type { ITableColumn } from "../../../components/VTable";
    import VTable from "../../../components/VTable/vTable.svelte";
    import { throttle } from "../../../core/helpers";
    import { Core } from "../../../core/store.svelte";
    import { formatN } from "../../../shared/main";
    import CategoriasMarcas from "./CategoriasMarcas.svelte";
    import { ListasCompartidasService, postProducto, ProductosService, type IProducto, type IProductoImage } from "./productos.svelte";

  let filterText = $state("")
  const productos = new ProductosService()
  const listas = new ListasCompartidasService([1,2])

  let view = $state(1)
  let layerView = $state(1)
  let productoForm = $state({} as IProducto)
  // svelte-ignore non_reactive_update
  let CategoriasLayer: CategoriasMarcas | null = null
  // svelte-ignore non_reactive_update
  let MarcasLayer: CategoriasMarcas | null = null

  let productoColumns: ITableColumn<IProducto>[] = [
    { header: "ID", css: "c-blue text-center", headerCss: "w-48",
      getValue: e => e.ID
    },
    { header: "Producto", highlight: true,
      getValue: e => e.Nombre
    },
    { header: "Categorías", highlight: true,
      getValue: e => {
        const nombres = []
        for(const id of e.CategoriasIDs){
          const nombre = listas.get(id)?.Nombre || `Categoría-${id}`
          nombres.push(nombre)
        }
        return nombres.join(", ")
      }
    },
    { header: "Precio", css: "text-right",
      getValue: e => formatN(e.Precio / 100,2)
    },
    { header: "Descuento", css: "text-right",
      getValue: e => e.Descuento ? String(e.Descuento) + "%" : ""
    },
    { header: "Precio Final", css: "text-right",
      getValue: e => formatN(e.PrecioFinal / 100,2)
    },
    { header: "Sub Unidades", css: "text-right",
      getValue: e => {
        if(!e.SbnUnidad) return ""
        return `${e.SbnCantidad} x ${e.SbnUnidad}`
      }
    },
  ]

  const categorias = $derived.by(() => {
    return listas.ListaRecordsMap.get(1) || []
  })

  const onSave = async () => {
    if((productoForm.Nombre?.length||0) < 4){
      Notify.failure("El nombre debe tener al menos 4 caracteres.")
      return
    }

    console.log("productor a enviar:",$state.snapshot(productoForm))
    // return

    Loading.standard("Guardando producto...")
    try {
      var result = await postProducto([productoForm])
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }
    Loading.remove()

    const producto = productos.productos.find(x => x.ID === productoForm.ID)
    if(producto){
      Object.assign(producto, productoForm)
    }
    Core.showSideLayer = 0
  }

  $effect(() => {
    console.log("productor form:", $state.snapshot(productoForm))
  })

</script>

<Page sideLayerSize={780}>
  <div class="flex items-center mb-8">
    <OptionsStrip selected={view}
      options={[[1,"Productos"],[2,"Categorías"],[3,"Marcas"]]} 
      onSelect={e => {
        Core.showSideLayer = 0
        productoForm = { ID: 0 } as IProducto
        view = e[0] as number
      }}
    />
    <div class="i-search w-200 ml-12">
      <div><i class="icon-search"></i></div>
      <input type="text" onkeyup={ev => {
        const value = String((ev.target as any).value||"")
        throttle(() => { filterText = value },150)
      }}>
    </div>

    <button class="bx-green ml-auto" onclick={ev => {
      ev.stopPropagation()
      if(view === 2){
        CategoriasLayer?.newRecord()
      } else if(view === 2) {
        MarcasLayer?.newRecord()
      } else {
        Core.showSideLayer = 1
      }
    }}>
      <i class="icon-plus"></i>Nuevo
    </button>
  </div>

  {#if view === 1}
    <Layer type="content">
      <VTable columns={productoColumns}
        data={productos.productos}
        filterText={filterText}
        selected={productoForm?.ID}
        isSelected={(e,id) => e.ID === id}
        getFilterContent={e => {
          return e.Nombre
        }}
        onRowClick={e => {
          productoForm = {...e}
          productoForm.CategoriasIDs = [...(e.CategoriasIDs||[])]
          productoForm.Propiedades = [...(e.Propiedades||[])]
          Core.showSideLayer = 1
        }}
      />
    </Layer>
  {/if}
  <Layer css="px-14 py-10" title={productoForm?.Nombre || ""} type="side"
    titleCss="h2 mb-6"
    options={[[1,"Información"],[2,"Ficha"],[3,"Parámetros"],[4,"Fotos"]]}
    selected={layerView}
    onSelect={e => layerView = e[0]}
    onSave={() => {
      onSave()
    }}
  >
    {#if layerView === 1}
      <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-16">
        <Input label="Nombre" saveOn={productoForm} css="col-span-24"
          required={true} save="Nombre"
        />
        <div class="col-span-9 row-span-4">
          <ImageUploader saveAPI="producto-image"
            clearOnUpload={true} types={["avif","webp"]}
            src={productoForm.Image ? `img-productos/${productoForm.Image.n}-x2` : ""}
            cardCss="w-full h-180  p-4"
            setDataToSend={e => {
              e.ProductoID = productoForm.ID
            }}
            onUploaded={(imagePath, description) => {
              if(imagePath.includes("/")){ imagePath = imagePath.split("/")[1] }
              productoForm.Image = { n: imagePath, d: description } as IProductoImage
              productoForm.Images = productoForm.Images || []
              productoForm.Images.unshift(productoForm.Image)
            }}
          />
        </div>
        <Input label="Precio Base" saveOn={productoForm} css="col-span-5"
          save="Precio" type="number" baseDecimals={2}
        />
        <Input label="Descuento" saveOn={productoForm} css="col-span-5"
          save="Descuento" postValue="%" type="number"
        />
        <Input label="Precio Final" saveOn={productoForm} css="col-span-5"
          save="PrecioFinal" type="number" baseDecimals={2}
        />
        <SearchSelect label="Moneda" saveOn={productoForm} css="col-span-5"
          save="MonedaID" keyId="i" keyName="v"
          options={[
            {i:1, v:"PEN (S/.)"},{i:2, v:"USD ($)"}
          ]}
        />
        <Input label="Peso" saveOn={productoForm} css="col-span-5"
          save="Peso" type="number"
        />
        <SearchSelect label="Unidad" saveOn={productoForm} css="col-span-5"
          save="UnidadID" keyId="i" keyName="v"
          options={[
            {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
          ]}
        />
        <Input label="Volumen" saveOn={productoForm} css="col-span-5"
          save="Volumen" type="number"
        />
        <SearchSelect label="Marca" saveOn={productoForm} css="col-span-10 mb-2"
          save="MarcaID" keyId="i" keyName="v"
          options={[
            {i:1, v:"PEN (S/.)"},{i:2, v:"g"},{i:3, v:"Libras"}
          ]}
        />
        <div class="col-span-5">
          <CheckboxOptions save="Params" saveOn={productoForm} type="multiple"
            options={[{i:1, v:"SKU Individual"}]} keyId="i" keyName="v"
          />
        </div>
        <Input saveOn={productoForm} save="Descripcion"
          css="col-span-24 mb-4" label="Descripción Corta" 
        />
        <SearchCard css="col-span-24 flex items-start" label="CATEGORÍAS ::"
          options={categorias} keyId="ID" keyName="Nombre"
          cardCss="grow" inputCss="w-180" bind:saveOn={productoForm}
          save="CategoriasIDs"
        />
        <div class="ff-bold h3 col-span-24 ml-8">
          Sub-Unidades
        </div>
        <Input saveOn={productoForm} save="SbnUnidad" 
          css="col-span-6" label="Nombre"
        />
        <Input saveOn={productoForm} save="SbnPrecio" baseDecimals={2}
          css="col-span-6" label="Precio Base" type="number"
        />
        <Input saveOn={productoForm} save="SbnDescuento" postValue="%"
          css="col-span-6" label="Descuento" type="number"
        />
        <Input saveOn={productoForm} save="SbnPreciFinal" baseDecimals={2}
          css="col-span-6" label="Precio Final" type="number"
        />
        <Input saveOn={productoForm} save="SbnCantidad" 
          css="col-span-6" label="Cantidad" type="number"
        />
      </div>
    {/if}
  </Layer>
  {#if view === 2}
    <CategoriasMarcas listas={listas} origin={1} bind:this={CategoriasLayer}/>
  {/if}
  {#if view === 3}
    <CategoriasMarcas listas={listas} origin={2} bind:this={MarcasLayer}/>
  {/if}
</Page>