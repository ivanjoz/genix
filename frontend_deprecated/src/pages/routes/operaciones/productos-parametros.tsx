import { createEffect, createMemo, createSignal } from "solid-js"
import { IListaRegistro, IListas, INewIDToID, listasCompartidas, postListaRegistros } from "~/services/admin/listas-compartidas"
import s1 from "./styles.module.css"
import { Modal, setOpenModals } from "~/components/Modals"
import { Input } from "~/components/Input"
import { ImageC, ImageUploader, uploadCurrentImages } from "~/components/Uploaders"
import { Loading } from "notiflix"
import { objectAssign } from "~/shared/main"

interface IProductosParametros {
  listas: IListas
  listaID: number
  handlerDerivedMap?: Map<number,()=>void>
  hideBar?: boolean
}

export const ProductosParametros = (props: IProductosParametros) => {
  
  const [view, setView] = createSignal(1)
  const [form, setForm] = createSignal({ Images: [] } as IListaRegistro)

  const [categorias, setCategorias] = createSignal([] as IListaRegistro[])
  createEffect(() => {
    setCategorias(props.listas?.Records?.filter(x => x.ListaID === props.listaID) || [])
  })

  const listaNombre = createMemo(() => listasCompartidas.find(x => x.id === props.listaID)?.name)

  const saveCategoria = async (isDelete?: boolean) => {
    const form_ = form()
    form_.ss = isDelete ? 0 : 1

    Loading.standard("Guardando...")
      await uploadCurrentImages()

      let result: INewIDToID[]
      try {
        result = await postListaRegistros([form_])
      } catch (error) {
        console.warn(error)
      }
    Loading.remove()

    const selected = categorias().find(x => x.ID === form_.ID)

    if(selected && isDelete){ 
      props.listas.Records = props.listas.Records.filter(x => x.ID !== form_.ID)
    } else if(selected) {
      objectAssign(selected, form_) 
    } else {
      form_.ID = result[0].NewID
      props.listas.RecordsMap.set(form_.ID, form_)
      props.listas.Records.push(form_)
    }

    setCategorias(props.listas?.Records?.filter(x => x.ListaID === props.listaID) || [])
    setOpenModals([])
  }

  const newRecord = () => {
    setOpenModals([110])
    setForm({ ID: -1, ListaID: props.listaID, Images: ["","",""] } as IListaRegistro)
  }

  props.handlerDerivedMap.set(props.listaID, newRecord)

  return <div class="w100">
    { !props.hideBar &&
      <div class="flex w100">
        <div class="h3 ff-bold mb-4 mr-auto"></div>
        <button class="bn1 d-green" onClick={ev => {
          ev.stopPropagation()
          newRecord()
        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    }
    <div class="w100 flex-wrap" style={{ "margin-left": "-8px", "margin-right": "-8px" }}>
    { categorias().map(e => {
        return <CategoriaCard e={e}
          onClick={() => {
            setOpenModals([110])
            e.Images = e.Images || []
            while(e.Images.length < 2){ e.Images.push("") }
            setForm({...e})
          }}
        />
      })
    }
    </div>
    <Modal title={form().ID > 0 ? "Editar "+listaNombre() : "Crear "+listaNombre()} 
      css="w56-78" id={110}
      onClose={() => setOpenModals([])}
      onSave={() => {
        saveCategoria()
      }}
      onDelete={form().ID > 0 ? () => {
        saveCategoria(true)
      } : null}
    >
      <Input label="Nombre" saveOn={form()} save="Nombre" required={true}
        css="mb-10"/>
      <Input label="Descripcion" saveOn={form()} save="Descripcion" 
        useTextArea={true} css="mb-10" rows={4}/>
      <div class="flex w100">
        { form().Images.map((image,i) => {
            return <ImageUploader  cardCss={"mr-8 "+s1.categoria_card_image_upload}
              types={["avif","webp"]} hideUploadButton={true}
              src={image ? `producto-categoria/${image}-x2` : ""}
              refreshIndexDBCache="productos"
              saveAPI={form().ID > 0 ? "producto-categoria-image" : ""}
              setDataToSend={e => {
                e.Order = i + 1
              }}
              onDelete={() => {
                form().Images[i] = ""
                setForm({...form()})
              }}
              hideFormUseMessage={`La imagen se subirá al presionar "Guardar"`}
              onUploaded={(imagePath) => {
                if(imagePath.includes("/")){ imagePath = imagePath.split("/")[1] }
                console.log("image path::", imagePath)
                form().Images[i] = imagePath
              }}
            />
          })
        }
      </div>
    </Modal>
  </div>
}

interface ICategoriaCard {
  e: IListaRegistro
  onClick: () => void
}

export const CategoriaCard = (props: ICategoriaCard) => {

  return <div class={"p-12 m-8 "+s1.categoria_card}
    onClick={ev => {
      ev.stopPropagation()
      props.onClick()
    }}
  >
    <div class={"mb-4 "+s1.categoria_card_image}>
      { props.e.Images[0] &&
        <ImageC src={`producto-categoria/${props.e.Images[0]}`} size={4} 
          types={["avif","webp"]} class="h100"/>
      }
    </div>
    <div class="h3 ff-semibold">{props.e.Nombre}</div>
    <div>{props.e.Descripcion}</div>
  </div>
}