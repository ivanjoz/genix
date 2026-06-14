<script lang="ts">
import { POST_XMLHR, GET } from '$libs/http.svelte';
import { untrack } from 'svelte';
import { Notify, fileToImage } from '$libs/helpers';
import { Env } from '$core/env';
import { tr } from '$core/store.svelte';
import { inMemoryImages, getInMemoryImageBase64, isImageInFlight, type InMemoryImage } from '$core/inMemoryImages.svelte';
import { addProcess, updateProcess } from '$core/notifications.svelte';
import { Agent } from '$components/agent/registry';
import s1 from '../components.module.css';

export interface ImageSource {
  src: string, base64: string, description?: string, types?: string[], name?: string,
  _id?: number
}

export interface IImageUploaderProps {
  src?: string;
  types?: string[];
  saveAPI?: string;
  refreshRoutes?: string[];
  onUploaded?: (image: { id: number; name: string; description?: string }) => void;
  setDataToSend?: (e: any) => void;
  clearOnUpload?: boolean;
  description?: string;
  cardStyle?: string;
  onDelete?: (src: string) => void;
  onChange?: (e: ImageSource, confirmImage?: () => Promise<void>) => void
  cardCss?: string;
  hideFormUseMessage?: string;
  hideUploadButton?: boolean;
  hideForm?: boolean;
  size?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9
  folder?: string
  useConvertAvif?: boolean
  convertResolutions?: Array<2 | 4 | 6 | 8>
  imageSource?: ImageSource
  // Label shown as the process name in the header notifications layer (e.g.
  // "Imagen Producto <name>"). Falls back to the image description / final name.
  processName?: string
  // Last image-ID digit identifying the resolution set; 0 uses the backend default.
  imageCounterConfig?: number
}

let {
  src = "",
  types = [],
  saveAPI = "",
  refreshRoutes = [],
  onChange,
  onUploaded = undefined,
  setDataToSend = undefined,
  clearOnUpload = false,
  description = "",
  cardStyle = "",
  onDelete = undefined,
  cardCss = "",
  hideFormUseMessage = "",
  hideForm = false,
  hideUploadButton = false,
  size, folder, useConvertAvif, convertResolutions = [8, 6, 4, 2],
  imageSource,
  processName = undefined,
  imageCounterConfig = 0
}: IImageUploaderProps = $props();

// State
// svelte-ignore state_referenced_locally
let imageSrc = $state(imageSource || { src, types, description } as ImageSource);
let progress = $derived((src||imageSrc?.base64) ? -1 : 0);

// Update imageSrc when props change
$effect(() => {
  src; imageSource;
  console.log("ImageUploader (Changed)::", src, imageSource)
  
  untrack(() => {
  	imageSrc = imageSource || { src, base64: "", types, description }
   	progress = (src || imageSource?.base64) ? -1 : 0
	  console.log("progress (1)::", $state.snapshot(progress),"|",$state.snapshot(src),"|",$state.snapshot(imageSource))
  })
})

// While the image is still converting/uploading, render its optimistic base64 from the shared
// map (folder/suffix tolerant). The map is the single authority: ANY ImageUploader on ANY page
// rendered with this image's name resolves the same in-flight entry — no parent wiring needed.
const liveBase64 = $derived(getInMemoryImageBase64(imageSrc?.src || ""))
// Drives the small corner loader so the user sees the background convert+upload is still running,
// even after the component unmounts and remounts (e.g. switching records and coming back).
const isInFlight = $derived(isImageInFlight(imageSrc?.src || ""))

const makeImageSrc = (format?: string) => {
  if(imageSrc.base64){ return imageSrc.base64 }
  if(liveBase64){ return liveBase64 }
  if(!imageSrc.src){ return "" }

  let srcUrl = imageSrc.src || ""
  // debugger
  if (srcUrl.substring(0, 8) !== "https://" && srcUrl.substring(0, 7) !== "http://") {
    srcUrl = Env.makeCDNRoute(folder as string, size ? `${srcUrl}-x${size}` :  srcUrl)
  }
  if(format){ srcUrl = srcUrl +"."+ format }
  return srcUrl
}

let imageFile: Blob
// True only while the ✓ click awaits the (fast) GET.image-id-counter reservation, so the card
// shows a dark overlay + spinner telling the user a request is in flight.
let isReserving = $state(false)

type IResulution = { i: number, r: number, fn: (c: string) => void }

interface IImagePayload {
  Content: string, Content_x6: string, Content_x4: string, Content_x2: string, Content_x8: string,
  Description?: string, ImageID?: number
}

