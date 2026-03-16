<script lang="ts" generics="T">
  import Renderer, { type ElementAST } from '$components/Renderer.svelte';
  import CellEditable from '$components/vTable/CellEditable.svelte';
  import CellSelector from '$components/vTable/CellSelector.svelte';
  import { highlString, include } from '$libs/helpers';
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list';
  import type { CardRendererSnippet, ICardButtonDeleteHandler, ICardCell } from './types';

  interface ICardCellContent {
    content: string;
    contentHTML: string;
    contentAST: ElementAST | ElementAST[];
    prefixHTML: string;
    prefixAST: ElementAST | ElementAST[];
    useSnippet: boolean;
    css: string;
  }

  interface CardsListProps<T> {
    cells: ICardCell<T>[];
    data: T[];
    maxHeight?: string;
    css?: string;
    cardCss?: string;
    viewportClass?: string;
    estimateSize?: number;
    overscan?: number;
    onRowClick?: (row: T, index: number) => void;
    selected?: T | number;
    isSelected?: (row: T, selected: T | number) => boolean;
    emptyMessage?: string;
    cellRenderer?: CardRendererSnippet<T>;
    filterText?: string;
    getFilterContent?: (row: T) => string;
    useFilterCache?: boolean;
    buttonDeleteHandler?: ICardButtonDeleteHandler<T>;
    buttonDeleteIf?: (row: T, index: number) => boolean;
  }

  let {
    cells,
    data,
    maxHeight = 'calc(100vh - 8rem)',
    css = '',
    cardCss = '',
    viewportClass = '',
    estimateSize = 180,
    overscan = 6,
    onRowClick,
    selected,
    isSelected,
    emptyMessage = 'No se encontraron registros.',
    cellRenderer,
    filterText,
    getFilterContent,
    useFilterCache = false,
    buttonDeleteHandler,
    buttonDeleteIf
  }: CardsListProps<T> = $props();

  const filterCache = new WeakMap<T & object, string>();

  // Keep the visible card configuration stable and skip hidden entries early.
  const visibleCells = $derived.by(() => {
    return cells.filter(cell => !cell.hidden);
  });

  const filterTextArray = $derived((filterText || '').toLowerCase().split(' ').filter(x => x.length > 1));

  // Reuse the same filter contract as VTable so consumers can switch components with minimal changes.
  const filteredData = $derived.by(() => {
    if (filterText && getFilterContent) {
      return data.filter(record => {
        let content = '';
        if (useFilterCache && typeof record === 'object' && record !== null) {
          const cached = filterCache.get(record as T & object);
          if (cached) {
            content = cached;
          } else {
            content = getFilterContent(record).toLowerCase();
            filterCache.set(record as T & object, content);
          }
        } else {
          content = getFilterContent(record).toLowerCase();
        }
        return include(content, filterTextArray);
      });
    }
    return data;
  });

  function getCellContent(cell: ICardCell<T>, record: T, index: number): ICardCellContent {
    const resolvedContent = {} as ICardCellContent;

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

    if (cellRenderer && cell.id) {
      resolvedContent.useSnippet = true;
    }

    resolvedContent.css = typeof cell.cellCss === 'string'
      ? cell.cellCss
      : (cell.onCellEdit || cell.onCellSelect ? 'cards-list-field-input-host' : '');

    const dynamicCellCss = cell.setCellCss?.(record);
    if (dynamicCellCss) {
      resolvedContent.css += ` ${dynamicCellCss}`;
    }
    if (cell.css) {
      resolvedContent.css += ` ${cell.css}`;
    }

    return resolvedContent;
  }

  function getLabelContent(cell: ICardCell<T>): string {
    return typeof cell.label === 'function' ? cell.label() : cell.label;
  }

  function isRowSelected(record: T): boolean {
    if (!selected || !isSelected) return false;
    return isSelected(record, selected);
  }

  function handleRowClick(record: T, index: number) {
    if (onRowClick) {
      onRowClick(record, getRecordIndex(record, index));
    }
  }

  function buildSelectorId(cell: ICardCell<T>, rowIndex: number, cellIndex: number): string {
    return `${String(cell.id || cell.field || cellIndex)}_${rowIndex}`;
  }

  function isInteractiveCell(cell: ICardCell<T>): boolean {
    return Boolean(cell.onCellEdit || cell.onCellSelect);
  }

  function getRecordIndex(record: T, fallbackIndex: number): number {
    const dataIndex = data.indexOf(record);
    return dataIndex >= 0 ? dataIndex : fallbackIndex;
  }
</script>

<div class="cards-list-container {css}"
  style="height: {maxHeight}; max-height: {maxHeight};"
