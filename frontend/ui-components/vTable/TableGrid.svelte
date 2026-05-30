<script lang="ts" generics="TRecord">
  import { onMount } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import Renderer, { type ElementAST } from '$components/misc/Renderer.svelte';
  import CellInput from '$components/vTable/CellInput.svelte';
  import { createFixedTableVirtualizer } from './vtable-virtual-fixed.svelte';
  import { splitTwoStrings } from '$libs/helpers';
  import MobileCardsVirtualList from '$components/vTable/MobileCardsVirtualList.svelte';
  import { Env } from '$core/env';
  import { tr } from '$core/store.svelte';
  import { Agent } from '$components/agent/registry';
  import {
    setVTableAgentContext,
    buildCellID,
    buildRowID,
    parseChildID,
    rowIndexFromRowID,
    type CellAgentMethods,
  } from '$components/vTable/agentContext';
  import type {
    IMobileCardsListCell,
    ITableColumn,
    TableGridCellAlign,
    TableGridCellRendererSnippet,
    TableGridHeaderRendererSnippet,
    TableGridRowRendererSnippet,
  } from './types';

  interface TableGridProps<TRecord> {
    columns: ITableColumn<TRecord>[];
    data: TRecord[];
    height?: string;
    rowHeight?: number;
    getRowHeight?: (rowRecord: TRecord, rowIndex: number) => number | undefined;
    bufferSize?: number;
    mobileBreakpointPx?: number;
    useInnerMobilePadding?: boolean;
    css?: string;
    headerCss?: string;
    rowCss?: string;
    cellCss?: string;
    mobileCardCss?: string;
    emptyMessage?: string;
    debug?: boolean;
    onRowClick?: (rowRecord: TRecord, rowIndex: number, rerender: () => void) => void;
    selectedRowId?: string | number;
    selectedRecord?: TRecord;
    getRowId?: (rowRecord: TRecord, rowIndex: number) => string | number;
    cellRenderer?: TableGridCellRendererSnippet<TRecord>;
    headerRenderer?: TableGridHeaderRendererSnippet<TRecord>;
    // When `useRowRenderer(record, idx)` returns true, the row's per-column cells are replaced
    // by `rowRenderer` rendered in a single full-row container (e.g. for section headers).
    useRowRenderer?: (record: TRecord, rowIndex: number) => boolean;
    rowRenderer?: TableGridRowRendererSnippet<TRecord>;
    cellInputType?: 'number';
  }

  interface TableGridPrefixContent {
    prefixHTML?: string;
    prefixAST?: ElementAST | ElementAST[];
  }

  let {
    columns,
    data,
    height = '460px',
    rowHeight = 36,
    getRowHeight,
    bufferSize = 12,
    mobileBreakpointPx = 580,
    useInnerMobilePadding = false,
    css = '',
    headerCss = '',
    rowCss = '',
    cellCss = '',
    mobileCardCss = '',
    emptyMessage = 'No records found.|No se encontraron registros.',
    debug = false,
    onRowClick,
    selectedRowId,
    selectedRecord,
    getRowId,
    cellRenderer,
    headerRenderer,
    useRowRenderer,
    rowRenderer,
    cellInputType,
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
    visibleColumns
      .map((columnDefinition) => columnDefinition.width || 'minmax(80px, 1fr)')
      .join(' '),
  );
  const normalizedRowHeight = $derived(Math.max(24, Math.round(rowHeight)));
  const resolveRowShellStyle = (rowRecord: TRecord, rowIndex: number): string => {
    const customHeight = getRowHeight?.(rowRecord, rowIndex);
    if (typeof customHeight === 'number' && customHeight > 0) {
      const px = Math.max(24, Math.round(customHeight));
      return `height: ${px}px; --table-grid-row-height: ${px}px;`;
    }
    return 'height: var(--table-grid-row-height);';
  };
  const estimatedMobileCardHeight = $derived(Math.max(128, normalizedRowHeight * 3 + 24));
  const useVirtualScroll = $derived(data.length >= 30);
  let windowWidth = $state(typeof window !== 'undefined' ? window.innerWidth : 1024);
  let shellElement = $state<HTMLDivElement | undefined>(undefined);
  let verticalScrollbarWidth = $state(0);

  // Resolve the selected record identity only when a resolver exists.
  const selectedRecordResolvedId = $derived.by(() => {
    if (!selectedRecord || !getRowId) return undefined;
    return getRowId(selectedRecord, -1);
  });

  const getCellValue = (
    rowRecord: TRecord,
    columnDefinition: ITableColumn<TRecord>,
    rowIndex: number,
  ): string | number => {
    if (!columnDefinition.getValue) return '';
    return columnDefinition.getValue(rowRecord, rowIndex);
  };

  const getSplitCellValue = (
    cellValue: string | number,
    columnDefinition: ITableColumn<TRecord>,
  ): [string, string] => {
    if (typeof cellValue !== 'string' || !columnDefinition.splitString) {
      return [String(cellValue ?? ''), ''];
    }

    // Split long labels into two balanced lines so adjacent columns remain visible.
    return splitTwoStrings(cellValue, columnDefinition.splitString);
  };

  const getAlignClassName = (align: TableGridCellAlign | undefined) => {
    if (align === 'center') return 'text-center';
    if (align === 'right') return 'text-right';
    return 'text-left';
  };

  // Reuse the same shared card renderer as VTable/CardsList while keeping grid-specific align classes.
  const mobileCardCells = $derived.by((): IMobileCardsListCell<TRecord, ITableColumn<TRecord>>[] => {
    return mobileColumns.map((columnDefinition) => ({
      ...columnDefinition,
      source: columnDefinition,
      itemCss: columnDefinition.mobile?.css || 'col-span-full',
      contentCss: columnDefinition.mobile?.contentCss || getAlignClassName(columnDefinition.align),
      labelTop: columnDefinition.mobile?.labelTop,
      labelLeft: columnDefinition.mobile?.labelLeft,
      icon: columnDefinition.mobile?.icon,
      iconCss: columnDefinition.mobile?.iconCss,
      elementLeft: columnDefinition.mobile?.elementLeft,
      elementRight: columnDefinition.mobile?.elementRight,
      mobileRender: columnDefinition.mobile?.render,
      if: columnDefinition.mobile?.if,
      onCellClick: columnDefinition.onCellClick,
      disableCellInteractions: columnDefinition.disableCellInteractions,
      showEditIcon: columnDefinition.showEditIcon,
      useRenderer: Boolean(cellRenderer && columnDefinition.useCellRenderer),
    }));
  });
  const isMobileView = $derived(windowWidth < mobileBreakpointPx && mobileCardCells.length > 0);

  const isHtmlContent = (contentValue: unknown): contentValue is string => {
    return typeof contentValue === 'string';
  };

  const getPrefixContent = (
    rowRecord: TRecord,
    columnDefinition: ITableColumn<TRecord>,
    rowIndex: number,
  ): TableGridPrefixContent => {
    const resolvedPrefix: TableGridPrefixContent = {};
    const renderedPrefix = columnDefinition.renderPrefix?.(rowRecord, rowIndex);

    // Match VTable's contract: strings are trusted HTML, AST values go through Renderer.
    if (typeof renderedPrefix === 'string') {
      resolvedPrefix.prefixHTML = renderedPrefix;
    } else if (renderedPrefix) {
      resolvedPrefix.prefixAST = renderedPrefix;
    }

    return resolvedPrefix;
  };

  const getHeaderContent = (columnDefinition: ITableColumn<TRecord>): string => {
    return tr(typeof columnDefinition.header === 'function'
      ? columnDefinition.header()
      : columnDefinition.header);
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

  const isSelectedRowByValue = (
    rowRecord: TRecord,
    selectedValue: TRecord | string | number,
  ): boolean => {
    // Reuse the same selection rules from desktop without mutating parent-bound state.
    if (selectedRecord && rowRecord === selectedRecord) {
      return true;
    }

    const resolvedIndex = data.indexOf(rowRecord);
    const rowIndex = resolvedIndex >= 0 ? resolvedIndex : -1;

    if (typeof selectedValue === 'string' || typeof selectedValue === 'number') {
      if (!getRowId) {
        return false;
      }
      return getRowId(rowRecord, rowIndex) === selectedValue;
    }

    if (selectedValue === rowRecord) {
      return true;
    }

    if (!getRowId) {
      return false;
    }

    const currentRowId = getRowId(rowRecord, rowIndex);
    return getRowId(selectedValue, -1) === currentRowId;
  };

  // Per-row version counters bumped when cell handlers invoke their `rerender` callback;
  // included in the cells' each-key so only the affected row remounts.
  const rowVersions = new SvelteMap<number, number>();

  const rerenderRow = (rowIndex: number) => {
    rowVersions.set(rowIndex, (rowVersions.get(rowIndex) || 0) + 1);
  };

  const handleRowClick = (rowRecord: TRecord, rowIndex: number) => {
    if (debug) {
      console.debug('[TableGrid] row click', { rowIndex, rowRecord });
    }
    onRowClick?.(rowRecord, rowIndex, () => rerenderRow(rowIndex));
  };

  // Custom virtualizer that drives the OUTER shell's scroll (vs. SvelteVirtualList's
  // internal viewport). The shell owns overflow: auto, so the scrollbar appears on
  // the rounded outer container instead of a nested div. Fixed-height variant —
  // rows are pinned via `.table-grid-row { height/max-height: var(--table-grid-row-height); overflow: hidden }`,
  // so no per-row measurement is needed.
  const virtualizer = createFixedTableVirtualizer({
    getScrollElement: () => shellElement ?? null,
    rowHeight: () => normalizedRowHeight,
    overscan: () => bufferSize,
  });

  // Attach the virtualizer once the shell mounts and we're in the virtual branch.
  // Re-runs if useVirtualScroll or isMobileView flips (e.g. row count crosses 30).
  $effect(() => {
    if (isMobileView || !useVirtualScroll) { return; }
    if (!shellElement) { return; }
    const ok = virtualizer.attach();
    if (!ok) { return; }
    return () => virtualizer.detach();
  });

  // Keep the virtualizer's row count in sync with the dataset.
  $effect(() => {
    if (isMobileView || !useVirtualScroll) { return; }
    virtualizer.setCount(data.length);
  });

  // Indices of rows inside the current visible window (with overscan).
  const visibleRowIndices = $derived.by(() => {
    const { start, end } = virtualizer.range;
    const indices: number[] = [];
    for (let i = start; i < end; i++) { indices.push(i); }
    return indices;
  });

  onMount(() => {
    const handleResize = () => {
      windowWidth = window.innerWidth;
    };
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
    };
  });

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

  // Register when there is any row interaction OR any column with cell editing
  // / selection. Without one of those there's nothing for the agent to drive.
  const hasInteractiveCell = $derived(
    columns.some((column) => column.onCellEdit || column.onCellSelect),
  );
  // Mobile branch hands the agent over to MobileCardsVirtualList (CardList type),
  // so skip registering a Table here to avoid a ghost handle with no rows.
  const shouldRegisterTable = $derived(!isMobileView && (Boolean(onRowClick) || hasInteractiveCell));

  // Row select drives the existing onRowClick. Row IDs are buildRowID(rowIndex)
  // = (rowIndex+1)*100 by convention; cells live in the same row above the row
  // id and below the next one.
  const dispatchRowSelect = (rowID: number) => {
    if (!onRowClick) { return; }
    const rowIndex = rowIndexFromRowID(rowID);
    if (rowIndex < 0 || rowIndex >= data.length) { return; }
    handleRowClick(data[rowIndex], rowIndex);
  };

  $effect(() => {
    if (!shouldRegisterTable) { return; }
    return Agent.register({
      id: componentID,
      type: "Table",
      label: "",
      // ids that are exact multiples of 100 are row ids; everything else is a
      // cell id, in which case the remaining args are forwarded to the cell's
      // own select() (e.g. CellSelect option ids). The first arg may arrive as
      // composite "<tableID>:<childID>" — parseChildID strips the prefix.
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
    });
  });