// Convert the picked file to AVIF: 3 resolutions when useConvertAvif, else a single 1200px image.
// onStep(done, total) is invoked once at the start (done = 0) and after each resolution finishes,
// so callers can report conversion progress ("Convirtiendo Imagen 1 / 3...").
const convertImageFile = async (
  file: Blob, onStep?: (done: number, total: number) => void,
): Promise<IImagePayload> => {
  const data: IImagePayload = {
    Content: "", Content_x8: "", Content_x6: "", Content_x4: "", Content_x2: "",
  }
  if(useConvertAvif){
    const resolutions = [
      { i: 8, r: 1200, fn: (content: string) => data.Content_x8 = content },
      { i: 6, r: 980, fn: (content: string) => data.Content_x6 = content },
      { i: 4, r: 670, fn: (content: string) => data.Content_x4 = content },
      { i: 2, r: 360, fn: (content: string) => data.Content_x2 = content }
    ].filter(resolution => convertResolutions.includes(resolution.i as 2 | 4 | 6 | 8)) as IResulution[]
    const total = resolutions.length
    let done = 0
    onStep?.(0, total)
    await Promise.all(resolutions.map(rs => fileToImage(file, rs.r, "avif").then(e => {
      rs.fn(e)
      onStep?.(++done, total)
    })))
  } else {
    onStep?.(0, 1)
    data.Content = await fileToImage(file, 1200, 'avif')
    onStep?.(1, 1)
  }
  return data
}

// Reserve the final ID, show the image immediately, then convert/upload in the background.
const confirmImage = async () => {
  if(!imageSrc.base64){
    Notify.failure("No hay nada que enviar.")
    return
  }
  if(!saveAPI){
    Notify.failure("No se configuró el endpoint para guardar la imagen.")
    return
  }
  const fileToUpload = imageFile
  const imageDescription = imageSrc.description
  const previewBase64 = imageSrc.base64

  // 1) Reserve the final imageID + base name (the only step the user waits on — fast).
  let reserved: { id: number, name: string }
  isReserving = true
  try {
    const counterRoute = imageCounterConfig > 0
      ? `image-id-counter?config=${imageCounterConfig}`
      : "image-id-counter"
    reserved = await GET({ route: counterRoute })
    console.log("ImageUploader (Reserved ID)::", reserved)
  } catch (error) {
    isReserving = false
    Notify.failure('Error al reservar el id de imagen: ' + String(error))
    return
  }
  isReserving = false
  console.log("image-id-counter reserved::", reserved)
  const finalName = reserved.name
  // Derive the id from the base name "<companyID>_<imageID>" so it never depends on how the
  // numeric `id` field survives response (de)serialization.
  const finalId = Number(reserved.id) || Number(String(finalName).split("_")[1])

  // 2) Capture the parent's fields ONCE, synchronously, right here — while the correct record is
  // still selected. This is the ONLY point setDataToSend is read: after it, convert+upload are
  // fully opaque to the parent (no callback ever re-enters it, so a later record switch can't
  // rebind this image's ProductID). For a NEW record the id isn't known yet (ProductID = 0); the
  // parent's flush then supplies it via upload(override).
  const parentData: Record<string, any> = {}
  if (setDataToSend) { setDataToSend(parentData) }

  // Track the whole image pipeline (convert → upload) as one process in the header layer.
  // The header row truncates long names with an ellipsis, so no length cap is needed here.
  const processLabel = processName || imageDescription || finalName
  const processID = addProcess(processLabel, tr('Converting image|Convirtiendo imagen') + '...', 1)

  // 3) Convert in the background (no ProductID needed); the upload step awaits this promise.
  // Report each converted resolution as a step: "Convirtiendo Imagen 1 / 3...".
  const convertPromise = convertImageFile(fileToUpload, (done, total) => {
    updateProcess(processID, '', tr('Converting Image|Convirtiendo Imagen') + ` ${Math.max(done, 1)} / ${total}...`, 1)
  })

  const entry: InMemoryImage = {
    name: finalName, id: finalId, folder: folder || "", base64: previewBase64,
    description: imageDescription, status: 'converting', progress: -1,
  }
  // Transparent background upload. `override` lets the parent inject fields unknown at confirm
  // time (e.g. a NEW product's real ProductID, available only after the product is saved).
  entry.upload = async (override?: Record<string, any>) => {
    if(entry.status === 'uploading'){ return }
    entry.status = 'uploading'
    const data = await convertPromise
    data.Description = imageDescription
    data.ImageID = finalId
    Object.assign(data, parentData, override) // override (real id from flush) wins over the snapshot
    console.log("uploading image::", { name: finalName, ImageID: data.ImageID, ProductID: (data as any).ProductID, route: saveAPI })
    updateProcess(processID, '', tr('Sending converted images...|Enviando imágenes convertidas...'), 1)
    try {
      await POST_XMLHR({
        data, route: saveAPI, refreshRoutes,
        onUploadProgress: e => {
          if(e.total){
            entry.progress = Math.round((e.loaded * 100) / e.total)
            updateProcess(processID, '', tr('Sending converted images...|Enviando imágenes convertidas...') + ` ${entry.progress}%`, 1)
          }
        },
      })
    } catch (error) {
      entry.status = 'error'; entry.error = String(error)
      updateProcess(processID, '', tr('Upload failed|Error al subir') + `: ${String(error)}`, 0)
      console.error("image upload failed::", finalName, error)
      Notify.failure('Error guardando la imagen: ' + String(error))
      throw error
    }
    updateProcess(processID, '', tr('Image uploaded|Imagen subida'), 2)
    inMemoryImages.delete(finalName) // free memory; renderers fall through to the CDN AVIF
  }
  // Mark "pending upload" once conversion finishes (unless the flush already started/failed).
  convertPromise
    .then(() => { if(entry.status === 'converting'){ entry.status = 'pending' } })
    .catch(err => { entry.status = 'error'; entry.error = String(err) })

  inMemoryImages.set(finalName, entry)

  // 4) Tell the parent the image is "saved" with its final id + name — its involvement ends here.
  if (onUploaded) { onUploaded({ id: finalId, name: finalName, description: imageDescription }) }

  // 5) Flip/reset the card exactly like a completed upload would. NOTE: do NOT call onChange
  // here — the parent's onChange resets its optimistic Image/_imageSource, wiping what onUploaded
  // just set.
  if (clearOnUpload) {
    imageSrc = { src: '', base64: '', types: [], description: '' }
    progress = 0
  } else {
    imageSrc = { src: size ? `${finalName}-x${size}` : finalName, base64: '', types: ['avif', 'webp'], description: imageDescription }
    progress = -1
  }

  // 6) If the parent record already exists (editing: ProductID known), upload now in the background
  // so the process finishes on its own. For a NEW record (id = 0) it stays pending until the parent
  // saves its record and flushes it with the real id via upload({ ProductID }).
  const parentRecordId = Number(parentData.ProductID || parentData.id || 0)
  if (!setDataToSend || parentRecordId > 0) { entry.upload?.().catch(() => {}) }
}

