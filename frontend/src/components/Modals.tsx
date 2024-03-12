import { JSX, JSXElement, Show, createEffect, createSignal } from "solid-js";
import { Portal } from "solid-js/web";
import { deviceType } from "~/app";

interface IModal {
  children: JSX.Element
  id: number
  title: string | JSX.Element
  css?: string
  isEdit?: boolean
  onSave?: () => void
  onDelete?: () => void
  onClose?: () => void
}

export const [openModals, setOpenModals] = createSignal([])

export function Modal(props: IModal) {

  const [isOpen, setIsOpen] = createSignal(openModals().includes(props.id))
  let modalDiv = undefined as unknown as HTMLDivElement

  createEffect(()=> {
    const isThisModalOpen = openModals().includes(props.id)
    if(isOpen() === isThisModalOpen){ return }

    if(isThisModalOpen){
      setIsOpen(true)
      setTimeout(()=> { 
        if(modalDiv){ modalDiv.classList.add("show"),0  }
      })
    } else {
      modalDiv?.classList?.remove("show")
      setTimeout(()=> { setIsOpen(false) },300)
    }
  })

  return <Show when={isOpen()}>
    <Portal mount={document.body}>
      <div class="modal-background flex-center" ref={modalDiv}>
        <div class={"modal-body flex-column p-rel " +(props.css||"")}>
          <div class="modal-title flex ai-center jc-between mb-auto">
            <div class="flex ai-center ff-bold h2 name">
              { props.title }
            </div>
            <div class="flex ai-center">
              { props.onDelete &&
                <button class="bn1 b-red2 mr-08 lh-10" onClick={ev => {
                  if(props.onDelete){
                    props.onDelete()
                    ev.stopPropagation()
                  }
                }}>
                  <i class="icon-trash"></i>
                </button>
              }
              { props.onSave &&
                <button class="bn1 b-blue2 mr-08 lh-10" onClick={ev => {
                  if(props.onSave){
                    props.onSave()
                    ev.stopPropagation()
                  }
                }}>
                  <i class="icon-floppy"></i>
                  { [1,2].includes(deviceType()) &&
                    <span>{ props.isEdit ? "Actualizar" : "Guardar" }</span>
                  }
                </button>
              }
              <button class="bn1 b-yellow2 h3 lh-10" onClick={ev => {
                ev.stopPropagation()
                if(props.onClose){ props.onClose() }
                setOpenModals([])
              }}>
                <i class="icon-cancel"></i>
              </button>
            </div>
          </div>
          {props.children}
        </div>
      </div>
    </Portal>
  </Show>
}

interface IContentLayer {
  children: JSXElement | JSXElement[]
  style?: JSX.CSSProperties
  class?: string
  showMobileLayer?: boolean
  onClose?: (() => void)
}

export const ContentLayer = (props: IContentLayer) => {

  const cN = () => {
    return "mob-layer1" + (props.class ? " " + props.class : "")
  }

  let layerRef: HTMLDivElement
  let bgLayerRef: HTMLDivElement

  createEffect(() => {
    if(props.showMobileLayer){
      setTimeout(()=> { 
        if(layerRef){ layerRef.classList.add("show") }
        if(bgLayerRef){ bgLayerRef.classList.add("show")  }
      },1)
    } else {
      if(layerRef){ layerRef.classList.remove("show") }
      if(bgLayerRef){ bgLayerRef.classList.remove("show")  }
    }
  })

  return <>
    <Show when={props.showMobileLayer || [1].includes(deviceType())}>
      { [2,3].includes(deviceType()) && 
        <div class="bg-layer1" ref={bgLayerRef} onClick={ev => {
          ev.stopPropagation()
          if(props.onClose){ props.onClose() }
        }}>
          <div class="flex-center"><i class="icon-cancel"></i></div>
        </div>
      }
      <div style={props.style} class={cN()} ref={layerRef}>
        {props.children}
      </div>
    </Show>
  </>
}

interface ISideLayer {
  id: number
  children: JSXElement | JSXElement[]
  style?: JSX.CSSProperties
  class?: string
  showMobileLayer?: boolean
  title: string | JSX.Element
  buttonSave?: JSX.Element
  css?: string
  isEdit?: boolean
  onSave?: () => void
  onDelete?: () => void
  onClose?: () => void
}

export const [openLayers, setOpenLayers] = createSignal([])

