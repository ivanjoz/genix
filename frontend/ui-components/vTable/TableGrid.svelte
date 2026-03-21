<script lang="ts" generics="TRecord">
  import { onMount } from 'svelte';
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import type { TableGridCellRendererSnippet, TableGridColumn } from './tableGridTypes';

  interface TableGridProps<TRecord> {
    columns: TableGridColumn<TRecord>[];
    data: TRecord[];
    height?: string;
    rowHeight?: number;
    bufferSize?: number;
    css?: string;
    headerCss?: string;
    rowCss?: string;
    emptyMessage?: string;
    debug?: boolean;
    onRowClick?: (rowRecord: TRecord, rowIndex: number) => void;
    selectedRowId?: string | number;
    selectedRecord?: TRecord;
    getRowId?: (rowRecord: TRecord, rowIndex: number) => string | number;
    cellRenderer?: TableGridCellRendererSnippet<TRecord>;
  }

  let {
    columns,
    data,
    height = '460px',
    rowHeight = 36,
    bufferSize = 12,
    css = '',
    headerCss = '',
    rowCss = '',
    emptyMessage = 'No se encontraron registros.',
    debug = false,
    onRowClick,
    selectedRowId,
    selectedRecord,
    getRowId,
    cellRenderer,
  }: TableGridProps<TRecord> = $props();

  // Keep a stable filtered list so hidden columns never affect row rendering logic.
  const visibleColumns = $derived(columns.filter((columnDefinition) => !columnDefinition.hidden));
  const gridTemplateColumns = $derived(
    visibleColumns.map((columnDefinition) => columnDefinition.width).join(' '),
  );
  const normalizedRowHeight = $derived(Math.max(24, Math.round(rowHeight)));
  let shellElement = $state<HTMLDivElement | undefined>(undefined);
  let verticalScrollbarWidth = $state(0);

  // Resolve the selected record identity only when a resolver exists.
  const selectedRecordResolvedId = $derived.by(() => {
    if (!selectedRecord || !getRowId) return undefined;
    return getRowId(selectedRecord, -1);
  });

  const debugVirtualListInfo = (virtualInfo: unknown) => {
    if (!debug) return;
    console.debug('[TableGrid] virtual-list debug info', virtualInfo);
  };

  const getCellValue = (
    rowRecord: TRecord,
    columnDefinition: TableGridColumn<TRecord>,
    rowIndex: number,
  ): string | number => {
    if (!columnDefinition.getValue) return '';
    return columnDefinition.getValue(rowRecord, rowIndex);
  };

  const getAlignClassName = (align: TableGridColumn<TRecord>['align']) => {
    if (align === 'center') return 'text-center';
    if (align === 'right') return 'text-right';
    return 'text-left';
  };

  const isSelectedRow = (rowRecord: TRecord, rowIndex: number): boolean => {
    // Fast path for record-reference selection used by existing VTable screens.
    if (selectedRecord && rowRecord === selectedRecord) {
      return true;
    }

    if (!getRowId) {
      return false;
    }

    const currentRowId = getRowId(rowRecord, rowIndex);

    // ID-based selection is useful when record references change between fetches.
    if (selectedRowId !== undefined && selectedRowId !== null && currentRowId === selectedRowId) {
      return true;
    }

    // Keep selectedRecord compatible even when parent sends a cloned object.
    if (
      selectedRecordResolvedId !== undefined
      && selectedRecordResolvedId !== null
      && currentRowId === selectedRecordResolvedId
    ) {
      return true;
    }

    return false;
  };

  const handleRowClick = (rowRecord: TRecord, rowIndex: number) => {
    if (debug) {
      console.debug('[TableGrid] row click', { rowIndex, rowRecord });
    }
    onRowClick?.(rowRecord, rowIndex);
  };

  const syncHeaderScrollbarCompensation = () => {
    if (!shellElement) return;

    const viewportElement = shellElement.querySelector('.table-grid-virtual-viewport') as HTMLDivElement | null;
    if (!viewportElement) return;

    // Match header width to the scrollable body viewport when the vertical scrollbar consumes space.
    verticalScrollbarWidth = Math.max(0, viewportElement.offsetWidth - viewportElement.clientWidth);
  };

  onMount(() => {
    if (!shellElement) return;

    const resizeObserver = new ResizeObserver(() => {
      syncHeaderScrollbarCompensation();
    });

    resizeObserver.observe(shellElement);

    const viewportElement = shellElement.querySelector('.table-grid-virtual-viewport') as HTMLDivElement | null;
    if (viewportElement) {
      resizeObserver.observe(viewportElement);
    }

    queueMicrotask(() => {
      syncHeaderScrollbarCompensation();
    });

    return () => {
      resizeObserver.disconnect();
    };
  });

  $effect(() => {
    data.length;
    normalizedRowHeight;
    bufferSize;

    queueMicrotask(() => {
      syncHeaderScrollbarCompensation();
    });
  });
</script>

<div class="table-grid-shell {css}"
  bind:this={shellElement}
  style="height: {height}; --table-grid-template-columns: {gridTemplateColumns}; --table-grid-row-height: {normalizedRowHeight}px; --table-grid-scrollbar-width: {verticalScrollbarWidth}px;"