const onFileChange = async (ev: Event) => {
  const target = ev.target as HTMLInputElement
  const files: FileList | null = target.files
  if (!files || files.length === 0){ return }

  imageFile = files[0] as Blob
  console.log('imagefile::', imageFile)

  try {
    const base64 = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = (e) => resolve(e.target?.result as string)
      reader.onerror = reject
      reader.readAsDataURL(imageFile)
    })
    progress = -1
    imageSrc = { src: "", base64: base64, types: [], description: imageSrc.description }
    onChange?.(imageSrc, confirmImage)
  } catch (error) {
    Notify.failure('Error procesando la imagen: ' + String(error))
    progress = 0
  }
}

$effect(() => {
  console.log("progress (2)::", $state.snapshot(progress))
})

const componentID = Env.getComponentID()
let fileInputElement = $state<HTMLInputElement | undefined>(undefined)

const removeImage = () => {
  if (onDelete) { onDelete(src); return }
  imageSrc = { src: '', base64: '', types: [], description: '' }
  onChange?.(imageSrc)
  progress = 0
}

$effect(() => {
  return Agent.register({
    id: componentID,
    type: "ImageUploader",
    label: description || "",
    click: () => { fileInputElement?.click() },
    deleted: () => { removeImage() },
  })
})
</script>

<div data-id="ImageUploader:{componentID}"
  data-value={imageSrc?.src || imageSrc?.base64 ? "uploaded" : "empty"}
  class="group relative overflow-hidden rounded-[10px] border border-gray-500 bg-white cursor-pointer
    {imageSrc?.src
      ? ''
      : 'hover:border-[#6e697a] hover:outline hover:outline-1 hover:outline-[#302e33]'}
    {cardCss}"
  style={cardStyle}
