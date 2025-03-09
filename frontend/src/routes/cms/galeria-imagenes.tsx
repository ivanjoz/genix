import { Image, ImageCard, ImageUploader } from "~/components/Uploaders"
import s1 from "./webpage.module.css";
import { IGaleriaImagen } from "~/services/cms/galeria-images";
import { createEffect, createSignal, For } from "solid-js";
import { Params } from "~/shared/security";
import { fechaUnixToSunix } from "~/core/main";
import { Modal, setOpenModals } from "~/components/Modals";

interface IImageGalery {
  imagenes: IGaleriaImagen[]
  useUploader?: boolean
}

export const ImageGalery = (props: IImageGalery) => {

  const [images, setImages] = createSignal(props.imagenes||[])
  createEffect(() => {
    setImages(props.imagenes||[])
  })

  return <div class={`flex grow-1 w100 ${s1.image_galery_page}`}>
    <div class="w100">
      
    </div>
    { props.useUploader &&
      <ImageUploader saveAPI="galeria-image"
        // refreshIndexDBCache="productos"
        clearOnUpload={true} cardCss={s1.image_upload_card}
        setDataToSend={e => {
          
        }}
        onUploaded={(Image, Description) => {
          const image = {
            Image, Description, upd: Params.sunixTime(), ss: 1
          } as IGaleriaImagen
          props.imagenes.unshift(image)
          setImages([...props.imagenes])
        }}
      />
    }
    {
    images().map(e => {
      return <ImageCard size={4} types={["avif","webp"]}
        src={`img-galeria/${e.Image}`}
        style={{ height: '12rem', width: 'auto', "min-width": '10rem', 
          "max-width": "18rem",
        }}
        imageStyle={{ "object-fit": 'contain' }}
      />
    })
    }
  </div>
}

interface IImageGalerySelector {
  imagenes: IGaleriaImagen[]
}

export const ImageGalerySelector = (props: IImageGalerySelector) => {
  return <>
    <div class={`p-rel ${s1.image_selector} flex-column ai-center jc-center`}
      onClick={ev => {
        ev.stopPropagation()
        setOpenModals([101])
      }}
    >
      <i class="icon-picture"></i>
      Seleccione Imagen
    </div>
    <Modal title="Galería Imágenes" id={101} css="w74-84">
      <ImageGalery imagenes={props.imagenes}/>
    </Modal>
  </>
}