<script lang="ts" generics="TRecord">
  import { onMount } from 'svelte';
  import Renderer from '$components/Renderer.svelte';
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import { splitTwoStrings } from '$libs/helpers';
  import type {
    TableGridCellRendererSnippet,
    TableGridColumn,
    TableGridHeaderRendererSnippet,
  } from './tableGridTypes';

  interface TableGridProps<TRecord> {
    columns: TableGridColumn<TRecord>[];
    data: TRecord[];
    height?: string;
    rowHeight?: number;
    bufferSize?: number;
    mobileBreakpointPx?: number;
    useInnerMobilePadding?: boolean;
    css?: string;
    headerCss?: string;
    rowCss?: string;
    mobileCardCss?: string;
    emptyMessage?: string;
    debug?: boolean;
    onRowClick?: (rowRecord: TRecord, rowIndex: number) => void;
    selectedRowId?: string | number;
    selectedRecord?: TRecord;
    getRowId?: (rowRecord: TRecord, rowIndex: number) => string | number;
    cellRenderer?: TableGridCellRendererSnippet<TRecord>;
    headerRenderer?: TableGridHeaderRendererSnippet<TRecord>;
  }

  let {
    columns,
    data,
    height = '460px',
    rowHeight = 36,
    bufferSize = 12,
    mobileBreakpointPx = 580,
    useInnerMobilePadding = false,
    css = '',
    headerCss = '',
    rowCss = '',
    mobileCardCss = '',
    emptyMessage = 'No se encontraron registros.',
    debug = false,
    onRowClick,
    selectedRowId,
    selectedRecord,
    getRowId,
    cellRenderer,
    headerRenderer,
  }: TableGridProps<TRecord> = $props();

  // Keep a stable filtered list so hidden columns never affect row rendering logic.
  const visibleColumns = $derived(columns.filter((columnDefinition) => !columnDefinition.hidden));
  // Reuse a `VTable`-style mobile contract so existing column definitions can opt into cards incrementally.
  const mobileColumns = $derived.by(() => {
    return visibleColumns
      .filter((columnDefinition) => columnDefinition.mobile)
      .sort((leftColumn, rightColumn) => {
        return (leftColumn.mobile?.order || 0) - (rightColumn.mobile?.order || 0);
      });
  });
  const gridTemplateColumns = $derived(
    visibleColumns.map((columnDefinition) => columnDefinition.width).join(' '),
  );
  const normalizedRowHeight = $derived(Math.max(24, Math.round(rowHeight)));
  const estimatedMobileCardHeight = $derived(Math.max(128, normalizedRowHeight * 3 + 24));
  const useVirtualScroll = $derived(data.length >= 30);
  let windowWidth = $state(typeof window !== 'undefined' ? window.innerWidth : 1024);
  const isMobileView = $derived(windowWidth < mobileBreakpointPx && mobileColumns.length > 0);
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

  const getSplitCellValue = (
    cellValue: string | number,
    columnDefinition: TableGridColumn<TRecord>,
  ): [string, string] => {
    if (typeof cellValue !== 'string' || !columnDefinition.splitString) {
      return [String(cellValue ?? ''), ''];
    }

    // Split long labels into two balanced lines so adjacent columns remain visible.
    return splitTwoStrings(cellValue, columnDefinition.splitString);
  };

  const getAlignClassName = (align: TableGridColumn<TRecord>['align']) => {
    if (align === 'center') return 'text-center';
    if (align === 'right') return 'text-right';
    return 'text-left';
  };

  const isHtmlContent = (contentValue: unknown): contentValue is string => {
    return typeof contentValue === 'string';
  };

  const shouldRenderMobileColumn = (
    rowRecord: TRecord,
    columnDefinition: TableGridColumn<TRecord>,
    rowIndex: number,
  ): boolean => {
    const mobileConfiguration = columnDefinition.mobile;
    if (!mobileConfiguration) return false;
    if (!mobileConfiguration.if) return true;
    return mobileConfiguration.if(rowRecord, rowIndex);
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

    if (isMobileView || !useVirtualScroll) {
      // Plain mode uses one sticky-scroll container, so the header aligns naturally without compensation.
      verticalScrollbarWidth = 0;
      return;
    }

    const viewportElement = shellElement.querySelector('.table-grid-virtual-viewport') as HTMLDivElement | null;
    if (!viewportElement) {
      verticalScrollbarWidth = 0;
      return;
    }

    // Match header width to the scrollable body viewport when the vertical scrollbar consumes space.
    verticalScrollbarWidth = Math.max(0, viewportElement.offsetWidth - viewportElement.clientWidth);
  };

  onMount(() => {
    if (!shellElement) return;

    const handleResize = () => {
      windowWidth = window.innerWidth;
    };
    window.addEventListener('resize', handleResize);

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
      window.removeEventListener('resize', handleResize);
      resizeObserver.disconnect();
    };
  });

  $effect(() => {
    data.length;
    normalizedRowHeight;
    bufferSize;
    isMobileView;

    queueMicrotask(() => {
      syncHeaderScrollbarCompensation();
    });
  });
