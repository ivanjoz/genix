import { For, JSX, createEffect, createSignal } from "solid-js";
import { on } from "solid-js";
import s1 from "./components.module.css"

export interface IInput<T> {
  id?: number
  saveOn: T
  save: keyof T
  label?: string
  css?: string
  inputCss?: string
  required?: boolean
  validator?: (v: (string|number)) => boolean
  type?: string
  placeholder?: string
  disabled?: boolean
  onChange?: (() => void)
  postValue?: JSX.Element
  baseDecimals?: number
  content?: (string|JSX.Element)
  transform?: (v: (string|number)) => (string|number)
}

const [inputUpdater, setInputUpdater] = createSignal(new Map as Map<number,number>)

export const refreshInput = (ids: number[]) => {
  const map = new Map(inputUpdater())
  for(let id of ids){
    const currentValue = map.get(id) || 0
    map.set(id, currentValue + 1)
  }
  setInputUpdater(map)
}

export function Input<T>(props: IInput<T>) {

  const baseDecimals = props.baseDecimals ? 10 ** props.baseDecimals : 0

  const chekIfInputIsValid = (_props: IInput<T>): number => {
    if(!_props.required || _props.disabled){ return 0 }
    if(!_props.saveOn || !_props.save) return 1
    const value = _props.saveOn[_props.save] as (string|number)

    let pass = !props.required
    if(props.validator){
      pass = props.validator(value as (string|number))
    } else {
      if(value || value === 0){ pass = true }
    } 
    return pass ? 2 : 1
  }

  const [inputValue, setInputValue] = createSignal("" as (string|number))
  const [isInputValid, setIsInputValid] = createSignal(chekIfInputIsValid(props))
  
  let isChange = 0

  const onKeyUp = (ev: KeyboardEvent, isBlur?: boolean) => {
    ev.stopPropagation()
    const target = ev.target as HTMLInputElement
    let value = target.value as (number|string)

    if(props.type === 'number'){
      if(!isBlur && !value && ev.key === "-"){ return }
      if(isNaN(value as number)){ value === undefined }
      else {
        value = parseFloat(value as string)
        if(baseDecimals){ value = Math.round(value * baseDecimals)}
      }
    }

    if(props.transform && isBlur){ value = props.transform(value) }

    if(props.saveOn && props.save){
      props.saveOn[props.save] = value as T[keyof T]
      setIsInputValid(chekIfInputIsValid(props))
    }
    setInputValue(value)
  }
  
  createEffect(on(
    () => [props.saveOn, props.id ? inputUpdater().get(props.id)|| 0 : 0], 
    () => {
      const v = props.saveOn[props.save]
      setInputValue(typeof v === 'number' ? v as number : (v as string||""))
      setIsInputValid(chekIfInputIsValid(props))
    }
  ))
  
  let cN = `${s1.input} p-rel`
  if(props.css){ cN += " " + props.css }
  
  const iconValid = () => {
    if(!isInputValid()) return null
    else if(isInputValid() === 2){ return <i class="v-icon icon-ok c-green"></i>  }
    else if(isInputValid() === 1){ 
      return <i class="v-icon icon-attention c-red"></i> 
    }
  }

  const getValue = () => {
    const value = inputValue()
    if(typeof value !== 'number'){ console.log("valor 1:", value) ; return value || "" }
   console.log("value 2:", value)
    return baseDecimals ? (value as number / baseDecimals) : value
  }

  return <div class={cN}>
    { props.label && <>
        <div class={`${s1.input_lab}`}>{props.label}{ iconValid() }</div>
        <div class={`${s1.input_lab1}`}>{props.label}{ iconValid() }</div>
      </>
    }
    <div class={`${s1.input_div} w100`}>
      <input class={s1.input_inp +" "+ (props.inputCss||"") } 
        value={getValue()} 
        onkeyup={ev => { onKeyUp(ev); isChange++ }}
        type={props.type || "text"}
        onBlur={(ev) => {
          onKeyUp(ev as unknown as any, true)
          if(props.onChange && isChange){ 
            props.onChange() 
            isChange = 0
          }
        }}
        placeholder={props.placeholder||""}
        disabled={props.disabled}
      />
      { !props.label && iconValid() }
      { props.postValue || null  }
    </div>
  </div>
}

interface ICheckBoxContainer<T> {
  saveOn: any
  save: string
  options: T[]
  keys: string
  label?: string
  css?: string
  required?: boolean
  onChange?: () => void
}

export function CheckBoxContainer<T>(props: ICheckBoxContainer<T>) {
  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]

  const [optionsSelected, setOptionsSelected] = createSignal([] as (number|string)[])

  createEffect(on(
    () => [props.saveOn], 
    () => { 
      setOptionsSelected([...(props.saveOn[props.save] || [])])
    }
  ))
  
  return <div class={"flex " + (props.css||"")}>
    <For each={props.options}>
      {(opt) => {
          const checked = () =>{
            return optionsSelected().includes(opt[keyId] as (number|string))
          }

          return <div class="checkbox-cnt mr-06 ">
            <div class={"checkbox-s1"+ (checked() ? " checked" : "")} 
              onClick={ev => {
                ev.stopPropagation()
                const id = opt[keyId] as (number|string)
                let newSelected = [...(optionsSelected())]
                if(newSelected.includes(id)){
                  newSelected = newSelected.filter(x => x !== id)
                } else {
                  newSelected.push(id)
                }
                setOptionsSelected(newSelected)
                if(props.saveOn && props.save){
                  props.saveOn[props.save] = newSelected
                }
              }}>
              <i class="icon-ok"></i>
            </div>
            <label>{opt[keyName] as string}</label>
          </div>
        }
      }
    </For>
  </div>
}

interface ICheckBox<T> {
  saveOn?: any
  save?: string
  label?: string
  css?: string
  required?: boolean
  onChange?: (b:boolean) => void
  checked?: boolean
}

export function CheckBox<T>(props: ICheckBox<T>) {
  const [checked, setChecked] = createSignal(false as boolean)

  createEffect(on(
    () => [props.saveOn, props.checked], 
    () => { 
      if(typeof props.checked === 'boolean'){
        setChecked(props.checked)
        return
      }
      if(props.saveOn && props.save){
        setChecked(props.saveOn[props.save] || false)
      }
    }
  ))

  const updateChecked = () =>{
    if(typeof props.checked === 'boolean'){ return }
    if(props.saveOn && props.save){
      props.saveOn[props.save] = checked()
    }
  }
  
  return <div class="checkbox-cnt mr-06 ">
    <div class={"checkbox-s1"+ (checked() ? " checked" : "")} 
      onClick={ev => {
        ev.stopPropagation()
        const newCheched = !checked()
        setChecked(newCheched)
        updateChecked()
        if(props.onChange){ props.onChange(newCheched) }
      }}>
      <i class="icon-ok"></i>
    </div>
    <label>{props.label}</label>
  </div>
}

export interface IInputDisabled<T> {
  label?: string
  css?: string
  inputCss?: string
  content?: (string|JSX.Element)
  getContent?: () => (string|JSX.Element)
  postValue?: JSX.Element
  decimals?: number
}

export function InputDisabled<T>(props: IInputDisabled<T>) {

  const getContent = props.getContent ? props.getContent : () => props.content

  let cN = "in-5c p-rel flex-column a-start"
  if(props.css){ cN += " " + props.css }

  return <div class={cN}>
    { props.label && 
      <div class="mr-auto label">
        {props.label}
      </div>
    }
    <div class={"in-5 flex ai-center " + (props.inputCss||"") }>
      {getContent()}
    </div>
    { props.postValue || null  }
  </div>
}

