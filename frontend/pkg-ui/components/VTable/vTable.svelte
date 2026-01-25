<script lang="ts" generics="T">
  import { untrack } from 'svelte';
  import { createVirtualizer } from './index.svelte';
  import type { ITableColumn, CellRendererSnippet } from "./types";
  import type { VirtualItem } from './index.svelte';
  import CellEditable from './CellEditable.svelte';
import { highlString, include } from '$core/lib/helpers';
  import Renderer, { type ElementAST } from '../micro/Renderer.svelte';
  import CellSelector from './CellSelector.svelte';
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';

  interface ICellContent { 
    content: string; 
    contentHTML: string; 
    contentAST: ElementAST | ElementAST[], 
    useSnippet: boolean; 
    css: string
  }

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
    filterText?: string;
    getFilterContent?: (row: T) => string;
    useFilterCache?: boolean;
    mobileCardCss?: string
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
    filterText,
    getFilterContent,
    useFilterCache = false,
    mobileCardCss = ''
  }: VTableProps<T> = $props();

  // State
  let containerRef = $state<HTMLDivElement>();
  let virtualItems = $state<VirtualItem[]>([]);
  let totalSize = $state(0);
  let virtualizerStore: ReturnType<typeof createVirtualizer> | null = null;
  let isInitialized = false;
  let dataVersion = $state(0);
  const filterCache = new WeakMap<T & object, string>();

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

  // Mobile columns sorted by order
  const mobileColumns = $derived.by(() => {
    return processedColumns.flatColumns
      .filter(col => col.mobile)
      .sort((a, b) => (a.mobile?.order || 0) - (b.mobile?.order || 0));
  });

  // Mobile view - only enable if columns have mobile config and screen is small
  const isMobileView = $derived(windowWidth < 580 && mobileColumns.length > 0);

  const filterTextArray = $derived((filterText||"").toLowerCase().split(" ").filter(x => x.length > 1))

  const filteredData = $derived.by(() => {
    console.log("filterText", filterText)

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
          return include(content, filterTextArray)
        })

      console.log("data filtrada::", filtered)
      return filtered
    } else {
      return data
    }
  })

  $effect(() => {
    if(selected || !selected){
      console.log("selected::", selected)
    }
  })

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

    if (column.render) {
      const renderedContent = column.render(record, index)
      if(typeof renderedContent === 'string'){
        rec.contentHTML = renderedContent
      } else if(renderedContent){
        rec.contentAST = renderedContent
      }
    }
    if (column.getValue) {
      rec.content = column.getValue(record, index) as string
    }

    // Check if we should use snippet renderer (takes priority over function renderer)
    if (cellRenderer && column.id) { rec.useSnippet = true; }

    rec.css = typeof column.cellCss === 'string' 
      ? column.cellCss
      : (column.onCellEdit ? "relative" : "px-8 py-4")
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
  function handleRowClick(record: T, index: number) {
    if (onRowClick) {
      onRowClick(record, index);
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
      {#if filteredData.length === 0}
        <div class="mobile-empty-message">
          {emptyMessage}
        </div>
      {:else}
        <SvelteVirtualList items={filteredData}>
          {#snippet renderItem(record, index)}
            <div class="mobile-card mb-6 {mobileCardCss || ''}" 
              role="button" 
              tabindex="0"
              onclick={() => handleRowClick(record, index)}
              onkeydown={(ev) => {
                if (ev.key === 'Enter' || ev.key === ' ') {
                  handleRowClick(record, index);
                }
              }}>
              <div class="mobile-card-grid">
                {#each mobileColumns as column}
                  {@const mobile = column.mobile}
                  {@const shouldRender = !mobile?.if || mobile.if(record, index)}
                  {#if shouldRender}
                    {@const cellData = getCellContent(column, record, index)}
                    {#if mobile?.labelTop}
                    <div class="mobile-card-item mobile-card-item-vertical {mobile?.css || 'col-span-full'}">
                      <div class="mobile-card-label-top">{mobile.labelTop}</div>
                      <div class="mobile-card-content-wrapper">
                        {#if mobile?.icon}
                          <i class="icon-{mobile.icon} {mobile?.iconCss || ''}"></i>
                        {/if}
                        {#if mobile?.labelLeft}
                          <span class="mobile-card-label-left">{mobile.labelLeft}</span>
                        {/if}
                        {#if mobile?.elementLeft}
                          <div class="mobile-card-left">
                            {#if typeof mobile.elementLeft === 'string'}
                              {@html mobile.elementLeft}
                            {:else}
                              <Renderer elements={mobile.elementLeft}/>
                            {/if}
                          </div>
                        {/if}
                        <div class="mobile-card-content">
                          {#if mobile?.render}
                            {@const renderedContent = mobile.render(record, index)}
                            {#if typeof renderedContent === 'string'}
                              {@html renderedContent}
                            {:else}
                              <Renderer elements={renderedContent}/>
                            {/if}
                          {:else if cellData.useSnippet && cellRenderer}
                            {@render cellRenderer(record, column, cellData.content, index)}
                          {:else if cellData.contentAST}
                            <Renderer elements={cellData.contentAST}/>
                          {:else if cellData.contentHTML}
                            {@html cellData.contentHTML}
                          {:else}
                            {cellData.content}
                          {/if}
                        </div>
                        {#if mobile?.elementRight}
                          <div class="mobile-card-right">
                            {#if typeof mobile.elementRight === 'string'}
                              {@html mobile.elementRight}
                            {:else}
                              <Renderer elements={mobile.elementRight}/>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    </div>
                  {:else}
                    <div class="mobile-card-item {mobile?.css || 'col-span-full'}">
                      {#if mobile?.icon}
                        <i class="icon-{mobile.icon} {mobile?.iconCss || ''}"></i>
                      {/if}
                      {#if mobile?.labelLeft}
                        <span class="mobile-card-label-left">{mobile.labelLeft}</span>
                      {/if}
                      {#if mobile?.elementLeft}
                        <div class="mobile-card-left">
                          {#if typeof mobile.elementLeft === 'string'}
                            {@html mobile.elementLeft}
                          {:else}
                            <Renderer elements={mobile.elementLeft}/>
                          {/if}
                        </div>
                      {/if}
                      <div class="mobile-card-content">
                        {#if mobile?.render}
                          {@const renderedContent = mobile.render(record, index)}
                          {#if typeof renderedContent === 'string'}
                            {@html renderedContent}
                          {:else}
                            <Renderer elements={renderedContent}/>
                          {/if}
                        {:else if cellData.useSnippet && cellRenderer}
                          {@render cellRenderer(record, column, cellData.content, index)}
                        {:else if cellData.contentAST}
                          <Renderer elements={cellData.contentAST}/>
                        {:else if cellData.contentHTML}
                          {@html cellData.contentHTML}
                        {:else}
                          {cellData.content}
                        {/if}
                      </div>
                      {#if mobile?.elementRight}
                        <div class="mobile-card-right">
                          {#if typeof mobile.elementRight === 'string'}
                            {@html mobile.elementRight}
                          {:else}
                            <Renderer elements={mobile.elementRight}/>
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
      {#if filteredData.length === 0}
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
          {@const record = filteredData[row.index]}
          
          {#if record}
            {@const selected = isRowSelected(record, row.index)}
            
            <tr class="vtable-row"
            class:vtable-row-even={row.index % 2 === 0}
            class:vtable-row-odd={row.index % 2 !== 0}
            class:vtable-row-selected={selected}
            style="transform: translateY({firstItemStart}px);"
            onclick={() => handleRowClick(record, row.index)}
          >
            {#each processedColumns.flatColumns as column, j (`${j}_${filterText||""}`)}
              {@const cellData = getCellContent(column, record, row.index)}
              <td class="{cellData.css}"
                style={column.cellStyle ? Object.entries(column.cellStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
                onclick={ev => {
                  if(column.onCellEdit){ ev.stopPropagation() }
                }}
              >
                {#if column.onCellEdit}
                  <CellEditable contentClass={column.css}
                    inputClass={column.inputCss}
                    getValue={() => cellData.content} 
                    render={
                      (column.render 
                      ? () => column.render?.(record, row.index) 
                      : undefined) as (value: number | string) => ElementAST[]
                    }
                    onChange={v => { 
                      column.onCellEdit?.(record,v)
                    }}
                  />
                {:else if column.onCellSelect}
                  <CellSelector options={column.cellOptions as any[]} 
                    keyId="" keyName=""
                    onChange={v => { 
                      column.onCellSelect?.(record,v)
                    }}
                  />
                {:else if cellData.useSnippet && cellRenderer}
                  {@render cellRenderer(record, column, cellData.content, row.index)}
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
                    {#if column.buttonEditHandler}
                      <button class="_11 _e" title="edit" onclick={ev => {
                        ev.stopPropagation()
                        column.buttonEditHandler?.(record)
                      }}>
                        <i class="icon-pencil"></i>
                      </button>
                    {/if}
                    {#if column.buttonDeleteHandler}
                      <button class="_11 _d" title="delete" onclick={ev => {
                        ev.stopPropagation()
                        column.buttonDeleteHandler?.(record)
                      }}>
                        <i class="icon-trash"></i>
                      </button>
                    {/if}
                  </div>
                {/if}
              </td>
            {/each}
          </tr>
          
          {#if isFinal}
            <tr style="height: {remainingSize}px; visibility: hidden;">
              <td style="border: none;"></td>
            </tr>
          {/if}
          {/if}
        {/each}
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

  .vtable-container:not(._14) {
    background-color: white;
    border: 1px solid #dee2e6;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 2px;
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

  .vtable-row > td:not(:last-of-type) {
    display: table-cell;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
    border-right: 1px solid #f1f3f5;
    position: relative;
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
    color: #666;
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
    color: #666;
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
  }
</style>
