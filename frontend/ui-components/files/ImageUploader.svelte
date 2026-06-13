<script lang="ts">
import { POST_XMLHR, GET } from '$libs/http.svelte';
import { onDestroy, untrack } from 'svelte';
import { Notify, fileToImage } from '$libs/helpers';
import { Env } from '$core/env';
import { imagesToUpload, tr } from '$core/store.svelte';
import { inMemoryImages, getInMemoryImageBase64, isImageInFlight, type InMemoryImage } from '$core/inMemoryImages.svelte';
import { addProcess, updateProcess } from '$core/notifications.svelte';
import { Agent } from '$components/agent/registry';

export interface IImageInput {
  content: string;
  name: string;
  folder: string;
}

export interface IImageResult {
  id: number;
  imageName: string;
  description?: string;
}

export interface ImageData {
  Content: string;
  Folder?: string;
  Description?: string;
}

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
  onChange?: (e: ImageSource, uploadImage?: () => Promise<IImageResult>) => void
  cardCss?: string;
  hideFormUseMessage?: string;
  hideUploadButton?: boolean;
  hideForm?: boolean;
  id?: number;
  size?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9
  folder?: string
  useConvertAvif?: boolean
  imageSource?: ImageSource
  // Opt-in optimistic flow: confirm (✓) reserves the final ID via GET.image-id-counter, shows
  // the image as "saved" instantly from the in-memory map, and defers convert+upload to the
  // background (the parent flushes it after saving its own record). Used for product images.
  useImageCounter?: boolean
}

let {
  src = "",
  types = [],
  saveAPI = "images",
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
  size, folder, useConvertAvif,
  id = undefined,
  imageSource,
  useImageCounter = false
}: IImageUploaderProps = $props();

// State
// svelte-ignore state_referenced_locally
let imageSrc = $state(imageSource || { src, types, description } as ImageSource);
let progress = $derived((src||imageSrc?.base64) ? -1 : 0);

Env.imageCounter++
// svelte-ignore state_referenced_locally
const imageID = id || Env.imageCounter

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
let isConverting = $state(false)
// True only while the ✓ click awaits the (fast) GET.image-id-counter reservation, so the card
// shows a dark overlay + spinner telling the user a request is in flight.
let isReserving = $state(false)

type IResulution = { i: number, r: number, fn: (c: string) => void }

interface IImagePayload {
  Content: string, Content_x6: string, Content_x4: string, Content_x2: string,
  Description?: string, ImageID?: number
}

// Convert the picked file to AVIF: 3 resolutions when useConvertAvif, else a single 1200px image.
const convertImageFile = async (file: Blob): Promise<IImagePayload> => {
  const data: IImagePayload = { Content: "", Content_x6: "", Content_x4: "", Content_x2: "" }
  if(useConvertAvif){
    const resolutions = [
      { i: 6, r: 980, fn: e => data.Content_x6 = e },
      { i: 4, r: 670, fn: e => data.Content_x4 = e },
      { i: 2, r: 360, fn: e => data.Content_x2 = e }
    ] as IResulution[]
    await Promise.all(resolutions.map(rs => fileToImage(file, rs.r, "avif").then(rs.fn)))
  } else {
    data.Content = await fileToImage(file, 1200, 'avif')
  }
  return data
}

