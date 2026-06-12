<script lang="ts" generics="T">
  import { untrack } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { createTableVirtualizer } from './vtable-virtual.svelte';
  import type { ITableColumn, CellRendererSnippet, IMobileCardsListCell } from "./types";
  import CellInput from '$components/vTable/CellInput.svelte';
  import CellSelect from '$components/vTable/CellSelect.svelte';
  import { highlString, wordInclude } from '$libs/helpers';
  import Renderer, { type ElementAST } from '$components/misc/Renderer.svelte';
  import MobileCardsVirtualList from '$components/vTable/MobileCardsVirtualList.svelte';
  import T from '$components/misc/T.svelte';
  import { Env } from '$core/env';
  import { Agent } from '$components/agent/registry';
  import {
    setVTableAgentContext,
    buildCellID,
    buildRowID,
    parseChildID,
    rowIndexFromRowID,
    type CellAgentMethods,
  } from '$components/vTable/agentContext';

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
    emptyMessage = 'No records found.|No se encontraron registros.',
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
  const filterCache = new WeakMap<T & object, string>();
  // Reactive cache of hydrated rows. SvelteMap reads are tracked per-key, so
  // resolving one row only re-evaluates templates that already read that key —
  // avoiding the full-table re-render storm that bumping a global version caused.
  const resolvedRowByKey = new SvelteMap<object | string, T>();
  // Internal bookkeeping (non-reactive): which keys have been queued/are loading
  // so the kickoff path stays idempotent without driving extra renders.
  const loadingRowKeys = new Set<object | string>();
  const queuedRowKeys = new Set<object | string>();
  // Per-row version counters used to force a specific row to unmount/remount
  // when handlers invoke the `rerender` callback. SvelteMap reactivity is keyed,
  // so bumping one entry only invalidates the matching row's keyed block.
  const rowVersions = new SvelteMap<number, number>();

  function rerenderRow(rowIndex: number) {
    const nextRowVersion = (rowVersions.get(rowIndex) || 0) + 1;
    rowVersions.set(rowIndex, nextRowVersion);
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
        contentCss: column.mobile?.contentCss,
        labelTop: column.mobile?.labelTop,
        labelLeft: column.mobile?.labelLeft,
        labelCss: 'color-label',
        icon: column.mobile?.icon,
        iconCss: column.mobile?.iconCss,
        elementLeft: column.mobile?.elementLeft,
        elementRight: column.mobile?.elementRight,
        mobileRender: column.mobile?.render,
        if: column.mobile?.if,
        onCellClick: column.onCellClick,
        disableCellInteractions: column.disableCellInteractions,
        showEditIcon: column.showEditIcon,
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

  // Resolve row objects lazily so tables can start from partial rows and hydrate
  // details on demand. The template-side read of resolvedRowByKey is what makes
  // this reactive — when the async hydration completes and we call .set(...) on
  // the SvelteMap, only the row(s) that consulted that key re-evaluate.
  function getResolvedRow(record: T, index: number): T | null {
    if (!getRowObject) {
      return record;
    }

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
          });
      });
    }

    return null;
  }

  // Virtualizer instance: shared between attach lifecycle and template reads.
  // Heights/offsets/ranges live inside it; the host only feeds it row count and
  // hands each rendered <tr> to `observeRow` for ResizeObserver-based measurement.
  const virtualizer = createTableVirtualizer({
    getScrollElement: () => containerRef ?? null,
    estimateSize: () => estimateSize,
    overscan: () => overscan,
  });

  // Bind the virtualizer to the scroll container once it's mounted (desktop view).
  $effect(() => {
    if (isMobileView) { return; }
    if (!containerRef) { return; }
    const ok = virtualizer.attach();
    if (!ok) { return; }
    return () => virtualizer.detach();
  });

  // Keep the virtualizer's row count in sync with the filtered dataset. Stored
  // heights are dropped on each change because the row at index `i` may now be
  // a different record.
  $effect(() => {
    const nextCount = filteredData.length;
    untrack(() => virtualizer.setCount(nextCount));
  });

  // Indices of rows inside the current visible window (with overscan). Driven by
  // the virtualizer's reactive range; scrolling updates this without bumping any
  // global table version.
  const visibleRowIndices = $derived.by(() => {
    const { start, end } = virtualizer.range;
    const indices: number[] = [];
    for (let i = start; i < end; i++) { indices.push(i); }
    return indices;
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

    /*
    rec.css = typeof column.cellCss === 'string'
      ? column.cellCss
      : (column.onCellEdit ? "relative" : "px-8 py-4")
    */
    rec.css = rec.css || ""
    // Append runtime classes returned by setCellCss for this row
    const dynamicCellCss = column.setCellCss?.(record)
    if (dynamicCellCss) {
      rec.css += ` ${dynamicCellCss}`
    }
    if(column.css){ rec.css += " " + column.css }

    return rec
  }

  // Resolve header string for use with <T> component
  function resolveHeader(column: ITableColumn<T>): string {
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

  const componentID = Env.getComponentID();

  // Cells (CellInput / CellSelect) hand their methods here keyed by cellID.
  // The table is the only agent handle; methods route by id.
  const cellRegistry = new Map<number, CellAgentMethods>();

  setVTableAgentContext({
    tableID: componentID,
    registerCell: (cellID, methods) => {
      cellRegistry.set(cellID, methods);
      return () => {
        if (cellRegistry.get(cellID) === methods) { cellRegistry.delete(cellID); }
      };
    },
  });

  const hasInteractiveCell = $derived(
    columns.some((column) => column.onCellEdit || column.onCellSelect || column.onCellClick),
  );
  // When the table flips to mobile view, MobileCardsVirtualList registers its
  // own CardList agent handle, so VTable steps back to avoid a ghost Table
  // handle that owns no visible rows.
  const shouldRegisterTable = $derived(!isMobileView && (Boolean(onRowClick) || hasInteractiveCell));

  // Cells with a column-level onCellClick aren't backed by a child component,
  // so they don't go through cellRegistry. We resolve the column from the
  // cellID's column slot directly when the agent clicks.
  const dispatchCellClick = (cellIDArg: number | string) => {
    const cellID = parseChildID(cellIDArg);
    if (!Number.isFinite(cellID) || cellID <= 0 || cellID % 100 === 0) { return; }
    const rowIndex = Math.floor((cellID - 1) / 100) - 1;
    const columnIndex = (cellID - 1) % 100;
    if (rowIndex < 0 || rowIndex >= filteredData.length) { return; }
    const column = processedColumns.flatColumns[columnIndex];
    if (!column?.onCellClick) { return; }
    const record = filteredData[rowIndex];
    if (!record) { return; }
    const resolved = getResolvedRow(record, rowIndex) || record;
    if (column.disableCellInteractions?.(resolved, rowIndex)) { return; }
    column.onCellClick(resolved, rowIndex, () => rerenderRow(rowIndex));
  };

  // ids that are exact multiples of 100 are row ids; everything else is a cell
  // id, and remaining args go to the cell's own select() (option ids).
  const dispatchRowSelect = (rowID: number) => {
    if (!onRowClick) { return; }
    const rowIndex = rowIndexFromRowID(rowID);
    if (rowIndex < 0 || rowIndex >= filteredData.length) { return; }
    const record = filteredData[rowIndex];
    if (!record) { return; }
    const resolved = getResolvedRow(record, rowIndex) || record;
    handleRowClick(resolved, rowIndex, () => rerenderRow(rowIndex));
  };

  $effect(() => {
    if (!shouldRegisterTable) { return; }
    return Agent.register({
      id: componentID,
      type: "Table",
      label: "",
      // The agent calls Table methods with the composite id stripped down by
      // the backend bridge (http.go::resolveTarget): "setValue('38:101', v)"
      // arrives here as setValueChild(101, v). select() still receives the
      // composite first arg because the table needs to disambiguate row vs
      // cell from the id alone.
      select: (...ids) => {
        if (ids.length === 0) { return; }
        const first = parseChildID(ids[0]);
        if (Number.isFinite(first) && first % 100 === 0) {
          for (const rid of ids) { dispatchRowSelect(parseChildID(rid)); }
          return;
        }
        cellRegistry.get(first)?.select?.(...ids.slice(1));
      },
      setValueChild: (cellID, value) => {
        cellRegistry.get(parseChildID(cellID))?.setValue?.(value);
      },
      searchChild: (cellID, text) => {
        cellRegistry.get(parseChildID(cellID))?.search?.(String(text ?? ''));
      },
      getOptionsChild: (cellID, max) => {
        return cellRegistry.get(parseChildID(cellID))?.getOptions?.(Number(max ?? 50)) ?? [];
      },
      clickChild: (cellID) => {
        dispatchCellClick(cellID);
      },
    });
  });
</script>

<div bind:this={containerRef}
  data-id={shouldRegisterTable ? `Table:${componentID}` : undefined}
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
              <T text={resolveHeader(column)} />
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
                <T text={resolveHeader(column)} />
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
              <T text={emptyMessage} />
            </div>
          </td>
        </tr>
      {:else}
        {@const range = virtualizer.range}
        {@const topSpacerHeight = Math.max(2, range.offsetAtStart)}
        {@const bottomSpacerHeight = Math.max(2, virtualizer.totalSize - range.offsetAtEnd)}

        <!-- Top spacer pushes the first rendered row to its real Y offset and
             preserves the 2px breathing room previously held by .vtable-edge-spacer. -->
        <tr class="vtable-virtual-spacer" aria-hidden="true" style="height: {topSpacerHeight}px;">
          <td colspan={processedColumns.flatColumns.length}></td>
        </tr>

        {#each visibleRowIndices as rowIndex (`${rowIndex}-${rowVersions.get(rowIndex) || 0}`)}
          {@const record = filteredData[rowIndex]}
          {@const resolvedRecord = record ? getResolvedRow(record, rowIndex) : null}

          {#if record}
            {@const selected = resolvedRecord ? isRowSelected(resolvedRecord, rowIndex) : false}

          <tr class="vtable-row"
            use:virtualizer.observeRow={rowIndex}
            data-id={onRowClick ? `TableRow:${componentID}:${buildRowID(rowIndex)}` : undefined}
            data-selected={selected ? "true" : undefined}
            class:vtable-row-even={rowIndex % 2 === 0}
            class:vtable-row-odd={rowIndex % 2 !== 0}
            class:vtable-row-selected={selected}
            onclick={() => resolvedRecord && handleRowClick(resolvedRecord, rowIndex, () => rerenderRow(rowIndex))}
          >
            {#if !resolvedRecord}
              <td colspan={processedColumns.flatColumns.length}
                class="vtable-loading"
                style="height: {estimateSize}px;"
              >
                Loading...
              </td>
            {:else}
              {#each processedColumns.flatColumns as column, j (j)}
                {@const cellData = getCellContent(column, resolvedRecord, rowIndex)}
                {@const css = cellCss ? cellCss + " " + (cellData.css||"") : cellData.css || ""}
                {@const cssFinal = [css, !/px-|pr-|pl-/.test(css) && "px-6", column.align === 'right' && 'text-right'].filter(Boolean).join(" ")}
                {@const cellInteractionsDisabled = column.disableCellInteractions?.(resolvedRecord, rowIndex)}
                {@const isAgentClickCell = !!column.onCellClick && !cellInteractionsDisabled && !column.onCellEdit && !column.onCellSelect}

                <td class="{cssFinal}"
                	class:clickable-cell={!!column.onCellClick && !cellInteractionsDisabled}
                  data-id={isAgentClickCell ? `${componentID}:${buildCellID(rowIndex, j)}` : undefined}
                  data-cell-type={isAgentClickCell ? 'CellClick' : undefined}
                  style={column.cellStyle ? Object.entries(column.cellStyle).map(([k, v]) => `${k}: ${v}`).join('; ') : ''}
                  onclick={ev => {
                    if(column.onCellEdit){ ev.stopPropagation() }
                    if(column.onCellClick){ 
                    	ev.stopPropagation() 
                      column.onCellClick(resolvedRecord, rowIndex, () => rerenderRow(rowIndex)) 
                    }
                  }}
                >
                  {#if column.showEditIcon && !column.disableCellInteractions?.(resolvedRecord, rowIndex)}
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
                  {#if column.onCellEdit && !column.disableCellInteractions?.(resolvedRecord, rowIndex)}
                  	{@const paddingCss = /px-|pr-|pl-/.test(column.inputCss||"") ? "" : "px-6"}
                   
                    <CellInput
                    	contentClass={cssFinal + (column.align === 'right' ? " justify-end" : "")}
                      inputClass={paddingCss +" "+ (column.inputCss||"") + (column.align === 'right' ? " text-right" : "")}
                      type={column.cellInputType || cellInputType}
                      cellID={buildCellID(rowIndex, j)}
                      getValue={() => cellData.content}
                      render={
                        (column.render
                        ? () => column.render?.(resolvedRecord, rowIndex)
                        : undefined) as (value: number | string) => ElementAST[]
                      }
                      onChange={(value: string | number) => {
                        column.onCellEdit?.(resolvedRecord, value, () => rerenderRow(rowIndex))
                      }}
                    />
                  {:else if column.onCellSelect}
                    <CellSelect
                      id={`${String(column.id || column.field || j)}_${rowIndex}`}
                      saveOn={resolvedRecord}
                      save={column.field as keyof T}
                      options={(column.cellOptions || []) as any[]}
                      keyId={(column.cellOptionsKeyId || 'ID') as never}
                      keyName={(column.cellOptionsKeyName || 'Name') as never}
                      cellID={buildCellID(rowIndex, j)}
                      contentClass={column.css}
                      onChange={(value: string | number) => {
                        column.onCellSelect?.(resolvedRecord, value, () => rerenderRow(rowIndex))
                      }}
                    />
                  {:else if cellData.useSnippet && cellRenderer}
                    {@render cellRenderer(resolvedRecord, column, cellData.content, rowIndex, false)}
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
          {/if}
        {/each}

        <!-- Bottom spacer extends the table to totalSize so scrollHeight reflects
             the entire dataset, not just the rendered window. -->
        <tr class="vtable-virtual-spacer" aria-hidden="true" style="height: {bottomSpacerHeight}px;">
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

  /* Spacer rows used by the virtualizer to extend tbody to the dataset's full
     height while only rendering the visible window. They must not paint a
     background or border or they'd show through under the rendered rows. */
  .vtable-virtual-spacer td {
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

  .clickable-cell {
    cursor: pointer;
    position: relative;
  }

  .clickable-cell:hover {
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
  ._11._d {
    color: #f04949;
    background-color: #ffe7e7;
    box-shadow: rgb(181 50 50 / 70%) 0px 1px 1px 0px;
  }
  ._11._d:hover {
    background-color: #f04949;
    color: #ffffff;
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
