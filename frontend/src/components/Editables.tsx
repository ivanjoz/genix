import { createEffect, createSignal } from "solid-js";

export interface ICellEditable<T> {
  saveOn: T,
  save: string
  class?: string
  contentClass?: string
  onChange?: (newValue: (string|number)) => void
}

export function CellEditable<T>(props: ICellEditable<T>) {
  const [isEditing, setIsEditing] = createSignal(false)
  const [currentValue, setCurrentValue] = createSignal(
    props.saveOn[props.save as keyof T] as (number|string)
  )

  const reSetValue = (newValue?: (string|number)) => {
    props.saveOn[props.save as keyof T] = newValue as any
    setCurrentValue(newValue as any)
  }

  let inputRef: HTMLInputElement = undefined as unknown as HTMLInputElement
  let prevValue = currentValue()
  
  createEffect(()=> {
    if(isEditing() && inputRef){ inputRef.focus() }
  })

  return <div class={"cell-ed-c flex ai-center p-rel " +(props.class||"")}>
    <div class={"h100 w100 " +(props.contentClass||"")} 
      style={{ visibility: isEditing() ? "hidden" : "visible" }}
      onClick={ev =>{
        ev.stopPropagation()
        prevValue = currentValue()
        setIsEditing(true)
      }}>
      { currentValue() }
    </div>
    { isEditing() && 
      <div class="flex ai-center cell-ed h100 w100">
        <input value={currentValue()||""} type="text" ref={inputRef}
          class="w100"
          autofocus={true}
          onKeyUp={ev => {
            ev.stopPropagation()
            const target = ev.target as HTMLInputElement
            reSetValue(target.value)
          }}
          onBlur={ev => {
            ev.stopPropagation()
            setIsEditing(false)
            const newValue = currentValue()
            if(prevValue !== newValue){
              prevValue = newValue
              if(props.onChange){ props.onChange(newValue) }
            }
          }}
        />
      </div>
    }
  </div>
}
