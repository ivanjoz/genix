import { createSignal } from "solid-js"
import styles from "./components.module.css"
import { POST } from "~/shared/http"

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

export const ImageUploader = () => {

  const [progress, setProgress] = createSignal(0)
  const [imageSrc, setImageSrc] = createSignal("")

  const onFileChange = async (ev: InputEvent) => {
    const files: FileList = ev.target.files
    const imageFile = files[0] as Blob
    console.log('imagefile::', imageFile)
    // const src = URL.createObjectURL(imageFile)
    const imageB64 = await resizeImageCanvasWebp({ 
      source: imageFile, size: 1.2, quality: 0.89 })
    setProgress(-1)
    setImageSrc(imageB64)
  }

  const uploadImage = async () => {
    try {
      await POST({
        data: { content: imageSrc(), folder: "images", name: "" },
        route: "images",
        onUploadProgress: e => {
          const progress = Math.round((e.loaded * 100) / e.total)
          console.log("progress:: ", progress)
          setProgress(progress)
        }
      })
    } catch (error) {
      
    }
  }

  return <div class={`p-rel ${styles.card_image_1}`}>
    <input onChange={ev => onFileChange(ev)} type="file" 
      accept="image/png, image/jpeg, image/webp" />
    { imageSrc().length > 0 &&
      <img class={`w100 h100 p-abs ${styles.card_image_img1}`} src={imageSrc()} alt="" />
    }
    { progress() != 0 &&
      <div class={`w100 h100 p-abs ${styles.card_image_layer}`}>
        <textarea class={`w100 ${styles.card_image_textarea}`} rows={3}
          placeholder="Nombre..."
        >

        </textarea>
        <div class={`w100 p-abs flex-center ${styles.card_image_layer_botton}`}>
          <button class="bnr2 b-red mr-04">
            <i class="icon-cancel"></i>
          </button>
          <button class="bnr2 b-blue" onClick={ev => {
            ev.stopPropagation()
            uploadImage()
          }}>
            <i class="icon-ok"></i>
          </button>
        </div>
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
        console.log('Base 64 image:::',base64data)
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