export const SideLayer = (props: ISideLayer) => {

  const [isOpen, setIsOpen] = createSignal(openLayers().includes(props.id))
  let modalDiv = undefined as unknown as HTMLDivElement

  createEffect(()=> {
    const isThisModalOpen = openLayers().includes(props.id)
    if(isOpen() === isThisModalOpen){ return }

    if(isThisModalOpen){
      setIsOpen(true)
      setTimeout(()=> { modalDiv?.classList?.add("show"),0 })
    } else {
      setIsOpen(false)
      /*
      modalDiv?.classList?.remove("show")
      setTimeout(()=> { setIsOpen(false) },300)
      */
    }
  })

  const cN = () => {
    return "side-layer1" + (props.class ? " " + props.class : "")
  }

  let layerRef: HTMLDivElement
  let bgLayerRef: HTMLDivElement

  return <>
    <Show when={isOpen()}>
      <div style={props.style} class={cN()} ref={layerRef}>
        <div class="flex w100 jc-between px-12 py-08">
          <div class="flex ai-center ff-bold h2 name">
            { props.title }
          </div>
          <div class="flex ai-center">
            { props.onDelete &&
              <button class="bn1 b-red2 mr-08 lh-10" onClick={ev => {
                if(props.onDelete){
                  props.onDelete()
                  ev.stopPropagation()
                }
              }}>
                <i class="icon-trash"></i>
              </button>
            }
            { props.onSave &&
              <button class="bn1 b-blue2 mr-08 lh-10" onClick={ev => {
                if(props.onSave){
                  props.onSave()
                  ev.stopPropagation()
                }
              }}>
                { props.buttonSave || <>
                  <i class="icon-floppy"></i>
                  { [1,2].includes(deviceType()) &&
                    <span>{ props.isEdit ? "Actualizar" : "Guardar" }</span>
                  }
                </>
                }
              </button>
            }
            <button class="bn1 b-yellow2 h3 lh-10" onClick={ev => {
              ev.stopPropagation()
              if(props.onClose){ props.onClose() }
              setOpenLayers([])
            }}>
              <i class="icon-cancel"></i>
            </button>
          </div>
        </div>
        <div class="w100 h100 p-rel px-12 py-04">
          {props.children}
        </div>
      </div>
    </Show>
  </>
}


export const CornerLayer = (props: ISideLayer) => {

  const [isOpen, setIsOpen] = createSignal(openLayers().includes(props.id))
  let modalDiv = undefined as unknown as HTMLDivElement

  createEffect(()=> {
    const isThisModalOpen = openLayers().includes(props.id)
    if(isOpen() === isThisModalOpen){ return }

    if(isThisModalOpen){
      setIsOpen(true)
      setTimeout(()=> { modalDiv?.classList?.add("show"),0 })
    } else {
      setIsOpen(false)
      /*
      modalDiv?.classList?.remove("show")
      setTimeout(()=> { setIsOpen(false) },300)
      */
    }
  })

  const cN = () => {
    return "corner-layer" + (props.class ? " " + props.class : "")
  }

  let layerRef: HTMLDivElement

  return <>
    <Show when={isOpen()}>
      <div style={props.style} class={cN()} ref={layerRef}>
        <div class="flex w100 jc-between px-12 py-08">
          <div class="flex ai-center ff-bold h2 name">
            { props.title }
          </div>
          <div class="flex ai-center">
            { props.onDelete &&
              <button class="bn1 b-red2 mr-08 lh-10" onClick={ev => {
                if(props.onDelete){
                  props.onDelete()
                  ev.stopPropagation()
                }
              }}>
                <i class="icon-trash"></i>
              </button>
            }
            { props.onSave &&
              <button class="bn1 b-blue2 mr-08 lh-10" onClick={ev => {
                if(props.onSave){
                  props.onSave()
                  ev.stopPropagation()
                }
              }}>
                { props.buttonSave || <>
                  <i class="icon-floppy"></i>
                  { [1,2].includes(deviceType()) &&
                    <span>{ props.isEdit ? "Actualizar" : "Guardar" }</span>
                  }
                </>
                }
              </button>
            }
            <button class="bn1 b-yellow2 h3 lh-10" onClick={ev => {
              ev.stopPropagation()
              if(props.onClose){ props.onClose() }
              setOpenLayers([])
            }}>
              <i class="icon-cancel"></i>
            </button>
          </div>
        </div>
        <div class="w100 h100 p-rel px-12 py-04">
          {props.children}
        </div>
      </div>
    </Show>
  </>
}