const uploadImage = async (): Promise<IImageResult> => {
  let result = {} as IImageResult
  if(!imageSrc.base64){
    Notify.failure("No hay nada que enviar.")
    return Promise.resolve(result)
  }

  isConverting = true
  let data: IImagePayload
  try {
    data = await convertImageFile(imageFile)
  } catch (error) {
    Notify.failure(`Error al convertir imagen: ${error}`)
    isConverting = false
    return Promise.resolve(result)
  }
  data.Description = imageSrc.description
  isConverting = false
  progress = 0.001

  if (setDataToSend) { setDataToSend(data); }

  console.log("data a enviar::", data)

  try {
    result = await POST_XMLHR({
      data,
      route: saveAPI || "images",
      refreshRoutes,
      onUploadProgress: e => {
        if(e.total){
          progress = Math.round((e.loaded * 100) / e.total)
          console.log("progress:: ", progress)
        }
      }
    })
  } catch (error) {
    Notify.failure('Error guardando la imagen: ' + String(error));
    return result;
  }

  result.id = imageID as number;
  result.description = imageSrc.description;
  // Base CDN name "<companyID>_<imageID>" (strip the folder prefix the backend returns).
  const savedBaseName = (result.imageName || "").split("/").pop() || "";

  if (clearOnUpload) {
    imageSrc = { src: '', base64: '', types: [], description: '' };
  } else {
    imageSrc = { src: `${result.imageName}-x2`, base64: '', types: ['webp', 'avif'], description: imageSrc.description };
  }

  console.log("image src 1::",$state.snapshot(progress), $state.snapshot(imageSrc),"clearOnUpload",clearOnUpload)
  if (onUploaded) {
    onUploaded({ id: Number(savedBaseName.split("_")[1]) || (imageID as number), name: savedBaseName, description: result.description });
  }

  progress = -1
  console.log("image src 2::",$state.snapshot(progress), $state.snapshot(imageSrc),"clearOnUpload",clearOnUpload)
  return result;
};

