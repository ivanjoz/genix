import { For, Show, createEffect, createMemo, createSignal, on, onMount } from "solid-js";
import { Portal } from "solid-js/web";
import { throttle } from "~/core/main";
import { deviceType } from "~/app";
import s1 from "./components.module.css"
import { Spinner4 } from "./Cards";

interface SearchSelect<T> {
  saveOn?: any
  save?: string | keyof T
  css?: string
  options: T[]
  keys: string
  label?: string
  placeholder?: string
  max?: number
  onChange?: (e: T) => void
  selected?: (number|string)
  notEmpty?: boolean
  required?: boolean
  disabled?: boolean
  clearOnSelect?: boolean
  avoidIDs?: number[]
  inputCss?: string
  icon?: string
  showLoading?: boolean
}

export function highlString(phrase: string, words: string[]) {
  if(typeof phrase !== 'string'){
    console.error("no es string")
    console.log(phrase)
    return "!"
  }
  const arr: (string|Element)[] = [phrase]
  if (!words || words.length === 0) return arr

  for (let word of words) {
    if (word.length < 2) continue
    for (let i = 0; i < arr.length; i++) {
      const str = arr[i]
      if (typeof str !== 'string') continue
      const idx = str.toLowerCase().indexOf(word)
      if (idx !== -1) {
        const ini = str.slice(0, idx)
        const middle = str.slice(idx, idx + word.length)
        const fin = str.slice(idx + word.length)
        arr.splice(i, 1, ini, <i>{middle}</i> as Element, fin)
        continue
      }
    }
  }
  return arr.filter(x => x)
}

export function makeHighlString(content: string, search: string){
  return <span class="_highlight">
    {highlString(content, search.split(" "))}
  </span>
}

