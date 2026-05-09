// Svelte context shared between every vTable variant and its inner cells.
// Cells (CellEditable / CellSelector) don't register themselves with the
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

// Conventional cellID = rowIndex * 100 + (columnIndex + 1). Column index is
// shifted by one so cell ids never collide with row ids (which are exact
// multiples of 100). A row of 99 columns therefore caps at *N100..*N99.
export function buildCellID(rowIndex: number, columnIndex: number): number {
  return rowIndex * 100 + columnIndex + 1;
}

export function buildRowID(rowIndex: number): number {
  return rowIndex * 100;
}
