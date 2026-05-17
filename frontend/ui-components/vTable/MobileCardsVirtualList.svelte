<script lang="ts" generics="TRecord, TCell extends IMobileCardsListCell<TRecord>">
  import Renderer, { type ElementAST } from '$components/misc/Renderer.svelte';
  import CellInput from '$components/vTable/CellInput.svelte';
  import CellSelect from '$components/vTable/CellSelect.svelte';
  import {
    buildCellID,
    buildRowID,
    parseChildID,
    rowIndexFromRowID,
    setVTableAgentContext,
    type CellAgentMethods,
  } from '$components/vTable/agentContext';
  import { Agent } from '$core/agent/registry';
  import { Env } from '$core/env';
  import { highlString, splitTwoStrings } from '$libs/helpers';
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import { SvelteMap } from 'svelte/reactivity';
  import type {
    CardRendererSnippet,
    CellRendererSnippet,
    ICardButtonDeleteHandler,
    IMobileCardsListCell,
    MobileCardsListRendererSnippet,
    MobileCardsListVariant,
    TableGridCellRendererSnippet,
  } from './types';

  interface IMobileCardCellContent {
    content: string;
    contentHTML: string;
    contentAST: ElementAST | ElementAST[];
    prefixHTML: string;
    prefixAST: ElementAST | ElementAST[];
    css: string;
  }

  interface MobileCardsVirtualListProps<TRecord, TCell extends IMobileCardsListCell<TRecord>> {
    data: TRecord[];
    cells: TCell[];
    variant?: MobileCardsListVariant;
    cardCss?: string;
    showSelectedCard?: boolean;
    viewportClass?: string;
    itemsClass?: string;
    estimateSize?: number;
    overscan?: number;
    nonVirtual?: boolean;
    emptyMessage?: string;
    loadingMessage?: string;
    filterText?: string;
    highlightPlainText?: boolean;
    onRowClick?: (row: TRecord, index: number, rerender: () => void) => void;
    selected?: TRecord | string | number;
    isSelected?: (row: TRecord, selected: TRecord | string | number) => boolean;
    getRecordIndex?: (record: TRecord, fallbackIndex: number) => number;
    resolveRecord?: (record: TRecord, index: number) => TRecord | null;
    buttonDeleteHandler?: ICardButtonDeleteHandler<TRecord>;
    buttonDeleteIf?: (row: TRecord, index: number) => boolean;
    debugName?: string;
    cardCellRenderer?: MobileCardsListRendererSnippet<TRecord, TCell>;
    legacyCardCellRenderer?: CardRendererSnippet<TRecord>;
    tableCellRenderer?: CellRendererSnippet<TRecord>;
    gridCellRenderer?: TableGridCellRendererSnippet<TRecord>;
  }

  let {
    data,
    cells,
    variant = 'compact',
    cardCss = '',
    showSelectedCard = false,
    viewportClass = '',
    itemsClass = '',
    estimateSize = 180,
    overscan = 6,
    nonVirtual = false,
    emptyMessage = 'No se encontraron registros.',
    loadingMessage = 'Loading...',
    filterText = '',
    highlightPlainText = false,
    onRowClick,
    selected,
    isSelected,
    getRecordIndex,
    resolveRecord,
    buttonDeleteHandler,
    buttonDeleteIf,
    debugName = 'MobileCardsVirtualList',
    cardCellRenderer,
    legacyCardCellRenderer,
    tableCellRenderer,
    gridCellRenderer,
  }: MobileCardsVirtualListProps<TRecord, TCell> = $props();

  // Keep the virtualizer hooks stable so parent containers can keep their own sizing rules.
  const virtualViewportClass = $derived(`virtual-list-viewport ${viewportClass}`.trim());
  const virtualItemsClass = $derived(`virtual-list-items ${itemsClass}`.trim());
  const visibleCells = $derived.by(() => cells.filter((cell) => !cell.hidden));
  const filterTextArray = $derived((filterText || '').toLowerCase().split(' ').filter((value) => value.length > 1));

  function getRecordListIndex(record: TRecord, fallbackIndex: number): number {
    return getRecordIndex ? getRecordIndex(record, fallbackIndex) : fallbackIndex;
  }

  function getResolvedRecord(record: TRecord, index: number): TRecord | null {
    if (!resolveRecord) {
      return record;
    }
    return resolveRecord(record, index);
  }

  function getCellContent(cell: TCell, record: TRecord, index: number): IMobileCardCellContent {
    const resolvedContent = {} as IMobileCardCellContent;

    if (cell.renderPrefix) {
      const renderedPrefix = cell.renderPrefix(record, index);
      if (typeof renderedPrefix === 'string') {
        resolvedContent.prefixHTML = renderedPrefix;
      } else if (renderedPrefix) {
        resolvedContent.prefixAST = renderedPrefix;
      }
    }

    if (cell.render) {
      const renderedContent = cell.render(record, index);
      if (typeof renderedContent === 'string') {
        resolvedContent.contentHTML = renderedContent;
      } else if (renderedContent) {
        resolvedContent.contentAST = renderedContent;
      }
    }

    if (cell.getValue) {
      resolvedContent.content = String(cell.getValue(record, index) ?? '');
    } else if (cell.field) {
      resolvedContent.content = String((record as Record<string, unknown>)?.[cell.field] ?? '');
    } else {
      resolvedContent.content = '';
    }

    resolvedContent.css = typeof cell.cellCss === 'string'
      ? cell.cellCss
      : (isInteractiveCell(cell, record, index) ? 'mobile-cards-value-host-interactive' : '');

    const dynamicCellCss = cell.setCellCss?.(record);
    if (dynamicCellCss) {
      resolvedContent.css += ` ${dynamicCellCss}`;
    }
    if (cell.css) {
      resolvedContent.css += ` ${cell.css}`;
    }

    return resolvedContent;
  }

  function getLabelContent(cell: TCell): string {
    if (typeof cell.label === 'function') {
      return cell.label();
    }
    return cell.label || '';
  }

  function isRowSelected(record: TRecord): boolean {
    if (!selected || !isSelected) return false;
    return isSelected(record, selected);
  }

  function handleRowClick(record: TRecord, recordIndex: number, sourceIndex: number) {
    onRowClick?.(record, recordIndex, () => rerenderRow(sourceIndex));
  }

  function buildSelectorId(cell: TCell, rowIndex: number, cellIndex: number): string {
    return `${String(cell.id || cell.field || cellIndex)}_${rowIndex}`;
  }

  // Per-row version counters bumped when handlers invoke their `rerender` callback;
  // read inside the renderItem snippet so only the affected card re-keys its cells.
  const rowVersions = new SvelteMap<number, number>();

  function rerenderRow(rowIndex: number) {
    rowVersions.set(rowIndex, (rowVersions.get(rowIndex) || 0) + 1);
  }

  // Keep the same disabled-state contract as desktop tables so mobile cards do not open detail layers accidentally.
  function isCellInteractionDisabled(cell: TCell, record: TRecord, index: number): boolean {
    return Boolean(cell.disableCellInteractions?.(record, index));
  }

  // A mobile value host is interactive when any cell-level handler is active for the current record.
  function isInteractiveCell(cell: TCell, record: TRecord, index: number): boolean {
    if (isCellInteractionDisabled(cell, record, index)) {
      return false;
    }
    return Boolean(cell.onCellEdit || cell.onCellSelect || cell.onCellClick);
  }

  // Route tap/keyboard activation to the same callback contract used by desktop table cells.
  function handleCellClick(event: MouseEvent, cell: TCell, record: TRecord, recordIndex: number, sourceIndex: number) {
    if (!cell.onCellClick || isCellInteractionDisabled(cell, record, recordIndex)) {
      return;
    }
    event.stopPropagation();
    logInteraction('onCellClick', {
      rowIndex: recordIndex,
      cellId: cell.id,
      field: cell.field,
    });
    cell.onCellClick(record, recordIndex, () => rerenderRow(sourceIndex));
  }

  function logInteraction(eventName: string, payload: Record<string, unknown>) {
    console.debug(`[${debugName}] ${eventName}`, payload);
  }

  function getSplitCellValue(content: string, splitString?: number): [string, string] {
    if (!splitString || !content) {
      return [content, ''];
    }
    return splitTwoStrings(content, splitString);
  }

  const componentID = Env.getComponentID();

  // Cells (CellInput / CellSelect) hand their methods here keyed by cellID.
  // The CardList is the only agent handle; methods route by id.
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
    cells.some((cell) => cell.onCellEdit || cell.onCellSelect || cell.onCellClick),
  );
  const shouldRegisterCardList = $derived(Boolean(onRowClick) || hasInteractiveCell);

  // Cells with a cell-level onCellClick aren't backed by a child component,
  // so they don't go through cellRegistry. We resolve the cell from the
  // cellID's column slot directly when the agent clicks.
  const dispatchCellClick = (cellIDArg: number | string) => {
    const cellID = parseChildID(cellIDArg);
    if (!Number.isFinite(cellID) || cellID <= 0 || cellID % 100 === 0) { return; }
    const sourceIndex = Math.floor((cellID - 1) / 100) - 1;
    const cellIndex = (cellID - 1) % 100;
    if (sourceIndex < 0 || sourceIndex >= data.length) { return; }
    const cell = visibleCells[cellIndex];
    if (!cell?.onCellClick) { return; }
    const sourceRecord = data[sourceIndex];
    if (!sourceRecord) { return; }
    const recordIndex = getRecordListIndex(sourceRecord, sourceIndex);
    const resolved = getResolvedRecord(sourceRecord, recordIndex) || sourceRecord;
    if (cell.disableCellInteractions?.(resolved, recordIndex)) { return; }
    cell.onCellClick(resolved, recordIndex, () => rerenderRow(sourceIndex));
  };

  // ids that are exact multiples of 100 are row ids; everything else is a cell
  // id, and remaining args go to the cell's own select() (option ids).
  const dispatchRowSelect = (rowID: number) => {
    if (!onRowClick) { return; }
    const sourceIndex = rowIndexFromRowID(rowID);
    if (sourceIndex < 0 || sourceIndex >= data.length) { return; }
    const sourceRecord = data[sourceIndex];
    if (!sourceRecord) { return; }
    const recordIndex = getRecordListIndex(sourceRecord, sourceIndex);
    const resolved = getResolvedRecord(sourceRecord, recordIndex) || sourceRecord;
    onRowClick(resolved, recordIndex, () => rerenderRow(sourceIndex));
  };

  $effect(() => {
    if (!shouldRegisterCardList) { return; }
    return Agent.register({
      id: componentID,
      type: "CardList",
      label: "",
      // The agent calls CardList methods with the composite id stripped down
      // by the backend bridge (http.go::resolveTarget): "setValue('38:101', v)"
      // arrives here as setValueChild(101, v). select() still receives the
      // composite first arg because we need to disambiguate row vs cell from
      // the id alone.
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

<div class="card-list-root"
  data-id={shouldRegisterCardList ? `CardList:${componentID}` : undefined}
>
{#if data.length === 0}
  <div class="mobile-cards-empty-message">
    {emptyMessage}
  </div>
{:else}
  {#snippet cardItem(sourceRecord: TRecord, sourceIndex: number)}
    {@const recordIndex = getRecordListIndex(sourceRecord, sourceIndex)}
    {@const resolvedRecord = getResolvedRecord(sourceRecord, recordIndex)}
    {@const selectedCard = resolvedRecord ? isRowSelected(resolvedRecord) : false}
    <div
        data-id={onRowClick ? `TableRow:${componentID}:${buildRowID(sourceIndex)}` : undefined}
        data-selected={selectedCard ? "true" : undefined}
        class="mobile-cards-card mobile-cards-card-{variant} {cardCss}"
        class:mobile-cards-card-selected={showSelectedCard && selectedCard}
        role="button"
        tabindex="0"
        onclick={() => resolvedRecord && handleRowClick(resolvedRecord, recordIndex, sourceIndex)}
        onkeydown={(event) => {
          if (event.target !== event.currentTarget) {
            return;
          }
          if (resolvedRecord && (event.key === 'Enter' || event.key === ' ')) {
            event.preventDefault();
            handleRowClick(resolvedRecord, recordIndex, sourceIndex);
          }
        }}
      >
        {#if !resolvedRecord}
          <div class="mobile-cards-loading-message" style="height: {estimateSize}px;">
            {loadingMessage}
          </div>
        {:else}
          {#if buttonDeleteHandler && (!buttonDeleteIf || buttonDeleteIf(resolvedRecord, recordIndex))}
            <button
              type="button"
              class="mobile-cards-delete-button"
              aria-label="eliminar"
              onclick={(event) => {
                event.stopPropagation();
                logInteraction('buttonDeleteHandler', { rowIndex: recordIndex, record: resolvedRecord });
                buttonDeleteHandler(resolvedRecord, recordIndex);
              }}
            >
              <i class="icon-trash"></i>
            </button>
          {/if}

          {@const rowVersion = rowVersions.get(sourceIndex) || 0}
          <div class="mobile-cards-grid mobile-cards-grid-{variant}">
            {#each visibleCells as cell, cellIndex (`${String(cell.id || cell.field || cellIndex)}_${filterText || ''}_${rowVersion}`)}
              {@const shouldRender = !cell.if || cell.if(resolvedRecord, recordIndex)}
              {#if shouldRender}
                {@const cellData = getCellContent(cell, resolvedRecord, recordIndex)}
                {@const cellInteractionsDisabled = isCellInteractionDisabled(cell, resolvedRecord, recordIndex)}
                {@const isAgentClickCell = !!cell.onCellClick && !cellInteractionsDisabled && !cell.onCellEdit && !cell.onCellSelect}
                {@const cellAgentDataID = isAgentClickCell ? `${componentID}:${buildCellID(sourceIndex, cellIndex)}` : undefined}
                {#if variant === 'cards'}
                  <div class="mobile-cards-item mobile-cards-item-cards {cell.itemCss || 'col-span-full'}">
                    <div class="mobile-cards-label {cell.labelCss || ''}">
                      {getLabelContent(cell)}
                    </div>
                    <div class="mobile-cards-content-row {cell.contentCss || ''}">
                      {#if cellData.prefixAST}
                        <span class="mobile-cards-prefix">
                          <Renderer elements={cellData.prefixAST} />
                        </span>
                      {:else if cellData.prefixHTML}
                        <span class="mobile-cards-prefix">
                          {@html cellData.prefixHTML}
                        </span>
                      {/if}

                      <div class="mobile-cards-value-host {cellData.css}"
                        class:mobile-cards-value-host-interactive={isInteractiveCell(cell, resolvedRecord, recordIndex)}
                        data-id={cellAgentDataID}
                        data-cell-type={isAgentClickCell ? 'CellClick' : undefined}
                        onpointerup={(event) => handleCellClick(event, cell, resolvedRecord, recordIndex, sourceIndex)}
                      >
                        {#if cell.onCellEdit}
                          <div class="mobile-cards-editable-border"><div></div></div>
                          <CellInput
                            contentClass={cell.contentCss || cell.css}
                            inputClass={cell.inputCss}
                            type={cell.type || 'text'}
                            cellID={buildCellID(sourceIndex, cellIndex)}
                            getValue={() => {
                              // Always use raw values inside edit mode so formatting stays display-only.
                              return cell.getValue
                                ? cell.getValue(resolvedRecord, recordIndex)
                                : cellData.content;
                            }}
                            render={
                              (cell.render
                                ? () => cell.render?.(resolvedRecord, recordIndex)
                                : undefined) as (value: number | string) => string | ElementAST | ElementAST[]
                            }
                            onChange={(value) => {
                              logInteraction('onCellEdit', {
                                rowIndex: recordIndex,
                                cellId: cell.id,
                                field: cell.field,
                                value,
                              });
                              cell.onCellEdit?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                            }}
                          />
                        {:else if cell.onCellSelect}
                          <div class="mobile-cards-editable-border"><div></div></div>
                          <CellSelect
                            id={buildSelectorId(cell, recordIndex, cellIndex)}
                            saveOn={resolvedRecord}
                            save={cell.field as keyof TRecord}
                            options={cell.cellOptions as any[]}
                            keyId={(cell.cellOptionsKeyId || 'ID') as never}
                            keyName={(cell.cellOptionsKeyName || 'Name') as never}
                            cellID={buildCellID(sourceIndex, cellIndex)}
                            contentClass={cell.contentCss}
                            onChange={(value) => {
                              logInteraction('onCellSelect', {
                                rowIndex: recordIndex,
                                cellId: cell.id,
                                field: cell.field,
                                value,
                                optionsLength: cell.cellOptions?.length || 0,
                              });
                              cell.onCellSelect?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                            }}
                          />
                        {:else if cell.useRenderer && cardCellRenderer}
                          {@render cardCellRenderer(resolvedRecord, cell, cellData.content, recordIndex)}
                        {:else if cell.useRenderer && legacyCardCellRenderer}
                          {@render legacyCardCellRenderer(resolvedRecord, cell.source as never, cellData.content, recordIndex)}
                        {:else if cellData.contentAST}
                          <Renderer elements={cellData.contentAST} />
                        {:else if cellData.contentHTML}
                          {@html cellData.contentHTML}
                        {:else}
                          {#if highlightPlainText && filterText}
                            {#each highlString(cellData.content, filterTextArray) as part}
                              <span class:mobile-cards-highlight={part.highl}>{part.text}</span>
                            {/each}
                          {:else}
                            {cellData.content}
                          {/if}
                        {/if}
                      </div>
                    </div>
                  </div>
                {:else if cell.labelTop}
                  <div class="mobile-cards-item mobile-cards-item-compact mobile-cards-item-vertical {cell.itemCss || 'col-span-full'}">
                    <div class="mobile-cards-label-top {cell.labelCss || ''}">{cell.labelTop}</div>
                    <div class="mobile-cards-content-wrapper">
                      {#if cell.icon}
                        <i class="icon-{cell.icon} {cell.iconCss || ''}"></i>
                      {/if}
                      {#if cell.labelLeft}
                        <span class="mobile-cards-label-left {cell.labelCss || ''}">{cell.labelLeft}</span>
                      {/if}
                      {#if cell.elementLeft}
                        <div class="mobile-cards-side">
                          {#if typeof cell.elementLeft === 'string'}
                            {@html cell.elementLeft}
                          {:else}
                            <Renderer elements={cell.elementLeft} />
                          {/if}
                        </div>
                      {/if}
                      {#if cellData.prefixAST}
                        <span class="mobile-cards-prefix">
                          <Renderer elements={cellData.prefixAST} />
                        </span>
                      {:else if cellData.prefixHTML}
                        <span class="mobile-cards-prefix">
                          {@html cellData.prefixHTML}
                        </span>
                      {/if}
                      <div class="mobile-cards-compact-content {cell.contentCss || ''} {cellData.css}"
                        class:mobile-cards-value-host-interactive={isInteractiveCell(cell, resolvedRecord, recordIndex)}
                        data-id={cellAgentDataID}
                        data-cell-type={isAgentClickCell ? 'CellClick' : undefined}
                        onpointerup={(event) => handleCellClick(event, cell, resolvedRecord, recordIndex, sourceIndex)}
                      >
                        {#if cell.onCellEdit}
                          <div class="mobile-cards-editable-border"><div></div></div>
                          <CellInput
                            contentClass={cell.contentCss || cell.css}
                            inputClass={cell.inputCss}
                            type={cell.type || 'text'}
                            cellID={buildCellID(sourceIndex, cellIndex)}
                            getValue={() => {
                              return cell.getValue
                                ? cell.getValue(resolvedRecord, recordIndex)
                                : cellData.content;
                            }}
                            render={
                              (cell.render
                                ? () => cell.render?.(resolvedRecord, recordIndex)
                                : undefined) as (value: number | string) => string | ElementAST | ElementAST[]
                            }
                            onChange={(value) => {
                              logInteraction('onCellEdit', {
                                rowIndex: recordIndex,
                                cellId: cell.id,
                                field: cell.field,
                                value,
                              });
                              cell.onCellEdit?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                            }}
                          />
                        {:else if cell.onCellSelect}
                          <div class="mobile-cards-editable-border"><div></div></div>
                          <CellSelect
                            id={buildSelectorId(cell, recordIndex, cellIndex)}
                            saveOn={resolvedRecord}
                            save={cell.field as keyof TRecord}
                            options={cell.cellOptions as any[]}
                            keyId={(cell.cellOptionsKeyId || 'ID') as never}
                            keyName={(cell.cellOptionsKeyName || 'Name') as never}
                            cellID={buildCellID(sourceIndex, cellIndex)}
                            contentClass={cell.contentCss}
                            onChange={(value) => {
                              logInteraction('onCellSelect', {
                                rowIndex: recordIndex,
                                cellId: cell.id,
                                field: cell.field,
                                value,
                                optionsLength: cell.cellOptions?.length || 0,
                              });
                              cell.onCellSelect?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                            }}
                          />
                        {:else if cell.mobileRender}
                          {@const renderedContent = cell.mobileRender(resolvedRecord, recordIndex)}
                          {#if typeof renderedContent === 'string'}
                            {@html renderedContent}
                          {:else if typeof renderedContent === 'number'}
                            {renderedContent}
                          {:else}
                            <Renderer elements={renderedContent} />
                          {/if}
                        {:else if cell.useRenderer && tableCellRenderer}
                          {@render tableCellRenderer(resolvedRecord, cell.source as never, cellData.content, recordIndex, true)}
                        {:else if cell.useRenderer && gridCellRenderer}
                          {@render gridCellRenderer(resolvedRecord, cell.source as never, recordIndex)}
                        {:else if cellData.contentAST}
                          <Renderer elements={cellData.contentAST} />
                        {:else if cellData.contentHTML}
                          {@html cellData.contentHTML}
                        {:else}
                          {@const [firstLine, secondLine] = getSplitCellValue(cellData.content, cell.splitString)}
                          {#if secondLine}
                            <div class="flex flex-col leading-[1.1]">
                              <div>{firstLine}</div>
                              <div>{secondLine}</div>
                            </div>
                          {:else}
                            {cellData.content}
                          {/if}
                        {/if}
                      </div>
                      {#if cell.elementRight}
                        <div class="mobile-cards-side">
                          {#if typeof cell.elementRight === 'string'}
                            {@html cell.elementRight}
                          {:else}
                            <Renderer elements={cell.elementRight} />
                          {/if}
                        </div>
                      {/if}
                    </div>
                  </div>
                {:else}
                  <div class="mobile-cards-item mobile-cards-item-compact {cell.itemCss || 'col-span-full'}">
                    {#if cell.icon}
                      <i class="icon-{cell.icon} {cell.iconCss || ''}"></i>
                    {/if}
                    {#if cell.labelLeft}
                      <span class="mobile-cards-label-left {cell.labelCss || ''}">{cell.labelLeft}</span>
                    {/if}
                    {#if cell.elementLeft}
                      <div class="mobile-cards-side">
                        {#if typeof cell.elementLeft === 'string'}
                          {@html cell.elementLeft}
                        {:else}
                          <Renderer elements={cell.elementLeft} />
                        {/if}
                      </div>
                    {/if}
                    {#if cellData.prefixAST}
                      <span class="mobile-cards-prefix">
                        <Renderer elements={cellData.prefixAST} />
                      </span>
                    {:else if cellData.prefixHTML}
                      <span class="mobile-cards-prefix">
                        {@html cellData.prefixHTML}
                      </span>
                    {/if}
                    <div class="mobile-cards-compact-content {cell.contentCss || ''} {cellData.css}"
                      class:mobile-cards-value-host-interactive={isInteractiveCell(cell, resolvedRecord, recordIndex)}
                      onpointerup={(event) => handleCellClick(event, cell, resolvedRecord, recordIndex, sourceIndex)}
                    >
                      {#if cell.onCellEdit}
                        <div class="mobile-cards-editable-border"><div></div></div>
                        <CellInput
                          contentClass={cell.contentCss || cell.css}
                          inputClass={cell.inputCss}
                          type={cell.type || 'text'}
                          cellID={buildCellID(sourceIndex, cellIndex)}
                          getValue={() => {
                            return cell.getValue
                              ? cell.getValue(resolvedRecord, recordIndex)
                              : cellData.content;
                          }}
                          render={
                            (cell.render
                              ? () => cell.render?.(resolvedRecord, recordIndex)
                              : undefined) as (value: number | string) => string | ElementAST | ElementAST[]
                          }
                          onChange={(value) => {
                            logInteraction('onCellEdit', {
                              rowIndex: recordIndex,
                              cellId: cell.id,
                              field: cell.field,
                              value,
                            });
                            cell.onCellEdit?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                          }}
                        />
                      {:else if cell.onCellSelect}
                        <div class="mobile-cards-editable-border"><div></div></div>
                        <CellSelect
                          id={buildSelectorId(cell, recordIndex, cellIndex)}
                          saveOn={resolvedRecord}
                          save={cell.field as keyof TRecord}
                          options={cell.cellOptions as any[]}
                          keyId={(cell.cellOptionsKeyId || 'ID') as never}
                          keyName={(cell.cellOptionsKeyName || 'Name') as never}
                          cellID={buildCellID(sourceIndex, cellIndex)}
                          contentClass={cell.contentCss}
                          onChange={(value) => {
                            logInteraction('onCellSelect', {
                              rowIndex: recordIndex,
                              cellId: cell.id,
                              field: cell.field,
                              value,
                              optionsLength: cell.cellOptions?.length || 0,
                            });
                            cell.onCellSelect?.(resolvedRecord, value, () => rerenderRow(sourceIndex));
                          }}
                        />
                      {:else if cell.mobileRender}
                        {@const renderedContent = cell.mobileRender(resolvedRecord, recordIndex)}
                        {#if typeof renderedContent === 'string'}
                          {@html renderedContent}
                        {:else if typeof renderedContent === 'number'}
                          {renderedContent}
                        {:else}
                          <Renderer elements={renderedContent} />
                        {/if}
                      {:else if cell.useRenderer && tableCellRenderer}
                        {@render tableCellRenderer(resolvedRecord, cell.source as never, cellData.content, recordIndex, true)}
                      {:else if cell.useRenderer && gridCellRenderer}
                        {@render gridCellRenderer(resolvedRecord, cell.source as never, recordIndex)}
                      {:else if cellData.contentAST}
                        <Renderer elements={cellData.contentAST} />
                      {:else if cellData.contentHTML}
                        {@html cellData.contentHTML}
                      {:else}
                        {@const [firstLine, secondLine] = getSplitCellValue(cellData.content, cell.splitString)}
                        {#if secondLine}
                          <div class="flex flex-col leading-[1.1]">
                            <div>{firstLine}</div>
                            <div>{secondLine}</div>
                          </div>
                        {:else}
                          {cellData.content}
                        {/if}
                      {/if}
                    </div>
                    {#if cell.elementRight}
                      <div class="mobile-cards-side">
                        {#if typeof cell.elementRight === 'string'}
                          {@html cell.elementRight}
                        {:else}
                          <Renderer elements={cell.elementRight} />
                        {/if}
                      </div>
                    {/if}
                  </div>
                {/if}
              {/if}
            {/each}
          </div>
        {/if}
      </div>
  {/snippet}

  {#if nonVirtual}
    <div class={virtualItemsClass}>
      {#each data as sourceRecord, sourceIndex (sourceIndex)}
        {@render cardItem(sourceRecord, sourceIndex)}
      {/each}
    </div>
  {:else}
    <SvelteVirtualList
      viewportClass={virtualViewportClass}
      itemsClass={virtualItemsClass}
      items={data}
      defaultEstimatedItemHeight={estimateSize}
      bufferSize={overscan}
    >
      {#snippet renderItem(sourceRecord, sourceIndex)}
        {@render cardItem(sourceRecord, sourceIndex)}
      {/snippet}
    </SvelteVirtualList>
  {/if}
{/if}
</div>

<style>
  .card-list-root {
    display: contents;
  }

  .mobile-cards-empty-message {
    color: #6c757d;
    text-align: center;
    padding: 32px 16px;
  }

  .mobile-cards-loading-message {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #6c757d;
    font-size: 0.875rem;
  }

  .mobile-cards-card {
    position: relative;
    cursor: pointer;
  }

  .mobile-cards-card-compact {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 12px;
    transition: box-shadow 0.2s ease;
  }

  .mobile-cards-card-compact:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }

  .mobile-cards-card-cards {
    border-radius: 10px;
    background-color: #f7f7fa;
    padding: 12px;
    transition: outline-color 0.15s ease, box-shadow 0.15s ease;
    box-shadow: rgb(69 68 93 / 22%) 0px 1px 3px;
    outline: 1px solid transparent;
  }

  .mobile-cards-card-cards:hover {
    outline-color: #cfd4e6;
  }

  .mobile-cards-card-selected {
    background-color: #f6f6ff;
    outline: 2px solid var(--color-11);
    outline-offset: -1px;
  }

  .mobile-cards-grid-compact {
    display: grid;
    grid-template-columns: repeat(24, 1fr);
    gap: 4px;
  }

  .mobile-cards-grid-cards {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 12px;
  }

  .mobile-cards-item-compact {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .mobile-cards-item-vertical {
    flex-direction: column;
    align-items: flex-start;
    row-gap: 0;
  }

  .mobile-cards-item-cards {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .mobile-cards-label-top {
    font-size: 14px;
    line-height: 1;
  }

  .mobile-cards-label-left {
    font-size: 14px;
    flex-shrink: 0;
  }

  .mobile-cards-label {
    font-size: 15px;
    color: #6d5dad;
    line-height: 1;
    margin-left: 8px;
  }

  .mobile-cards-content-wrapper,
  .mobile-cards-content-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 0;
  }

  .mobile-cards-content-row {
    gap: 6px;
    border-top: 0;
    border-radius: 4px;
    margin-top: -6px;
  }

  .mobile-cards-prefix {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
  }

  .mobile-cards-side {
    flex-shrink: 0;
  }

  .mobile-cards-compact-content,
  .mobile-cards-value-host {
    flex: 1;
    min-width: 0;
    word-break: break-word;
  }

  .mobile-cards-compact-content {
    position: relative;
  }

  .mobile-cards-value-host-interactive {
    position: relative;
    min-height: 32px;
    cursor: pointer;
  }

  .mobile-cards-highlight {
    color: #da3c3c;
    text-decoration: underline;
  }

  .mobile-cards-delete-button {
    position: absolute;
    top: 3px;
    right: 3px;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background-color: #ffe8ea;
    color: #e55757;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: rgb(144 35 35 / 51%) 0 1px 1px 0;
    z-index: 2;
    font-size: 15px;
  }

  .mobile-cards-delete-button:hover {
    background-color: #c82333;
    color: white;
  }

  .mobile-cards-editable-border {
    overflow: hidden;
    position: absolute;
    bottom: -4px;
    width: calc(100% + 4px);
    height: 18px;
    left: -2px;
  }

  .mobile-cards-editable-border > div {
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

  @media (max-width: 579px) {
    .mobile-cards-grid-cards {
      grid-template-columns: 1fr;
    }
  }
</style>
