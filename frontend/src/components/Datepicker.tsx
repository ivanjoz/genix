import { addDays, addMonths, endOfMonth, format, getDay, getISOWeek, getISOWeekYear, getMonth, isFirstDayOfMonth, startOfISOWeek, startOfMonth } from "date-fns";
import { For, Show, createEffect, createSignal, on } from "solid-js";
import { throttle } from "~/core/main";
import { arrayToMapN } from "~/shared/main";
import { Params } from "~/shared/security";

export interface IDatePicker<T> {
  label?: string
  css?: string
  saveOn?: any
  save?: string | keyof T
  onChange?: (e: T) => void
  required?: boolean
  clearOnSelect?: boolean
  inputCss?: string
}

const weekDaysNames = [
  { n: 1, name: 'LU' },
  { n: 2, name: 'MA' },
  { n: 3, name: 'MI' },
  { n: 4, name: 'JU' },
  { n: 5, name: 'VI' },
  { n: 6, name: 'SA' },
  { n: 7, name: 'DO' },
]

const monthsNames = [
  { n: 1, name: 'Enero' },
  { n: 2, name: 'Febrero' },
  { n: 3, name: 'Marzo' },
  { n: 4, name: 'Abril' },
  { n: 5, name: 'Mayo' },
  { n: 6, name: 'Junio' },
  { n: 7, name: 'Julio' },
  { n: 8, name: 'Agusto' },
  { n: 9, name: 'Septiembre' },
  { n: 10, name: 'Octubre' },
  { n: 11, name: 'Noviembre' },
  { n: 12, name: 'Diciembre' },
]

