<script lang="ts" generics="T">
  import { SvelteMap } from 'svelte/reactivity';
  import Renderer, { type ElementAST } from '$components/Renderer.svelte';
  import type { ITableColumn } from './types';

  interface TableStreamProps<T> {
    columns: ITableColumn<T>[];
    data?: T[];
    maxRecords: number;
    maxHeight?: string;
    css?: string;
    tableCss?: string;
    onRowClick?: (row: T, index: number, rerender: () => void) => void;
    selected?: T | number;
    isSelected?: (row: T, selected: T | number) => boolean;
    emptyMessage?: string;
  }

  let {
    columns,
    data,
    maxRecords,
    maxHeight = '430px',
    css = '',
    tableCss = '',
    onRowClick,
    selected,
    isSelected,
    emptyMessage = 'No se encontraron registros.'
  }: TableStreamProps<T> = $props();

  let streamRecords = $state<T[]>([]);
  // Per-row version counters bumped when `onRowClick` invokes its `rerender` callback;
  // included in the row `#each` key so only the affected row remounts.
  const rowVersions = new SvelteMap<number, number>();

  const rerenderRow = (rowIndex: number) => {
    rowVersions.set(rowIndex, (rowVersions.get(rowIndex) || 0) + 1);
  };

  // Keep an internal bounded buffer so append operations always enforce maxRecords.
  const normalizeRecords = (incomingRecords: T[]) => {
    const normalizedMaxRecords = Math.max(1, maxRecords || 1);
    return incomingRecords.slice(0, normalizedMaxRecords);
  };

  // Mirrors external data updates while preserving bounded behavior.
  $effect(() => {
    if (data === undefined) return;
    streamRecords = normalizeRecords(data || []);
  });

  const getVisibleColumns = () => columns.filter((columnDefinition) => !columnDefinition.hidden);

  const getHeaderContent = (columnDefinition: ITableColumn<T>) => {
    return typeof columnDefinition.header === 'function'
      ? columnDefinition.header()
      : columnDefinition.header;
  };

  const getCellValue = (columnDefinition: ITableColumn<T>, rowRecord: T, rowIndex: number) => {
    if (columnDefinition.getValue) {
      return columnDefinition.getValue(rowRecord, rowIndex) as string | number;
    }
    return '';
  };

  const getCellRenderedContent = (columnDefinition: ITableColumn<T>, rowRecord: T, rowIndex: number) => {
    if (!columnDefinition.render) return null;
    return columnDefinition.render(rowRecord, rowIndex) as string | ElementAST | ElementAST[];
  };

  const getRowSelected = (rowRecord: T) => {
    if (!selected || !isSelected) return false;
    return isSelected(rowRecord, selected);
  };

  const appendTop = (incomingRecordOrList: T | T[]) => {
    const incomingRecords = Array.isArray(incomingRecordOrList)
      ? incomingRecordOrList
      : [incomingRecordOrList];

    // Insert newest records at the beginning and trim the tail in one pass.
    streamRecords = normalizeRecords([...incomingRecords, ...streamRecords]);
  };

  const replaceRecords = (incomingRecords: T[]) => {
    streamRecords = normalizeRecords(incomingRecords || []);
  };

  const clearRecords = () => {
    streamRecords = [];
  };

  // Expose imperative handlers so streaming modules can append records without array cloning in parents.
  export { appendTop, replaceRecords, clearRecords };
</script>

<div class="stream-table-card {css}">
  <div class="stream-table-scroll" style="max-height: {maxHeight};">
    <table class="stream-table {tableCss}">
      <thead>
        <tr>
          {#each getVisibleColumns() as columnDefinition}
            <th class={columnDefinition.headerCss || ''}>
              {getHeaderContent(columnDefinition)}
            </th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#if streamRecords.length === 0}
          <tr>
            <td colspan={Math.max(1, getVisibleColumns().length)} class="stream-empty">{emptyMessage}</td>
          </tr>
        {:else}
          {#each streamRecords as rowRecord, rowIndex (`${rowIndex}_${rowVersions.get(rowIndex) || 0}`)}
            <tr
              class:stream-row-selected={getRowSelected(rowRecord)}
              class:stream-row-even={rowIndex % 2 === 0}
              class:stream-row-odd={rowIndex % 2 !== 0}
              onclick={() => onRowClick?.(rowRecord, rowIndex, () => rerenderRow(rowIndex))}
            >
              {#each getVisibleColumns() as columnDefinition}
                {@const renderedCellContent = getCellRenderedContent(columnDefinition, rowRecord, rowIndex)}
                <td class={columnDefinition.cellCss || columnDefinition.css || ''}>
                  {#if typeof renderedCellContent === 'string'}
                    {@html renderedCellContent}
                  {:else if renderedCellContent}
                    <Renderer elements={renderedCellContent}/>
                  {:else}
                    {getCellValue(columnDefinition, rowRecord, rowIndex)}
                  {/if}
                </td>
              {/each}
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

<style>
  .stream-table-card {
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    background: #ffffff;
    overflow: hidden;
  }

  .stream-table-scroll {
    overflow: auto;
  }

  .stream-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
  }

  .stream-table thead th {
    position: sticky;
    top: 0;
    z-index: 2;
    background: #0f172a;
    color: #e2e8f0;
    text-align: left;
    font-weight: 700;
    padding: 10px 8px;
    border-bottom: 1px solid #1e293b;
    white-space: nowrap;
  }

  .stream-table tbody td {
    padding: 8px;
    border-bottom: 1px solid #e2e8f0;
    white-space: nowrap;
  }

  .stream-row-even {
    background: #f8fafc;
  }

  .stream-row-selected {
    outline: 2px solid #2563eb;
    outline-offset: -2px;
  }

  .stream-empty {
    color: #64748b;
    text-align: center;
    padding: 22px 8px !important;
    font-family: inherit !important;
  }
</style>
