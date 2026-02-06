<script lang="ts">
import { POST_XMLHR } from '$libs/http.svelte';
import { onDestroy, untrack } from 'svelte';
import { Notify, fileToImage } from '$libs/helpers';
import { Env } from '$core/env';
import { imagesToUpload } from '$core/store.svelte';
import type { IImageResult } from '$core/types/common';
import Page from '$domain/Page.svelte';

export interface IImageInput {
  content: string;
  name: string;
  folder: string;
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
  refreshIndexDBCache?: string;
  onUploaded?: (imagePath: string, description?: string) => void;
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
}

let {
  src = "",
  types = [],
  saveAPI = "images",
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
  imageSource
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
  untrack(() => {
  	imageSrc = imageSource || { src, base64: "", types, description }
   	progress = (src || imageSource?.base64) ? -1 : 0
	  console.log("progress (1)::", $state.snapshot(progress),"|",$state.snapshot(src),"|",$state.snapshot(imageSource))
  })
})

const makeImageSrc = (format?: string) => {
  if(imageSrc.base64){ return imageSrc.base64 }
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

type IResulution = {
  i: number, r: number, fn: (c: string) => {}, promise?: Promise<any>
}

const uploadImage = async (): Promise<IImageResult> => {
  let result = {} as IImageResult
  if(!imageSrc.base64){
    Notify.failure("No hay nada que enviar.")
    return Promise.resolve(result)
  }

  progress = 0.001

  const data = {
    Content: "", Content_x6: "",  Content_x4: "",  Content_x2: "",
    Description: imageSrc.description
  }

  if(useConvertAvif){
    const resolutions = [
      { i: 6, r: 980, fn: e => data.Content_x6 = e },
      { i: 4, r: 670, fn: e => data.Content_x4 = e },
      { i: 2, r: 360, fn: e => data.Content_x2 = e }
    ] as IResulution[]

    for(const rs of resolutions){
      rs.promise = new Promise(resolve => {
        fileToImage(imageFile, rs.r, "avif").then(d => {
          // console.log("image b64",d)
          rs.fn(d), resolve(0)
        })
      })
    }

    try {
      await Promise.all(resolutions.map(x => x.promise))
    } catch (error) {
      Notify.failure(`Error al convertir imagen: ${error}`)
      return Promise.resolve(result)
    }

  } else {
    data.Content = imageSrc.base64
  }

  if (setDataToSend) { setDataToSend(data); }

  console.log("data a enviar::", data)

  try {
    result = await POST_XMLHR({
      data,
      route: saveAPI || "images",
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

  if (clearOnUpload) {
    imageSrc = { src: '', base64: '', types: [], description: '' };
  } else {
    imageSrc = { src: `${result.imageName}-x2`, base64: '', types: ['webp', 'avif'], description: imageSrc.description };
  }

  console.log("image src 1::",$state.snapshot(progress), $state.snapshot(imageSrc),"clearOnUpload",clearOnUpload)
  if (onUploaded) {
    onUploaded(result.imageName, result.description);
  }

  progress = -1
  console.log("image src 2::",$state.snapshot(progress), $state.snapshot(imageSrc),"clearOnUpload",clearOnUpload)
  return result;
};

const onFileChange = async (ev: Event) => {
  const target = ev.target as HTMLInputElement
  const files: FileList | null = target.files
  if (!files || files.length === 0){ return }

  imageFile = files[0] as Blob
  console.log('imagefile::', imageFile)

  try {
    // Use web worker-based image conversion (1.2 MP resolution, WebP format)
    const imageB64 = await fileToImage(imageFile, 1200, 'avif')
    console.log("imageB64", imageB64)
    progress = -1
    imageSrc = { src: "", base64: imageB64, types: [], description: imageSrc.description }
    onChange?.(imageSrc, uploadImage)
  } catch (error) {
    Notify.failure('Error procesando la imagen: ' + String(error))
    progress = 0
  }
}

// Effect to manage imagesToUpload map
$effect(() => {
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


</script>

<div class="relative {cardCss} card_image_1 {imageSrc?.src ? '' : 'card_input'}"
  style={cardStyle}
>
  {#if (imageSrc?.src || '').length === 0}
    <div class="w-full h-full relative flex flex-col items-center justify-center card_input_layer">
      <input onchange={onFileChange} type="file" accept="image/png, image/jpeg, image/webp" />
      <div style="font-size: 2.4rem">
        <i class="icon-upload"></i>
      </div>
      <div class="h5">Subir Imagen</div>
    </div>
  {/if}

  {#if (imageSrc.src||imageSrc.base64||"").length > 0}
    <picture class="contents">
      {#if imageSrc.types?.includes('avif')}
        <source type="image/avif" srcset={makeImageSrc("avif")} />
      {/if}
      {#if imageSrc.types?.includes('webp')}
        <source type="image/webp" srcset={makeImageSrc("webp")} />
      {/if}
      <img class="w-full h-full absolute card_image_img1"
        src={makeImageSrc(imageSrc?.types?.[0])}
        alt="Upload preview"
      />
    </picture>
  {/if}

  {#if progress === -1}
    <div class="w-full h-full absolute card_image_layer{imageSrc.base64 ? ' s1' : ''}">
      {#if imageSrc.base64 && !hideFormUseMessage && !hideForm}
        <textarea class="w-full card_image_textarea"
          rows={3} placeholder="Nombre..."
          onblur={(ev) => {
            ev.stopPropagation();
            imageSrc.description = ev.currentTarget.value || '';
            onChange?.(imageSrc, uploadImage);
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
          aria-label="Eliminar imagen"
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
            aria-label="Subir imagen"
            onclick={(ev) => {
              ev.stopPropagation();
              uploadImage();
            }}
          >
            <i class="icon-ok"></i>
          </button>
        {/if}
      </div>
    </div>
  {/if}

  {#if progress > 0}
    <div class="w-full h-full absolute flex flex-col items-center justify-center card_image_layer_loading">
      <div class="c-white h3 ff-bold">Loading...</div>
      <div class="flex relative items-center justify-center h-22 lh-10 w-[calc(100%-16px)] left-0 right-0 mt-8 _8 mr-8 ml-9 p-2">
        <div class="absolute _9 left-2 h-18"
          style="width: calc({Math.round(progress)}% - 4px);"
        ></div>
        <div class="absolute fs14 ff-bold text-white">{Math.round(progress)} %</div>
      </div>
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
