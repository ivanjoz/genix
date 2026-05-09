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
    // Render the calendar through Popover2 (Portal-to-body) so it can escape clipped/overflow ancestors like table cells.
    usePopover?: boolean
    // Strip chrome (background/border/outline/shadow); fills parent with w-full h-full. For table-cell embedding.
    useInlineStyle?: boolean
  }
</script>

<script lang="ts" generics="T">
  import { Core } from "$core/store.svelte";
  import { untrack } from "svelte";
  import Popover2 from "$components/popover2/Popover2.svelte";
  import {
    buildCalendarWeeks,
    createDateInputContext,
    dateFromUnixDay,
    formatUnixDay,
    getMonthKey,
    parseMonthKey,
    parseTypedDate,
    weekDaysNames,
  } from "./date-input.helpers";
  import s1 from "./components.module.css";
  import { Env } from "$core/env";
  import { Agent } from "$core/agent/registry";

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
    type = "unix",
    usePopover = false,
    useInlineStyle = false,
  }: IDateInputProps<T> = $props()

  const {
    todayDate,
    timezoneOffsetSeconds,
    todayUnixDay: fechaTodayUnix,
    currentMonthKey,
  } = createDateInputContext()

  let monthSelected = $state(currentMonthKey)
  let fechaSelected = $state(0)
  let fechaFocus = $state(0)
  let showCalendar = $state(false)
  let inputValue = $state("")
  let avoidCloseOnBlur = false
  let inputElement = $state<HTMLInputElement>()
  const isMobile = $derived(Core.deviceType === 3)

  const semanasDias = $derived.by(() => buildCalendarWeeks(monthSelected, timezoneOffsetSeconds))
  const monthName = $derived(parseMonthKey(monthSelected))

  const changeMonth = (count: number) => {
    const mn = monthName
    const fecha = new Date(mn.year, mn.month - 1, 1, 0, 0, 0)
    fecha.setMonth(fecha.getMonth() + count)
    const month = fecha.getFullYear() * 100 + (fecha.getMonth() + 1)
    monthSelected = month
    inputElement?.focus()
  }

  const setAutocompletedValue = (value: string) => {
    const parsedDate = parseTypedDate(value, todayDate, timezoneOffsetSeconds)
    if (parsedDate.autoCompletedDate && parsedDate.autoCompletedUnixDay) {
      monthSelected = getMonthKey(parsedDate.autoCompletedDate)
      fechaFocus = parsedDate.autoCompletedUnixDay
    } else {
      fechaFocus = 0
    }
  }

  const changeFechaSelected = (fechaUnix: number) => {
    untrack(() => {
      if (save && saveOn) {
        if (!fechaUnix) {
          delete saveOn[save]
          return
        }
        saveOn[save] = fechaUnix as NonNullable<T>[keyof T]
      }
    })
    fechaSelected = fechaUnix || 0
    inputValue = formatUnixDay(fechaUnix, timezoneOffsetSeconds)

    if (inputElement) {
      inputElement.value = inputValue
    }

    if (fechaUnix) {
      monthSelected = getMonthKey(dateFromUnixDay(fechaUnix, timezoneOffsetSeconds))
    } else {
      monthSelected = currentMonthKey
    }
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
      const parsedDate = parseTypedDate(value, todayDate, timezoneOffsetSeconds)
      if (value.length === 10 && parsedDate.isCompleted && parsedDate.autoCompletedUnixDay) {
        changeFechaSelected(parsedDate.autoCompletedUnixDay)
        if (onChange) onChange()
      } else {
        (ev.target as HTMLInputElement).value = ""
        changeFechaSelected(0)
      }
      fechaFocus = 0
    }
  }

  const openMobileLayer = () => {
    if (disabled) { return }

    const selectedMonthKey = fechaSelected
      ? getMonthKey(dateFromUnixDay(fechaSelected, timezoneOffsetSeconds))
      : monthSelected || currentMonthKey

    // Delegate the mobile picker to the shared top layer so it can escape clipped form containers.
    Core.showMobileDateLayer = {
      selectedUnixDay: fechaSelected || 0,
      focusedUnixDay: fechaFocus || fechaSelected || 0,
      selectedMonthKey,
      label: label || undefined,
      placeholder,
      onSelect: (unixDay) => {
        changeFechaSelected(unixDay)
        if (onChange) onChange()
      },
      onClose: () => {
        fechaFocus = 0
      }
    }
  }

  // Effect to sync with external changes
  $effect(() => {
    if (saveOn && save) {
      const fechaUnix = saveOn[save] as number
      if (fechaUnix) {
        const value = formatUnixDay(fechaUnix, timezoneOffsetSeconds)
        setAutocompletedValue(value)
        inputValue = value
        if (inputElement) inputElement.value = value
      } else {
        inputValue = ""
        if (inputElement) inputElement.value = ""
        monthSelected = currentMonthKey
      }
      fechaSelected = fechaUnix || 0
    }
  })

  $effect(() => {
    if (isMobile) {
      showCalendar = false
    }
  })

  let cN = $derived(`${s1.input} relative date-input-container` + (css ? " " + css : ""))

  // YYYY-MM-DD mirror of the selected day, exposed as data-value for the agent.
  const agentDataValue = $derived.by(() => {
    if (!fechaSelected) { return "" }
    const d = dateFromUnixDay(fechaSelected, timezoneOffsetSeconds)
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
  })
  const agentDataLabel = $derived(label || placeholder || "")

  const componentID = Env.getComponentID()

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: "DateInput",
      label: label || placeholder || "",
      setValue: (value: string | number) => {
        if (typeof value === "number" && Number.isFinite(value) && value > 0) {
          changeFechaSelected(value)
          if (onChange) onChange()
          return
        }
        const text = String(value || "")
        const parsed = parseTypedDate(text, todayDate, timezoneOffsetSeconds)
        if (parsed.isCompleted && parsed.autoCompletedUnixDay) {
          changeFechaSelected(parsed.autoCompletedUnixDay)
          if (onChange) onChange()
        } else if (text === "") {
          changeFechaSelected(0)
          if (onChange) onChange()
        }
      },
    })
  })

  // Shared close-on-mouseleave: keep open while the input still owns focus (user is typing); otherwise dismiss.
  const handleCalendarMouseLeave = (ev: MouseEvent) => {
    ev.stopPropagation()
    if (inputElement !== document.activeElement) {
      avoidCloseOnBlur = false
      showCalendar = false
      fechaFocus = 0
    }
  }
