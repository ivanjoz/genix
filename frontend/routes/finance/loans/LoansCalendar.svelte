<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import Calendar from '#components/calendar/Calendar.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, todayUnixDay } from '#libs/helpers.ts'
  import type { ILoan } from '#services/finance/loans.svelte.ts'
  import { loanCalendarActivities } from './loans.ts'

  // A month of installments of every loan: green paid, blue due, red overdue. The cash flow's
  // daily view is the full calendar of every obligation.
  let { loans }: { loans: ILoan[] } = $props()

  const today = todayUnixDay()
  const todayDate = new Date(today * 86400000)
  // YYMM, the Calendar's month code.
  let monthCode = $state((todayDate.getUTCFullYear() % 100) * 100 + todayDate.getUTCMonth() + 1)

  const shiftMonth = (monthsAhead: number) => {
    const monthIndex = (monthCode % 100) - 1 + monthsAhead
    const year = 2000 + Math.floor(monthCode / 100) + Math.floor(monthIndex / 12)
    monthCode = (year % 100) * 100 + (((monthIndex % 12) + 12) % 12) + 1
  }
  const monthLabel = $derived(`${String(monthCode % 100).padStart(2, '0')}/${2000 + Math.floor(monthCode / 100)}`)
  const activities = $derived(loanCalendarActivities(loans, today, (cents) => formatN(cents / 100, 2)))
</script>

<div class="flex items-center gap-8 mb-8" aria-label="Loans calendar month">
  <Button mode="subtle" icon="icon-[fa--chevron-left]" label="Shows the previous month." onClick={() => shiftMonth(-1)} />
  <div class="h3 ff-bold w-100 text-center">{monthLabel}</div>
  <Button mode="subtle" icon="icon-[fa--chevron-right]" label="Shows the next month." onClick={() => shiftMonth(1)} />
  <div class="text-fg-muted ml-12">
    {tr('Green paid · blue due · red overdue|Verde pagada · azul por vencer · rojo vencida')}
  </div>
</div>
<Calendar mode="month" month={monthCode} {activities} css="mb-12" />