>
  {#if filteredData.length === 0}
    <div class="cards-list-empty-message">
      {emptyMessage}
    </div>
  {:else}
    <SvelteVirtualList viewportClass={viewportClass}
      items={filteredData}
      defaultEstimatedItemHeight={estimateSize}
      bufferSize={overscan}
    >
      {#snippet renderItem(record, index)}
        {@const recordIndex = getRecordIndex(record, index)}
        <div
          class="cards-list-card {cardCss}"
          class:cards-list-card-selected={isRowSelected(record)}
          role="button"
          tabindex="0"
          onclick={() => handleRowClick(record, recordIndex)}
          onkeydown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              handleRowClick(record, recordIndex);
            }
          }}
        >
          {#if buttonDeleteHandler && (!buttonDeleteIf || buttonDeleteIf(record, recordIndex))}
            <button
              type="button"
              class="cards-list-delete-button"
              aria-label="eliminar"
              onclick={event => {
                event.stopPropagation();
                console.debug('[CardsList] buttonDeleteHandler', { rowIndex: recordIndex, record });
                buttonDeleteHandler(record, recordIndex);
              }}
            >
              <i class="icon-trash"></i>
            </button>
          {/if}
          <div class="cards-list-grid">
            {#each visibleCells as cell, cellIndex (`${String(cell.id || cell.field || cellIndex)}_${filterText || ''}`)}
              {@const shouldRender = !cell.if || cell.if(record, recordIndex)}
              {#if shouldRender}
                {@const cellData = getCellContent(cell, record, recordIndex)}
                <div class="cards-list-item {cell.itemCss || 'col-span-full'}">
                  <div class="cards-list-label {cell.labelCss || ''}">
                    {getLabelContent(cell)}
                  </div>
                  <div class="cards-list-content {cell.contentCss || ''}">
                    {#if cellData.prefixAST}
                      <span class="cards-list-prefix">
                        <Renderer elements={cellData.prefixAST} />
                      </span>
                    {:else if cellData.prefixHTML}
                      <span class="cards-list-prefix">
                        {@html cellData.prefixHTML}
                      </span>
                    {/if}

                    <div class="cards-list-value-host {cellData.css}" class:cards-list-value-host-interactive={isInteractiveCell(cell)}>
                      {#if cell.onCellEdit}
                      	<div class="cell-editable-border"><div></div></div>
                        <CellEditable
                          contentClass={cell.contentCss}
                          inputClass={cell.inputCss}
                          type={cell.type || 'text'}
                          getValue={() => cellData.content}
                          render={
                            (cell.render
                              ? () => cell.render?.(record, recordIndex)
                              : undefined) as (value: number | string) => ElementAST[]
                          }
                          onChange={value => {
                            console.debug('[CardsList] onCellEdit', {
                              rowIndex: recordIndex,
                              cellId: cell.id,
                              field: cell.field,
                              value
                            });
                            cell.onCellEdit?.(record, value);
                          }}
                        />
                      {:else if cell.onCellSelect}
                     		<div class="cell-editable-border"><div></div></div>
                        <CellSelector
                          id={buildSelectorId(cell, recordIndex, cellIndex)}
                          saveOn={record}
                          save={cell.field as keyof T}
                          options={cell.cellOptions as any[]}
                          keyId={(cell.cellOptionsKeyId || 'ID') as never}
                          keyName={(cell.cellOptionsKeyName || 'Name') as never}
                          contentClass={cell.contentCss}
                          onChange={value => {
                            console.debug('[CardsList] onCellSelect', {
                              rowIndex: recordIndex,
                              cellId: cell.id,
                              field: cell.field,
                              value,
                              optionsLength: cell.cellOptions?.length || 0
                            });
                            cell.onCellSelect?.(record, value);
                          }}
                        />
                      {:else if cellData.useSnippet && cellRenderer}
                        {@render cellRenderer(record, cell, cellData.content, recordIndex)}
                      {:else if cellData.contentAST}
                        <Renderer elements={cellData.contentAST} />
                      {:else if cellData.contentHTML}
                        {@html cellData.contentHTML}
                      {:else}
                        {#if filterText}
                          {#each highlString(cellData.content, filterTextArray) as part}
                            <span class:cards-list-highlight={part.highl}>{part.text}</span>
                          {/each}
                        {:else}
                          {cellData.content}
                        {/if}
                      {/if}
                    </div>
                  </div>
                </div>
              {/if}
            {/each}
          </div>
        </div>
      {/snippet}
    </SvelteVirtualList>
  {/if}
</div>

<style>
  .cards-list-container {
    overflow: hidden;
    background-color: white;
    border: none;
    border-radius: 0;
    box-shadow: none;
    padding: 0;
  }

  .cards-list-container :global(.virtual-list-items) {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .cards-list-container :global(.virtual-list-viewport) {
    overflow-y: auto;
  }

  .cards-list-card {
    position: relative;
    border-radius: 10px;
    background-color: #f7f7fa;
    padding: 12px;
    cursor: pointer;
    transition: outline-color 0.15s ease, box-shadow 0.15s ease;
    box-shadow: rgb(69 68 93 / 22%) 0px 1px 3px;
    outline: 1px solid transparent;
  }

  .cards-list-card:hover {
    outline-color: #cfd4e6;
  }

  .cards-list-card-selected {
    background-color: #f6f6ff;
    outline: 2px solid var(--color-11);
  }

  .cards-list-grid {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 12px;
  }

  .cards-list-delete-button {
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

  .cards-list-delete-button:hover {
    background-color: #c82333;
    color: white;
  }

  .cards-list-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .cards-list-label {
	  font-size: 15px;
	  color: #6d5dad;
    line-height: 1;
    margin-left: 8px;
  }

  .cards-list-content {
  	display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    border-top: 0;
    border-radius: 4px;
    margin-top: -6px;
  }

  .cards-list-prefix {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
  }

  .cards-list-value-host {
    min-width: 0;
    flex: 1;
  }

  .cards-list-value-host-interactive {
    position: relative;
    min-height: 32px;
  }

  .cards-list-empty-message {
    color: #6c757d;
    text-align: center;
    padding: 32px 16px;
  }

  .cards-list-highlight {
    color: #da3c3c;
    text-decoration: underline;
  }

  .cell-editable-border {
 		overflow: hidden;
 		position: absolute;
    bottom: -4px;
    width: calc(100% + 4px);
    height: 18px;
    left: -2px;
  }
  
  .cell-editable-border > div {
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
    .cards-list-container {
      width: calc(100% + 8px);
      margin-left: -4px;
      margin-right: -4px;
      border: none;
      box-shadow: none;
      padding: 4px;
    }

    .cards-list-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
