import { JSX, createSignal } from "solid-js"
import styles from "./components.module.css"
import { POST, POST_XMLHR } from "~/shared/http"
import { Notify } from "~/core/main"

export interface IImageInput {
  content: string
  name: string
  folder: string
}

interface InputEvent extends Event {
  target: HTMLInputElement
  currentTarget: HTMLInputElement
}

export interface ImageData {
  Content: string
  Folder?: string
  Description?: string
}

export interface IImageUploader {
  src?: string
  types?: string[]
  saveAPI?: string
  refreshIndexDBCache?: string
  onUploaded?: (imagePath: string, description?: string) => void
  setDataToSend?: (e: any) => void
  clearOnUpload?: boolean
  description?: string
  cardStyle?: JSX.CSSProperties
  onDelete?: (src: string) => void
}

export const ImageUploader = (props?: IImageUploader) => {

  const [imageSrc, setImageSrc] = createSignal(props || { src: "" } as IImageUploader)
  const [progress, setProgress] = createSignal(props.src ? -1 : 0)

  const onFileChange = async (ev: InputEvent) => {
    const files: FileList = ev.target.files
    const imageFile = files[0] as Blob
    console.log('imagefile::', imageFile)
    // const src = URL.createObjectURL(imageFile)
    const imageB64 = await resizeImageCanvasWebp({ 
      source: imageFile, size: 1.2, quality: 0.89 })
    setProgress(-1)
    setImageSrc({ src: imageB64 })
  }

  const uploadImage = async () => {
    let result: { imageName: string }
    setProgress(0.001)
    const data = { 
      Content: imageSrc().src, Folder: "img-uploads", 
      Description: imageSrc().description }
    if(props.setDataToSend){ props.setDataToSend(data) }

    try {
      result = await POST_XMLHR({
        data,
        route: props.saveAPI || "images",
        refreshIndexDBCache: props.refreshIndexDBCache || "",
        onUploadProgress: e => {
          const progress = Math.round((e.loaded * 100) / e.total)
          console.log("progress:: ", progress)
          setProgress(progress)
        }
      })
    } catch (error) {
      Notify.failure("Error guardando la imagen:", error)
      return
    }
    if(props.clearOnUpload){
      setImageSrc({ src: "" })
      setProgress(0)
    } else {
      setProgress(-1)
      setImageSrc({ src: `${result.imageName}-x2`, types: ["webp","avif"] })
    }
    if(props.onUploaded){ 
      props.onUploaded(result.imageName, imageSrc().description) 
    }
  }

  const isImageBase64 = () => {
    return imageSrc().src?.length > 0 && (imageSrc()?.types||[]).length === 0
  }

  const makeImageSrc = () => {
    let src = imageSrc().src
    if(src.substring(0,5) !== "data:"){
      if(src.substring(0,8) !== "https://" && src.substring(0,7) !== "http://"){
        src = window.S3_URL + src
      }
    }
    return src
  }
  
  return <div class={`p-rel ${styles.card_image_1} ${imageSrc()?.src ? "" : styles.card_input}`}
    style={props.cardStyle}
  >
    { (imageSrc()?.src||"").length === 0 &&
      <div class={`w100 h100 p-rel flex-column ai-center jc-center ${styles.card_input_layer}`}>
        <input onChange={ev => onFileChange(ev)} type="file" 
          accept="image/png, image/jpeg, image/webp" 
        />
        <div style={{ "font-size": "2.4rem" }}>
          <i class="icon-upload"></i>
        </div>
        <div class="h5">Subir Imagen</div>
      </div>
    }
    { imageSrc()?.src?.length > 0 &&
      <picture class="dsp-cont">
        { imageSrc().types?.includes("avif") &&
          <source type="image/avif" srcset={makeImageSrc() + ".avif"} />
        }
        { imageSrc().types?.includes("webp") &&
          <source type="image/webp" srcset={makeImageSrc() + ".webp"} />
        }
        <img class={`w100 h100 p-abs ${styles.card_image_img1}`}
          src={makeImageSrc() + 
            (imageSrc().types?.length > 0 ? `.${imageSrc().types[0]}` : "")}/>
      </picture>
    }
    { progress() == -1 &&
      <div class={`w100 h100 p-abs ${styles.card_image_layer}${isImageBase64() ? " s1" : ""}`}>
        { isImageBase64() &&
          <textarea class={`w100 ${styles.card_image_textarea}`} rows={3}
            placeholder="Nombre..."
            onBlur={ev => {
              ev.stopPropagation()
              imageSrc().description = ev.target.value || ""
            }}
          />
        }
        <div class={`w100 p-abs flex-center ${styles.card_image_layer_botton}`}>
          <button class={`bnr2 b-red mr-04 ${isImageBase64() 
            ? "" : styles.card_image_layer_bn_close2}`} 
            onClick={ev => {
              ev.stopPropagation()
              if(props.onDelete){
                props.onDelete(props.src); return
              }
              setImageSrc({ src: "" })
              setProgress(0)
            }}>
            <i class="icon-cancel"></i>
          </button>
          { isImageBase64() &&
            <button class="bnr2 b-blue" onClick={ev => {
              ev.stopPropagation()
              uploadImage()
            }}>
              <i class="icon-ok"></i>
            </button>
          }
        </div>
      </div>
    }
    { progress() > 0 &&
      <div class={`w100 h100 p-abs flex-center ${styles.card_image_layer_loading}`}>
        <div class="c-white h3 ff-bold">Loading...</div>
      </div>
    }
  </div>
}