// Optimistic confirm (useImageCounter): reserve the final ID (fast), show the image as "saved"
// immediately from the in-memory map, and convert in the background. The byte-upload is deferred
// into entry.upload() so the parent can run it AFTER saving its own record (when ProductID exists).
const confirmOptimistic = async () => {
  if(!imageSrc.base64){
    Notify.failure("No hay nada que enviar.")
    return
  }
  const fileToUpload = imageFile
  const imageDescription = imageSrc.description
  const previewBase64 = imageSrc.base64

  // 1) Reserve the final imageID + base name (the only step the user waits on — fast).
  let reserved: { id: number, name: string }
  isReserving = true
  try {
    reserved = await GET({ route: "image-id-counter" })
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

  // 3) Convert in the background (no ProductID needed); the upload step awaits this promise.
  const convertPromise = convertImageFile(fileToUpload)

  const entry: InMemoryImage = {
    name: finalName, id: finalId, folder: folder || "", base64: previewBase64,
    description: imageDescription, status: 'converting', progress: -1,
  }
  // Transparent background upload. `override` lets the parent inject fields unknown at confirm
  // time (e.g. a NEW product's real ProductID, available only after the product is saved).
  entry.upload = async (override?: Record<string, any>) => {
    if(entry.status === 'uploading'){ return }
    entry.status = 'uploading'
    // Track this background upload as a process in the header notifications layer.
    const processName = imageDescription || finalName
    const processID = addProcess(processName, tr('Uploading image...|Subiendo imagen...'), 1)
    const data = await convertPromise
    data.Description = imageDescription
    data.ImageID = finalId
    Object.assign(data, parentData, override) // override (real id from flush) wins over the snapshot
    console.log("uploading image::", { name: finalName, ImageID: data.ImageID, ProductID: (data as any).ProductID, route: saveAPI })
    try {
      await POST_XMLHR({
        data, route: saveAPI || "images", refreshRoutes,
        onUploadProgress: e => {
          if(e.total){
            entry.progress = Math.round((e.loaded * 100) / e.total)
            updateProcess(processID, '', tr(`Uploading|Subiendo`) + ` ${entry.progress}%`, 1)
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
  // just set. (The legacy uploadImage success path doesn't call onChange either.)
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
  if (parentRecordId > 0) { entry.upload?.().catch(() => {}) }
}

// ✓ button / textarea-blur entry point: optimistic flow when useImageCounter, else legacy upload.
const onConfirm = () => useImageCounter ? confirmOptimistic() : uploadImage()

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
    onChange?.(imageSrc, onConfirm as unknown as () => Promise<IImageResult>)
  } catch (error) {
    Notify.failure('Error procesando la imagen: ' + String(error))
    progress = 0
  }
}

// Legacy deferred-upload registry (e.g. categoria images). The optimistic flow uses the
// in-memory map instead, so it skips this registration.
$effect(() => {
  if (useImageCounter) { return }
  if (imageSrc.base64) {
    imagesToUpload.set(imageID as number, uploadImage);
  } else {
    imagesToUpload.delete(imageID as number);
  }
});

onDestroy(() => {
  imagesToUpload.delete(imageID as number);
});

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
  class="relative {cardCss} card_image_1 {imageSrc?.src ? '' : 'card_input'}"
  style={cardStyle}
>
  {#if (imageSrc?.src || '').length === 0}
    <div class="w-full h-full relative flex flex-col items-center justify-center card_input_layer">
      <input bind:this={fileInputElement} onchange={onFileChange} type="file" accept="image/png, image/jpeg, image/webp" />
      <div style="font-size: 2.4rem">
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
      <img class="w-full h-full absolute card_image_img1"
        src={makeImageSrc(imageSrc?.types?.[0])}
        alt="Upload preview"
      />
    </picture>
  {/if}

  {#if isInFlight}
    <div class="absolute card_image_corner_loader" title={tr('Processing image...|Procesando imagen...')}>
      <span class="loader"></span>
    </div>
  {/if}

  {#if progress === -1}
    <div class="w-full h-full absolute card_image_layer{imageSrc.base64 ? ' s1' : ''}">
      {#if imageSrc.base64 && !hideFormUseMessage && !hideForm}
        <textarea class="w-full card_image_textarea"
          rows={3} placeholder={tr("Name...|Nombre...")}
          onblur={(ev) => {
            ev.stopPropagation();
            imageSrc.description = ev.currentTarget.value || '';
            onChange?.(imageSrc, onConfirm as unknown as () => Promise<IImageResult>);
          }}
        ></textarea>
      {/if}
      {#if imageSrc.base64 && hideFormUseMessage}
        <div class="absolute w-full card_image_upload_text">
          {hideFormUseMessage}
        </div>
      {/if}
      <div class="w-full absolute flex items-center justify-center card_image_layer_botton">
        <button class="bnr-1 _4 mr-12 {imageSrc.base64
            ? ''
            : 'card_image_layer_bn_close2'} card_image_btn"
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
          <button class="bnr-1 _5 card_image_btn"
            aria-label={tr("Upload image|Subir imagen")}
            onclick={(ev) => {
              ev.stopPropagation();
              onConfirm();
            }}
          >
            <i class="icon-ok"></i>
          </button>
        {/if}
      </div>
    </div>
  {/if}

  {#if isReserving || isConverting || progress > 0}
    <div class="w-full h-full absolute flex flex-col items-center justify-center card_image_layer_loading">
      {#if isReserving}
        <div class="card_image_spinner"></div>
      {/if}
      <div class="c-white h3 ff-bold">{isReserving ? tr('Loading...|Cargando...') : isConverting ? tr('Converting...|Convirtiendo...') : tr('Saving...|Guardando...')}</div>
      {#if progress > 0}
        <div class="flex relative items-center justify-center h-22 lh-10 w-[calc(100%-16px)] left-0 right-0 mt-8 _8 mr-8 ml-9 p-2">
          <div class="absolute _9 left-2 h-18"
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
  ._8 {
    background-color: #00000054;
  }
  ._9 {
    background-color: #29b15d;
    width: 80px;
  }

  /* ImageUploader styles */
  .card_image_1 {
    overflow: hidden;
    border-radius: 7px;
    border: 1px solid gray;
    background-color: white;
    cursor: pointer;
  }

  :global(body.dark) .card_image_1 {
    background-color: #474a57;
  }

  .card_image_1.card_input:hover {
    border: 1px solid rgb(110, 105, 122);
    outline: 1px solid rgb(48, 46, 51);
  }

  :global(body.dark) .card_image_1.card_input:hover {
    border-color: white;
    outline-color: white;
  }

  .card_image_img1 {
    top: 0;
    left: 0;
    object-fit: contain;
  }

  .card_image_layer {
    top: 0;
    padding: 6px;
    left: 0;
  }

  .card_image_layer:not(:hover) .card_image_layer_bn_close2 {
    display: none;
  }

  .card_image_layer:hover,
  .card_image_layer.s1 {
    background: linear-gradient(0deg, rgba(0, 0, 0, 0.4) 0%, rgba(255, 255, 255, 0) 25%);
  }

  .card_image_btn {
    outline: 2px solid rgba(255, 255, 255, 0.5);
  }

  .card_image_btn:hover {
    outline: 2px solid rgba(0, 0, 0, 0.7) !important;
  }

  .card_image_layer_loading {
    top: 0px;
    left: 0px;
    backdrop-filter: blur(4px);
    background-color: rgba(0, 0, 0, 0.34);
  }

  .card_image_spinner {
    width: 38px;
    height: 38px;
    margin-bottom: 10px;
    border: 4px solid rgba(255, 255, 255, 0.35);
    border-top-color: #ffffff;
    border-radius: 50%;
    animation: card_image_spin 0.8s linear infinite;
  }

  @keyframes card_image_spin {
    to { transform: rotate(360deg); }
  }

  /* Small dual-ring loader pinned top-right while the image converts/uploads in the background. */
  .card_image_corner_loader {
    top: 9px;
    right: 9px;
    width: 20px;
    height: 20px;
    z-index: 2;
    pointer-events: none;
  }

  .card_image_corner_loader .loader {
    display: inline-block;
    width: 20px;
    height: 20px;
    background: #FF3D00;
    border-radius: 50%;
    position: relative;
    box-sizing: border-box;
    animation: card_image_rotation 2s linear infinite;
  }

  .card_image_corner_loader .loader::after {
    content: '';
    box-sizing: border-box;
    position: absolute;
    left: 50%;
    top: 50%;
    border: 10px solid;
    border-color: transparent #FFF;
    border-radius: 50%;
    transform: translate(-50%, -50%);
  }

  @keyframes card_image_rotation {
    0%   { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .card_image_upload_text {
    font-family: 'bold';
    color: white;
    text-shadow: 2px 2px 6px #000000;
    text-align: center;
    bottom: 3rem;
    background-color: rgba(0, 0, 0, 0.2);
    left: 0;
    line-height: 1.2;
    padding: 0 8px;
  }

  .card_image_layer_botton {
    bottom: 0;
    padding: 6px;
  }

  .card_image_textarea {
    background-color: #0000005c;
    border: none;
    border-radius: 7px;
    color: white;
    font-family: 'bold';
    text-shadow: -1px 0 #000000b8, 0 1px #000000b8, 1px 0 #000000b8, 0 -1px #000000b8;
    line-height: 1.1;
    overflow: hidden;
    height: auto;
    backdrop-filter: blur(4px);
    outline-offset: 0;
    outline: 2px solid #ffffff60;
    resize: none;
    padding: 4px 4px;
  }

  .card_image_textarea:focus {
    outline-color:rgb(189, 141, 253);
    outline-offset: 0;
  }

  .card_image_textarea::placeholder {
    text-shadow: none;
    color: white;
  }

  .card_image_1 input {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    width: 100%;
    cursor: pointer;
    opacity: 0;
  }

  .card_input_layer {
    pointer-events: none;
  }

  .card_image_1 input {
    pointer-events: all;
  }

  .image_loading_layer {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background-color: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
  }

  .image_card_default {
    width: 10rem;
    height: 10rem;
    border-radius: 7px;
    overflow: hidden;
  }

  .image_card_desc {
    bottom: 0;
    left: 0;
    padding: 6px;
    background: linear-gradient(0deg, rgba(0, 0, 0, 0.7) 0%, rgba(0, 0, 0, 0) 100%);
    color: white;
  }
</style>
