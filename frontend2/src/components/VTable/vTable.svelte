<script lang="ts" generics="T">
  import { untrack } from 'svelte';
  import { createVirtualizer } from './index.svelte';
  import type { ITableColumn, CellRendererSnippet } from "./types";
  import type { VirtualItem } from './index.svelte';
    import CellEditable from '../CellEditable.svelte';

  interface VTableProps<T> {
    columns: ITableColumn<T>[];
    data: T[];
    maxHeight?: string;
    css?: string;
    tableCss?: string;
    estimateSize?: number;
    overscan?: number;
    onRowClick?: (row: T, index: number) => void;
    selected?: T | number;
    isSelected?: (row: T, selected: T | number) => boolean;
    emptyMessage?: string;
    cellRenderer?: CellRendererSnippet<T>;
  }

  let {
    columns,
    data,
    maxHeight = 'calc(100vh - 8rem)',
    css = '',
    tableCss = '',
    estimateSize = 34,
    overscan = 15,
    onRowClick,
    selected,
    isSelected,
    emptyMessage = 'No se encontraron registros.',
    cellRenderer,
  }: VTableProps<T> = $props();

  // State
  let containerRef = $state<HTMLDivElement>();
  let virtualItems = $state<VirtualItem[]>([]);
  let totalSize = $state(0);
  let virtualizerStore: ReturnType<typeof createVirtualizer> | null = null;
  let isInitialized = false;
  let dataVersion = $state(0);

  // Process columns to calculate colspans and separate header levels
  const processedColumns = $derived.by(() => {
    const level1: ITableColumn<T>[] = [];
    const level2: ITableColumn<T>[] = [];
    const flatColumns: ITableColumn<T>[] = [];

    for (const col of columns) {
      const colWithSpan = { ...col };
      colWithSpan._colspan = col.subcols?.length || 0;
      
      level1.push(colWithSpan);
      
      if (col.subcols && col.subcols.length > 0) {
        for (const subcol of col.subcols) {
          level2.push(subcol);
          flatColumns.push(subcol);
        }
      } else {
        flatColumns.push(colWithSpan);
      }
    }

    return {
      level1,
      level2,
      flatColumns,
      hasSubcols: level2.length > 0
    };
  });

  // Initialize virtualizer
  $effect(() => {
    if (containerRef && !virtualizerStore) {
      virtualizerStore = createVirtualizer({
        count: 0,
        getScrollElement: () => containerRef!,
        estimateSize: () => estimateSize,
        overscan: overscan,
        getCount: () => data.length
      });

      const updateVirtualItems = () => {
        const items = virtualizerStore!.getVirtualItems();
        const size = virtualizerStore!.getTotalSize();
        
        virtualItems = [...items];
        totalSize = size;
      };

      const unsubscribe = virtualizerStore.subscribe(updateVirtualItems);

      requestAnimationFrame(() => {
        updateVirtualItems();
        isInitialized = true;
      });

      return () => {
        isInitialized = false;
        virtualizerStore = null;
        unsubscribe();
      };
    }
  });

  // Watch for data changes
  let lastDataRef: any = null;
  $effect(() => {
    const currentData = data;
    
    if (lastDataRef !== null && currentData !== lastDataRef && isInitialized && virtualizerStore) {
      untrack(() => {
        dataVersion++;
        
        const items = virtualizerStore!.getVirtualItems();
        const size = virtualizerStore!.getTotalSize();
        
        virtualItems = [...items];
        totalSize = size;
      });
    }
    
    lastDataRef = currentData;
  });

  // Helper to get cell content
  function getCellContent(column: ITableColumn<T>, record: T, index: number): { 
    content: any; 
    isHTML: boolean; 
    useSnippet: boolean;
    css: string
  } {
    let content: any = '';
    let isHTML = false;
    let useSnippet = false;

    if (column.renderHTML) {
      // Explicitly render as HTML
      content = column.renderHTML(record, index, () => {
        dataVersion++;
      });
      isHTML = true;
    } else if (column.render) {
      // Render function - may return HTML or other content
      content = column.render(record, index, () => {
        dataVersion++;
      });
      // Assume it's HTML if it's a string containing HTML tags
      isHTML = typeof content === 'string' && /<[^>]+>/.test(content);
    } else if (column.getValue) {
      content = column.getValue(record, index);
      isHTML = false;
    } else if (column.field) {
      content = (record as any)[column.field];
      isHTML = false;
    }

    // Check if we should use snippet renderer (takes priority over function renderer)
    if (cellRenderer) {
      useSnippet = true;
    }

    let css = typeof column.cellCss === 'string' 
      ? column.cellCss
      : (column.onEditChange ? "relative" : "px-8 py-4")
    if(column.css){ css += " " + column.css }

    return { content, isHTML, useSnippet, css };
  }

  // Helper to get header content
  function getHeaderContent(column: ITableColumn<T>): string {
    return typeof column.header === 'function' ? column.header() : column.header;
  }

  // Check if row is selected
  function isRowSelected(record: T, index: number): boolean {
    if (!selected || !isSelected) return false;
    return isSelected(record, selected);
  }

  // Handle row click
  function handleRowClick(record: T, index: number) {
    if (onRowClick) {
      onRowClick(record, index);
    }
  }
</script>

