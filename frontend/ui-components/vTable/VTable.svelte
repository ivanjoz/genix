<script lang="ts" generics="T">
  import { untrack } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { createVirtualizer } from './index.svelte';
  import type { ITableColumn, CellRendererSnippet, IMobileCardsListCell } from "./types";
  import type { VirtualItem } from './index.svelte';
  import CellEditable from '$components/vTable/CellEditable.svelte';
  import CellSelector from '$components/vTable/CellSelector.svelte';
  import { highlString, wordInclude } from '$libs/helpers';
  import Renderer, { type ElementAST } from '$components/Renderer.svelte';
  import MobileCardsVirtualList from '$components/vTable/MobileCardsVirtualList.svelte';

  interface ICellContent {
    content: string;
    contentHTML: string;
    contentAST: ElementAST | ElementAST[],
    prefixHTML: string;
    prefixAST: ElementAST | ElementAST[],
    useSnippet: boolean;
    css: string
  }

  interface VTableProps<T> {
    columns: ITableColumn<T>[];
    data: T[];
    maxHeight?: string;
    css?: string;
    tableCss?: string;
    cellCss?: string;
    estimateSize?: number;
    overscan?: number;
    onRowClick?: (row: T, index: number, rerender: () => void) => void;
    selected?: T | number;
    isSelected?: (row: T, selected: T | number) => boolean;
    emptyMessage?: string;
    cellRenderer?: CellRendererSnippet<T>;
    filterText?: string;
    getFilterContent?: (row: T) => string;
    useFilterCache?: boolean;
    mobileCardCss?: string
    getRowObject?: (row: Partial<T>) => Promise<T>
    cellInputType?: 'number';
  }

  let {
    columns,
    data,
    maxHeight = 'calc(100vh - 8rem)',
    css = '',
    tableCss = '',
    cellCss = '',
    estimateSize = 34,
    overscan = 15,
    onRowClick,
    selected,
    isSelected,
    emptyMessage = 'No se encontraron registros.',
    cellRenderer,
    filterText,
    getFilterContent,
    useFilterCache = false,
    mobileCardCss = '',
    getRowObject,
    cellInputType,
  }: VTableProps<T> = $props();

  // State
  let containerRef = $state<HTMLDivElement>();
  let virtualItems = $state<VirtualItem[]>([]);
  let totalSize = $state(0);
  let virtualizerStore: ReturnType<typeof createVirtualizer> | null = null;
  let isInitialized = false;
  let dataVersion = $state(0);
  let rowObjectVersion = $state(0);
  const filterCache = new WeakMap<T & object, string>();
  const resolvedRowByKey = new Map<object | string, T>();
  const loadingRowKeys = new Set<object | string>();
  const queuedRowKeys = new Set<object | string>();
  // Per-row version counters used to force a specific row to unmount/remount
  // when handlers invoke the `rerender` callback.
  const rowVersions = new SvelteMap<number, number>();

  function rerenderRow(rowIndex: number) {
    const nextRowVersion = (rowVersions.get(rowIndex) || 0) + 1;
    rowVersions.set(rowIndex, nextRowVersion);
    // SvelteMap updates alone have not been enough to invalidate the keyed row
    // in all call paths, so also bump the table version to force reconciliation.
    dataVersion++;
  }

  // Mobile view state
  let windowWidth = $state(typeof window !== 'undefined' ? window.innerWidth : 1024);

  // Handle window resize
  $effect(() => {
    const handleResize = () => {
      windowWidth = window.innerWidth;
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  });

  // Process columns to calculate colspans and separate header levels
  const processedColumns = $derived.by(() => {
    const level1: ITableColumn<T>[] = [];
    const level2: ITableColumn<T>[] = [];
    const flatColumns: ITableColumn<T>[] = [];

    for (const col of columns) {
      if (col.hidden) continue;

      const colWithSpan = { ...col };
      const visibleSubcols = (col.subcols || []).filter((subcol) => !subcol.hidden);
      colWithSpan._colspan = visibleSubcols.length || 0;
      colWithSpan.subcols = visibleSubcols;

      level1.push(colWithSpan);

      if (visibleSubcols.length > 0) {
        for (const subcol of visibleSubcols) {
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

  // Flatten the mobile-only configuration so the shared renderer can treat table cards like any other card list.
  const mobileColumns = $derived.by((): IMobileCardsListCell<T, ITableColumn<T>>[] => {
    return processedColumns.flatColumns
      .filter(col => col.mobile)
      .sort((a, b) => (a.mobile?.order || 0) - (b.mobile?.order || 0))
      .map((column) => ({
        ...column,
        source: column,
        itemCss: column.mobile?.css || 'col-span-full',
        labelTop: column.mobile?.labelTop,
        labelLeft: column.mobile?.labelLeft,
        labelCss: 'color-label',
        icon: column.mobile?.icon,
        iconCss: column.mobile?.iconCss,
        elementLeft: column.mobile?.elementLeft,
        elementRight: column.mobile?.elementRight,
        mobileRender: column.mobile?.render,
        if: column.mobile?.if,
        useRenderer: Boolean(cellRenderer && column.id),
      }));
  });

  // Mobile view - only enable if columns have mobile config and screen is small
  const isMobileView = $derived(windowWidth < 580 && mobileColumns.length > 0);

  const filterTextArray = $derived((filterText||"").toLowerCase().split(" ").filter(x => x.length > 1))

  const filteredData = $derived.by(() => {
    if(filterText && getFilterContent){
      const filtered = data.filter(
        x => {
          let content = ""
          if (useFilterCache && typeof x === 'object' && x !== null) {
            const cached = filterCache.get(x);
            if (cached) {
              content = cached;
            } else {
              content = getFilterContent(x).toLowerCase();
              filterCache.set(x, content);
            }
          } else {
            content = getFilterContent(x).toLowerCase();
          }
          return wordInclude(content, filterTextArray)
        })
      return filtered
    } else {
      return data
    }
  })

  function getRowKey(record: T, index: number): object | string {
    return typeof record === 'object' && record !== null ? record : `row_${index}`;
  }

  // Resolve row objects lazily so tables can start from partial rows and hydrate details on demand.
  function getResolvedRow(record: T, index: number): T | null {
    if (!getRowObject) {
      return record;
    }

    rowObjectVersion;
    const rowKey = getRowKey(record, index);
    const cachedRow = resolvedRowByKey.get(rowKey);
    if (cachedRow) {
      return cachedRow;
    }

    if (!loadingRowKeys.has(rowKey) && !queuedRowKeys.has(rowKey)) {
      queuedRowKeys.add(rowKey);

      // Defer hydration kickoff so template evaluation stays side-effect free.
      queueMicrotask(() => {
        queuedRowKeys.delete(rowKey);
        if (loadingRowKeys.has(rowKey) || resolvedRowByKey.has(rowKey)) {
          return;
        }

        loadingRowKeys.add(rowKey);
        rowObjectVersion++;

        void getRowObject(record as Partial<T>)
          .then((resolvedRow) => {
            // Keep the source row visible when hydration returns nothing.
            resolvedRowByKey.set(rowKey, resolvedRow || record);
          })
          .catch((resolveError) => {
            console.error('[VTable] getRowObject error', resolveError);
            // Stop the loading loop and keep the original row if the async hydration fails.
            resolvedRowByKey.set(rowKey, record);
          })
          .finally(() => {
            loadingRowKeys.delete(rowKey);
            rowObjectVersion++;
          });
      });
    }

    return null;
  }

  // Initialize virtualizer
  $effect(() => {
    if (isMobileView) return;
    if (containerRef && !virtualizerStore) {
      virtualizerStore = createVirtualizer({
        count: 0,
        getScrollElement: () => containerRef!,
        estimateSize: () => estimateSize,
        overscan: overscan,
        getCount: () => filteredData.length
      });

      const updateVirtualItems = () => {
        if(!virtualizerStore){ return }
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
  $effect(() => {
    // Track both filteredData and its length to ensure changes are detected
    const currentData = filteredData;
    const currentLength = filteredData.length;

    if (isInitialized && virtualizerStore) {
      untrack(() => {
        dataVersion++;

        // Notify virtualizer of the change
        virtualizerStore!.refresh();

        const items = virtualizerStore!.getVirtualItems();
        const size = virtualizerStore!.getTotalSize();

        virtualItems = [...items];
        totalSize = size;
      });
    }
  });

  // Helper to get cell content
  function getCellContent(column: ITableColumn<T>, record: T, index: number): ICellContent {

    const rec = { } as ICellContent

    // getValue runs first so any side-effects (e.g. lazy compute) are visible to render
    if (column.getValue) {
      rec.content = column.getValue(record, index) as string
    }

    if (column.renderPrefix) {
      const renderedPrefix = column.renderPrefix(record, index)
      if (typeof renderedPrefix === 'string') {
        rec.prefixHTML = renderedPrefix
      } else if (renderedPrefix) {
        rec.prefixAST = renderedPrefix
      }
    }

    if (column.render) {
      const renderedContent = column.render(record, index)
      if(typeof renderedContent === 'string'){
        rec.contentHTML = renderedContent
      } else if(renderedContent){
        rec.contentAST = renderedContent
      }
    }

    // Check if we should use snippet renderer (takes priority over function renderer)
    if (cellRenderer && column.id) { rec.useSnippet = true; }

    rec.css = typeof column.cellCss === 'string'
      ? column.cellCss
      : (column.onCellEdit ? "relative" : "px-8 py-4")
    // Append runtime classes returned by setCellCss for this row
    const dynamicCellCss = column.setCellCss?.(record)
    if (dynamicCellCss) {
      rec.css += ` ${dynamicCellCss}`
    }
    if(column.css){ rec.css += " " + column.css }

    return rec
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
  function handleRowClick(record: T, index: number, rerender: () => void) {
    if (onRowClick) {
      onRowClick(record, index, rerender);
    }
  }
</script>

<div bind:this={containerRef}
  class="vtable-container {css}" class:_14={isMobileView}
  style="max-height: {maxHeight}; overflow: {isMobileView ? 'hidden' : 'auto'};"
>
  {#if isMobileView}
    <!-- Mobile Card View -->
    <div class="mobile-card-container" style="height: {maxHeight};">
      <MobileCardsVirtualList
        data={filteredData}
        cells={mobileColumns}
        variant="compact"
        cardCss={`mb-6 ${mobileCardCss}`.trim()}
        estimateSize={estimateSize}
        overscan={overscan}
        emptyMessage={emptyMessage}
        loadingMessage="Loading..."
        onRowClick={onRowClick ? handleRowClick : undefined}
        resolveRecord={getRowObject ? getResolvedRow : undefined}
        debugName="VTable"
        tableCellRenderer={cellRenderer}
      />
    </div>
  {:else}
    <!-- Desktop Table View -->
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
            <div class={column.headerInnerCss || ''}>
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
              <div class={column.headerInnerCss || ''}>
                {getHeaderContent(column)}
              </div>
            </th>
          {/each}
        </tr>
      {/if}
    </thead>

    <!-- Virtual Body -->
    <tbody class="vtable-body">
      {#if filteredData.length === 0}
        <tr>
          <td colspan={processedColumns.flatColumns.length} class="vtable-empty">
            <div class="vtable-empty-message">
              {emptyMessage}
            </div>
          </td>
        </tr>
      {:else}
        <!-- Spacer row to keep selected outline visible under sticky header -->
        <tr class="vtable-edge-spacer" aria-hidden="true">
          <td colspan={processedColumns.flatColumns.length}></td>
        </tr>

        {#each virtualItems as row, i (`${row.index}-${dataVersion}-${rowVersions.get(row.index) || 0}`)}
          {@const firstItemStart = virtualItems[0]?.start || 0}
          {@const isFinal = i === virtualItems.length - 1}
          {@const remainingSize = totalSize - (virtualItems[0]?.size || estimateSize) * virtualItems.length}
          {@const record = filteredData[row.index]}
          {@const resolvedRecord = record ? getResolvedRow(record, row.index) : null}

          {#if record}
            {@const selected = resolvedRecord ? isRowSelected(resolvedRecord, row.index) : false}

          <tr class="vtable-row"
            class:vtable-row-even={row.index % 2 === 0}
            class:vtable-row-odd={row.index % 2 !== 0}
            class:vtable-row-selected={selected}
            style="transform: translateY({firstItemStart}px); height: {estimateSize}px;"
            onclick={() => resolvedRecord && handleRowClick(resolvedRecord, row.index, () => rerenderRow(row.index))}
          >
            {#if !resolvedRecord}
              <td colspan={processedColumns.flatColumns.length}
                class="vtable-loading"
                style="height: {estimateSize}px;"
              >
                Loading...
              </td>
            {:else}
              {#each processedColumns.flatColumns as column, j (`${j}_${filterText||""}`)}
                {@const cellData = getCellContent(column, resolvedRecord, row.index)}                
                
                <td class="{cellCss} {cellData.css} {column.onCellClick && !column.disableCellInteractions?.(resolvedRecord, row.index) ? '_clickable-cell' : ''}"
                  style={column.cellStyle ? Object.entries(column.cellStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
                  onclick={ev => {
                    if(column.onCellEdit){ ev.stopPropagation() }
                    if(column.onCellClick){ 
                    	ev.stopPropagation() 
                      column.onCellClick(resolvedRecord, row.index, () => rerenderRow(row.index)) 
                    }
                  }}
                >
                  {#if column.showEditIcon && !column.disableCellInteractions?.(resolvedRecord, row.index)}
                    <i class="icon-pencil _edit-icon"></i>
                  {/if}
                  {#if cellData.prefixAST}
                    <span class="vtable-cell-prefix">
                      <Renderer elements={cellData.prefixAST}/>
                    </span>
                  {:else if cellData.prefixHTML}
                    <span class="vtable-cell-prefix">
                      {@html cellData.prefixHTML}
                    </span>
                  {/if}
                  {#if column.onCellEdit && !column.disableCellInteractions?.(resolvedRecord, row.index)}
                    <CellEditable contentClass={"px-6 "+(column.css||"")}
                      inputClass={"px-6 "+(column.inputCss)}
                      type={column.cellInputType || cellInputType}
                      getValue={() => cellData.content}
                      render={
                        (column.render
                        ? () => column.render?.(resolvedRecord, row.index)
                        : undefined) as (value: number | string) => ElementAST[]
                      }
                      onChange={(value: string | number) => {
                        column.onCellEdit?.(resolvedRecord, value, () => rerenderRow(row.index))
                      }}
                    />
                  {:else if column.onCellSelect}
                    {console.log('[VTable] CellSelector props', {
                      selectorId: `${String(column.id || column.field || j)}_${row.index}`,
                      rowIndex: row.index,
                      columnField: column.field,
                      columnId: column.id,
                      optionsLength: column.cellOptions?.length || 0,
                      firstOption: column.cellOptions?.[0] || null,
                      currentFieldValue: column.field ? (resolvedRecord as any)?.[column.field] : undefined,
                    })}
                    <CellSelector
                      id={`${String(column.id || column.field || j)}_${row.index}`}
                      saveOn={resolvedRecord}
                      save={column.field as keyof T}
                      options={(column.cellOptions || []) as any[]}
                      keyId={(column.cellOptionsKeyId || 'ID') as never}
                      keyName={(column.cellOptionsKeyName || 'Name') as never}
                      contentClass={column.css}
                      onChange={(value: string | number) => {
                        column.onCellSelect?.(resolvedRecord, value, () => rerenderRow(row.index))
                      }}
                    />
                  {:else if cellData.useSnippet && cellRenderer}
                    {@render cellRenderer(resolvedRecord, column, cellData.content, row.index, false)}
                  {:else if cellData.contentAST}
                    <Renderer elements={cellData.contentAST}/>
                  {:else if cellData.contentHTML}
                    {@html cellData.contentHTML}
                  {:else}
                    {#if filterText && column.highlight}
                      {#each highlString(cellData.content, filterTextArray) as part }
                        <span class:_2={part.highl}>{part.text}</span>
                      {/each }
                    {:else}
                      { cellData.content }
                    {/if}
                  {/if}
                  {#if column.buttonEditHandler || column.buttonDeleteHandler}
                    <div class="flex gap-4 items-center justify-center">
                      {#if column.buttonEditHandler && (!column.buttonEditIf || column.buttonEditIf(resolvedRecord))}
                        <button class="_11 _e" title="edit" onclick={ev => {
                          ev.stopPropagation()
                          column.buttonEditHandler?.(resolvedRecord)
                        }}>
                          <i class="icon-pencil"></i>
                        </button>
                      {/if}
                      {#if column.buttonDeleteHandler && (!column.buttonDeleteIf || column.buttonDeleteIf(resolvedRecord))}
                        <button class="_11 _d" title="delete" onclick={ev => {
                          ev.stopPropagation()
                          column.buttonDeleteHandler?.(resolvedRecord)
                        }}>
                          <i class="icon-trash"></i>
                        </button>
                      {/if}
                    </div>
                  {/if}
                </td>
              {/each}
            {/if}
          </tr>

          {#if isFinal}
            <tr style="height: {remainingSize}px; visibility: hidden;">
              <td style="border: none;"></td>
            </tr>
          {/if}
          {/if}
        {/each}

        <!-- Spacer row to keep selected outline visible at table bottom -->
        <tr class="vtable-edge-spacer" aria-hidden="true">
          <td colspan={processedColumns.flatColumns.length}></td>
        </tr>
      {/if}
    </tbody>
  </table>
  {/if}
</div>

<style>
  ._2 {
    color: #da3c3c;
    text-decoration: underline;
  }

  .hsc > div {
    background-color: #e9ecef;
  }

  .vtable-cell-prefix {
    display: inline-flex;
    align-items: center;
    vertical-align: middle;
  }

  .vtable-container:not(._14) {
    background-color: white;
    border: 1px solid #dee2e6;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    /* No inner top inset: prevents body rows from peeking above sticky header while scrolling */
    padding: 0 2px;
  }

  .vtable-container._14 {
    width: calc(100% + 8px);
    margin-left: -4px;
    margin-right: -4px;
  }

  @media (max-width: 579px) {
    .vtable-container :global(.virtual-list-items) {
      padding: 4px;
    }
  }

  .vtable {
    width: 100%;
    border-collapse: separate;
    background-color: white;
    border-spacing: 0;
    background-color: white;
    isolation: isolate;
  }

  .vtable-header {
    position: sticky;
    top: 0;
    /* Keep header fully above transformed virtual rows */
    z-index: 30;
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
    /* Prevent body rows from bleeding through sticky header cells */
    background-color: #f8f9fa;
  }

  .vtable-header-cell:last-child {
    border-right: none;
  }

  .vtable-header-cell > div {
    height: 100%;
    min-height: 36px;
    border-bottom: 1px solid rgb(204, 204, 204);
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1.1;
    padding: 0 4px;
    background-color: #f8f9fa;
  }

  .vtable-body {
    position: relative;
  }

  .vtable-row {
    display: table-row;
    width: 100%;
    border-bottom: 1px solid #e9ecef;
    transition: background-color 0.15s ease;
    cursor: pointer;
    height: var(--row-height);
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

  .vtable-row-selected, .vtable-row-selected.vtable-row:hover {
    background-color: #f6f6ff;
    outline: 2px solid var(--color-11);
    border-radius: 5px;
    position: relative;
    z-index: 12;
  }

  .vtable-edge-spacer td {
    height: 2px;
    padding: 0;
    border: none !important;
    background: transparent;
  }

  .vtable-row > td:not(:last-of-type) {
    display: table-cell;
    text-overflow: ellipsis;
   /*  white-space: nowrap; */
    vertical-align: middle;
    border-right: 1px solid #f1f3f5;
    position: relative;
  }

  .vtable-cell:last-child {
    border-right: none;
  }

  ._edit-icon {
    position: absolute;
    left: 4px;
    top: 50%;
    transform: translateY(-50%);
    font-size: 0.75rem;
    color: #4f6ef7;
    opacity: 0;
    pointer-events: none;
  }

  .vtable-row:hover ._edit-icon {
    opacity: 1;
  }

  ._clickable-cell {
    cursor: pointer;
    position: relative;
  }

  ._clickable-cell:hover {
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.596);
    z-index: 1;
  }

  .vtable-empty {
    padding: 2rem;
    text-align: center;
  }

  .vtable-loading {
    padding: 0 8px;
    text-align: center;
    vertical-align: middle;
  }

  .vtable-empty-message {
    color: #6c757d;
    font-size: 0.875rem;
  }

  ._11 {
    border-radius: 50%;
    width: 30px;
    height: 30px;
    font-size: 15px;
    color: #5243c2;
    box-shadow: rgb(67 61 110 / 62%) 0px 1px 2px 0px;
  }
  ._11._e {
    color: #5243c2;
    background-color: #e7e4ff;
    box-shadow: rgb(67 61 110 / 62%) 0px 1px 2px 0px;
  }
  ._11._e:hover {
    outline: 1px solid #5243c2;
    background-color: #f5f4ff;
  }

  /* Mobile Card Styles */
  .mobile-card-container {
    width: 100%;
  }

  .mobile-empty-message {
    text-align: center;
    padding: 2rem;
    color: #6c757d;
    font-size: 0.875rem;
  }

  .mobile-loading-message {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #6c757d;
    font-size: 0.875rem;
  }

  .mobile-card {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 12px;
    cursor: pointer;
    transition: box-shadow 0.2s ease;
  }

  .mobile-card:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }

  .mobile-card-grid {
    display: grid;
    grid-template-columns: repeat(24, 1fr);
    gap: 4px;
  }

  .mobile-card-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .mobile-card-item-vertical {
    flex-direction: column;
    align-items: flex-start;
    row-gap: 0;
  }

  .mobile-card-label-top {
    font-size: 14px;
   /*  color: #666;*/
    line-height: 1;
  }

  .mobile-card-content-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }

  .mobile-card-label-left {
    font-size: 14px;
    /*  color: #666;*/
    flex-shrink: 0;
  }

  .mobile-card-left,
  .mobile-card-right {
    flex-shrink: 0;
  }

  .mobile-card-content {
    flex: 1;
    min-width: 0;
    word-break: break-word;
    position: relative;
  }

  .mobile-card-content-interactive {
    min-height: 32px;
  }

  .mobile-cell-editable-border {
    overflow: hidden;
    position: absolute;
    bottom: -4px;
    width: calc(100% + 4px);
    height: 18px;
    left: -2px;
  }

  .mobile-cell-editable-border > div {
    width: calc(100% - 4px);
    border: 1px solid #d2d5e7;
    height: 24px;
    border-top: none;
    box-shadow: #706e9021 0 1px 2px 1px;
    position: absolute;
    bottom: 4px;
    left: 2px;
    border-radius: 4px;
  }
</style>
