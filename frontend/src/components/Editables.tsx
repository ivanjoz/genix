import { For, JSX, createEffect, createSignal } from "solid-js";

export interface ICellEditable<T> {
  saveOn: T,
  save: string
  class?: string
  contentClass?: string
  inputClass?: string
  onChange?: (newValue: (string|number)) => void
  render?: (value: (number|string), isEditing: boolean) => JSX.Element
  getValue?: (e: T) => (number|string)
  required?: boolean
  type?: string
}

export function CellEditable<T>(props: ICellEditable<T>) {
  const [isEditing, setIsEditing] = createSignal(false)

  let prevValue = props.getValue
    ? props.getValue(props.saveOn)
    : (props.saveOn||{} as T)[props.save as keyof T] as (number|string)
  
  const [currentValue, setCurrentValue] = createSignal(prevValue)

  const extractValue = (newValue?: (string|number)) => {
    if(props.type === 'number'){ 
      newValue = parseFloat(newValue as string||'0') 
    }
    return newValue
  }

  const reSetValue = (newValue?: (string|number)) => {
    props.saveOn[props.save as keyof T] = extractValue(newValue) as any
    setCurrentValue(newValue as any)
  }

  let inputRef: HTMLInputElement = undefined as unknown as HTMLInputElement
  
  createEffect(()=> {
    if(isEditing() && inputRef){ inputRef.focus() }
  })

  const renderContent = () => {
    return props.render ? props.render(currentValue(), isEditing()) : currentValue() 
  }

  return <div class={"cell-ed-c flex ai-center p-rel " +(props.class||"")}>
    <div class={"h100 w100 " +(props.contentClass||"")} 
      style={{ visibility: isEditing() ? "hidden" : "visible" }}
      onClick={ev =>{
        ev.stopPropagation()
        prevValue = currentValue()
        setIsEditing(true)
      }}>
      { renderContent() }
      { !currentValue() && props.required &&
        <i class="icon-attention c-red"></i>
      }
    </div>
    { isEditing() && 
      <div class="flex ai-center cell-ed h100 w100">
        <input value={currentValue()||""} type={props.type || "text"}
          class={"w100 " + (props.inputClass||"")}
          autofocus={true} ref={inputRef}
          onKeyUp={ev => {
            ev.stopPropagation()
            reSetValue((ev.target as HTMLInputElement).value)
          }}
          onBlur={ev => {
            ev.stopPropagation()
            
            const newValue = extractValue(ev.target.value)
            if(prevValue !== newValue){
              prevValue = newValue
              if(props.onChange){ props.onChange(newValue) }
            }
            setIsEditing(false)
          }}
        />
      </div>
    }
  </div>
}

interface ICellTextOptions<T,Y> {
  saveOn: T
  class?: string
  save?: string
  getValue?: (e: Y) => string
  setValue?: (e: Y, value: string) => Y
  skipt?: (e: Y) => boolean
}

export function CellTextOptions<T,Y>(props: ICellTextOptions<T,Y>) {
  const getValue = props.getValue || ((e: Y) => e)  
  const setValue = props.setValue || ((_: Y, v: string) => v)  

  const values = ((props.saveOn[props.save as keyof T] || []) as (Y|string)[]
    ).filter(x => x)
  
  values.push(setValue({} as Y,""))

  const [currentValues, setCurrentValues] = createSignal(values)

  return <div class={"flex-wrap ai-center p-rel " +(props.class||"")}>
    { currentValues().map((e,i) => {
      if(props.skipt && props.skipt(e as Y)){ return null }
      return <CellTextOption 
        defaultValue={getValue(e as Y) as string}
        onBlur={content => {
          let values = [...currentValues()]
          values[i] = setValue(values[i] as Y, content)
          values = values.filter(x => getValue(x as Y))
          props.saveOn[props.save as keyof T] = values as never
          values.push(setValue({} as Y,""))
          console.log("nuevos valores:: ", [...values])
          setCurrentValues(values)
        }}/>
      })
    }
  </div>
}

interface ICellTextOption {
  defaultValue?: string
  onBlur: ((e: string) => void)
}

export function CellTextOption<Y>(props: ICellTextOption) {

  const [value, setValue] = createSignal(props.defaultValue)
  const [show, setShow] = createSignal(false)

  createEffect(() => {
    setValue(props.defaultValue)
  })

  let input: HTMLInputElement

  createEffect(() => {
    if(show() && input){ input.focus() }
  })

  return <div class="card-c3 flex ai-center mt-03 mb-03 ml-04 mr-04 p-rel overflow-hidden"
    onClick={ev => {
      ev.stopPropagation()
      setShow(true)
      console.log("hizo click()")
    }}
  >
    { show() && 
      <input class="p-abs" autofocus={true} ref={input}
        style={{ "padding-right": '0' }}
        value={value()}
        onKeyUp={ev => {
          ev.stopPropagation()
          setValue(ev.target['value'])
        }}
        onBlur={()=> {
          props.onBlur(value())
          setShow(false)
        }}
      /> 
    }
    <div class="nowrap px-06 mt-01"
      style={{ 'padding-right': show() ? '0.8rem' : undefined }}
    >
      { value() }
    </div>
  </div>
}