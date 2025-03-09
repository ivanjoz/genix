import { ImageUploader } from "~/components/Uploaders"
import s1 from "./webpage.module.css";

interface IImageGalery {

}

export const ImageGalery = (props: IImageGalery) => {

  return <div class={`flex grow-1 w100 ${s1.image_galery_page}`}>
    <ImageUploader saveAPI="galeria-image"
      // refreshIndexDBCache="productos"
      clearOnUpload={true} cardCss={s1.image_upload_card}
      setDataToSend={e => {
        
      }}
      onUploaded={src => {
        console.log("imagen cargada::",src)
      }}
    />
  </div>

}