// How a value is written inside a SUNAT flat file. Formatting only — no book rule lives here.
//
// The rules are Tabla 5 of RS 112-2021: `|` separates the fields, so no field may contain one;
// free text may not contain `/` or `\` either; amounts carry exactly two decimals.

// sunatDate writes a UnixDay as DD/MM/AAAA (fields 5, 29).
//
// Read in UTC, like the rest of the book: a UnixDay is a day number, and interpreting its
// midnight in a negative offset lands on the day before.
export function sunatDate(unixDay: number): string {
  if (!unixDay) return ""
  const date = new Date(unixDay * 86400 * 1000)
  const day = String(date.getUTCDate()).padStart(2, "0")
  const month = String(date.getUTCMonth() + 1).padStart(2, "0")
  return `${day}/${month}/${date.getUTCFullYear()}`
}

// sunatAmount writes cents as the 12,2 decimal the annex asks for: always two decimals, no
// thousands separator, the minus sign attached to the number.
export const sunatAmount = (cents: number): string => ((cents || 0) / 100).toFixed(2)

// sunatText makes a free-text field safe to put between two pipes.
//
// The three forbidden characters are dropped rather than replaced: a razón social that arrived
// with a slash in it is a data-entry accident, and a substitute character would travel into the
// filed book as if the taxpayer had written it. Line breaks would split the record in two, so
// they collapse to a space.
export function sunatText(value: string, maxLength: number): string {
  const cleaned = (value || "")
    .replace(/[|/\\]/g, "")
    .replace(/\s+/g, " ")
    .trim()
  return cleaned.length > maxLength ? cleaned.slice(0, maxLength) : cleaned
}
