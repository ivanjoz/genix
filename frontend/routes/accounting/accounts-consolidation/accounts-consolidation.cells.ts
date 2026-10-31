// The Debe / Haber cells Movimientos and the Diario de gestión share.
import { tr } from '#core/store.svelte.ts'
import { pcgeAccountLabel } from '#core/pcge.ts'
import { formatN } from '#libs/helpers.ts'
import { OPERATION_STATUS_INCOMPLETE, type IAccountingOperation, type IGroupAccountLine } from './accounts-consolidation'

// A side's accounts and amounts stack one per line. Every line has the same fixed height, so an
// account stays level with its amount in the next cell; a long account name ends in an ellipsis.
// The table's cells are top-aligned at that same line height (ENTRY_TABLE_CELL_CSS), so a one-line
// cell sits level with the first line of its neighbours.
const ACCOUNT_LINE_CSS = "h-26 leading-26 truncate"
export const ENTRY_TABLE_CELL_CSS = "!align-top leading-26 py-2"

// The accounts carry the status: an account still to pick is a red warning, and an incomplete
// entry paints its accounts red, its problems on hover.
export function accountLinesCell(lines: IGroupAccountLine[], status: number, operations: IAccountingOperation[]): string {
  const isIncomplete = status === OPERATION_STATUS_INCOMPLETE
  const problems = isIncomplete
    ? [...new Set(operations.map(operation => tr(operation.Problem)))].join(" ").replaceAll('"', "&quot;")
    : ""
  return lines.map(line => line.AccountCode
    ? `<div class="${ACCOUNT_LINE_CSS} w-240${isIncomplete ? " text-red-fg" : ""}" title="${problems}">${pcgeAccountLabel(line.AccountCode)}</div>`
    : `<div class="${ACCOUNT_LINE_CSS} w-240 text-red-fg" title="${tr("Unclassified|Sin clasificar")}"><i class="icon-[fa--exclamation-triangle]"></i></div>`,
  ).join("")
}

export const amountLinesCell = (lines: IGroupAccountLine[]): string =>
  lines.map(line => `<div class="${ACCOUNT_LINE_CSS}">${formatN(line.Amount / 100, 2)}</div>`).join("")