export function SearchSelect<T>(props: SearchSelect<T>) {

  const [show, setShow] = createSignal(false)
  const [filteredOptions, setFilteredOptions] = createSignal([...props.options])
  const [arrowSelected, setArrowSelected] = createSignal(-1)
  const [avoidhover, setAvoidhover] = createSignal(false)
  const [isValid, setIsValid] = createSignal(0)

  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]

  let inputRef: HTMLInputElement = undefined as unknown as HTMLInputElement
  let words: string[] = []

  const getSelectedFromProps = (): T => {
    let currValue = props.selected
    if(typeof currValue === 'undefined' && props.save && props.saveOn){
      currValue = props.saveOn[props.save as keyof T] as (number|string)
    }
    let selected: T
    if(currValue){
      selected = props.options.find(x => x[keyId] === currValue)
    }
    return selected
  }

  const isRequired = () => {
    return props.required && !props.disabled
  }

  createEffect(on(
    () => [props.saveOn, props.selected||"", props.options], 
    () => {
      if(!inputRef) return
      const selected = getSelectedFromProps()
      if(selected){
        inputRef.value = selected[keyName] as string
      } else {
        inputRef.value = ""
      }
      if(isRequired()){ setIsValid(selected ? 1 : 2) }
    }
  ))
  
  let prevSetValueTime = Date.now()

  const setValueSaveOn = (selected?: T, setOnInput?: boolean) => {
    const nowTime = Date.now()
    if(nowTime - prevSetValueTime < 80){ return }
    prevSetValueTime = nowTime

    if(props.notEmpty && !selected){ 
      selected = getSelectedFromProps()
    }

    if(setOnInput && inputRef){
      if(selected && !props.clearOnSelect){
        inputRef.value = selected[keyName] as string
      } else {
        inputRef.value = ""
      }  
    }

    const newValue = selected ? selected[keyId] : null
    if(isRequired()){ setIsValid(newValue ? 1 : 2) }

    if(props.clearOnSelect){
      if(props.onChange){ props.onChange(selected) }
    } else if(props.saveOn && props.save){
      const current = (props.saveOn[props.save as keyof T] || null) as number
      if(current !== newValue){
        props.saveOn[props.save as keyof T] = newValue
        if(props.onChange){ props.onChange(selected) }
      }
    } else if(typeof props.selected !== 'undefined') {
      if((props.selected || null) !== newValue){
        if(props.onChange){ props.onChange(selected) }
      }
    }
  }

  const filter = (text: string) => {
    if(!text && ((props.avoidIDs?.length||0) === 0)){ return [...props.options] }
    const avoidIDSet = new Set(props.avoidIDs||[])

    const filteredOptions: T[] = []
    for(let op of props.options){
      if(avoidIDSet.size > 0 && avoidIDSet.has(op[keyId as keyof T] as number)){ 
        continue 
      }
      const name = op[keyName] as string
      if(typeof name === 'string'){
        const nameN = name.toLowerCase()
        if(!text || nameN.includes(text)){ filteredOptions.push(op) }
      }
    }
    return filteredOptions
  }

  const onKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    const target = ev.target as HTMLInputElement
    const text = String(target.value||"").toLowerCase()

    throttle(() => {
      words = String(inputRef.value).split(" ")
      setFilteredOptions(filter(text))
    },120)
  }

  const onKeyDown = (ev: KeyboardEvent) => {
    console.log("avoid hover:: ", avoidhover())
    ev.stopPropagation()

    let nroR = filteredOptions().length
    if (nroR > props.max) nroR = props.max

    if (!show() || filteredOptions().length === 0) return
    // console.log('keycode', ev.keyCode)
    // Fecha arriba
    if (ev.key === "ArrowUp") {
      let arrow = (arrowSelected() || 0) - 1
      if (arrow < 0) arrow = nroR
      setArrowSelected(arrow)
      if (!avoidhover()) setAvoidhover(true)
    }
    else if (ev.key === "ArrowDown") {
      let arrow = (arrowSelected() || 0) + 1
      if (arrow > nroR) arrow = 1
      setArrowSelected(arrow)
      if (!avoidhover()) setAvoidhover(true)
    }
  }

  const onOptionClick = (opt: T) => {
    if(inputRef){
      setValueSaveOn(opt, true)
    }
  }

  // let cN = "in-5c p-rel flex-column a-start"
  let cN = `${s1.input} p-rel`
  if(props.css){ cN += " " + props.css }
  if(!props.label){ cN += " no-label" }

  const iconValid = (): any => {
    if(!isValid()){ return null }
    if(isValid() === 1){ return <i class="v-icon icon-ok c-green"></i>  }
    else { 
      return <i class="v-icon icon-attention c-red"></i> 
    }
  }

  const disabled = createMemo(() => {
    return props.disabled || props.showLoading
  })

  return <div class={cN}>
    { props.label && <>
        <div class={`${s1.input_lab}`}>{props.label}{ iconValid() }</div>
        <div class={`${s1.input_lab1}`}>{props.label}{ iconValid() }</div>
      </>
    }
    <div class={`${s1.input_div} w100`}>
      <input class={`${s1.input_inp +" "+ (props.inputCss||"")}`} 
        onkeyup={onKeyUp} ref={inputRef}
        onPaste={onKeyUp as any}
        onCut={onKeyUp as any}
        onKeyDown={onKeyDown}
        placeholder={
          props.showLoading ? "" : props.placeholder || ":: seleccione ::"
        }
        onkeydown={ev => {
          ev.stopPropagation()
          console.log(ev)
        }}
        disabled={disabled()}
        onFocus={ev=> {
          ev.stopPropagation()
          words = []
          setFilteredOptions(filter(""))
          setShow(true)
        }}
        onBlur={ev => {
          ev.stopPropagation()
          // Revisa si la opción seleccionada existe
          let inputValue = String(inputRef.value||"").toLowerCase()
          const selected = props.options.find(
            x => ((x[keyName]||"") as string).toLowerCase() === inputValue
          )
          setValueSaveOn(selected,true)
          setShow(false)
        }}
      />
    </div>
    { props.showLoading &&
      <Spinner4 style={{ position: 'absolute', left: '0.7rem', bottom: "8px" }}/>
    }
    <Show when={!disabled()}>
      <div class={`${s1.input_icon1} p-abs ` + ((show() && !props.icon) ? " show" : "")}>
        <i class={props.icon || "icon-down-open-1"}></i>
      </div>
    </Show>
    { show() &&
      <div class={"search-ctn z20 w100" + (arrowSelected() >= 0 ? " on-arrow" : "")}
        onMouseMove={avoidhover() ? (ev) => {
          console.log("hover aqui:: ", arrowSelected())
          ev.stopPropagation()
          if(avoidhover()) {
            setArrowSelected(-1)
            setAvoidhover(false)
          }
        } : undefined}
      >
        { filteredOptions().map((e,i) => {
            let cN = "flex ai-center _highlight"
            if(arrowSelected() === i){ cN += " _selected" }

            const name = e[keyName as keyof T] as string
            return <div class={cN}
              onMouseDown={ev => {
                ev.stopPropagation()
                onOptionClick(e)
              }}>
              <div class="txt">{highlString(name, words)}</div>
            </div>
          })
        }
      </div>
    } 
 </div>
}

