<script lang="ts" generics="T">
  import { wordInclude } from '$libs/helpers';
  import MobileCardsVirtualList from '$components/vTable/MobileCardsVirtualList.svelte';
  import type {
    CardRendererSnippet,
    ICardButtonDeleteHandler,
    ICardCell,
    IMobileCardsListCell,
  } from './types';

  interface CardsListProps<T> {
    cells: ICardCell<T>[];
    data: T[];
    height?: string;
    css?: string;
    cardCss?: string;
    viewportClass?: string;
    itemsClass?: string;
    estimateSize?: number;
    overscan?: number;
    onRowClick?: (row: T, index: number, rerender: () => void) => void;
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
    height = 'calc(100vh - 8rem)',
    css = '',
    cardCss = '',
    viewportClass = '',
    itemsClass = '',
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
  // Keep the virtualizer base classes and append custom classes without breaking layout internals.
  const virtualViewportClass = $derived(`virtual-list-viewport ${viewportClass}`.trim());
  const virtualItemsClass = $derived(`virtual-list-items ${itemsClass}`.trim());

  // Adapt the card cells once so the shared renderer can keep the edit/select behavior centralized.
  const mobileCells = $derived.by((): IMobileCardsListCell<T, ICardCell<T>>[] => {
    return cells.map((cell) => ({
      ...cell,
      source: cell,
      useRenderer: Boolean(cellRenderer && cell.id),
    }));
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
        return wordInclude(content, filterTextArray);
      });
    }
    return data;
  });

  function isRowSelected(record: T): boolean {
    if (!selected || !isSelected) return false;
    return isSelected(record, selected);
  }

  function handleRowClick(record: T, index: number, rerender: () => void) {
    if (onRowClick) {
      onRowClick(record, getRecordIndex(record, index), rerender);
    }
  }
  function getRecordIndex(record: T, fallbackIndex: number): number {
    const dataIndex = data.indexOf(record);
    return dataIndex >= 0 ? dataIndex : fallbackIndex;
  }
</script>

<div class="cards-list-container {css}"
  style="height: {height};"
>
  {#if filteredData.length === 0}
    <div class="cards-list-empty-message">
      {emptyMessage}
    </div>
  {:else}
    <MobileCardsVirtualList
      data={filteredData}
      cells={mobileCells}
      variant="cards"
      cardCss={`mb-6 ${cardCss}`.trim()}
      showSelectedCard={true}
      viewportClass={virtualViewportClass}
      itemsClass={virtualItemsClass}
      estimateSize={estimateSize}
      overscan={overscan}
      emptyMessage={emptyMessage}
      filterText={filterText}
      highlightPlainText={true}
      onRowClick={onRowClick ? handleRowClick : undefined}
      selected={selected}
      isSelected={isSelected ? isRowSelected : undefined}
      getRecordIndex={getRecordIndex}
      buttonDeleteHandler={buttonDeleteHandler}
      buttonDeleteIf={buttonDeleteIf}
      debugName="CardsList"
      legacyCardCellRenderer={cellRenderer}
    />
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
    min-height: 0;
  }

  /* Force the virtual list wrapper to inherit the card list height so its viewport can scroll. */
  .cards-list-container :global(.virtual-list) {
    height: 100%;
    min-height: 0;
  }

  /* Avoid flex gap here because the virtualizer does not include it in scroll height calculations. */
  .cards-list-container :global(.virtual-list-items) {
    display: block;
  }

  .cards-list-container :global(.virtual-list-items > div) {
    margin-bottom: 8px;
  }

  .cards-list-container :global(.virtual-list-items > div:last-child) {
    margin-bottom: 0;
  }

  .cards-list-container :global(.virtual-list-viewport) {
    height: 100%;
    min-height: 0;
    overflow-y: auto;
  }

  .cards-list-empty-message {
    color: #6c757d;
    text-align: center;
    padding: 32px 16px;
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
  }
</style>
