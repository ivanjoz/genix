import { Image, ImageCard, ImageUploader } from "~/components/Uploaders"
import s1 from "./webpage.module.css";
import { IGaleriaImagen } from "~/services/cms/galeria-images";
import { createEffect, createMemo, createSignal, For, JSX } from "solid-js";
import { Params } from "~/shared/security";
import { fechaUnixToSunix, include, throttle } from "~/core/main";
import { Modal, setOpenModals } from "~/components/Modals";

interface IImageGalery {
  imagenes: IGaleriaImagen[]
  useUploader?: boolean
  selectable?: boolean
  showTitle?: boolean
  barStyle?: JSX.CSSProperties
  onSelect?: (e: IGaleriaImagen) => void
}

export const ImageGalery = (props: IImageGalery) => {

  const [images, setImages] = createSignal(props.imagenes||[])
  createEffect(() => {
    setImages(props.imagenes||[])
  })

  const [filterText, setFilterText] = createSignal([] as string[])

  const filteredImages = createMemo(() => {
    const text = filterText()
    if(text.length === 0){ return images() }
    else {
      return images().filter(x => include(x.Description, text))
    }
  })

  return <div class={`flex-column grow-1 w100 ${s1.image_galery_page}`}>
    <div class="flex ai-center w100" style={props.barStyle}>
      { props.showTitle &&
        <div class="ff-bold h2 mr-16">Galería Imágenes</div>
      }
      <div class="search-c4 mr-16 w14rem mr-auto">
        <div><i class="icon-search"></i></div>
        <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
          ev.stopPropagation()
          throttle(() => {
            const text = ((ev.target as HTMLInputElement).value||"").toLowerCase().trim() as string
            setFilterText(text.split(" ").filter(x => x))
          },150)
        }}/>
      </div>
    </div>
    <div class="flex-wrap w100">
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
      filteredImages().map(e => {
        return <ImageCard size={4} types={["avif","webp"]}
          src={`img-galeria/${e.Image}`}
          description={e.Description}
          style={{ height: '12rem', width: 'auto', 
            "min-width": '10rem', "max-width": "18rem",
          }}        
          imageStyle={{ "object-fit": 'contain' }}
          selectable={true}
          onSelect={() => {
            if(props.onSelect){ props.onSelect(e) }
          }}
        />
      })
      }
    </div>
  </div>
}

interface IImageGalerySelector {
  imagenes: IGaleriaImagen[]
  imageSelected?: string
  onSelect: (e: IGaleriaImagen) => void
}

export const ImageGalerySelector = (props: IImageGalerySelector) => {
  return <>
    <div class={`p-rel ${s1.image_selector}`}
      onClick={ev => {
        ev.stopPropagation()
        setOpenModals([101])
      }}
    >
      <div class={`p-abs w100 h100 flex-column ai-center jc-center ${s1.image_selector_seleccione}`}
        classList={{[s1.image_selector_cambiar]: !!props.imageSelected }}
      >
        { props.imageSelected &&
          <>
            <i class="icon-picture-1"></i>
            <div>Cambiar Imagen</div> 
          </>
        }
        { !props.imageSelected &&
          <>
            <i class="icon-picture-1"></i>
            <div>Seleccione Imagen</div> 
          </>
        }
      </div>
      { props.imageSelected &&
        <Image src={`img-galeria/${props.imageSelected}`} 
          types={["avif","webp"]} size={4}
          class="w100 h100"
        />
      }
    </div>
    <Modal title="" id={101} css="w74-84">
      <ImageGalery imagenes={props.imagenes} showTitle={true}
        barStyle={{ position: 'absolute', top: '0.6rem', width: 'fit-content' }}
        selectable={true}
        onSelect={img => {
          setOpenModals([])
          props.onSelect(img)
        }}
      />
    </Modal>
  </>
}