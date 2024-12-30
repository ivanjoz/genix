import { createEffect, createSignal, JSX, JSXElement, on, Show } from "solid-js";


export interface ILayerAutoHide {
  children: JSXElement
  icon: JSXElement
  buttonClass: string
  containerClass?: string
  layerClass?: string
  layerStyle?: JSX.CSSProperties
}

export const LayerSelect = (props: ILayerAutoHide) => {
  const [show, setShow] = createSignal(false)
  
  let refInput: HTMLInputElement
  let refLayer: HTMLDivElement
  let avoidHideOnBlur = false
  
  createEffect(() => {
    if(show()){
      refInput?.focus()
    }
  },[show()])

  return <div class={"flex-center flex-column" + 
    (props.containerClass ? " " + props.containerClass : "")}
  >
    <button class={props.buttonClass} 
      onMouseDown={ev => {
        ev.stopPropagation()
        avoidHideOnBlur = true
      }}
      onClick={ev => {
        ev.stopPropagation()
        if(avoidHideOnBlur){ avoidHideOnBlur = false }
        setShow(!show())
      }}>
      {props.icon}
    </button>
    <div class="p-rel" style={{ width: '0px', height: '0px' }}>
      <Show when={show()}>
        <input ref={refInput} autofocus={true} class="input-hide1"
          onBlur={ev => {
            ev.stopPropagation()
            if(avoidHideOnBlur){
              avoidHideOnBlur = false; refInput?.focus()
            } else {
              setShow(false)
            }
          }}
        />
        <div class={"p-abs layer-c2 layer-angle1" + (props.layerClass ? " " + props.layerClass : "")}
          style={props.layerStyle}
          ref={refLayer}
          onMouseDown={ev => {
            ev.stopPropagation()
            avoidHideOnBlur = true
          }}
        >
          { props.children }
        </div>
      </Show>
    </div>
  </div>
}

export interface ILayerLoading<T> {
  baseObject: T
  startPromise: (e: T) => Promise<any>
  lodingElement?: JSX.Element
  children: JSX.Element | JSX.Element[]
}

export function LayerLoading<T>(props: ILayerLoading<T>){
  const [isLoading, setIsLoading] = createSignal(false)
  
  createEffect(on(() => [props.baseObject], 
    () => {
      setIsLoading(true)
      props.startPromise(props.baseObject)
      .then(() => {
        setIsLoading(false)
      })
    }
  ))

  return <div class={"w100"}>
    { isLoading() &&
      <div class="pm-loading mt-12" style={{ padding: '7px' }}>
        <div class="bg"></div>
        <span>{"Cargando..."}</span>
      </div>
    }
    { !isLoading() &&
      props.children
    }
  </div>
}
