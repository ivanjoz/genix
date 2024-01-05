import { For, createEffect, createSignal } from "solid-js";
import { on } from "solid-js";

interface IInput<T> {
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
}

export function Input<T>(props: IInput<T>) {

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

  const onKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    const target = ev.target as HTMLInputElement
    setInputValue(target.value||"")
    if(props.saveOn){
      let value = target.value as (number|string)
      if(props.type === 'number'){
        if(isNaN(value as number)){ value === undefined }
        else {
          value = parseFloat(value as string)
        }
      }
      props.saveOn[props.save] = value as T[keyof T]
      // console.log("is valid:: ", chekIfInputIsValid(props))
      setIsInputValid(chekIfInputIsValid(props))
    }
  }
  
  createEffect(on(
    () => props.saveOn, 
    () => {
      setInputValue((props.saveOn[props.save]||"") as string)
      setIsInputValid(chekIfInputIsValid(props))
    }
  ))

  let cN = "in-5c p-rel flex-column a-start"
  if(props.css){ cN += " " + props.css }
    
  const iconValid = () => {
    if(!isInputValid()) return null
    else if(isInputValid() === 2){ return <i class="v-icon icon-ok c-green"></i>  }
    else if(isInputValid() === 1){ 
      return <i class="v-icon icon-attention c-red"></i> 
    }
  }

  return <div class={cN}>
    { props.label && 
      <div class="mr-auto label">
        {props.label} { iconValid() }
      </div>
    }
    <input class={"in-5 " + (props.inputCss||"") } value={inputValue()} 
      onkeyup={ev => {
        onKeyUp(ev)
        isChange++
      }}
      type={props.type || "text"}
      onBlur={(ev) => {
        onKeyUp(ev as unknown as any)
        if(props.onChange && isChange){ 
          props.onChange() 
          isChange = 0
        }
      }}
      placeholder={props.placeholder||""}
      disabled={props.disabled}
    />
    { !props.label && iconValid() }
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