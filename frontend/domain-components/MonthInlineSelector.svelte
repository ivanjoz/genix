<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import T from '#components/misc/T.svelte';
  import { calendarMonthCodeName, calendarMonthCodeOfDay, shiftCalendarMonthCode, todayUnixDay } from '#libs/helpers.ts';

  // A month stepper: ‹ Octubre 2026 › in one pill, plus `Hoy` when another month is shown.
  // The month is the yymm code genix-ui's Calendar takes (2610 = October 2026).
  interface Props {
    monthCode: number
    css?: string
  }

  let { monthCode = $bindable(), css = '' }: Props = $props()

  const currentMonthCode = calendarMonthCodeOfDay(todayUnixDay())
</script>

<div class="flex items-center gap-8 {css}">
  <div class="flex items-center h-40 px-4 rounded-full bg-surface border border-line shadow-sm">
    <Button icon="icon-[fa--chevron-left] text-[12px]" color="blue" mode="minimal" height="small" useCircle shadow="none"
      label="Previous month|Mes anterior" onClick={() => { monthCode = shiftCalendarMonthCode(monthCode, -1) }} />
    <div class="flex items-center justify-center gap-8 w-180 px-8 select-none">
      <i class="icon-[fa--calendar-o] text-accent-solid"></i>
      <span class="ff-bold text-[16px] text-fg"><T text={calendarMonthCodeName(monthCode)} /></span>
    </div>
    <Button icon="icon-[fa--chevron-right] text-[12px]" color="blue" mode="minimal" height="small" useCircle shadow="none"
      label="Next month|Mes siguiente" onClick={() => { monthCode = shiftCalendarMonthCode(monthCode, 1) }} />
  </div>
  {#if monthCode !== currentMonthCode}
    <Button name="Today|Hoy" icon="icon-[fa--dot-circle-o]" color="blue" mode="subtle" height="compact"
      label="Back to this month.|Vuelve a este mes." onClick={() => { monthCode = currentMonthCode }} />
  {/if}
</div>
