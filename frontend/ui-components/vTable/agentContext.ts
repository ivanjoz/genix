// Svelte context shared between every vTable variant and its inner cells.
// Cells (CellInput / CellSelect) don't register themselves with the
// global Agent registry: they hand their methods to the parent table through
// `registerCell`, and the table is the sole agent handle. The agent addresses
// individual cells via the composite data-id `<tableID>:<cellID>`; the table
// dispatches by cellID against the map of registered cells.

import { getContext, setContext } from 'svelte';
import type { AgentOption } from '$core/agent/registry';

const VTABLE_AGENT_KEY = Symbol.for('vTable.agentContext');

export interface CellAgentMethods {
  setValue?: (value: string | number) => void;
  search?: (text: string) => void;
  select?: (...optionIds: (number | string)[]) => void;
  getOptions?: (max?: number) => AgentOption[];
}

export interface VTableAgentContext {
  // Component id of the parent Table. Cells use it to build their data-id.
  tableID: number;
  // Returns a cleanup fn the cell calls in its $effect teardown.
  registerCell: (cellID: number, methods: CellAgentMethods) => () => void;
}

export function setVTableAgentContext(ctx: VTableAgentContext): void {
  setContext(VTABLE_AGENT_KEY, ctx);
}

export function getVTableAgentContext(): VTableAgentContext | undefined {
  return getContext<VTableAgentContext | undefined>(VTABLE_AGENT_KEY);
}

// Row id = (rowIndex + 1) * 100 — shifted by one so the lowest visible row id
// is 100, never 0 (an id of 0 reads as "missing" to many code paths). Cell id
// = rowID + columnIndex + 1, so the column slot is 1-based and cell ids never
// collide with row ids (which are exact multiples of 100). Cap of 99 cells
// per row.
export function buildRowID(rowIndex: number): number {
  return (rowIndex + 1) * 100;
}

export function buildCellID(rowIndex: number, columnIndex: number): number {
  return buildRowID(rowIndex) + columnIndex + 1;
}

// Inverse of buildRowID. Returns -1 when the id is not a valid row id.
export function rowIndexFromRowID(rowID: number): number {
  if (!Number.isFinite(rowID) || rowID <= 0 || rowID % 100 !== 0) { return -1; }
  return rowID / 100 - 1;
}

// Extract the numeric child portion from either a composite id ("38:101") or
// a bare child id ("101" / 101). Used by the Table's agent methods so the
// agent can address cells/rows with the same composite id it sees in the
// HTML snapshot.
export function parseChildID(raw: number | string): number {
  if (typeof raw === 'number') { return raw; }
  const s = String(raw);
  const colon = s.indexOf(':');
  return Number(colon >= 0 ? s.slice(colon + 1) : s);
}
