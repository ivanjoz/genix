import { createSignal } from "solid-js"
import styles from "./components.module.css"
import { POST } from "~/shared/http"
import { Notify } from "notiflix"

export interface IImageInput {
  content: string
  name: string
  folder: string
}

export const postProducto = (data: IImageInput) => {
  return POST({
    data,
    route: "images",
  })
}

interface InputEvent extends Event {
  target: HTMLInputElement
  currentTarget: HTMLInputElement
}

interface ImageSrc {
  src?: string
  types?: string[]
}

export const ImageUploader = () => {

  const [progress, setProgress] = createSignal(0)
  const [imageSrc, setImageSrc] = createSignal({ src: "" } as ImageSrc)

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
    try {
      result = await POST({
        data: { content: imageSrc().src, folder: "images", name: "" },
        route: "images",
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

    const imageName = `${window.S3_URL}${result.imageName}-x2`
    setImageSrc({ src: imageName, types: ["webp","avif"] })
    setProgress(-1)
  }

  const isImageBase64 = () => {
    return (imageSrc()?.types||[]).length === 0
  }

  return <div class={`p-rel ${styles.card_image_1}`}>
    { (imageSrc()?.src||"").length === 0 &&
      <input onChange={ev => onFileChange(ev)} type="file" 
        accept="image/png, image/jpeg, image/webp" 
      />
    }
    { imageSrc()?.src?.length > 0 &&
      <picture class="dsp-cont">
        { imageSrc().types?.includes("avif") &&
          <source type="image/avif" srcset={imageSrc().src + ".avif"} />
        }
        { imageSrc().types?.includes("webp") &&
          <source type="image/webp" srcset={imageSrc().src + ".webp"} />
        }
        <img class={`w100 h100 p-abs ${styles.card_image_img1}`}
          src={imageSrc().src + 
            (imageSrc().types?.length > 0 ? `.${imageSrc().types[0]}` : "")}/>
      </picture>
    }
    { progress() == -1 &&
      <div class={`w100 h100 p-abs ${styles.card_image_layer}${isImageBase64() ? " s1" : ""}`}>
        { isImageBase64() &&
          <textarea class={`w100 ${styles.card_image_textarea}`} rows={3}
            placeholder="Nombre..."
          />
        }
        <div class={`w100 p-abs flex-center ${styles.card_image_layer_botton}`}>
          <button class={`bnr2 b-red mr-04 ${isImageBase64() 
            ? "" : styles.card_image_layer_bn_close2}`} 
            onClick={ev => {
              ev.stopPropagation()
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