let cardCounterID = 0

export function SearchCard<T>(props: SearchSelect<T>) {

  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]
  const [optionsSelected, setOptionsSelected] = createSignal([] as (number|string)[])

  cardCounterID++
  const id = cardCounterID
  const optionsMap: Map<(number|string),T> = new Map()
  
  const processOptions = () => {
    for(let e of props.options){
      optionsMap.set(e[keyId] as (number|string),e)
    }
  }

  createEffect(on(
    () => [props.options], 
    () => { processOptions() }
  ))

  createEffect(on(
    () => [props.saveOn], 
    () => { 
      setOptionsSelected([...(props.saveOn[props.save] || [])])
    }
  ))

  const doSetOptionsSelected = (ids: (number|string)[]) => {
    setOptionsSelected([...ids])
    if(props.saveOn && props.save){  
      props.saveOn[props.save as keyof T] = ids as T[keyof T]
    }
  }

  return <div class={"flex p-rel mr-06 " + (props.css||"")}
    style={{ 'padding-top': [3].includes(deviceType()) ? '0.8rem' : undefined }}
    classList={{ 'ml-06': [3].includes(deviceType()) }}
  >
    <Show when={[3].includes(deviceType())}>
      <div class="in-label p-abs" style={{ top: '-2px' }}>
        {props.label||props.placeholder}
      </div>
    </Show>
    <Show when={[1,2].includes(deviceType())}>
      <SearchSelect css={props.inputCss || "w-08x"} label={props.label}
        options={props.options} keys={props.keys} placeholder={props.placeholder}
        avoidIDs={optionsSelected() as number[]}
        clearOnSelect={true} icon="icon-search"
        onChange={e => {
          if(!e){ return }
          optionsSelected().push(e[keyId] as (number|string))
          doSetOptionsSelected(optionsSelected())
        }}
      />
    </Show>
    <Show when={[3].includes(deviceType())}>
      <button class="bn1 d-green s1" style={{ "margin-top": '2px' }} onClick={ev => {
        ev.stopPropagation()
        setOpenSearchLayer(id)
      }}>
        <i class="icon-plus"></i>
      </button>
    </Show>
    <div class="grow-1 flex-wrap card-c12p ml-06">
      <For each={optionsSelected()}>
        {(id) => {
            const opt = optionsMap.get(id)
            const name = opt ? opt[keyName] : `ITEM-${id}`
            return <div class="card-c12">
              { name as string }
              <button class="bnr1 b-trash" onClick={ev => {
                ev.stopPropagation()
                const newSelected = optionsSelected().filter(x => x !== id)
                doSetOptionsSelected(newSelected)
              }}>
                <i class="icon-trash"></i>
              </button>
            </div>
          }
        }
      </For>   
    </div>
    <SearchMobileLayer id={id} options={props.options} keys={props.keys}
      avoidIDs={optionsSelected()}
      onSelect={e => {
        if(!e){ return }
        optionsSelected().push(e[keyId] as (number|string))
        doSetOptionsSelected(optionsSelected())
      }}
    />
  </div>
}