</script>

{#snippet calendarInner()}
  <div class="flex justify-between items-center mb-[2px]">
    <button class="h2 bn-d1 flex items-center justify-center p-0 bg-transparent border-0"
      type="button"
      onmousedown={(ev) => { ev.stopPropagation(); avoidCloseOnBlur = true }}
      onclick={(ev) => { ev.stopPropagation(); changeMonth(-12) }}>«</button>
    <button class="h2 bn-d1 flex items-center justify-center p-0 bg-transparent border-0"
      type="button"
      onmousedown={(ev) => { ev.stopPropagation(); avoidCloseOnBlur = true }}
      onclick={(ev) => { ev.stopPropagation(); changeMonth(-1) }}>‹</button>
    <div class="bn-d2 flex items-center justify-center font-semibold">
      <div class="mr-[4px]">{monthName.name}</div>
      <div>{monthName.year}</div>
    </div>
    <button class="h2 bn-d1 flex items-center justify-center p-0 bg-transparent border-0"
      type="button"
      onmousedown={(ev) => { ev.stopPropagation(); avoidCloseOnBlur = true }}
      onclick={(ev) => { ev.stopPropagation(); changeMonth(1) }}>›</button>
    <button class="h2 bn-d1 flex items-center justify-center p-0 bg-transparent border-0"
      type="button"
      onmousedown={(ev) => { ev.stopPropagation(); avoidCloseOnBlur = true }}
      onclick={(ev) => { ev.stopPropagation(); changeMonth(12) }}>»</button>
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
        {@const isOutMonth = day.monthKey !== monthSelected}
        {@const isSelected = day.unixDay === fechaSelected}
        {@const isFocused = day.unixDay === fechaFocus}
        {@const isToday = fechaTodayUnix === day.unixDay}
        <button
          class="relative dp-day text-[14px] text-center flex items-center justify-center p-0 bg-transparent border-0 {isOutMonth ? 'is-out' : ''} {isSelected ? 'selected' : ''} {isFocused ? 'focused' : ''}"
          type="button"
          onclick={(ev) => {
            ev.stopPropagation()
            changeFechaSelected(day.unixDay)
            showCalendar = false
            fechaFocus = 0
            avoidCloseOnBlur = false
            if (onChange) onChange()
          }}
          onmousedown={(ev) => { avoidCloseOnBlur = true; ev.stopPropagation() }}
        >
          {day.day}
          {#if isToday}
            <div class="ln-today"></div>
          {/if}
        </button>
      {/each}
    </div>
  {/each}
{/snippet}

{#snippet calendarBlock()}
  {#if showCalendar && !isMobile}
    {#if usePopover}
      <!-- Portal-rendered: positioning handled by Popover2; .in-popover strips the inline-mode absolute positioning. -->
      <Popover2 referenceElement={inputElement ?? null} open={showCalendar} placement="bottom-start" offset={4}>
        <div class="date-picker-c in-popover" role="presentation" onmouseleave={handleCalendarMouseLeave}>
          {@render calendarInner()}
        </div>
      </Popover2>
    {:else}
      <div class="date-picker-c" role="presentation" onmouseleave={handleCalendarMouseLeave}>
        {@render calendarInner()}
      </div>
    {/if}
  {/if}
{/snippet}

{#if label && !useInlineStyle}
  <div data-id="DateInput:{componentID}" data-value={agentDataValue} data-label={agentDataLabel} data-type="other" class={cN}>
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
      {#if !isMobile}
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
      {:else}
        <div
          class={`w-full flex items-center ${s1.input_inp} ff-mono ${inputCss || ""} ${disabled ? "opacity-60" : ""}`}
          role="button"
          tabindex={disabled ? -1 : 0}
          aria-disabled={disabled}
          onclick={(ev) => {
            ev.stopPropagation()
            openMobileLayer()
          }}
          onkeydown={(ev) => {
            if (ev.key === 'Enter' || ev.key === ' ') {
              ev.preventDefault()
              openMobileLayer()
            }
          }}
        >
          <div class={`w-full ${inputValue ? "" : "_mobile-placeholder"}`}>
            {inputValue || placeholder}
          </div>
        </div>
      {/if}
    </div>

    {@render calendarBlock()}
  </div>
{:else}
  <!-- useInlineStyle swaps the chrome (s1.input/input_div/input_inp) for bare classes that fill the parent cell. -->
  <div data-id="DateInput:{componentID}" data-value={agentDataValue} data-label={agentDataLabel} data-type="other"
    class={(useInlineStyle ? "di-bare relative w-full h-full date-input-container" : `${s1.input} no-label relative date-input-container`) + (css ? " " + css : "")}>
    <div class={useInlineStyle ? "flex w-full h-full" : `${s1.input_div} flex w-full`}>
      {#if !isMobile}
        <input
          bind:this={inputElement}
          type="text"
          class={(useInlineStyle ? "di-bare-input ff-mono w-full h-full" : `w-full ${s1.input_inp} ff-mono`) + (inputCss ? " " + inputCss : "")}
          value={inputValue}
          placeholder={placeholder}
          disabled={disabled}
          onfocus={handleFocus}
          onblur={handleBlur}
          onkeydown={handleKeyDown}
          onkeyup={handleKeyUp}
        />
      {:else}
        <div
          class={`w-full flex items-center ${useInlineStyle ? "di-bare-input h-full" : s1.input_inp} ff-mono ${inputCss || ""} ${disabled ? "opacity-60" : ""}`}
          role="button"
          tabindex={disabled ? -1 : 0}
          aria-disabled={disabled}
          onclick={(ev) => {
            ev.stopPropagation()
            openMobileLayer()
          }}
          onkeydown={(ev) => {
            if (ev.key === 'Enter' || ev.key === ' ') {
              ev.preventDefault()
              openMobileLayer()
            }
          }}
        >
          <div class={`w-full ${inputValue ? "" : "_mobile-placeholder"}`}>
            {inputValue || placeholder}
          </div>
        </div>
      {/if}
    </div>

    {@render calendarBlock()}
  </div>
{/if}

<style>
  .date-input-container {
    position: relative;
  }

  ._mobile-placeholder {
    color: #6d5dad;
  }

  /* Bare/inline mode: strip chrome so the parent cell owns the visuals. */
  .di-bare-input {
    background: transparent;
    border: 0;
    outline: 0;
    box-shadow: none;
    padding: 0;
  }
  .di-bare-input:focus {
    outline: 0;
    box-shadow: none;
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

  /* In popover mode the Popover2 wrapper handles absolute positioning; reset our inline-mode offsets. */
  .date-picker-c.in-popover {
    position: static;
    top: auto;
    left: auto;
    margin-top: 0;
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
    color: #6d5dad;
  }
</style>
