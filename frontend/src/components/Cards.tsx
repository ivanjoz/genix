import { For, JSX, JSXElement, Show, createEffect, createMemo, createSignal, on } from "solid-js";
import { VList } from "../components/virtua/solid";
import { arrayToMapN } from "~/shared/main";
import { SearchSelect } from "./SearchSelect";
import s1 from "./cards.module.css"

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
      props.saveOn[props.save] = props.saveOn[props.save] || [] as Y[keyof Y]
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

export interface ICardsList<T> {
  data: T[]
  render: (d: T, i: number) => JSX.Element
}

export function CardsList<T>(props: ICardsList<T>){

  const records = createMemo(() => props.data)

  return <VList data={records()}>
    {(d, i) => {
      return props.render(records()[i],i)
    }}
  </VList>
}

export interface IButtonList<T> {
  options: T[]
  keys: string
  onClick: (e: T) => void
  selected?: keyof T
}

export function ButtonList<T>(props: IButtonList<T>){

  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]
  const [selected, setSelected] = createSignal(props.selected)

  return <div class="flex ai-center">
    <For each={props.options}>
    {e => {
      const isSelected = createMemo(() => e[keyId] === selected())

      return <button class="bn1 mr-06" 
        classList={{ "b-blue": isSelected() }}
        onClick={ev => {
        ev.stopPropagation()
        setSelected(e[keyId] as any)
        if(props.onClick){ props.onClick(e) }
      }}>
        {e[keyName] as string}
      </button>
    }}
    </For>
  </div>
}

interface ISpinnerProps {
  mensaje?: string, className?: string
  size?: string
}

export const Spinner = (props: ISpinnerProps) => {
  let className = "spinner1 flex a-center j-center"
  if (props.className) { className += (" " + props.className) }

  let mensaje = props.mensaje || ''

  if (mensaje.includes(' ...')) {
    mensaje = mensaje.replace(' ...', '...')
  }

  if (mensaje.includes('...')) {
    // Comprobar si los puntos suspensivos están al final
    if (mensaje.indexOf('...') === mensaje.length - 3) {
      // Si es así, quitarlos, traducir y volver a añadirlos
      mensaje = mensaje.replace('...', '')
      mensaje = mensaje + '...'
    }
  }

  return (
    <div class={className}>
      {mensaje &&
        <div class="mr-08">
          {mensaje || `Cargando...`}
        </div>
      }
    <div class="lds-spinner">
      <div></div><div></div><div></div><div></div><div></div>
      <div></div><div></div><div></div><div></div><div></div>
      <div></div><div></div></div>
    </div>
  )
}

import arrow1Svg from "../assets/flecha_inicio.svg?raw"
import arrow2Svg from "../assets/flecha_fin.svg?raw"
import { parseSVG } from "~/core/main";

interface IICardArrowStepsOption {
  id: number, name: string, icon?: string
}

export interface ICardArrowSteps {
  options: IICardArrowStepsOption[]
  onSelect?: (e: IICardArrowStepsOption) => void
  selected?: number
  optionRender?: (e: IICardArrowStepsOption) => JSX.Element
}

export const CardArrowSteps = (props: ICardArrowSteps) => {

  return <div class="grid mr-08"
    style={{ "grid-template-columns": props.options.map(x => "1fr").join(" ") }}
  >
  { props.options.map(e => {
      return <div class={`flex p-rel ai-center ${s1.card_arrow_ctn}`}
        classList={{ [s1.card_arrow_ctn_selected]: e.id === props.selected }}
        onClick={ev => {
          ev.stopPropagation()
          if(props.onSelect){ 
            props.onSelect(e)
          }
        }}
      >
        <img class={`h100 ${s1.card_arrow_svg}`} 
          src={parseSVG(arrow1Svg)} alt=""
        />
        <div class={`h100 flex-center ${s1.card_arrow_name}`}>
          { props.optionRender 
              ? props.optionRender(e) 
              : <div class="ff-semibold">{e.name}</div>
          }          
        </div>
        <img class={`h100 ${s1.card_arrow_svg}`} src={parseSVG(arrow2Svg)} alt="" 
        />
        <div class={s1.card_arrow_line}></div>
      </div>
    })
  }
  </div>
}


export interface ISpinner4 {
  style: JSX.CSSProperties
}

export const Spinner4 = (props: ISpinner4) => {
  return <div style={props.style}>
    <div class="progress1">  
    </div>
  </div>
}