export function DatePicker<T>(props: IDatePicker<T>) {
  const fechaToday = new Date()
  const offset = fechaToday.getTimezoneOffset() * 60
  const fechaTodayUnix = Params.getFechaUnix()
  const month_ = fechaToday.getFullYear() * 100 + (fechaToday.getMonth() + 1)

  const [monthSelected, setMonthSelected] = createSignal(month_)
  const [fechaSelected, setFechaSelected] = createSignal(0)
  const [fechaFocus, setFechaFocus] = createSignal(0)
  const [showCalendar, setShowCalendar] = createSignal(false)

  const parseMonth = (yearMonth: number) => {
    const yearMonthString = String(yearMonth)
    const year = parseInt(yearMonthString.substring(0,4))
    const month = parseInt(yearMonthString.substring(4,6))
    const name = monthsNamesMap.get(month)?.name || "-"
    return { name, year, month }
  }

  const monthsNamesMap = arrayToMapN(monthsNames,'n')

  const semanasDias = () => {
    let fecha: Date
    if(monthSelected()){
      const { year, month } = parseMonth(monthSelected())
      fecha = new Date(year, month - 1, 1, 0, 0, 0)
    } else {
      fecha = new Date()
    }
    
    const monthStart = startOfMonth(fecha)
    const monthEnd = endOfMonth(fecha)
    const monthCurrent = fecha.getMonth()
    let fechaStart = startOfISOWeek(monthStart)

    const semanas = []
    while(fechaStart.getTime() <= monthEnd.getTime()){
      const fechaStarTime = fechaStart.getTime()
      
      const fechaStartUnix = Math.floor((fechaStarTime - (offset * 1000)) / 86400000)
      // const fechaStartUnix = (fechaStarTime - (offset * 1000)) / 86400000
      const year = getISOWeekYear(fechaStart)
      const week = getISOWeek(fechaStart)
      const weekDays = []
      for(let i = 0; i < 7; i++){
        const date = new Date(fechaStarTime + (86400000 * i))
        const fechaUnix = fechaStartUnix + i
        const day = date.getDate()
        const month = date.getFullYear() * 100 + (date.getMonth() + 1)
        weekDays.push({ date, fechaUnix, day, month })
      }
      semanas.push({ year, week, weekDays, monthCurrent })
      fechaStart = new Date(fechaStarTime + (86400000 * 7))
    }

    console.log("semanas:: ", semanas)
    return semanas
  }

  const monthName = () => {
    return parseMonth(monthSelected())
  }

  let avoidCloseOnBlur = false
  let input: HTMLInputElement

  const changeMonth = (count: number) => {
    const mn = monthName()
    const fecha = new Date(mn.year, mn.month - 1, 1, 0, 0, 0)
    const fechaNew = addMonths(fecha, count)
    const month = fechaNew.getFullYear() * 100 + (fechaNew.getMonth() + 1)
    setMonthSelected(month)
    input.focus()
  }

  const makeButton = (icon: string, count: number) => {
    return <div class="h2 bn-d1 flex-center" 
      onMouseDown={ev => {
        ev.stopPropagation()
        avoidCloseOnBlur = true
      }}
      onClick={ev => {
        ev.stopPropagation()
        changeMonth(count)
      }}>{icon}</div>
  }

  const makeFechaFormat = (fechaUnix: number): string => {
    if(!fechaUnix){ return "" }
    const fechaHoraUnix = fechaUnix * 86400 + offset
    return format(fechaHoraUnix * 1000, "dd-MM-yyyy")
  }

  const parseValue = (value: string) => {
    value = value.trim().substring(0, 10).replaceAll("/","-")
    const todayYear = String(fechaToday.getFullYear()) 

    const arrValue = value.split("-").filter(x => x)
    console.log("arr values::", arrValue, value)

    let day = isNaN(arrValue[0] as unknown as number) ? 0 : parseInt(arrValue[0])
    let month = isNaN(arrValue[1] as unknown as number) ? 0 : parseInt(arrValue[1])

    let yearString = arrValue[2] || ""
    if (yearString.length !== 4){
      for (let i = 0; i < todayYear.length; i++) {
        if (!yearString[i]) { yearString += todayYear[i] }
      }
    }

    let year = isNaN(yearString as unknown as number) ? 0 : parseInt(yearString)

    const isCompleted = day && month && yearString.length === 4
    let fechaUnixAutocomplated = 0
    let fechaAutocomplated: Date = null

    if(day){
      if(!month){ month = fechaToday.getMonth() + 1 }
      if(!year){ year = fechaToday.getFullYear() }
      fechaAutocomplated = new Date(year, month - 1, day, 0, 0, 0)
      if(fechaAutocomplated.getTime){
        fechaUnixAutocomplated = 
          Math.floor((fechaAutocomplated.getTime() - (offset * 1000)) / 86400000)
      } else {
        fechaAutocomplated = null
      }
    }

    return { isCompleted, fechaUnixAutocomplated, fechaAutocomplated, day, month, year }
  }

  const setAutocompletedValue = (value: string) => {
    const acv = parseValue(value)
    console.log("autocompleted value::", acv)
    if(acv.fechaAutocomplated){
      const month_ = acv.fechaAutocomplated.getFullYear() * 100 + (acv.fechaAutocomplated.getMonth() + 1)
      setMonthSelected(month_)
      setFechaFocus(acv.fechaUnixAutocomplated)
    } else {
      setFechaFocus(0)
    }
  }

  const changeFechaSelected = (fechaUnix: number) => {
    if(props.save && props.saveOn){
      props.saveOn[props.save] = fechaUnix
    }
    setFechaSelected(fechaUnix)
  }
  
  createEffect(() => {
    if(props.saveOn && props.save){
      const fechaUnix = props.saveOn[props.save]
      if(fechaUnix){
        const value = makeFechaFormat(fechaUnix)
        setAutocompletedValue(value)
        input.value = value
      } else {
        input.value = ""
        setMonthSelected(0)
      }
      setFechaSelected(0)
    }
  })

  const regexKeys = new Set(['1','2','3','4','5','6','7','8','9','0','-','/'])
  const regexKeysPress = new Set([...(regexKeys),'Backspace','Control','c','v','x'])

  let cN = "in-5c p-rel flex-column a-start"
  if(props.css){ cN += " " + props.css }

  return <div class={cN}>
    <div class="label">{props.label}</div>
    <input ref={input} type="text" class="ff-mono in-5" 
      value={makeFechaFormat(fechaSelected())}
      placeholder="DD-MM-YYYY"
      onFocus={ev => {
        ev.stopPropagation()
        setShowCalendar(true)
      }}
      onBlur={ev => {
        ev.stopPropagation()
        if(avoidCloseOnBlur){ avoidCloseOnBlur = false; return }
        setShowCalendar(false)
        if(fechaFocus() !== 0){
          const value = (ev.target.value||"").trim()
          const acv = parseValue(value)
          console.log("autocompleted value::", acv, value)
          if(value.length === 10 && acv.isCompleted){
            changeFechaSelected(acv.fechaUnixAutocomplated)
          } else {
            ev.target.value = ""
            if(props.saveOn && props.save){
              delete props.saveOn[props.save]
            }
          }
          setFechaFocus(0)
        }
      }}
      onkeydown={ev => {
        console.log(`User pressed: ${ev.key} | ${ev.altKey}`);
        if(!regexKeysPress.has(ev.key)){
          ev.preventDefault()
        }
      }}
      onKeyUp={ev => {
        ev.stopPropagation()
        let value = (ev.target as HTMLInputElement).value
        let valueCleaned = ""
        for(let key of value){
          if(regexKeys.has(key)){ valueCleaned += key }
        }
        if(value !== valueCleaned){
          (ev.target as HTMLInputElement).value = valueCleaned
        }
        throttle(() => { setAutocompletedValue(valueCleaned) },150)
      }}
    />
    <Show when={showCalendar()}>
      <div class="date-picker-c" onMouseLeave={ev => {
        ev.stopPropagation()
        if(input !== document.activeElement){ 
          avoidCloseOnBlur = false
          setShowCalendar(false)
          setFechaFocus(0)
        }
      }}>
        <div class="flex jc-between ai-center mb-02">
          { makeButton("«", -12) }
          { makeButton("‹", -1) }
          <div class="bn-d2 flex-center">
            <div class="mr-04">{ monthName().name }</div>
            <div>{ monthName().year }</div>
          </div>
          { makeButton("›", 1) }
          { makeButton("»", 12) }
        </div>
        <div class="flex">
          <div class="dp-week base h6 ff-bold c-purple"></div>
          { weekDaysNames.map(e => {
              return <div class="dp-col ta-c flex-center h6 ff-bold">{e.name}</div>
            })
          } 
        </div>
        <For each={semanasDias()}>
        {week => {
          return <div class="flex">
            <div class="dp-week h6 ff-bold ta-c flex-center c-purple">{week.week}</div>
            { week.weekDays.map(e => {
                let cN = "p-rel dp-day ta-c flex-center"
                if(e.month !== monthSelected()){ cN += " is-out" }
                
                if(e.fechaUnix === fechaSelected()){ cN += " selected" }
                else if(e.fechaUnix === fechaFocus()){ cN += " focused" }

                return <div class={cN}
                  onClick={ev => {
                    ev.stopPropagation()
                    changeFechaSelected(e.fechaUnix)
                    setShowCalendar(false)
                    setFechaFocus(0)
                    avoidCloseOnBlur = false
                  }}
                  onMouseDown={ev => {
                    avoidCloseOnBlur = true
                    console.log("onMouseDown close on blur:: " + avoidCloseOnBlur)
                    ev.stopPropagation()
                  }}
                >
                  {e.day}
                  { fechaTodayUnix === e.fechaUnix &&
                    <div class="ln-today"></div>
                  }
                </div>
              })
            }
          </div>
        }}
        </For>
      </div>
    </Show>
  </div>
}