const resizeImageInCanvas = (
  canvas: HTMLCanvasElement, width: number, height: number, type: string, quality: number 
): Promise<string> => {
  
  const scaledCanvas = document.createElement('canvas')
  scaledCanvas.width = width
  scaledCanvas.height = height
  const ctx2 = scaledCanvas.getContext('2d') as CanvasRenderingContext2D
  ctx2.drawImage(canvas, 0, 0, width, height)

  return new Promise<string>(resolve => {
    scaledCanvas.toBlob(blob => {
      blob = (blob as Blob)
      const reader = new FileReader()
      reader.readAsDataURL(blob)
      reader.onloadend = () => {
        const base64data = reader.result  
        resolve((base64data as string))
      }
    },type,quality)
  })
}

export interface ConvertImageArgs {
  source: Uint8Array | Blob
  originalSize?: number[]
  size?: number
  thumbnailSize?: number
  quality?: number
}

const resizeImageCanvasWebp = (args: ConvertImageArgs): Promise<string> => {
  const canvas = document.createElement('canvas')
  const img = new Image
  // const img = document.createElement('img')

  return new Promise((resolve,reject) => {   
    img.onload = () => {
      args.originalSize = [img.width, img.height]
      
      canvas.width = (args.originalSize as number[])[0]
      canvas.height = (args.originalSize as number[])[1]
      const maxAreaPx = (args.size as number) * 1000 * 1000
      
      const imageAreaPx = canvas.width * canvas.height
      let newWidth = canvas.width
      let newHeight = canvas.height
      if(imageAreaPx > maxAreaPx){
        const coef = (imageAreaPx / maxAreaPx)**(0.5)
        newWidth = Math.floor(canvas.width / coef)
        newHeight =  Math.floor(canvas.height / coef)
      }

      const ctx = canvas.getContext('2d') as CanvasRenderingContext2D
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
      console.log('args 11::', args)

      resizeImageInCanvas(canvas,newWidth,newHeight,'image/webp',args.quality)
      .then(imageSrc => {
        resolve(imageSrc)
      })
      .catch(error => {
        reject(error)
      })
    }

    img.src = URL.createObjectURL(args.source as Blob)
    console.log('Data-URL::', img.src)
  })
}

export interface IImageCtn {
  src?: string
  types?: string[]
  size?: 2 | 4 | 6 | 8 | 9
  class?: string
  style?: JSX.CSSProperties
}

export const ImageCtn = (props: IImageCtn) => {

  const makeImageSrc = () => {
    let src = props.src
    if(src.substring(0,5) !== "data:"){
      if(src.substring(0,8) !== "https://" && src.substring(0,7) !== "http://"){
        src = window.S3_URL + src
      }
      if(props.size){ src = `${src}-x${props.size}` }
    }
    return src
  }


  return <picture class={"dsp-cont"} style={props.style}>
    { props.types?.includes("avif") &&
      <source type="image/avif" srcset={makeImageSrc() + ".avif"} />
    }
    { props.types?.includes("webp") &&
      <source type="image/webp" srcset={makeImageSrc() + ".webp"} />
    }
    <img class={`${props.class||""}`}
      style={props.style}
      src={makeImageSrc() + (props.types?.length > 0 ? `.${props.types[0]}` : "")}
    />
  </picture>
}

