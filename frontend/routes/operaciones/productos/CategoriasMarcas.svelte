<script lang="ts">
import ImageUploader from '$components/ImageUploader.svelte';
import Input from '$components/Input.svelte';
import Modal from '$components/Modal.svelte';
import { closeAllModals, closeModal, imagesToUpload, openModal } from '$core/store.svelte';
import { Loading, Notify } from '$libs/helpers/helpers';
    import { postListaRegistros, type IListaRegistro, type INewIDToID, type ListasCompartidasService } from "./productos.svelte";

  const {
    listas, origin
  }: {
    listas: ListasCompartidasService,
    origin: 1 /* Categorías */ | 2 /* Marcas */,
  } = $props()

  const categorias = $derived.by(() => {
    console.log("listas getted:", $state.snapshot(listas))
    return listas.ListaRecordsMap.get(origin) || []
  })

  let form = $state({} as IListaRegistro)
  const imagesIDs = [151,152,153]
  let images = $state([{ _id: 151 },{ _id: 152 },{ _id: 153 }] as ImageSource[])

  const onSave = async (isDelete?: boolean) => {
    console.log("form a enviar 1::",$state.snapshot(form), isDelete)

    if((form.Nombre||"").length < 4 || !form.ListaID){
      Notify.failure("Debe colocar un nombre de al menos 4 caracteres.")
      return
    }

    Loading.standard("Guardando categoría...")
    form.Images = form.Images || []
    form.ss = isDelete ? 0 : 1

    for(let i = 0; i < imagesIDs.length; i++){
      const imageID = imagesIDs[i]
      // debugger
      if(imagesToUpload.has(imageID)){
        Loading.change(`Guardando Imagen ${i+1}`)
        const result = await imagesToUpload.get(imageID)?.()

        let imageName = result?.imageName || ""
        if(imageName.includes("/")){ imageName = imageName.split("/")[1] }

        form.Images[i] = imageName
      } else {
        form.Images[i] = form.Images?.[i] || ""
      }
    }

    console.log("form a enviar 2::",$state.snapshot(form))
    Loading.change("Guardando categoría...")

    let result: INewIDToID[]
    try {
      result = await postListaRegistros([form])
    } catch (error) {
      Notify.failure(error as string)
      // Loading.remove()
      return
    }

    const selected = categorias.find(x => x.ID === form.ID)
    let newCategorias = [...categorias]

    if(selected && isDelete){
      newCategorias = newCategorias.filter(x => x.ID !== form.ID)
    } else if(selected) {
      Object.assign(selected, form)
    } else {
      form.ID = result[0].NewID
      newCategorias.push(form)
      listas.RecordsMap.set(form.ID, form)
    }

    console.log("newCategorias", newCategorias.length,"|",categorias.length)

    listas.ListaRecordsMap.set(origin, newCategorias)
    listas.ListaRecordsMap = new Map(listas.ListaRecordsMap)
    closeAllModals()
    Loading.remove()
  }

  export const newRecord = () => {
    form = { ListaID: origin as number } as IListaRegistro
    openModal(2)
  }

  $effect(() => {
    console.log("form a enviar::",$state.snapshot(form))
  })

  const selectCategoria = (e: IListaRegistro) => {
    // console.log("form a enviar::",$state.snapshot(e))
    // return
    form = {...e}
    images = imagesIDs.map((_id,i) => {
      return { _id, src: (form.Images||[])[i] || "" } as ImageSource
    })
    console.log("categoría getted:", $state.snapshot(form),$state.snapshot(images))
    openModal(2)
  }

</script>

<div class="w-full flex flex-wrap gap-12">
  {#each categorias as e }
    <div class="_1 px-12 py-10 w-250 min-h-140" role="button" tabindex="0"
      onkeydown={ev => {
        ev.stopPropagation()
        if (ev.key === 'Enter' || ev.key === ' ') {
          ev.preventDefault()
          selectCategoria(e)
        }
      }}
      onclick={ev => {
        ev.stopPropagation()
        selectCategoria(e)
      }}>
      <div class="min-h-70 _2 mb-2">

      </div>
      <div class="fs17 ff-semibold">{e.Nombre}</div>
      <div class="fs15">{e.Descripcion}</div>
    </div>
  {/each}
</div>
<Modal title="CATEGORÍA" id={2} size={6}
  onClose={() => {
    // form = {} as IListaRegistro
    closeModal(2)
  }}
  onSave={() => { onSave()  }}
  onDelete={form.ID > 0 ? () => {
    onSave(true)
  } : undefined}
  >
  <div class="grid grid-cols-12 gap-10 p-6">
    <Input label="Nombre" css="col-span-12"
      saveOn={form} save="Nombre" required={true}
    />
    <Input label="Descripción" css="col-span-12 mb-16"
      saveOn={form} save="Descripcion"
    />
    {#each images as image, index }
      <ImageUploader clearOnUpload={true} id={image._id}
        folder="img-public" src={image.src} size={2}
        cardCss="w-full h-180 p-4 col-span-4" types={["avif","webp"]}
        saveAPI="producto-categoria-image"
        hideUploadButton={true} hideForm={true}
        onChange={e => {
          Object.assign(image, e)
        }}
        setDataToSend={e => {
          e.Order = index + 1
        }}
      />
    {/each }
  </div>
</Modal>

<style>
  ._1 {
    background-color: rgb(255, 255, 255);
    border-radius: 8px;
    box-shadow: rgba(0, 0, 0, 0.1) 0px 4px 6px -1px, rgba(0, 0, 0, 0.06) 0px 2px 4px -1px;
  }
  ._1:hover {
    outline: 2px solid rgba(0, 0, 0, 0.5);
  }

  ._2 {
    background-color: rgb(235, 233, 245);
  }
</style>
