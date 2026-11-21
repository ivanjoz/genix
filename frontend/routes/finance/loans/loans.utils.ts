// Generic date helpers for the loans page's editable cells. No business logic.

const MS_PER_DAY = 86400000

/** UnixDay → "dd-mm-yyyy" (empty for 0). */
export function unixDayToDMY(unixDay: number): string {
  if (!unixDay) return ''
  const date = new Date(unixDay * MS_PER_DAY)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${pad(date.getUTCDate())}-${pad(date.getUTCMonth() + 1)}-${date.getUTCFullYear()}`
}

/** "dd-mm-yyyy" (or "dd/mm/yyyy") → UnixDay; 0 when it is not a real date. */
export function dmyToUnixDay(text: string): number {
  const match = /^\s*(\d{1,2})[-/](\d{1,2})[-/](\d{4})\s*$/.exec(text)
  if (!match) return 0
  const [day, month, year] = [Number(match[1]), Number(match[2]), Number(match[3])]
  const date = new Date(Date.UTC(year, month - 1, day))
  if (date.getUTCDate() !== day || date.getUTCMonth() !== month - 1) return 0
  return Math.round(date.getTime() / MS_PER_DAY)
}