>
  {#if (imageSrc?.src || '').length === 0}
    <div class="relative flex flex-col items-center justify-center w-full h-full pointer-events-none">
      <input
        class="absolute inset-0 w-full h-full cursor-pointer opacity-0 pointer-events-auto"
        bind:this={fileInputElement}
        onchange={onFileChange}
        type="file"
        accept="image/png, image/jpeg, image/webp"
      />
      <div class="text-[2.4rem]">
        <i class="icon-upload"></i>
      </div>
      <div class="h5">{tr("Upload Image|Subir Imagen")}</div>
    </div>
  {/if}

  {#if (imageSrc.src||imageSrc.base64||"").length > 0}
    <picture class="contents">
      <!-- Skip the typed <source>s while showing optimistic base64: its bytes aren't avif/webp. -->
      {#if !imageSrc.base64 && !liveBase64 && imageSrc.types?.includes('avif')}
        <source type="image/avif" srcset={makeImageSrc("avif")} />
      {/if}
      {#if !imageSrc.base64 && !liveBase64 && imageSrc.types?.includes('webp')}
        <source type="image/webp" srcset={makeImageSrc("webp")} />
      {/if}
      <img class="absolute inset-0 w-full h-full object-contain"
        src={makeImageSrc(imageSrc?.types?.[0])}
        alt="Upload preview"
      />
    </picture>
  {/if}

  {#if isInFlight}
    <div
      class="absolute top-9 right-9 z-4 w-20 h-20 pointer-events-none"
      title={tr('Processing image...|Procesando imagen...')}
    >
      <span class={s1.card_image_corner_loader_ring}></span>
    </div>
  {/if}

  {#if progress === -1}
    <div
      class="absolute inset-0 w-full h-full p-6 hover:bg-[linear-gradient(0deg,rgba(0,0,0,0.4)_0%,rgba(255,255,255,0)_25%)]
        {imageSrc.base64
          ? 'bg-[linear-gradient(0deg,rgba(0,0,0,0.4)_0%,rgba(255,255,255,0)_25%)]'
          : ''}"
    >
      {#if imageSrc.base64 && !hideFormUseMessage && !hideForm}
        <textarea
          class="w-full h-auto overflow-hidden resize-none rounded-[7px] border-0 bg-[#0000005c] p-4 text-white
            outline-2 outline-[#ffffffa3] backdrop-blur-[4px] focus:outline-1 focus:outline-[#7e4dbd]
            placeholder:text-white [font-family:bold] [line-height:1.1]
            [text-shadow:-1px_0_#000000b8,0_1px_#000000b8,1px_0_#000000b8,0_-1px_#000000b8]
            [&::placeholder]:[text-shadow:none]"
          rows={3} placeholder={tr("Name...|Nombre...")}
          onblur={(ev) => {
            ev.stopPropagation();
            imageSrc.description = ev.currentTarget.value || '';
            onChange?.(imageSrc, confirmImage);
          }}
        ></textarea>
      {/if}
      {#if imageSrc.base64 && hideFormUseMessage}
        <div
          class="absolute bottom-[3rem] left-0 w-full bg-black/20 px-8 text-center text-white
            [font-family:bold] [line-height:1.2] [text-shadow:2px_2px_6px_#000]"
        >
          {hideFormUseMessage}
        </div>
      {/if}
      <div class="absolute bottom-0 flex items-center justify-center w-full p-6">
        <button class="bnr-1 _4 mr-12 {imageSrc.base64
            ? ''
            : 'hidden group-hover:block'} outline-2 outline-white/50 hover:outline-black/70"
          aria-label={tr("Delete image|Eliminar imagen")}
          onclick={(ev) => {
            ev.stopPropagation();
            if (onDelete) {
              onDelete(src);
              return;
            }
            imageSrc = { src: '', base64: '', types: [], description: '' };
            onChange?.(imageSrc);
            progress = 0;
          }}
        >
          <i class="icon-cancel"></i>
        </button>
        {#if imageSrc.base64 && !hideUploadButton}
          <button class="bnr-1 _5 outline-2 outline-white/50 hover:outline-black/70"
            aria-label={tr("Upload image|Subir imagen")}
            onclick={(ev) => {
              ev.stopPropagation();
              confirmImage();
            }}
          >
            <i class="icon-ok"></i>
          </button>
        {/if}
      </div>
    </div>
  {/if}

  {#if isReserving || progress > 0}
    <div class="absolute inset-0 flex flex-col items-center justify-center w-full h-full bg-black/35 backdrop-blur-[4px]">
      {#if isReserving}
        <div class={s1.card_image_spinner}></div>
      {/if}
      <div class="c-white h3 ff-bold">{isReserving ? tr('Loading...|Cargando...') : tr('Saving...|Guardando...')}</div>
      {#if progress > 0}
        <div class="relative left-0 right-0 flex items-center justify-center h-22 w-[calc(100%-16px)] mt-8 mr-8 ml-9 p-2 bg-black/35 lh-10">
          <div class="absolute left-2 h-18 bg-[#29b15d]"
            style="width: calc({Math.round(progress)}% - 4px);"
          ></div>
          <div class="absolute fs14 ff-bold text-white">{Math.round(progress)} %</div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  /* Button styles */
  ._4 {
    background-color: #e75c5c;
    color: white;
  }
  ._4:hover {
    background-color: #f77d7d;
  }
  ._5 {
    background-color: #1277f5;
    color: white;
  }
  ._5:hover {
    background-color: #52a0ff;
  }

</style>