export const [openSearchLayer, setOpenSearchLayer] = createSignal(0)

export interface ISearchMobileLayer<T> {
  id: number
  css?: string
  options: T[]
  keys: string
  label?: string
  placeholder?: string
  max?: number
  onSelect?: (e: T) => void
  selected?: (number|string)
  notEmpty?: boolean
  required?: boolean
  clearOnSelect?: boolean
  avoidIDs?: (number|string)[]
  inputCss?: string
}

export function SearchMobileLayer<T>(props: ISearchMobileLayer<T>) {

  const show = () => {
    return openSearchLayer() === props.id 
  }

  let inputRef: HTMLInputElement = undefined as unknown as HTMLInputElement
  let onMouseStatus = 1

  const [keyId, keyName] = props.keys.split(".") as [keyof T, keyof T]
  const [filteredOptions, setFilteredOptions] = createSignal([...props.options])
  const [words, setWords] = createSignal([] as string[])

  const filter = (text: string) => {
    if(!text && ((props.avoidIDs?.length||0) === 0)){ return [...props.options] }
    const avoidIDSet = new Set(props.avoidIDs||[])

    const filteredOptions: T[] = []
    for(let op of props.options){
      if(avoidIDSet.size > 0 && avoidIDSet.has(op[keyId as keyof T] as number)){ 
        continue 
      }
      const name = op[keyName] as string
      if(typeof name === 'string'){
        const nameN = name.toLowerCase()
        if(!text || nameN.includes(text)){ filteredOptions.push(op) }
      }
    }
    return filteredOptions
  }

  const onKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    const target = ev.target as HTMLInputElement

    throttle(() => {
      setWords(String(inputRef.value).split(" ").map(x => x.toLocaleLowerCase()))
      setFilteredOptions(filter(String(target.value||"").toLowerCase()))
    },120)
  }
  
  createEffect(() => {
    if(show()){
      inputRef?.focus()
      setWords([])
      setFilteredOptions(filter(""))
      onMouseStatus = 0 
    }
  })

  return <Show when={show()}>
    <Portal mount={document.body}>
      <div class="search-background">
        <div class="p-rel w100 px-08 mt-06">
          <i class="i-search icon-search p-abs"></i>
          <input class="in-5 s2 increment" onkeyup={onKeyUp} ref={inputRef}
            style={{ "margin-bottom": '6px' }}
            onBlur={ev => {
              ev.stopPropagation()
              console.log("blur:: mouse status", onMouseStatus)
              if(onMouseStatus === 1){ 
                onMouseStatus = 0
              } else {
                setOpenSearchLayer(0)
              }
            }}
          />
        </div>
        <div class="w100 px-08 options-c2">
          <For each={filteredOptions()}>
            {(opt,i) => {
              let cN = "flex ai-center jc-center card-c2"
              if(i() % 2 === 0){ cN += " mr-08" }
              return <div class={cN} style={{ width: '100%' }}
                onMouseDown={ev => {
                  ev.stopPropagation()
                  onMouseStatus = 1
                }}
                onClick={ev => {
                  ev.stopPropagation()
                  if(props.onSelect){ props.onSelect(opt) }
                  setOpenSearchLayer(0)
                }}
              >
                <div class="_highlight s1">
                  {highlString(opt[keyName] as unknown as string, words())}
                </div>
              </div>
            }}
          </For>
        </div>
      </div>
    </Portal>
  </Show>
}