</script>

<div data-id={shouldRegisterTable ? `Table:${componentID}` : undefined}
  class="table-grid-shell {css}"
  class:table-grid-shell-mobile={isMobileView}
  bind:this={shellElement}
  style="height: {isMobileView || useVirtualScroll ? height : 'auto'}; max-height: {height}; --table-grid-template-columns: {gridTemplateColumns}; --table-grid-row-height: {normalizedRowHeight}px; --table-grid-scrollbar-width: {verticalScrollbarWidth}px;"
>
  {#if isMobileView}
    <div class="table-grid-mobile-shell"
      class:table-grid-mobile-shell-inner-padding={useInnerMobilePadding}
    >
      <MobileCardsVirtualList
        data={data}
        cells={mobileCardCells}
        variant="compact"
        cardCss={`mb-6 ${mobileCardCss}`.trim()}
        showSelectedCard={true}
        estimateSize={estimatedMobileCardHeight}
        overscan={bufferSize}
        emptyMessage={emptyMessage}
        onRowClick={onRowClick ? handleRowClick : undefined}
        selected={selectedRecord || selectedRowId}
        isSelected={isSelectedRowByValue}
        getRecordIndex={(rowRecord, fallbackIndex) => {
          const resolvedIndex = data.indexOf(rowRecord);
          return resolvedIndex >= 0 ? resolvedIndex : fallbackIndex;
        }}
        debugName="TableGrid"
        gridCellRenderer={cellRenderer}
      />
    </div>
  {:else if !useVirtualScroll}
    <div class="table-grid-plain-scroll">
      <div class="table-grid-header table-grid-header-sticky {headerCss}" role="row">
        {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id || columnIndex)}
          {@const headerPaddingCss = /px-|pr-|pl-/.test(columnDefinition.headerCss || "") ? "" : "px-6"}
          <div class="table-grid-header-cell {headerPaddingCss} {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
            role="columnheader"
          >
            {#if headerRenderer}
              {@render headerRenderer(columnDefinition, columnIndex)}
            {:else}
              {getHeaderContent(columnDefinition)}
            {/if}
          </div>
        {/each}
      </div>

      {#if data.length === 0}
        <div class="table-grid-empty">{tr(emptyMessage)}</div>
      {:else}
        <div class="table-grid-edge-spacer" aria-hidden="true"></div>
        {#each data as rowRecord, rowIndex (getRowId ? getRowId(rowRecord, rowIndex) : rowIndex)}
          {@const selected = isSelectedRow(rowRecord, rowIndex)}
          <div class="table-grid-row {rowCss}"
            class:table-grid-row-even={rowIndex % 2 === 0}
            class:table-grid-row-odd={rowIndex % 2 !== 0}
            class:table-grid-row-selected={selected}
            data-id={onRowClick ? `TableRow:${componentID}:${buildRowID(rowIndex)}` : undefined}
            data-selected={selected ? "true" : undefined}
            style={resolveRowShellStyle(rowRecord, rowIndex)}
            role="row"
            tabindex="0"
            onclick={() => handleRowClick(rowRecord, rowIndex)}
            onkeydown={(eventInfo) => {
              if (eventInfo.key === 'Enter' || eventInfo.key === ' ') {
                handleRowClick(rowRecord, rowIndex);
              }
            }}
          >
            {#if useRowRenderer?.(rowRecord, rowIndex) && rowRenderer}
              <div class="table-grid-cell tg-row-custom" style="grid-column: 1 / -1;" role="cell">
                {@render rowRenderer(rowRecord, rowIndex)}
              </div>
            {:else}
            {#each visibleColumns as colDef, columnIndex (`${colDef.id || columnIndex}_${rowVersions.get(rowIndex) || 0}`)}
              {@const defaultCellValue = getCellValue(rowRecord, colDef, rowIndex)}
              {@const [splitCellFirstLine, splitCellSecondLine] = getSplitCellValue(defaultCellValue, colDef)}
              {@const prefixContent = getPrefixContent(rowRecord, colDef, rowIndex)}
              {@const combinedCellCss = `${cellCss || ""} ${colDef.css || ""} ${colDef.setCellCss?.(rowRecord) || ""} ${colDef.css || ""}`}
              {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? "" : "px-6"}
              {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || "") ? "" : "px-6"}
              {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || "") ? "" : "px-6"}

              <div class="table-grid-cell [&:last-child]:border-r-0 {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                  class:tg-cell-hover-effect={colDef.showHoverEffect}
                role="cell"
                title={`${defaultCellValue}`}
              >
                <div class="tg-cell-layout">
                  {#if prefixContent.prefixAST}
                    <span class="table-grid-cell-prefix">
                      <Renderer elements={prefixContent.prefixAST}/>
                    </span>
                  {:else if prefixContent.prefixHTML}
                    <span class="table-grid-cell-prefix">
                      {@html prefixContent.prefixHTML}
                    </span>
                  {/if}
                  {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(rowRecord, rowIndex)}
                    <CellInput contentClass={`${contentPaddingCss} ${colDef.css || ""}${colDef.align === 'right' ? ' justify-end' : ''}`}
                      inputClass={`${inputPaddingCss} ${colDef.inputCss || ""}${colDef.align === 'right' ? ' text-right' : ''}`}
                      type={colDef.cellInputType || cellInputType}
                      cellID={buildCellID(rowIndex, columnIndex)}
                      getValue={() => String(defaultCellValue)}
                      render={colDef.render ? () => colDef.render!(rowRecord, rowIndex) : undefined}
                      onBeforeCellChange={colDef.onBeforeCellChange ? (value) => colDef.onBeforeCellChange!(rowRecord, value) : undefined}
                      onChange={(value) => colDef.onCellEdit?.(rowRecord, value, () => rerenderRow(rowIndex))}
                    />
                  {:else if cellRenderer && colDef.useCellRenderer}
                    {@render cellRenderer(rowRecord, colDef, rowIndex)}
                  {:else if colDef.buttonEditHandler || colDef.buttonDeleteHandler}
                    <div class="flex gap-4 items-center justify-center w-full">
                      {#if colDef.buttonEditHandler && (!colDef.buttonEditIf || colDef.buttonEditIf(rowRecord))}
                        <button class="_11 _e" title="edit" onclick={(ev) => {
                          ev.stopPropagation();
                          colDef.buttonEditHandler?.(rowRecord);
                        }}>
                          <i class="icon-pencil"></i>
                        </button>
                      {/if}
                      {#if colDef.buttonDeleteHandler && (!colDef.buttonDeleteIf || colDef.buttonDeleteIf(rowRecord))}
                        <button class="_11 _d" title="delete" onclick={(ev) => {
                          ev.stopPropagation();
                          colDef.buttonDeleteHandler?.(rowRecord);
                        }}>
                          <i class="icon-trash"></i>
                        </button>
                      {/if}
                    </div>
                  {:else if colDef.render}
                    {@const renderedContent = colDef.render(rowRecord, rowIndex)}
                    <div class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>
                      {#if isHtmlContent(renderedContent)}
                        {@html renderedContent}
                      {:else}
                        <Renderer elements={renderedContent}/>
                      {/if}
                    </div>
                  {:else if splitCellSecondLine}
                    <div class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>
                      <div>{splitCellFirstLine}</div>
                      <div>{splitCellSecondLine}</div>
                    </div>
                  {:else}
                    <span class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>{defaultCellValue}</span>
                  {/if}
                </div>
              </div>
            {/each}
            {/if}
          </div>
        {/each}
        <div class="table-grid-edge-spacer" aria-hidden="true"></div>
      {/if}
    </div>
  {:else}
    {@const range = virtualizer.range}
    {@const topSpacerHeight = range.offsetAtStart}
    {@const bottomSpacerHeight = Math.max(0, virtualizer.totalSize - range.offsetAtEnd)}
    <div class="table-grid-scroll-host use-virtual-scroll">
      <div class="table-grid-header table-grid-header-sticky {headerCss}" role="row">
        {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id || columnIndex)}
          {@const headerPaddingCss = /px-|pr-|pl-/.test(columnDefinition.headerCss || "") ? "" : "px-6"}
          <div class="table-grid-header-cell {headerPaddingCss} {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
            role="columnheader"
          >
            {#if headerRenderer}
              {@render headerRenderer(columnDefinition, columnIndex)}
            {:else}
              {getHeaderContent(columnDefinition)}
            {/if}
          </div>
        {/each}
      </div>

      <div class="table-grid-body">
        <div class="table-grid-virtual-spacer" aria-hidden="true" style="height: {topSpacerHeight}px;"></div>

        {#each visibleRowIndices as rowIndex (`${getRowId ? getRowId(data[rowIndex], rowIndex) : rowIndex}_${rowVersions.get(rowIndex) || 0}`)}
          {@const rowRecord = data[rowIndex]}
          {#if rowRecord}
            {@const selected = isSelectedRow(rowRecord, rowIndex)}
            <div class="table-grid-row {rowCss}"
              use:virtualizer.observeRow={rowIndex}
              class:table-grid-row-even={rowIndex % 2 === 0}
              class:table-grid-row-odd={rowIndex % 2 !== 0}
              class:table-grid-row-selected={selected}
              data-id={onRowClick ? `TableRow:${componentID}:${buildRowID(rowIndex)}` : undefined}
              data-selected={selected ? "true" : undefined}
              style={resolveRowShellStyle(rowRecord, rowIndex)}
              role="row"
              tabindex="0"
              onclick={() => handleRowClick(rowRecord, rowIndex)}
              onkeydown={(eventInfo) => {
                if (eventInfo.key === 'Enter' || eventInfo.key === ' ') {
                  handleRowClick(rowRecord, rowIndex);
                }
              }}
            >
              {#if useRowRenderer?.(rowRecord, rowIndex) && rowRenderer}
                <div class="table-grid-cell tg-row-custom" style="grid-column: 1 / -1;" role="cell">
                  {@render rowRenderer(rowRecord, rowIndex)}
                </div>
              {:else}
              {#each visibleColumns as colDef, columnIndex (`${colDef.id || columnIndex}_${rowVersions.get(rowIndex) || 0}`)}
                {@const defaultCellValue = getCellValue(rowRecord, colDef, rowIndex)}
                {@const [splitCellFirstLine, splitCellSecondLine] = getSplitCellValue(defaultCellValue, colDef)}
                {@const prefixContent = getPrefixContent(rowRecord, colDef, rowIndex)}
                {@const combinedCellCss = `${cellCss || ""} ${colDef.css || ""} ${colDef.setCellCss?.(rowRecord) || ""} ${colDef.css || ""}`}
                {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? "" : "px-6"}
                {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || "") ? "" : "px-6"}
                {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || "") ? "" : "px-6"}
                <div class="table-grid-cell [&:last-child]:border-r-0 {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                  class:tg-cell-hover-effect={colDef.showHoverEffect}
                  role="cell"
                  title={`${defaultCellValue}`}
                >
                  <div class="tg-cell-layout">
                    {#if prefixContent.prefixAST}
                      <span class="table-grid-cell-prefix">
                        <Renderer elements={prefixContent.prefixAST}/>
                      </span>
                    {:else if prefixContent.prefixHTML}
                      <span class="table-grid-cell-prefix">
                        {@html prefixContent.prefixHTML}
                      </span>
                    {/if}
                    {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(rowRecord, rowIndex)}
                      <CellInput contentClass={`${contentPaddingCss} ${colDef.css || ""}${colDef.align === 'right' ? ' justify-end' : ''}`}
                        inputClass={`${inputPaddingCss} ${colDef.inputCss || ""}${colDef.align === 'right' ? ' text-right' : ''}`}
                        type={colDef.cellInputType || cellInputType}
                        cellID={buildCellID(rowIndex, columnIndex)}
                        getValue={() => String(defaultCellValue)}
                        render={colDef.render ? () => colDef.render!(rowRecord, rowIndex) : undefined}
                        onBeforeCellChange={colDef.onBeforeCellChange ? (value) => colDef.onBeforeCellChange!(rowRecord, value) : undefined}
                        onChange={(value) => colDef.onCellEdit?.(rowRecord, value, () => rerenderRow(rowIndex))}
                      />
                    {:else if cellRenderer && colDef.useCellRenderer}
                      {@render cellRenderer(rowRecord, colDef, rowIndex)}
                    {:else if colDef.buttonEditHandler || colDef.buttonDeleteHandler}
                      <div class="flex gap-4 items-center justify-center w-full">
                        {#if colDef.buttonEditHandler && (!colDef.buttonEditIf || colDef.buttonEditIf(rowRecord))}
                          <button class="_11 _e" title="edit" onclick={(ev) => {
                            ev.stopPropagation();
                            colDef.buttonEditHandler?.(rowRecord);
                          }}>
                            <i class="icon-pencil"></i>
                          </button>
                        {/if}
                        {#if colDef.buttonDeleteHandler && (!colDef.buttonDeleteIf || colDef.buttonDeleteIf(rowRecord))}
                          <button class="_11 _d" title="delete" onclick={(ev) => {
                            ev.stopPropagation();
                            colDef.buttonDeleteHandler?.(rowRecord);
                          }}>
                            <i class="icon-trash"></i>
                          </button>
                        {/if}
                      </div>
                    {:else if colDef.render}
                      {@const renderedContent = colDef.render(rowRecord, rowIndex)}
                      <div class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>
                        {#if isHtmlContent(renderedContent)}
                          {@html renderedContent}
                        {:else}
                          <Renderer elements={renderedContent}/>
                        {/if}
                      </div>
                    {:else if splitCellSecondLine}
                      <div class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>
                        <div>{splitCellFirstLine}</div>
                        <div>{splitCellSecondLine}</div>
                      </div>
                    {:else}
                      <span class="tg-cell-content" class:tg-cell-line-clamp={colDef.useLineClamp}>{defaultCellValue}</span>
                      {/if}
                  </div>
                </div>
              {/each}
              {/if}
            </div>
          {/if}
        {/each}

        <div class="table-grid-virtual-spacer" aria-hidden="true" style="height: {bottomSpacerHeight}px;"></div>
      </div>
    </div>
  {/if}
</div>

<style>
  .table-grid-shell {
    border: 1px solid #dee2e6;
    border-radius: 8px;
    background-color: #ffffff;
    overflow: auto;
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
    /* Passive wrapper — the shell owns scrolling now, so this just needs to
       grow with its content (sticky header + spacers + visible rows). */
    min-height: 0;
    border-radius: inherit;
  }

  .table-grid-plain-scroll {
    max-height: inherit;
    /*
    scrollbar-gutter: stable;
    scrollbar-width: auto;
    scrollbar-color: #94a3b8 #f1f5f9;
    */
  }

  .table-grid-edge-spacer {
    height: 2px;
  }
  
  .table-grid-header,
  .table-grid-row {
    display: grid;
    grid-template-columns: var(--table-grid-template-columns);
    width: 100%;
    /* Grow to fit fixed column tracks so the scroll container can scroll horizontally
       when the viewport is narrower than the sum of fixed widths. */
    min-width: min-content;
  }

  .table-grid-header {
    border-bottom: 1px solid #e9ecef;
    background: #f8f9fa;
    position: relative;
    z-index: 2;
    padding-left: 2px;
    padding-right: calc(2px + var(--table-grid-scrollbar-width));
    box-sizing: border-box;
  }

  .table-grid-header-sticky {
    position: sticky;
    top: 0;
    z-index: 4;
    padding-left: 0;
    padding-right: 0;
  }

  .table-grid-header-cell {
    font-family: bold;
    color: #495057;
    border-right: 1px solid #edf2f7;
    line-height: 1.1;
    display: grid;
    align-content: center;
    min-width: 0;
  }

  .table-grid-header-cell:last-child {
    border-right: none;
  }

  .table-grid-body {
    min-height: 0;
    position: relative;
    box-sizing: border-box;
  }

  /* Spacers above/below the rendered window; inline `height` extends the body
     to the dataset's full virtual height so the shell's scrollbar reflects it. */
  .table-grid-virtual-spacer {
    width: 100%;
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

  /* Container for caller-provided row renderer; spans every grid column. */
  .tg-row-custom {
    display: flex;
    align-items: center;
    width: 100%;
    border-right: 0;
  }

  /* Edit/delete action-button styles mirror VTable so action columns look consistent. */
  ._11 {
    border-radius: 50%;
    width: 26px;
    height: 26px;
    font-size: 13px;
    color: #5243c2;
    box-shadow: rgb(67 61 110 / 62%) 0px 1px 2px 0px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 0;
    cursor: pointer;
    flex-shrink: 0;
  }
  ._11._e {
    color: #5243c2;
    background-color: #e7e4ff;
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

  .table-grid-row {
    height: var(--table-grid-row-height);
    max-height: var(--table-grid-row-height);
    overflow: hidden;
    cursor: pointer;
    border-bottom: 1px solid #edf2f7;
    transition: background-color 0.15s ease;
    position: relative;
    box-sizing: border-box;
  }

  .table-grid-row:focus-visible {
    outline: none;
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
    position: relative;
  }

  .table-grid-cell-prefix {
    display: inline-flex;
    align-items: center;
    flex: 0 0 auto;
  }

  .tg-cell-layout {
    display: flex;
    align-items: center;
    gap: 4px;
    width: 100%;
    min-width: 0;
  }

  .tg-cell-content {
    flex: 1 1 auto;
    min-width: 0;
  }

  .text-right .tg-cell-content {
    text-align: right;
  }

  .text-center .tg-cell-content {
    text-align: center;
  }

  /* Clamp only the text wrapper; the cell shell keeps sizing, borders, and alignment stable. */
  .tg-cell-line-clamp {
    white-space: normal;
    overflow: hidden;
    text-overflow: clip;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    line-height: 1.2;
  }

  .tg-cell-hover-effect:hover {
    outline: 1px solid rgba(0, 0, 0, 0.596);
    outline-offset: -1px;
  }
  .tg-cell-hover-effect:focus-within {
    outline: none;
    box-shadow: inset 0 0 0 1px #b17bff, inset 0 0 0 2px #dbc1ff;
    background-color: #f9f4ff;
  }
  
  .table-grid-shell::-webkit-scrollbar {
    width: 12px;
    height: 12px;
  }

</style>