>
  <div class="table-grid-scroll-host">
    <div class="table-grid-header {headerCss}">
      <div class="table-grid-header-row" role="row">
        {#each visibleColumns as columnDefinition (columnDefinition.id)}
          <div class="table-grid-header-cell {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
            role="columnheader"
          >
            {columnDefinition.header}
          </div>
        {/each}
      </div>
    </div>

    <div class="table-grid-body">
      {#if data.length === 0}
        <div class="table-grid-empty">{emptyMessage}</div>
      {:else}
        <SvelteVirtualList items={data}
          bufferSize={bufferSize}
          defaultEstimatedItemHeight={normalizedRowHeight}
          debug={debug}
          debugFunction={debugVirtualListInfo}
          containerClass="table-grid-virtual-container h-full"
          viewportClass="table-grid-virtual-viewport"
          contentClass="table-grid-virtual-content w-full"
          itemsClass="table-grid-virtual-items w-full [&>div]:w-full"
        >
          {#snippet renderItem(rowRecord, rowIndex)}
            {@const selected = isSelectedRow(rowRecord, rowIndex)}
            <div class="table-grid-row-shell" style="height: var(--table-grid-row-height);">
              <div class="table-grid-row {rowCss}"
                class:table-grid-row-even={rowIndex % 2 === 0}
                class:table-grid-row-odd={rowIndex % 2 !== 0}
                class:table-grid-row-selected={selected}
                role="row"
                tabindex="0"
                onclick={() => handleRowClick(rowRecord, rowIndex)}
                onkeydown={(eventInfo) => {
                  if (eventInfo.key === 'Enter' || eventInfo.key === ' ') {
                    handleRowClick(rowRecord, rowIndex);
                  }
                }}
              >
                {#each visibleColumns as columnDefinition (columnDefinition.id)}
                  {@const defaultCellValue = getCellValue(rowRecord, columnDefinition, rowIndex)}
                  <div class="table-grid-cell [&:last-child]:border-r-0 {getAlignClassName(columnDefinition.align)} {columnDefinition.cellCss || ''}"
                    role="cell"
                    title={`${defaultCellValue}`}
                  >
                    {#if cellRenderer && columnDefinition.useCellRenderer}
                      {@render cellRenderer(rowRecord, columnDefinition, rowIndex)}
                    {:else}
                      {defaultCellValue}
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/snippet}
        </SvelteVirtualList>
      {/if}
    </div>
  </div>
</div>

<style>
  .table-grid-shell {
    border: 1px solid #dee2e6;
    border-radius: 8px;
    background-color: #ffffff;
    overflow: hidden;
    min-height: 0;
    box-sizing: border-box;
  }

  .table-grid-scroll-host {
    display: grid;
    grid-template-rows: auto minmax(0, 1fr);
    height: 100%;
    min-height: 0;
    overflow-x: hidden;
    overflow-y: hidden;
  }

  .table-grid-header {
    border-bottom: 1px solid #e9ecef;
    background: #f8f9fa;
    position: relative;
    z-index: 2;
    padding-right: var(--table-grid-scrollbar-width);
    box-sizing: border-box;
  }

  .table-grid-header-row,
  .table-grid-row {
    display: grid;
    grid-template-columns: var(--table-grid-template-columns);
    width: 100%;
    min-width: max-content;
  }

  .table-grid-header-cell {
    padding: 8px 10px;
    font-family: bold;
    color: #495057;
    border-right: 1px solid #edf2f7;
    white-space: nowrap;
  }

  .table-grid-header-cell:last-child {
    border-right: none;
  }

  .table-grid-body {
    height: 100%;
    min-height: 0;
    position: relative;
    box-sizing: border-box;
  }

  .table-grid-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 140px;
    color: #6c757d;
    padding: 16px;
  }

  .table-grid-row-shell {
    width: 100%;
    padding: 1px 2px;
    box-sizing: border-box;
  }

  .table-grid-row {
    height: var(--table-grid-row-height);
    cursor: pointer;
    border-bottom: 1px solid #edf2f7;
    transition: background-color 0.15s ease;
    position: relative;
  }

  .table-grid-row:focus-visible {
    outline: 2px solid #3b82f6;
    outline-offset: -2px;
  }

  .table-grid-row:hover {
    background-color: #f1f3f5;
  }

  .table-grid-row-even {
    background: #ffffff;
  }

  .table-grid-row-odd {
    background: #f8f9fa;
  }

  .table-grid-row-selected,
  .table-grid-row-selected.table-grid-row:hover {
    background-color: #f6f6ff;
    outline: 2px solid var(--color-11);
    outline-offset: -1px;
    border-radius: 4px;
    border-bottom-color: transparent;
    z-index: 12;
  }

  .table-grid-cell {
    border-right: 1px solid #f1f5f9;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    align-content: center;
    min-width: 0;
  }

  :global(.table-grid-virtual-viewport) {
    height: 100% !important;
    overflow-y: auto !important;
    overflow-x: hidden !important;
  }

  :global(.table-grid-virtual-container) {
    height: 100% !important;
    min-height: 0 !important;
  }

  :global(.table-grid-virtual-content) {
    min-height: 100%;
  }
</style>
