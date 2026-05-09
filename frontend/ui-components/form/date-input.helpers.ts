export interface DateInputWeekDay {
  date: Date
  unixDay: number
  day: number
  monthKey: number
}

export interface DateInputWeek {
  year: number
  week: number
  weekDays: DateInputWeekDay[]
  monthCurrent: number
}

export const weekDaysNames = [
  { n: 1, name: 'LU' },
  { n: 2, name: 'MA' },
  { n: 3, name: 'MI' },
  { n: 4, name: 'JU' },
  { n: 5, name: 'VI' },
  { n: 6, name: 'SA' },
  { n: 7, name: 'DO' },
]

export const monthsNames = [
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

const monthsNamesMap = new Map(monthsNames.map((monthRecord) => [monthRecord.n, monthRecord]))

export const createDateInputContext = () => {
  // Keep timezone handling centralized so desktop and mobile render the same unix-day values.
  const todayDate = new Date()
  const timezoneOffsetSeconds = todayDate.getTimezoneOffset() * 60

  return {
    todayDate,
    timezoneOffsetSeconds,
    todayUnixDay: unixDayFromDate(todayDate, timezoneOffsetSeconds),
    currentMonthKey: getMonthKey(todayDate),
  }
}

export const getMonthKey = (date: Date): number => {
  return date.getFullYear() * 100 + (date.getMonth() + 1)
}

export const parseMonthKey = (yearMonth: number) => {
  const yearMonthString = String(yearMonth)
  const year = parseInt(yearMonthString.substring(0, 4))
  const month = parseInt(yearMonthString.substring(4, 6))
  const name = monthsNamesMap.get(month)?.name || "-"

  return { name, year, month }
}

export const unixDayFromDate = (date: Date, timezoneOffsetSeconds: number): number => {
  return Math.floor((date.getTime() - (timezoneOffsetSeconds * 1000)) / 86400000)
}

export const dateFromUnixDay = (unixDay: number, timezoneOffsetSeconds: number): Date => {
  return new Date((unixDay * 86400 + timezoneOffsetSeconds) * 1000)
}

const startOfISOWeek = (date: Date): Date => {
  const weekStartDate = new Date(date)
  const currentDay = weekStartDate.getDay()
  const diffToMonday = (currentDay === 0 ? -6 : 1) - currentDay

  weekStartDate.setDate(weekStartDate.getDate() + diffToMonday)
  weekStartDate.setHours(0, 0, 0, 0)

  return weekStartDate
}

const getISOWeek = (date: Date): number => {
  const normalizedDate = new Date(date)
  normalizedDate.setHours(0, 0, 0, 0)
  normalizedDate.setDate(normalizedDate.getDate() + 4 - (normalizedDate.getDay() || 7))

  const yearStartDate = new Date(normalizedDate.getFullYear(), 0, 1)
  return Math.ceil((((normalizedDate.getTime() - yearStartDate.getTime()) / 86400000) + 1) / 7)
}

const getISOWeekYear = (date: Date): number => {
  const normalizedDate = new Date(date)
  normalizedDate.setDate(normalizedDate.getDate() + 4 - (normalizedDate.getDay() || 7))
  return normalizedDate.getFullYear()
}

export const buildCalendarWeeks = (
  monthKey: number,
  timezoneOffsetSeconds: number,
): DateInputWeek[] => {
  const { year, month } = parseMonthKey(monthKey)
  const monthFirstDate = new Date(year, month - 1, 1, 0, 0, 0)
  const monthLastDate = new Date(year, month, 0)
  const visibleMonthIndex = monthFirstDate.getMonth()
  let currentWeekStartDate = startOfISOWeek(monthFirstDate)

  const calendarWeeks: DateInputWeek[] = []

  while (currentWeekStartDate.getTime() <= monthLastDate.getTime()) {
    const currentWeekStartTime = currentWeekStartDate.getTime()
    const currentWeekStartUnixDay = unixDayFromDate(currentWeekStartDate, timezoneOffsetSeconds)
    const currentWeekYear = getISOWeekYear(currentWeekStartDate)
    const currentWeekNumber = getISOWeek(currentWeekStartDate)
    const weekDays: DateInputWeekDay[] = []

    // Build the 7 visible cells explicitly so the calendar grid stays stable.
    for (let dayOffset = 0; dayOffset < 7; dayOffset++) {
      const currentDate = new Date(currentWeekStartTime + (86400000 * dayOffset))
      weekDays.push({
        date: currentDate,
        unixDay: currentWeekStartUnixDay + dayOffset,
        day: currentDate.getDate(),
        monthKey: getMonthKey(currentDate),
      })
    }

    calendarWeeks.push({
      year: currentWeekYear,
      week: currentWeekNumber,
      weekDays,
      monthCurrent: visibleMonthIndex,
    })

    currentWeekStartDate = new Date(currentWeekStartTime + (86400000 * 7))
  }

  return calendarWeeks
}

export const formatUnixDay = (unixDay: number, timezoneOffsetSeconds: number): string => {
  if (!unixDay) { return "" }

  const formattedDate = dateFromUnixDay(unixDay, timezoneOffsetSeconds)
  const day = String(formattedDate.getDate()).padStart(2, '0')
  const month = String(formattedDate.getMonth() + 1).padStart(2, '0')
  const year = formattedDate.getFullYear()

  return `${day}-${month}-${year}`
}

export const parseTypedDate = (
  rawValue: string,
  todayDate: Date,
  timezoneOffsetSeconds: number,
) => {
  const normalizedValue = rawValue.trim().substring(0, 10).replaceAll("/", "-")
  const currentYearText = String(todayDate.getFullYear())
  const valueParts = normalizedValue.split("-").filter((part) => part)

  let day = isNaN(valueParts[0] as unknown as number) ? 0 : parseInt(valueParts[0])
  let month = isNaN(valueParts[1] as unknown as number) ? 0 : parseInt(valueParts[1])

  let yearText = valueParts[2] || ""
  if (yearText.length !== 4) {
    for (let currentIndex = 0; currentIndex < currentYearText.length; currentIndex++) {
      if (!yearText[currentIndex]) { yearText += currentYearText[currentIndex] }
    }
  }

  let year = isNaN(yearText as unknown as number) ? 0 : parseInt(yearText)
  const isCompleted = day > 0 && month > 0 && yearText.length === 4
  let autoCompletedUnixDay = 0
  let autoCompletedDate: Date | null = null

  if (day > 0) {
    if (!month) { month = todayDate.getMonth() + 1 }
    if (!year) { year = todayDate.getFullYear() }

    const candidateDate = new Date(year, month - 1, day, 0, 0, 0)
    const isSameCalendarDate =
      candidateDate.getFullYear() === year &&
      candidateDate.getMonth() === month - 1 &&
      candidateDate.getDate() === day

    // Reject overflowed dates such as 31-02-2026 before updating focus/save state.
    if (isSameCalendarDate && !Number.isNaN(candidateDate.getTime())) {
      autoCompletedDate = candidateDate
      autoCompletedUnixDay = unixDayFromDate(candidateDate, timezoneOffsetSeconds)
    }
  }

  return {
    normalizedValue,
    isCompleted,
    autoCompletedUnixDay,
    autoCompletedDate,
    day,
    month,
    year,
  }
}

export const resolvePreservedUnixDay = (
  referenceUnixDay: number,
  targetMonthKey: number,
  timezoneOffsetSeconds: number,
): number => {
  if (!referenceUnixDay) { return 0 }

  const referenceDate = dateFromUnixDay(referenceUnixDay, timezoneOffsetSeconds)
  const { year, month } = parseMonthKey(targetMonthKey)
  const monthLastDay = new Date(year, month, 0).getDate()
  const preservedDay = Math.min(referenceDate.getDate(), monthLastDay)
  const preservedDate = new Date(year, month - 1, preservedDay, 0, 0, 0)

  return unixDayFromDate(preservedDate, timezoneOffsetSeconds)
}
