import { Confirm, IConfirmOptions } from "notiflix";
import { For, JSX, JSXElement, Show, createEffect, createMemo, createSignal, on } from "solid-js";
import { SearchSelect } from "./SearchSelect";
import { arrayToMapN } from "~/shared/main";

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
  buttonStyle?: JSX.CSSProperties
  buttonClass?: string
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
        if(props.buttonClass){ cn += " " + props.buttonClass }
        return cn
      }

      return <div class={getClass()} 
        style={props.buttonStyle}
        onClick={ev =>{
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

interface ICardSelect<T,Y> {
  label: string
  options: T[]
  keys: string
  css: string
  saveOn?: Y
  save?: keyof Y
}

export function CardSelect<T,Y>(props: ICardSelect<T,Y>){

  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]

  const [selected, setSelected] = createSignal([])
  const optionsMap = createMemo(() => {
    return arrayToMapN(props.options, keyId as string)
  })

  const setSelectedOnSave = () => {
    if(props.saveOn && props.save){
      props.saveOn[props.save] = selected() as Y[keyof Y]
    }
  }

  createEffect(() => {
    if(props.saveOn && props.save){
      setSelected(props.saveOn[props.save] as (number|string)[])
    }
  })

  return <div class={"p-rel search-c1 "+(props.css)}>
    <div class="flex">
      <SearchSelect options={props.options||[]} icon="icon-search"
        label={""} placeholder={props.label} keys={props.keys}
        clearOnSelect={true}
        avoidIDs={selected()}
        onChange={e => {
          if(!e){ return }
          console.log(e)
          selected().push(e[keyId])
          setSelected([...selected()])
          setSelectedOnSave()
        }}
      />
    </div>
    <div class="search-opts ac-center p-rel z10 flex-wrap">
      <For each={selected()}>
      {e => {
        const opt = optionsMap().get(e)
        let name = ""
        if(opt){  name = opt[keyName] as string }
        else {
          name = `key-${keyId as string}`
        }
        return <div class="search-opt flex-center p-rel">
          { name }
          <div class="bn-close flex-center" onclick={ev =>{
            ev.stopPropagation()
            const newSelected = selected().filter(id => id !== e)
            setSelected(newSelected)
            setSelectedOnSave()
          }}>
            <i class="icon-cancel h5"></i>
          </div>
        </div>
      }}
      </For>
      <div class="ln-1 p-abs z10"></div>
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
