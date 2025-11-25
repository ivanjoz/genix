<script lang="ts">
    import CheckboxOptions from "$components/CheckboxOptions.svelte";
    import Input from "$components/Input.svelte";
    import HTMLEditor from "$components/micro/HTMLEditor.svelte";
    import { ConfirmWarn, Loading, Notify } from "$lib/helpers";
    import { POST } from "$lib/http";
    import ImageUploader from "../../../components/ImageUploader.svelte";
    import Layer from "../../../components/Layer.svelte";
    import OptionsStrip from "../../../components/micro/OptionsStrip.svelte";
    import Page from "$components/Page.svelte";
    import SearchCard from "../../../components/SearchCard.svelte";
    import SearchSelect from "../../../components/SearchSelect.svelte";
    import type { ITableColumn } from "../../../components/VTable";
    import VTable from "../../../components/VTable/vTable.svelte";
    import { throttle } from "../../../core/helpers";
    import { Core } from "../../../core/store.svelte";
    import { formatN } from "../../../shared/main";
    import Atributos from "./Atributos.svelte";
    import CategoriasMarcas from "./CategoriasMarcas.svelte";
    import { ListasCompartidasService, postProducto, productoAtributos, ProductosService, type IProducto, type IProductoImage } from "./productos.svelte";

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
    Core.setSideLayer(0)
  }

  $effect(() => {
    console.log("productor form:", $state.snapshot(productoForm))
  })

  const deleteProductoImage = async (ImageToDelete: string) => {
    Loading.standard("Eliminando Imagen...")
    try {
      await POST({
        data: { ProductoID: productoForm.ID, ImageToDelete },
        route: "producto-image",
        refreshRoutes: ["productos"],
      }) 
    } catch (error) {
      Notify.failure(`Error al eliminar imagen: ${error}`)
      Loading.remove()
      return
    }    
    Loading.remove()
    productoForm.Images = (productoForm.Images||[]).filter(e => e.n !== ImageToDelete)
  }

</script>

