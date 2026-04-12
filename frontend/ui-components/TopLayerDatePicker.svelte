<script lang="ts">
import { Core } from '$core/store.svelte';
import { untrack } from 'svelte';
import {
  buildCalendarWeeks,
  createDateInputContext,
  formatUnixDay,
  getMonthKey,
  monthsNames,
  parseMonthKey,
  parseTypedDate,
  resolvePreservedUnixDay,
  weekDaysNames,
} from './date-input.helpers';

const {
  todayDate,
  timezoneOffsetSeconds,
  todayUnixDay,
  currentMonthKey,
} = createDateInputContext()

const monthRows = [
  monthsNames.slice(0, 4),
  monthsNames.slice(4, 8),
  monthsNames.slice(8, 12),
]

const allowedDateInputChars = new Set(['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '/'])
const allowedDateInputKeys = new Set([
  ...allowedDateInputChars,
  'Backspace',
  'Delete',
  'ArrowLeft',
  'ArrowRight',
  'ArrowUp',
  'ArrowDown',
  'Tab',
  'Enter',
  'Home',
  'End',
])

let typedValue = $state("")
let localSelectedUnixDay = $state(0)
let localFocusedUnixDay = $state(0)
let localMonthKey = $state(currentMonthKey)
let inputElement = $state<HTMLInputElement>()
let previewTimeoutId: number | undefined

const isOpen = $derived(!!Core.showMobileDateLayer)
const visibleMonth = $derived(parseMonthKey(localMonthKey))
const calendarWeeks = $derived(buildCalendarWeeks(localMonthKey, timezoneOffsetSeconds))

const sanitizeDateInputValue = (rawValue: string) => {
  let cleanedValue = ""
  for (const currentChar of rawValue) {
    if (allowedDateInputChars.has(currentChar)) {
      cleanedValue += currentChar
    }
  }
  return cleanedValue.substring(0, 10)
}

const syncLayerState = () => {
  const mobileDateLayer = Core.showMobileDateLayer
  const selectedUnixDay = mobileDateLayer?.selectedUnixDay || 0
  const selectedMonthKey = mobileDateLayer?.selectedMonthKey ||
    (selectedUnixDay ? getMonthKey(new Date((selectedUnixDay * 86400 + timezoneOffsetSeconds) * 1000)) : currentMonthKey)

  localSelectedUnixDay = selectedUnixDay
  localFocusedUnixDay = mobileDateLayer?.focusedUnixDay || selectedUnixDay || 0
  localMonthKey = selectedMonthKey
  typedValue = formatUnixDay(selectedUnixDay, timezoneOffsetSeconds)
}

const closeLayer = () => {
  const mobileDateLayer = Core.showMobileDateLayer
  Core.showMobileDateLayer = null
  mobileDateLayer?.onClose?.()
}

const clearSelectedDate = () => {
  typedValue = ""
  localSelectedUnixDay = 0
  localFocusedUnixDay = 0
  localMonthKey = currentMonthKey
  Core.showMobileDateLayer?.onSelect(0)
  closeLayer()
}

const previewTypedDate = (nextValue: string) => {
  const parsedDate = parseTypedDate(nextValue, todayDate, timezoneOffsetSeconds)
  if (!parsedDate.autoCompletedDate || !parsedDate.autoCompletedUnixDay) {
    localFocusedUnixDay = 0
    return
  }

  // Mirror the typed draft into the visible month so the grid previews the target date.
  localMonthKey = getMonthKey(parsedDate.autoCompletedDate)
  localFocusedUnixDay = parsedDate.autoCompletedUnixDay
}

const queueTypedDatePreview = (nextValue: string) => {
  if (previewTimeoutId) { clearTimeout(previewTimeoutId) }
  previewTimeoutId = setTimeout(() => {
    previewTypedDate(nextValue)
  }, 150) as unknown as number
}

const commitTypedDate = () => {
  const cleanedValue = sanitizeDateInputValue(typedValue)
  const parsedDate = parseTypedDate(cleanedValue, todayDate, timezoneOffsetSeconds)
  typedValue = cleanedValue

  if (cleanedValue.trim() === "") {
    localSelectedUnixDay = 0
    localFocusedUnixDay = 0
    Core.showMobileDateLayer?.onSelect(0)
    return
  }

  if (parsedDate.isCompleted && parsedDate.autoCompletedDate && parsedDate.autoCompletedUnixDay) {
    // Save the typed value immediately on blur while keeping the overlay available for further changes.
    localSelectedUnixDay = parsedDate.autoCompletedUnixDay
    localFocusedUnixDay = parsedDate.autoCompletedUnixDay
    localMonthKey = getMonthKey(parsedDate.autoCompletedDate)
    typedValue = formatUnixDay(parsedDate.autoCompletedUnixDay, timezoneOffsetSeconds)
    Core.showMobileDateLayer?.onSelect(parsedDate.autoCompletedUnixDay)
    return
  }

  // Invalid drafts should not leak into the saved state, so restore the last committed value.
  typedValue = formatUnixDay(localSelectedUnixDay, timezoneOffsetSeconds)
  localFocusedUnixDay = localSelectedUnixDay || 0
}

const updateVisibleMonth = (targetMonthKey: number) => {
  localMonthKey = targetMonthKey

  const referenceUnixDay = localFocusedUnixDay || localSelectedUnixDay
  if (!referenceUnixDay) { return }

  // Preserve the current day number when browsing months so the calendar stays anchored.
  localFocusedUnixDay = resolvePreservedUnixDay(referenceUnixDay, targetMonthKey, timezoneOffsetSeconds)
  typedValue = formatUnixDay(localFocusedUnixDay, timezoneOffsetSeconds)
}

const changeVisibleYear = (yearDelta: number) => {
  const { year, month } = visibleMonth
  updateVisibleMonth((year + yearDelta) * 100 + month)
}

const selectMonth = (monthNumber: number) => {
  updateVisibleMonth(visibleMonth.year * 100 + monthNumber)
}

const selectDay = (unixDay: number) => {
  localSelectedUnixDay = unixDay
  localFocusedUnixDay = unixDay
  typedValue = formatUnixDay(unixDay, timezoneOffsetSeconds)
  Core.showMobileDateLayer?.onSelect(unixDay)
  closeLayer()
}

$effect(() => {
  if (isOpen) {
    untrack(() => {
      syncLayerState()
    })
    return
  }

  if (previewTimeoutId) {
    clearTimeout(previewTimeoutId)
    previewTimeoutId = undefined
  }

  inputElement?.blur()
})
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="_1" class:_2={isOpen} data-button-layer-protected="true">
  <div class="p-6 pb-4">
    <div class="flex items-center">
      <i class="icon-calendar _3"></i>
      <input
        bind:this={inputElement}
        bind:value={typedValue}
        type="text"
        class="h-38 min-w-0 grow ff-mono _4"
        placeholder={Core.showMobileDateLayer?.placeholder || "DD-MM-YYYY"}
        onkeydown={(ev) => {
          if (!allowedDateInputKeys.has(ev.key)) {
            ev.preventDefault()
            return
          }

          if (ev.key === 'Enter') {
            ev.preventDefault()
            inputElement?.blur()
          }
        }}
        oninput={(ev) => {
          const cleanedValue = sanitizeDateInputValue((ev.currentTarget as HTMLInputElement).value || "")
          typedValue = cleanedValue
          queueTypedDatePreview(cleanedValue)
        }}
        onblur={() => {
          commitTypedDate()
        }}
      />
      <div class="ml-8 flex shrink-0 items-center gap-8">
        <button
          class="h-40 w-40 shrink-0 _5 _5b"
          aria-label="Limpiar fecha"
          onclick={() => {
            clearSelectedDate()
          }}
        >
          <i class="icon-ccw h1"></i>
        </button>
        <button
          class="h-40 w-40 shrink-0 _5"
          aria-label="Cerrar selector de fecha"
          onclick={() => {
            commitTypedDate()
            closeLayer()
          }}
        >
          <i class="icon-cancel h1"></i>
        </button>
      </div>
    </div>

    {#if Core.showMobileDateLayer?.label}
      <div class="mt-6 px-4 text-[13px] tracking-[0.4px] text-white/70">
        {Core.showMobileDateLayer.label}
      </div>
    {/if}

    <div class="_6 mt-8 p-8">
      <div class="flex items-center justify-between gap-8">
        <button
          class="_7 h-38 w-38"
          type="button"
          aria-label="Año anterior"
          onclick={() => {
            changeVisibleYear(-1)
          }}
        >
          ‹
        </button>
        <div class="flex min-w-0 grow items-center justify-center gap-8 px-8">
          <div class="truncate text-[17px] font-medium text-white/82">
            {visibleMonth.name}
          </div>
          <div class="text-[20px] font-semibold tracking-[0.4px] text-white">
            {visibleMonth.year}
          </div>
        </div>
        <button
          class="_7 h-38 w-38"
          type="button"
          aria-label="Año siguiente"
          onclick={() => {
            changeVisibleYear(1)
          }}
        >
          ›
        </button>
      </div>

      <div class="mt-8 grid grid-cols-4 gap-6">
        {#each monthRows as monthRow}
          {#each monthRow as monthRecord}
            <button
              class="_8 h-38 px-6 text-[14px]"
              class:_9={visibleMonth.month === monthRecord.n}
              type="button"
              onclick={() => {
                selectMonth(monthRecord.n)
              }}
            >
              {monthRecord.name.substring(0, 3)}
            </button>
          {/each}
        {/each}
      </div>
    </div>

    <div class="_10 mt-10 p-8">
      <div class="flex">
        <div class="dp-week text-center text-[12px] font-semibold text-white/35"></div>
        {#each weekDaysNames as dayName}
          <div class="dp-col flex items-center justify-center text-[12px] font-semibold text-white/65">
            {dayName.name}
          </div>
        {/each}
      </div>

      {#each calendarWeeks as weekRecord}
        <div class="flex">
          <div class="dp-week flex items-center justify-center text-[12px] font-semibold text-[rgba(158,140,212,0.58)]">
            {weekRecord.week}
          </div>
          {#each weekRecord.weekDays as weekDay}
            {@const isOutsideVisibleMonth = weekDay.monthKey !== localMonthKey}
            {@const isSelectedDay = weekDay.unixDay === localSelectedUnixDay}
            {@const isFocusedDay = weekDay.unixDay === localFocusedUnixDay}
            {@const isToday = weekDay.unixDay === todayUnixDay}

            <button
              class="dp-day relative flex items-center justify-center border-0 bg-transparent p-0 text-sm"
              class:is-out={isOutsideVisibleMonth}
              class:selected={isSelectedDay}
              class:focused={isFocusedDay && !isSelectedDay}
              type="button"
              onclick={() => {
                selectDay(weekDay.unixDay)
              }}
            >
              {weekDay.day}
              {#if isToday}
                <div class="ln-today"></div>
              {/if}
            </button>
          {/each}
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  ._1 {
    width: 100vw;
    max-height: calc(100vh - 8px);
    background-color: #000000b3;
    overflow-y: auto;
    overflow-x: hidden;
    z-index: 300;
    position: fixed;
    top: 0;
    backdrop-filter: blur(8px);
    border-radius: 0 0 16px 16px;
    border-bottom: 6px solid rgba(0, 0, 0, .25);
    opacity: 0;
    pointer-events: none;
    padding-bottom: 6px;
  }

  ._1._2 {
    opacity: 1;
    pointer-events: all;
  }

  ._3 {
    position: absolute;
    left: 14px;
    color: #ffffff82;
    margin-bottom: 2px;
  }

  ._4 {
    border-radius: 9px;
    color: #fff;
    margin-bottom: 2px;
    background-color: #464672d4;
    border: 1px solid transparent;
    line-height: 1;
    appearance: none;
    outline: none;
    padding-left: 34px;
    padding-right: 10px;
  }

  ._4:focus {
    border: 1px solid rgb(117, 118, 214);
    outline: 1px solid rgb(150, 152, 255);
    outline-offset: 0;
  }

  ._5 {
    border-radius: 50%;
    background-color: #e06868;
    color: #fff;
    border: none;
    outline: none;
  }

  ._5b {
    background-color: #30303d;
  }

  ._6,
  ._10 {
    border-radius: 16px;
    border: 1px solid #ffffff1f;
    outline: 2px solid #00000059;
    background: linear-gradient(180deg, rgba(40, 41, 52, 0.94) 0%, rgba(24, 25, 31, 0.94) 100%);
  }

  ._7,
  ._8 {
    border-radius: 9px;
    border: 1px solid #ffffff1f;
    outline: 2px solid #00000059;
    background-color: #0000003d;
    color: #fff;
  }

  ._9 {
    background-color: #6861ec;
    border-color: #b3b0ff;
    color: #fff;
  }

  .dp-week {
    width: 28px;
    height: 30px;
    flex-shrink: 0;
  }

  .dp-col {
    width: calc((100% - 28px) / 7);
    height: 28px;
    flex-shrink: 0;
  }

  .dp-day {
    width: calc((100% - 28px) / 7);
    height: 36px;
    flex-shrink: 0;
    cursor: pointer;
    border-radius: 8px;
    color: #f4f4ff;
  }

  .dp-day.is-out {
    color: #ffffff45;
  }

  .dp-day.selected {
    background-color: #6d5dad;
    color: white;
    outline: 2px solid #b6b1ff;
  }

  .dp-day.focused {
    background-color: #ffffff14;
    outline: 1px solid #8d89ce;
  }

  .ln-today {
    position: absolute;
    bottom: 4px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    background-color: #ffd96d;
    border-radius: 50%;
  }

  .dp-day.selected .ln-today {
    background-color: white;
  }
</style>