</script>

<div class="table-grid-shell {css}"
  class:table-grid-shell-mobile={isMobileView}
  bind:this={shellElement}
  style="height: {isMobileView || useVirtualScroll ? height : 'auto'}; max-height: {height}; --table-grid-template-columns: {gridTemplateColumns}; --table-grid-row-height: {normalizedRowHeight}px; --table-grid-scrollbar-width: {verticalScrollbarWidth}px;"
>
  {#if isMobileView}
    <div class="table-grid-mobile-shell"
      class:table-grid-mobile-shell-inner-padding={useInnerMobilePadding}
    >
      {#if data.length === 0}
        <div class="table-grid-empty">{emptyMessage}</div>
      {:else}
        <SvelteVirtualList items={data}
          bufferSize={bufferSize}
          defaultEstimatedItemHeight={estimatedMobileCardHeight}
          debug={debug}
          debugFunction={debugVirtualListInfo}
        >
          {#snippet renderItem(rowRecord, rowIndex)}
            {@const selected = isSelectedRow(rowRecord, rowIndex)}
            <div class="table-grid-mobile-card mb-6 {mobileCardCss || ''}"
                class:table-grid-mobile-card-selected={selected}
                role="button"
                tabindex="0"
                onclick={() => handleRowClick(rowRecord, rowIndex)}
                onkeydown={(eventInfo) => {
                  if (eventInfo.key === 'Enter' || eventInfo.key === ' ') {
                    handleRowClick(rowRecord, rowIndex);
                  }
                }}
              >
                <div class="table-grid-mobile-card-grid">
                  {#each mobileColumns as columnDefinition (columnDefinition.id)}
                    {@const mobileConfiguration = columnDefinition.mobile}
                    {@const defaultCellValue = getCellValue(rowRecord, columnDefinition, rowIndex)}
                    {@const [splitCellFirstLine, splitCellSecondLine] = getSplitCellValue(defaultCellValue, columnDefinition)}
                    {#if mobileConfiguration && shouldRenderMobileColumn(rowRecord, columnDefinition, rowIndex)}
                      {#if mobileConfiguration.labelTop}
                      <div class="table-grid-mobile-item table-grid-mobile-item-vertical {mobileConfiguration.css || 'col-span-full'}">
                        <div class="table-grid-mobile-label-top">{mobileConfiguration.labelTop}</div>
                        <div class="table-grid-mobile-content-wrapper">
                          {#if mobileConfiguration.icon}
                            <i class="icon-{mobileConfiguration.icon} {mobileConfiguration.iconCss || ''}"></i>
                          {/if}
                          {#if mobileConfiguration.labelLeft}
                            <span class="table-grid-mobile-label-left">{mobileConfiguration.labelLeft}</span>
                          {/if}
                          {#if mobileConfiguration.elementLeft}
                            <div class="table-grid-mobile-left">
                              {#if isHtmlContent(mobileConfiguration.elementLeft)}
                                {@html mobileConfiguration.elementLeft}
                              {:else}
                                <Renderer elements={mobileConfiguration.elementLeft}/>
                              {/if}
                            </div>
                          {/if}
                          <div class="table-grid-mobile-content {getAlignClassName(columnDefinition.align)} {columnDefinition.cellCss || ''}">
                            {#if mobileConfiguration.render}
                              {@const renderedContent = mobileConfiguration.render(rowRecord, rowIndex)}
                              {#if isHtmlContent(renderedContent)}
                                {@html renderedContent}
                              {:else}
                                <Renderer elements={renderedContent}/>
                              {/if}
                            {:else if cellRenderer && columnDefinition.useCellRenderer}
                              {@render cellRenderer(rowRecord, columnDefinition, rowIndex)}
                            {:else if splitCellSecondLine}
                              <div class="flex flex-col leading-[1.1]">
                                <div>{splitCellFirstLine}</div>
                                <div>{splitCellSecondLine}</div>
                              </div>
                            {:else}
                              {defaultCellValue}
                            {/if}
                          </div>
                          {#if mobileConfiguration.elementRight}
                            <div class="table-grid-mobile-right">
                              {#if isHtmlContent(mobileConfiguration.elementRight)}
                                {@html mobileConfiguration.elementRight}
                              {:else}
                                <Renderer elements={mobileConfiguration.elementRight}/>
                              {/if}
                            </div>
                          {/if}
                        </div>
                      </div>
                    {:else}
                      <div class="table-grid-mobile-item {mobileConfiguration.css || 'col-span-full'}">
                        {#if mobileConfiguration.icon}
                          <i class="icon-{mobileConfiguration.icon} {mobileConfiguration.iconCss || ''}"></i>
                        {/if}
                        {#if mobileConfiguration.labelLeft}
                          <span class="table-grid-mobile-label-left">{mobileConfiguration.labelLeft}</span>
                        {/if}
                        {#if mobileConfiguration.elementLeft}
                          <div class="table-grid-mobile-left">
                            {#if isHtmlContent(mobileConfiguration.elementLeft)}
                              {@html mobileConfiguration.elementLeft}
                            {:else}
                              <Renderer elements={mobileConfiguration.elementLeft}/>
                            {/if}
                          </div>
                        {/if}
                        <div class="table-grid-mobile-content {getAlignClassName(columnDefinition.align)} {columnDefinition.cellCss || ''}">
                          {#if mobileConfiguration.render}
                            {@const renderedContent = mobileConfiguration.render(rowRecord, rowIndex)}
                            {#if isHtmlContent(renderedContent)}
                              {@html renderedContent}
                            {:else}
                              <Renderer elements={renderedContent}/>
                            {/if}
                          {:else if cellRenderer && columnDefinition.useCellRenderer}
                            {@render cellRenderer(rowRecord, columnDefinition, rowIndex)}
                          {:else if splitCellSecondLine}
                            <div class="flex flex-col leading-[1.1]">
                              <div>{splitCellFirstLine}</div>
                              <div>{splitCellSecondLine}</div>
                            </div>
                          {:else}
                            {defaultCellValue}
                          {/if}
                        </div>
                        {#if mobileConfiguration.elementRight}
                          <div class="table-grid-mobile-right">
                            {#if isHtmlContent(mobileConfiguration.elementRight)}
                              {@html mobileConfiguration.elementRight}
                            {:else}
                              <Renderer elements={mobileConfiguration.elementRight}/>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    {/if}
                    {/if}
                  {/each}
                </div>
            </div>
          {/snippet}
        </SvelteVirtualList>
      {/if}
    </div>
  {:else if !useVirtualScroll}
    <div class="table-grid-plain-scroll">
      <div class="table-grid-header table-grid-header-sticky {headerCss}">
        <div class="table-grid-header-row" role="row">
          {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id)}
            <div class="table-grid-header-cell {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
              role="columnheader"
            >
              {#if headerRenderer}
                {@render headerRenderer(columnDefinition, columnIndex)}
              {:else}
                {columnDefinition.header}
              {/if}
            </div>
          {/each}
        </div>
      </div>

      {#if data.length === 0}
        <div class="table-grid-empty">{emptyMessage}</div>
      {:else}
        {#each data as rowRecord, rowIndex (getRowId ? getRowId(rowRecord, rowIndex) : rowIndex)}
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
                {@const [splitCellFirstLine, splitCellSecondLine] = getSplitCellValue(defaultCellValue, columnDefinition)}
                <div class="table-grid-cell [&:last-child]:border-r-0 {getAlignClassName(columnDefinition.align)} {columnDefinition.cellCss || ''}"
                  role="cell"
                  title={`${defaultCellValue}`}
                >
                  {#if cellRenderer && columnDefinition.useCellRenderer}
                    {@render cellRenderer(rowRecord, columnDefinition, rowIndex)}
                  {:else if splitCellSecondLine}
                    <div class="flex flex-col leading-[1.1]">
                      <div>{splitCellFirstLine}</div>
                      <div>{splitCellSecondLine}</div>
                    </div>
                  {:else}
                    {defaultCellValue}
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {:else}
    <div class="table-grid-scroll-host use-virtual-scroll">
      <div class="table-grid-header {headerCss}">
        <div class="table-grid-header-row" role="row">
          {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id)}
            <div class="table-grid-header-cell {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
              role="columnheader"
            >
              {#if headerRenderer}
                {@render headerRenderer(columnDefinition, columnIndex)}
              {:else}
                {columnDefinition.header}
              {/if}
            </div>
          {/each}
        </div>
      </div>

      <div class="table-grid-body use-virtual-scroll">
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
                  {@const [splitCellFirstLine, splitCellSecondLine] = getSplitCellValue(defaultCellValue, columnDefinition)}
                  <div class="table-grid-cell [&:last-child]:border-r-0 {getAlignClassName(columnDefinition.align)} {columnDefinition.cellCss || ''}"
                    role="cell"
                    title={`${defaultCellValue}`}
                  >
                    {#if cellRenderer && columnDefinition.useCellRenderer}
                      {@render cellRenderer(rowRecord, columnDefinition, rowIndex)}
                    {:else if splitCellSecondLine}
                      <div class="flex flex-col leading-[1.1]">
                        <div>{splitCellFirstLine}</div>
                        <div>{splitCellSecondLine}</div>
                      </div>
                    {:else}
                      {defaultCellValue}
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/snippet}
        </SvelteVirtualList>
      </div>
    </div>
  {/if}
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

  .table-grid-shell-mobile {
    border: none;
    width: calc(100% + 8px);
    margin-left: -4px;
    margin-right: -4px;
  }

  .table-grid-scroll-host {
    display: grid;
    grid-template-rows: auto minmax(0, 1fr);
    height: auto;
    min-height: 0;
    overflow: hidden;
    border-radius: inherit;
  }

  .table-grid-scroll-host.use-virtual-scroll {
    height: 100%;
  }

  .table-grid-plain-scroll {
    max-height: inherit;
    overflow-y: auto;
    overflow-x: hidden;
    /*
    scrollbar-gutter: stable;
    scrollbar-width: auto;
    scrollbar-color: #94a3b8 #f1f5f9;
    */
  }
  
  .table-grid-header {
    border-bottom: 1px solid #e9ecef;
    background: #f8f9fa;
    position: relative;
    z-index: 2;
    padding-right: var(--table-grid-scrollbar-width);
    box-sizing: border-box;
  }

  .table-grid-header-sticky {
    position: sticky;
    top: 0;
    z-index: 4;
    padding-right: 0;
  }

  .table-grid-header-row,
  .table-grid-row {
    display: grid;
    grid-template-columns: var(--table-grid-template-columns);
    width: 100%;
    /* Let fractional columns shrink inside the viewport so cell ellipsis can work. */
    min-width: 0;
  }

  .table-grid-header-cell {
    font-family: bold;
    color: #495057;
    border-right: 1px solid #edf2f7;
    line-height: 1.1;
  }

  .table-grid-header-cell:last-child {
    border-right: none;
  }

  .table-grid-body {
    height: auto;
    min-height: 0;
    position: relative;
    box-sizing: border-box;
  }

  .table-grid-body.use-virtual-scroll {
    height: 100%;
    overflow: hidden;
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

  .table-grid-mobile-shell {
    height: 100%;
    min-height: 0;
  }

  .table-grid-mobile-shell-inner-padding {
    box-sizing: border-box;
  }

  .table-grid-mobile-shell-inner-padding :global(.virtual-list-viewport) {
    padding: 4px;
  }

  .table-grid-mobile-card {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 12px;
    cursor: pointer;
    transition: box-shadow 0.2s ease;
  }

  .table-grid-mobile-card:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }

  .table-grid-mobile-card-selected,
  .table-grid-mobile-card-selected.table-grid-mobile-card:hover {
    background-color: #f6f6ff;
    outline: 2px solid var(--color-11);
    outline-offset: -1px;
  }

  .table-grid-mobile-card:focus-visible {
    outline: 2px solid #3b82f6;
    outline-offset: -2px;
  }

  .table-grid-mobile-card-grid {
    display: grid;
    grid-template-columns: repeat(24, 1fr);
    gap: 4px;
  }

  .table-grid-mobile-item {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .table-grid-mobile-item-vertical {
    flex-direction: column;
    align-items: flex-start;
    row-gap: 0;
  }

  .table-grid-mobile-label-top {
    font-size: 14px;
    color: #64748b;
    line-height: 1;
  }

  .table-grid-mobile-content-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 0;
  }

  .table-grid-mobile-label-left {
    font-size: 14px;
    color: #64748b;
    flex-shrink: 0;
  }

  .table-grid-mobile-left,
  .table-grid-mobile-right {
    flex-shrink: 0;
  }

  .table-grid-mobile-content {
    flex: 1;
    min-width: 0;
    word-break: break-word;
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
    /*
    scrollbar-gutter: stable;
    scrollbar-width: auto;
    scrollbar-color: #94a3b8 #f1f5f9;
    */
  }
  
  .table-grid-plain-scroll::-webkit-scrollbar {
    width: 12px;
  }

  :global(.table-grid-virtual-viewport::-webkit-scrollbar) {
    width: 12px;
  }
  
   /*


  :global(.table-grid-virtual-viewport::-webkit-scrollbar-track) {
    background: #f1f5f9;
  }

  :global(.table-grid-virtual-viewport::-webkit-scrollbar-thumb) {
    background: #94a3b8;
    border-radius: 999px;
    border: 2px solid #f1f5f9;
  }

  :global(.table-grid-virtual-viewport::-webkit-scrollbar-thumb:hover) {
    background: #64748b;
  }


  .table-grid-plain-scroll::-webkit-scrollbar-track {
    background: #f1f5f9;
  }

  .table-grid-plain-scroll::-webkit-scrollbar-thumb {
    background: #94a3b8;
    border-radius: 999px;
    border: 2px solid #f1f5f9;
  }

  .table-grid-plain-scroll::-webkit-scrollbar-thumb:hover {
    background: #64748b;
  }
  */
  
  :global(.table-grid-virtual-container) {
    height: 100% !important;
    min-height: 0 !important;
  }

  :global(.table-grid-virtual-content) {
    min-height: 100%;
  }

</style>