<Page sideLayerSize={780} title="Productos">
  <div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
    <OptionsStrip selected={view} css="col-span-12 mb-6 md:mb-0"
      options={[[1,"Productos"],[2,"Categorías"],[3,"Marcas"]]} 
      useMobileGrid={true}
      onSelect={e => {
        Core.setSideLayer(0)
        productoForm = { ID: 0 } as IProducto
        view = e[0] as number
      }}
    />
    <div class="i-search w-full md:w-200 md:ml-12 col-span-5">
      <div><i class="icon-search"></i></div>
      <input type="text" onkeyup={ev => {
        const value = String((ev.target as any).value||"")
        throttle(() => { filterText = value },150)
      }}>
    </div>

    <button class="bx-green ml-auto col-span-7" onclick={ev => {
      ev.stopPropagation()
      if(view === 2){
        CategoriasLayer?.newRecord()
      } else if(view === 3) {
        MarcasLayer?.newRecord()
      } else {
        Core.setSideLayer(1)
      }
    }}>
      <i class="icon-plus"></i><span class="hidden md:block">Nuevo</span>
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
          Core.setSideLayer(1)
        }}
      />
    </Layer>
  {/if}
  <Layer css="px-8 py-8 md:px-14 md:py-10" title={productoForm?.Nombre || ""} type="side"
    titleCss="h2 mb-6" contentCss="px-0 md:px-0" id={1}
    options={[
      [1,"Información",["Informa-","ción"]],
      [2,"Ficha"],
      [3,"Presentaciones",["Presenta-","ciones"]],
      [4,"Fotos"]
    ]}
    selected={layerView}
    onSelect={e => layerView = e[0]}
    onClose={() => { productoForm = {} as IProducto }}
    onSave={() => { onSave() }}
  >
    {#if layerView === 1}
      <div class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16">
        <Input label="Nombre" saveOn={productoForm} css="col-span-24"
          required={true} save="Nombre"
        />
        <div class="col-span-24 md:col-span-9 md:row-span-4">
          <ImageUploader saveAPI="producto-image"
            clearOnUpload={true} types={["avif","webp"]}
            folder="img-productos" size={2} src={productoForm.Image?.n}
            cardCss="w-full h-180 p-4"
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
        <Input label="Precio Base" saveOn={productoForm} 
          css="col-span-12 md:col-span-5"
          save="Precio" type="number" baseDecimals={2}
          dependencyValue={productoForm.Precio}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm))
            productoForm.PrecioFinal = Math.round((productoForm.Precio * (100 - (productoForm.Descuento||0))) / 100)
          }}
        />
        <Input label="Descuento" saveOn={productoForm} 
          css="col-span-12 md:col-span-5"
          save="Descuento" postValue="%" type="number"
        />
        <Input label="Precio Final" saveOn={productoForm} 
          css="col-span-12 md:col-span-5"
          dependencyValue={productoForm.PrecioFinal}
          save="PrecioFinal" type="number" baseDecimals={2}
          onChange={() => {
            console.log("productoForm", $state.snapshot(productoForm))
            productoForm.Precio = Math.round((productoForm.PrecioFinal / (100 - (productoForm.Descuento||0))) * 100)
          }}
        />
        <SearchSelect label="Moneda" saveOn={productoForm} 
          css="col-span-12 md:col-span-5"
          save="MonedaID" keyId="i" keyName="v"
          options={[
            {i:1, v:"PEN (S/.)"},{i:2, v:"USD ($)"}
          ]}
        />
        <Input label="Peso" saveOn={productoForm} css="col-span-12 md:col-span-5"
          save="Peso" type="number"
        />
        <SearchSelect label="Unidad" saveOn={productoForm} css="col-span-12 md:col-span-5"
          save="UnidadID" keyId="i" keyName="v"
          options={[
            {i:1, v:"Kg"},{i:2, v:"g"},{i:3, v:"Libras"}
          ]}
        />
        <Input label="Volumen" saveOn={productoForm} css="col-span-12 md:col-span-5"
          save="Volumen" type="number"
        />
        <SearchSelect label="Marca" saveOn={productoForm} css="col-span-12 md:col-span-10 mb-2"
          save="MarcaID" keyId="ID" keyName="Nombre"
          options={listas.ListaRecordsMap.get(2)||[]}
        />
        <div class="col-span-24 md:col-span-5">
          <CheckboxOptions save="Params" saveOn={productoForm} type="multiple"
            options={[{i:1, v:"SKU Individual"}]} keyId="i" keyName="v"
          />
        </div>
        <Input saveOn={productoForm} save="Descripcion"
          css="col-span-24 mb-4" label="Descripción Corta" 
        />
        <SearchCard css="col-span-24 flex items-start" label="CATEGORÍAS ::"
          options={categorias} keyId="ID" keyName="Nombre"
          cardCss="grow" inputCss="w-[35%] md:w-180" bind:saveOn={productoForm}
          save="CategoriasIDs" optionsCss="w-280"
        />
        <div class="ff-bold h3 col-span-24 ml-8">
          Sub-Unidades
        </div>
        <Input saveOn={productoForm} save="SbnUnidad" 
          css="col-span-12 md:col-span-6" label="Nombre"
        />
        <Input bind:saveOn={productoForm} save="SbnPrecio" baseDecimals={2}
          css="col-span-12 md:col-span-6" label="Precio Base" type="number"
          onChange={() => {
            console.log()
            productoForm.PrecioFinal = Math.round((productoForm.Precio * (100 - (productoForm.Descuento||0))) / 100)
            productoForm = {...productoForm}
          }}
        />
        <Input saveOn={productoForm} save="SbnDescuento" postValue="%"
          css="col-span-12 md:col-span-6" label="Descuento" type="number"
        />
        <Input saveOn={productoForm} save="SbnPreciFinal" baseDecimals={2}
          css="col-span-12 md:col-span-6" label="Precio Final" type="number"
        />
        <Input saveOn={productoForm} save="SbnCantidad" 
          css="col-span-12 md:col-span-6" label="Cantidad" type="number"
        />
      </div>
    {/if}
    {#if layerView === 2}
      <HTMLEditor saveOn={productoForm} save="ContentHTML" 
        css="mt-12"/>
    {/if}
    {#if layerView === 3}
      <div class="mt-16 w-full">
        <Atributos bind:producto={productoForm}></Atributos>
      </div>
    {/if}
    {#if layerView === 4}
      <div class="grid grid-cols-2 md:grid-cols-4 items-start gap-x-10 gap-y-10 mt-16">
        <ImageUploader saveAPI="producto-image"
          clearOnUpload={true} types={["avif","webp"]} folder="img-productos"
          cardCss="w-full h-170 p-4"
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
        {#each (productoForm.Images||[]) as image }
          <ImageUploader saveAPI="producto-image" size={2}
            clearOnUpload={true} types={["avif","webp"]} folder="img-productos"
            cardCss="w-full h-170 p-4" src={image?.n}
            onDelete={() => {
              ConfirmWarn("ELIMINAR IMAGEN",
                `Eliminar la imagen ${image.d ? `"${image.d}"` : "seleccionada"}`,
                "SI","NO",
                () => {
                  deleteProductoImage(image.n)
                }
              )
            }}
          />
        {/each}
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