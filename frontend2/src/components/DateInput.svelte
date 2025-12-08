<script lang="ts" module>
  export interface IDateInputProps<T> {
    saveOn?: T
    save?: keyof T
    label?: string
    css?: string
    inputCss?: string
    placeholder?: string
    required?: boolean
    disabled?: boolean
    onChange?: () => void
    type?: "unix" | "sunix"
  }

  interface WeekDay {
    date: Date
    fechaUnix: number
    day: number
    month: number
  }

  interface Week {
    year: number
    week: number
    weekDays: WeekDay[]
    monthCurrent: number
  }
</script>

<script lang="ts" generics="T">
  import { untrack } from "svelte";
  import s1 from "./components.module.css";

  let {
    saveOn = $bindable(),
    save,
    label = "",
    css = "",
    inputCss = "",
    placeholder = "DD-MM-YYYY",
    required = false,
    disabled = false,
    onChange,
    type = "unix"
  }: IDateInputProps<T> = $props()

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
    { n: 8, name: 'Agosto' },
    { n: 9, name: 'Septiembre' },
    { n: 10, name: 'Octubre' },
    { n: 11, name: 'Noviembre' },
    { n: 12, name: 'Diciembre' },
  ]

  const monthsNamesMap = new Map(monthsNames.map(m => [m.n, m]))

  const fechaToday = new Date()
  const offset = fechaToday.getTimezoneOffset() * 60
  const fechaTodayUnix = Math.floor((fechaToday.getTime() - (offset * 1000)) / 86400000)
  const month_ = fechaToday.getFullYear() * 100 + (fechaToday.getMonth() + 1)

  let monthSelected = $state(month_)
  let fechaSelected = $state(0)
  let fechaFocus = $state(0)
  let showCalendar = $state(false)
  let inputValue = $state("")
  let avoidCloseOnBlur = false
  let inputElement: HTMLInputElement

  const parseMonth = (yearMonth: number) => {
    const yearMonthString = String(yearMonth)
    const year = parseInt(yearMonthString.substring(0, 4))
    const month = parseInt(yearMonthString.substring(4, 6))
    const name = monthsNamesMap.get(month)?.name || "-"
    return { name, year, month }
  }

  const startOfISOWeek = (date: Date): Date => {
    const d = new Date(date)
    const day = d.getDay()
    const diff = (day === 0 ? -6 : 1) - day
    d.setDate(d.getDate() + diff)
    d.setHours(0, 0, 0, 0)
    return d
  }

  const getISOWeek = (date: Date): number => {
    const d = new Date(date)
    d.setHours(0, 0, 0, 0)
    d.setDate(d.getDate() + 4 - (d.getDay() || 7))
    const yearStart = new Date(d.getFullYear(), 0, 1)
    return Math.ceil((((d.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
  }

  const getISOWeekYear = (date: Date): number => {
    const d = new Date(date)
    d.setDate(d.getDate() + 4 - (d.getDay() || 7))
    return d.getFullYear()
  }

  const semanasDias = $derived.by((): Week[] => {
    let fecha: Date
    if (monthSelected) {
      const { year, month } = parseMonth(monthSelected)
      fecha = new Date(year, month - 1, 1, 0, 0, 0)
    } else {
      fecha = new Date()
    }

    const monthStart = new Date(fecha.getFullYear(), fecha.getMonth(), 1)
    const monthEnd = new Date(fecha.getFullYear(), fecha.getMonth() + 1, 0)
    const monthCurrent = fecha.getMonth()
    let fechaStart = startOfISOWeek(monthStart)

    const semanas: Week[] = []
    while (fechaStart.getTime() <= monthEnd.getTime()) {
      const fechaStarTime = fechaStart.getTime()
      const fechaStartUnix = Math.floor((fechaStarTime - (offset * 1000)) / 86400000)
      const year = getISOWeekYear(fechaStart)
      const week = getISOWeek(fechaStart)
      const weekDays: WeekDay[] = []
      
      for (let i = 0; i < 7; i++) {
        const date = new Date(fechaStarTime + (86400000 * i))
        const fechaUnix = fechaStartUnix + i
        const day = date.getDate()
        const month = date.getFullYear() * 100 + (date.getMonth() + 1)
        weekDays.push({ date, fechaUnix, day, month })
      }
      semanas.push({ year, week, weekDays, monthCurrent })
      fechaStart = new Date(fechaStarTime + (86400000 * 7))
    }
    return semanas
  })

  const monthName = $derived(parseMonth(monthSelected))

  const changeMonth = (count: number) => {
    const mn = monthName
    const fecha = new Date(mn.year, mn.month - 1, 1, 0, 0, 0)
    fecha.setMonth(fecha.getMonth() + count)
    const month = fecha.getFullYear() * 100 + (fecha.getMonth() + 1)
    monthSelected = month
    inputElement?.focus()
  }

  const makeFechaFormat = (fechaUnix: number): string => {
    if (!fechaUnix) return ""
    const fechaHoraUnix = fechaUnix * 86400 + offset
    const date = new Date(fechaHoraUnix * 1000)
    const day = String(date.getDate()).padStart(2, '0')
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const year = date.getFullYear()
    return `${day}-${month}-${year}`
  }

  const parseValue = (value: string) => {
    value = value.trim().substring(0, 10).replaceAll("/", "-")
    const todayYear = String(fechaToday.getFullYear())

    const arrValue = value.split("-").filter(x => x)
    
    let day = isNaN(arrValue[0] as unknown as number) ? 0 : parseInt(arrValue[0])
    let month = isNaN(arrValue[1] as unknown as number) ? 0 : parseInt(arrValue[1])

    let yearString = arrValue[2] || ""
    if (yearString.length !== 4) {
      for (let i = 0; i < todayYear.length; i++) {
        if (!yearString[i]) yearString += todayYear[i]
      }
    }

    let year = isNaN(yearString as unknown as number) ? 0 : parseInt(yearString)

    const isCompleted = day && month && yearString.length === 4
    let fechaUnixAutocomplated = 0
    let fechaAutocomplated: Date | null = null

    if (day) {
      if (!month) month = fechaToday.getMonth() + 1
      if (!year) year = fechaToday.getFullYear()
      fechaAutocomplated = new Date(year, month - 1, day, 0, 0, 0)
      if (fechaAutocomplated.getTime) {
        fechaUnixAutocomplated = Math.floor((fechaAutocomplated.getTime() - (offset * 1000)) / 86400000)
      } else {
        fechaAutocomplated = null
      }
    }

    return { isCompleted, fechaUnixAutocomplated, fechaAutocomplated, day, month, year }
  }

  const setAutocompletedValue = (value: string) => {
    const acv = parseValue(value)
    if (acv.fechaAutocomplated) {
      const month_ = acv.fechaAutocomplated.getFullYear() * 100 + (acv.fechaAutocomplated.getMonth() + 1)
      monthSelected = month_
      fechaFocus = acv.fechaUnixAutocomplated
    } else {
      fechaFocus = 0
    }
  }

  const changeFechaSelected = (fechaUnix: number) => {
    untrack(() => {
      if (save && saveOn) {
        saveOn[save] = fechaUnix as NonNullable<T>[keyof T]
      }
    })
    fechaSelected = fechaUnix
  }

  const regexKeys = new Set(['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '/'])
  const regexKeysPress = new Set([...regexKeys, 'Backspace', 'Control', 'c', 'v', 'x', 'Tab'])

  let timeoutId: number | undefined
  const throttle = (fn: () => void, delay: number) => {
    if (timeoutId) clearTimeout(timeoutId)
    timeoutId = setTimeout(fn, delay) as unknown as number
  }

  const handleKeyDown = (ev: KeyboardEvent) => {
    if (!regexKeysPress.has(ev.key)) {
      ev.preventDefault()
    }
  }

  const handleKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    let value = (ev.target as HTMLInputElement).value
    let valueCleaned = ""
    for (let key of value) {
      if (regexKeys.has(key)) valueCleaned += key
    }
    if (value !== valueCleaned) {
      (ev.target as HTMLInputElement).value = valueCleaned
    }
    inputValue = valueCleaned
    throttle(() => { setAutocompletedValue(valueCleaned) }, 150)
  }

  const handleFocus = (ev: FocusEvent) => {
    ev.stopPropagation()
    showCalendar = true
  }

  const handleBlur = (ev: FocusEvent) => {
    ev.stopPropagation()
    if (avoidCloseOnBlur) {
      avoidCloseOnBlur = false
      return
    }
    showCalendar = false
    if (fechaFocus !== 0) {
      const value = ((ev.target as HTMLInputElement).value || "").trim()
      const acv = parseValue(value)
      if (value.length === 10 && acv.isCompleted) {
        changeFechaSelected(acv.fechaUnixAutocomplated)
        if (onChange) onChange()
      } else {
        (ev.target as HTMLInputElement).value = ""
        if (save && saveOn) {
          delete saveOn[save]
        }
      }
      fechaFocus = 0
    }
  }

  // Effect to sync with external changes
  $effect(() => {
    if (saveOn && save) {
      const fechaUnix = saveOn[save] as number
      if (fechaUnix) {
        const value = makeFechaFormat(fechaUnix)
        setAutocompletedValue(value)
        inputValue = value
        if (inputElement) inputElement.value = value
      } else {
        inputValue = ""
        if (inputElement) inputElement.value = ""
        monthSelected = month_
      }
      fechaSelected = fechaUnix || 0
    }
  })

  let cN = $derived(`${s1.input} relative date-input-container` + (css ? " " + css : ""))
</script>

{#if label}
  <div class={cN}>
    <div class={s1.input_lab_cell_left}><div></div></div>
    <div class={s1.input_lab}>{label}</div>
    <div class={s1.input_lab_cell_right}><div></div></div>
    <div class={s1.input_shadow_layer}>
      <div></div>
    </div>
    <div class={`${s1.input_div} flex w-full`}>
      <div class={s1.input_div_1}>
        <div></div>
      </div>
      <input
        bind:this={inputElement}
        type="text"
        class="w-full {s1.input_inp} ff-mono {inputCss || ""}"
        value={inputValue}
        placeholder={placeholder}
        disabled={disabled}
        onfocus={handleFocus}
        onblur={handleBlur}
        onkeydown={handleKeyDown}
        onkeyup={handleKeyUp}
      />
    </div>

    {#if showCalendar}
      <div class="date-picker-c" onmouseleave={(ev) => {
        ev.stopPropagation()
        if (inputElement !== document.activeElement) {
          avoidCloseOnBlur = false
          showCalendar = false
          fechaFocus = 0
        }
      }}>
        <div class="flex justify-between items-center mb-[2px]">
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(-12)
            }}>«</div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(-1)
            }}>‹</div>
          <div class="bn-d2 flex items-center justify-center">
            <div class="mr-[4px]">{monthName.name}</div>
            <div>{monthName.year}</div>
          </div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(1)
            }}>›</div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(12)
            }}>»</div>
        </div>
        <div class="flex">
          <div class="dp-week base text-[13px] ff-bold c-purple"></div>
          {#each weekDaysNames as dayName}
            <div class="dp-col text-center flex items-center justify-center text-[13px] ff-bold">{dayName.name}</div>
          {/each}
        </div>
        {#each semanasDias as week}
          <div class="flex">
            <div class="dp-week text-[13px] ff-bold text-center flex items-center justify-center c-purple">{week.week}</div>
            {#each week.weekDays as day}
              {@const isOutMonth = day.month !== monthSelected}
              {@const isSelected = day.fechaUnix === fechaSelected}
              {@const isFocused = day.fechaUnix === fechaFocus}
              {@const isToday = fechaTodayUnix === day.fechaUnix}
              <div
                class="relative dp-day text-center flex items-center justify-center {isOutMonth ? 'is-out' : ''} {isSelected ? 'selected' : ''} {isFocused ? 'focused' : ''}"
                onclick={(ev) => {
                  ev.stopPropagation()
                  changeFechaSelected(day.fechaUnix)
                  showCalendar = false
                  fechaFocus = 0
                  avoidCloseOnBlur = false
                  if (onChange) onChange()
                }}
                onmousedown={(ev) => {
                  avoidCloseOnBlur = true
                  ev.stopPropagation()
                }}
              >
                {day.day}
                {#if isToday}
                  <div class="ln-today"></div>
                {/if}
              </div>
            {/each}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else}
  <div class={`${s1.input} no-label relative date-input-container` + (css ? " " + css : "")}>
    <div class={s1.input_shadow_layer}>
      <div></div>
    </div>
    <div class={`${s1.input_div} flex w-full`}>
      <div class={s1.input_div_1}>
        <div></div>
      </div>
      <input
        bind:this={inputElement}
        type="text"
        class="w-full {s1.input_inp} ff-mono {inputCss || ""}"
        value={inputValue}
        placeholder={placeholder}
        disabled={disabled}
        onfocus={handleFocus}
        onblur={handleBlur}
        onkeydown={handleKeyDown}
        onkeyup={handleKeyUp}
      />
    </div>

    {#if showCalendar}
      <div class="date-picker-c" onmouseleave={(ev) => {
        ev.stopPropagation()
        if (inputElement !== document.activeElement) {
          avoidCloseOnBlur = false
          showCalendar = false
          fechaFocus = 0
        }
      }}>
        <div class="flex justify-between items-center mb-[2px]">
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(-12)
            }}>«</div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(-1)
            }}>‹</div>
          <div class="bn-d2 flex items-center justify-center">
            <div class="mr-[4px]">{monthName.name}</div>
            <div>{monthName.year}</div>
          </div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(1)
            }}>›</div>
          <div class="h2 bn-d1 flex items-center justify-center"
            onmousedown={(ev) => {
              ev.stopPropagation()
              avoidCloseOnBlur = true
            }}
            onclick={(ev) => {
              ev.stopPropagation()
              changeMonth(12)
            }}>»</div>
        </div>
        <div class="flex">
          <div class="dp-week base text-[13px] ff-bold c-purple"></div>
          {#each weekDaysNames as dayName}
            <div class="dp-col text-center flex items-center justify-center text-[13px] ff-bold">{dayName.name}</div>
          {/each}
        </div>
        {#each semanasDias as week}
          <div class="flex">
            <div class="dp-week text-[13px] ff-bold text-center flex items-center justify-center c-purple">{week.week}</div>
            {#each week.weekDays as day}
              {@const isOutMonth = day.month !== monthSelected}
              {@const isSelected = day.fechaUnix === fechaSelected}
              {@const isFocused = day.fechaUnix === fechaFocus}
              {@const isToday = fechaTodayUnix === day.fechaUnix}
              <div
                class="relative dp-day text-center flex items-center justify-center {isOutMonth ? 'is-out' : ''} {isSelected ? 'selected' : ''} {isFocused ? 'focused' : ''}"
                onclick={(ev) => {
                  ev.stopPropagation()
                  changeFechaSelected(day.fechaUnix)
                  showCalendar = false
                  fechaFocus = 0
                  avoidCloseOnBlur = false
                  if (onChange) onChange()
                }}
                onmousedown={(ev) => {
                  avoidCloseOnBlur = true
                  ev.stopPropagation()
                }}
              >
                {day.day}
                {#if isToday}
                  <div class="ln-today"></div>
                {/if}
              </div>
            {/each}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .date-input-container {
    position: relative;
  }

  .date-picker-c {
    position: absolute;
    top: 100%;
    left: 0;
    margin-top: 4px;
    background: white;
    border: 1px solid #c1c5dc;
    border-radius: 8px;
    padding: 12px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 1000;
    min-width: 280px;
  }

  .dp-week {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
  }

  .dp-col {
    width: 32px;
    height: 28px;
    flex-shrink: 0;
  }

  .dp-day {
    width: 32px;
    height: 32px;
    flex-shrink: 0;
    cursor: pointer;
    border-radius: 4px;
    transition: background-color 0.2s;
    font-size: 14px;
  }

  .dp-day:hover {
    background-color: #f0f0f5;
  }

  .dp-day.is-out {
    color: #b0b0c0;
  }

  .dp-day.selected {
    background-color: #6d5dad;
    color: white;
    font-weight: 600;
  }

  .dp-day.focused {
    background-color: #e8e7f5;
    outline: 2px solid #9794d6;
  }

  .ln-today {
    position: absolute;
    bottom: 2px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    background-color: #6d5dad;
    border-radius: 50%;
  }

  .dp-day.selected .ln-today {
    background-color: white;
  }

  .bn-d1, .bn-d2 {
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
    user-select: none;
  }

  .bn-d1:hover, .bn-d2:hover {
    background-color: #f0f0f5;
  }

  .bn-d2 {
    font-weight: 600;
    color: #6d5dad;
  }
</style>