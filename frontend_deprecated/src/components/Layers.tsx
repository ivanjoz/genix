import { children, createEffect, createMemo, createSignal, JSX, JSXElement, on, onCleanup, Show } from "solid-js";


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
            return
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

/* Background Parallax */
export interface IBackgroundParallax {
  children: JSX.Element
  reverse?: boolean
  offsetTop?: number
  keyUpdated?: string
}

export const BackgroundParallax = (props: IBackgroundParallax) => {

  let scrollContainer: HTMLElement
  let maxHeight = 0, apear = 0, disapear = 0
  console.log("volviendo a renderizar??")

  const THIS = {
    clear: null as () => void,
    parent: null as HTMLElement,
    element: null as HTMLElement,
    onWindowScroll: null as (() => void)
  }

  const childNodes = children(() => props.children)

  const onWindowScroll = () => {
    // console.log("Ejecutando :)", props.keyUpdated,"|",THIS.element)
    THIS.onWindowScroll() 
  }

  createEffect(on(
    () => [props.keyUpdated],
    () => {
      THIS.element = childNodes() as HTMLElement
      THIS.parent = THIS.element.parentElement as HTMLElement
    
      if(THIS.element.tagName.toLowerCase() === "picture"){    
        THIS.element = THIS.element.querySelector('img') as unknown as HTMLElement
      }
  
      // Define cual es el elemento que hace scroll
      let parentContCurrent = THIS.element.offsetParent as HTMLElement
      while(parentContCurrent){
        if(parentContCurrent.scrollHeight > maxHeight){
          maxHeight = parentContCurrent.scrollHeight
          scrollContainer = parentContCurrent
        }
        parentContCurrent = parentContCurrent.offsetParent as HTMLElement
      }
  
      // console.log("scrollContainer", THIS.parent, scrollContainer)
      scrollContainer = scrollContainer || document.documentElement


      const recalcVariables = () => {
        // console.log("Element for parallax::", THIS.element)
    
        THIS.element.style.position = "absolute"
    
        // Distancia del margen superior del elemento hasta el Top
        if(!THIS.element || !THIS.parent) return
        THIS.element.style.transition = "top 250ms"
        
        const wH = window.innerHeight // Altura de la pantalla
        const conRect = THIS.parent.getBoundingClientRect()
        // .y = distancia del elemento hacia el top
        apear = conRect.y - wH - scrollContainer.offsetTop
        if(apear < 0){ apear = 0 }
        // .bottom = distancia del bottom del elemento hacia el top
        disapear = conRect.bottom - (props.offsetTop||0)
        /*
        console.log("parallax apear | disapear::", apear,"|",disapear)
    
        // Máximo valor posible de window.scrollY;
        let maxScroll = document.body.scrollHeight - wH
        // Si [disapear] es mayor que el máximo, se iguala al máximo
        if(disapear > maxScroll) disapear = maxScroll
        // Rango donde se ejecutará el parallax
        range = disapear - apear
        // Compense es el porcentaje fijo que debe agregarse al backgroud position para que la imagen esté centrada en caso de offset menor a 100
        // Cuando el offset = 100; compense = 0;
        compense = ( range * ( 1 - offset / 100 ) ) / 2
        */
      }
      
      THIS.onWindowScroll = () => {
        const element = THIS.element
        // Posición del scroll
        const wy = scrollContainer.scrollTop || window.scrollY
        // Si el elemento no está en el viewport no hace nada
        console.log("is scroll | apear:",apear,"| disapear:",disapear,"| scroll:",wy)
  
        if(apear > wy || wy > disapear){ return }
  
        const displacementMax = element.offsetHeight - THIS.parent.offsetHeight
        const scrollRange = disapear - apear
  
        let scollPartial = wy - apear
        if(scollPartial > scrollRange){ scollPartial = scrollRange }
  
        console.log("parallax: está en el viewport::", displacementMax)
        
        const offset = Math.floor((scollPartial / scrollRange) * displacementMax)
  
        /*
        let perc = ( (wy - apear + compense) / range) * offset
        // Si el parallax es inverso
        if(props.reverse) perc = 100 - perc
        // Agrega el estilo al objeto
        */
        element.style.top = `-${offset}px`
        // console.log("setting top:: ", element.style.top, element)
      }
  
      const updateAndScroll = () => {  recalcVariables(); onWindowScroll() }
      updateAndScroll()

      let scroller: Window | HTMLElement = window
      if(!["html","body"].includes(scrollContainer.tagName.toLowerCase())){
        scroller = scrollContainer
      }
      
      console.log("Agregando eventos::", scroller)
      scroller.addEventListener("scroll", onWindowScroll)
      scroller.addEventListener("load", updateAndScroll)
      scroller.addEventListener("resize", updateAndScroll)

      THIS.clear = () => {
        scroller.removeEventListener("scroll", onWindowScroll)
        scroller.removeEventListener("load", updateAndScroll)
        scroller.removeEventListener("resize", updateAndScroll)
      }
    }
  ))

  onCleanup(() => {
    console.log("limpiando eventos!!")
    THIS.clear()
  })
  
  return <>
    { childNodes() }
  </>
}

