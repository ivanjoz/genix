import { For, JSX, JSXElement, Show, createEffect, createSignal, on } from "solid-js";

interface ILayerAutoHide {
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

interface IBarOptions {
  options: [number,string][]
  selectedID: number
  class?: string
  onSelect: (e: number) => void
}

export const BarOptions = (props: IBarOptions) => {

  return <div class={"bar-1 flex p-rel" + (props.class ? " " + props.class : "")}>
    <For each={props.options}>
    {e => {
      const getClass = () => {
        let cn = "bn-e1 s1 flex-center"
        if(props.selectedID === e[0]){ cn += " selected" }
        return cn
      }

      return <div class={getClass()} onClick={ev =>{
        ev.stopPropagation()
        if(props.onSelect){ props.onSelect(e[0]) }
      }}>
        {e[1]}
      </div>
    }}
    </For>
    <div class="ln-1 p-abs z10"></div>
  </div>
}