<div
  bind:this={containerRef}
  class="vtable-container {css}"
  style="max-height: {maxHeight}; overflow: auto;"
>
  <table class="vtable {tableCss}">
    <!-- Header -->
    <thead class="vtable-header">
      <!-- First level headers -->
      <tr class="vtable-header-row">
        {#each processedColumns.level1 as column}
          <th class="vtable-header-cell {column.headerCss || ''}"
            class:hsc={(column.subcols||[]).length > 0}
            style={column.headerStyle ? Object.entries(column.headerStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
            colspan={column._colspan || 1}
            rowspan={column._colspan ? 1 : (processedColumns.hasSubcols ? 2 : 1)}
          >
            <div>
              {getHeaderContent(column)}
            </div>
          </th>
        {/each}
      </tr>
      
      <!-- Second level headers (if subcolumns exist) -->
      {#if processedColumns.hasSubcols}
        <tr class="vtable-header-row vtable-header-row-sub">
          {#each processedColumns.level2 as column}
            <th
              class="vtable-header-cell {column.headerCss || ''}"
              style={column.headerStyle ? Object.entries(column.headerStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
            >
              <div>
                {getHeaderContent(column)}
              </div>
            </th>
          {/each}
        </tr>
      {/if}
    </thead>

    <!-- Virtual Body -->
    <tbody class="vtable-body">
      {#if data.length === 0}
        <tr>
          <td colspan={processedColumns.flatColumns.length} class="vtable-empty">
            <div class="vtable-empty-message">
              {emptyMessage}
            </div>
          </td>
        </tr>
      {:else}
        {#each virtualItems as row, i (`${row.index}-${dataVersion}`)}
          {@const firstItemStart = virtualItems[0]?.start || 0}
          {@const isFinal = i === virtualItems.length - 1}
          {@const remainingSize = totalSize - (virtualItems[0]?.size || estimateSize) * virtualItems.length}
          {@const record = data[row.index]}
          {@const selected = isRowSelected(record, row.index)}
          
          <tr class="vtable-row"
            class:vtable-row-even={row.index % 2 === 0}
            class:vtable-row-odd={row.index % 2 !== 0}
            class:vtable-row-selected={selected}
            style="height: {row.size}px; transform: translateY({firstItemStart}px);"
            onclick={() => handleRowClick(record, row.index)}
          >
            {#each processedColumns.flatColumns as column}
              {@const cellData = getCellContent(column, record, row.index)}
              <td class="{cellData.css}"
                style={column.cellStyle ? Object.entries(column.cellStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
                onclick={ev => {
                  if(column.onEditChange){ ev.stopPropagation() }
                }}
              >
                {#if cellData.isHTML}
                  {@html cellData.content}
                {:else if column.onEditChange}
                  <CellEditable saveOn={record} 
                    getValue={() => cellData.content} 
                    onChange={v => { 
                      column.onEditChange?.(record,v)
                    }}
                  />
                {:else if cellData.useSnippet && cellRenderer}
                  {@render cellRenderer(record, column, cellData.content, row.index)}
                {:else if cellData.isHTML}
                  {@html cellData.content}
                {:else}
                  {cellData.content}
                {/if}
              </td>
            {/each}
          </tr>
          
          {#if isFinal}
            <tr style="height: {remainingSize}px; visibility: hidden;">
              <td style="border: none;"></td>
            </tr>
          {/if}
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .hsc > div {
    background-color: #e9ecef;
  }
  .vtable-container {
    border: 1px solid #dee2e6;
    border-radius: 8px;
    background-color: white;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .vtable {
    width: 100%;
    border-collapse: collapse;
    background-color: white;
  }

  .vtable-header {
    position: sticky;
    top: 0;
    z-index: 10;
    background-color: #f8f9fa;
  }

  .vtable-header-row {
    width: 100%;
    background-color: #f8f9fa;
  }

  .vtable-header-row-sub {
    border-bottom: 1px solid #dee2e6;
  }

  .vtable-header-cell {
    height: 24px;
    font-weight: 600;
    font-size: 15px;
    text-align: left;
    border: none;
    border-right: 1px solid #e9ecef;
    position: relative;
  }

  .vtable-header-cell:last-child {
    border-right: none;
  }

  .vtable-header-cell > div {
    height: 100%;
    width: 100%;
    min-height: 36px;
    border-bottom: 1px solid rgb(204, 204, 204);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .vtable-body {
    position: relative;
  }

  .vtable-row {
    display: table-row;
    width: 100%;
    border-bottom: 1px solid #e9ecef;
    transition: background-color 0.15s ease;
    will-change: transform;
    cursor: pointer;
  }

  .vtable-row:hover {
    background-color: #f1f3f5;
  }

  .vtable-row-even {
    background-color: #ffffff;
  }

  .vtable-row-odd {
    background-color: #f8f9fa;
  }

  .vtable-row-selected {
    background-color: #e7f3ff !important;
  }

  .vtable-row > td {
    display: table-cell;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
    border-right: 1px solid #f1f3f5;
  }

  .vtable-cell:last-child {
    border-right: none;
  }

  .vtable-empty {
    padding: 2rem;
    text-align: center;
  }

  .vtable-empty-message {
    color: #6c757d;
    font-size: 0.875rem;
  }
</style>
