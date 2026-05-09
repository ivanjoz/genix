<script lang="ts" module>
  export interface TableTreeNode<TRecord> {
    id: string | number
    record: TRecord
    children: TRecord[]
    isOpen: boolean
  }
</script>

<script lang="ts" generics="TRecord">
  import CellEditable from '$components/vTable/CellEditable.svelte'
  import Renderer, { type ElementAST } from '$components/Renderer.svelte'
  import { Env } from '$core/env'
  import { Agent } from '$core/agent/registry'
  import type { ITableColumn } from './types'

  interface TableTreeProps<TRecord> {
    columns: ITableColumn<TRecord>[]
    data: TableTreeNode<TRecord>[]
    css?: string
    rowCss?: string
    cellCss?: string
    emptyMessage?: string
    selectedId?: string | number
    selectedChildId?: string | number
    getChildId?: (record: TRecord, index: number, parent: TableTreeNode<TRecord>) => string | number
    onNodeClick?: (node: TableTreeNode<TRecord>, index: number) => void
    onChildClick?: (record: TRecord, index: number, parent: TableTreeNode<TRecord>) => void
  }

  let {
    columns,
    data,
    css = '',
    rowCss = '',
    cellCss = '',
    emptyMessage = 'No se encontraron registros.',
    selectedId,
    selectedChildId,
    getChildId,
    onNodeClick,
    onChildClick,
  }: TableTreeProps<TRecord> = $props()

  const visibleColumns = $derived(columns.filter((column) => !column.hidden))
  const gridTemplateColumns = $derived(
    visibleColumns.map((column) => column.width || 'minmax(120px, 1fr)').join(' '),
  )

  function getCellValue(record: TRecord, column: ITableColumn<TRecord>, rowIndex: number) {
    // Match TableGrid column value resolution so callers can reuse column definitions.
    if (column.getValue) { return column.getValue(record, rowIndex) }
    if (column.field) { return (record as Record<string, unknown>)[column.field] as string | number }
    return ''
  }

  function isHtmlContent(value: unknown) {
    return typeof value === 'string' && /<[a-z][\s\S]*>/i.test(value)
  }

  function renderCell(record: TRecord, column: ITableColumn<TRecord>, rowIndex: number) {
    // Render callbacks replace the default cell text, same as TableGrid.
    if (column.render) { return column.render(record, rowIndex) }
    return getCellValue(record, column, rowIndex)
  }

  function getAlignClassName(align?: 'left' | 'center' | 'right') {
    if (align === 'center') { return 'justify-center text-center' }
    if (align === 'right') { return 'justify-end text-right' }
    return 'justify-start text-left'
  }

  function toggleNode(node: TableTreeNode<TRecord>, nodeIndex: number) {
    node.isOpen = !node.isOpen
    data = [...data]
    console.debug('[table-tree] toggle node', { id: node.id, isOpen: node.isOpen, childCount: node.children.length })
    onNodeClick?.(node, nodeIndex)
  }

  function resolveChildId(record: TRecord, childIndex: number, parent: TableTreeNode<TRecord>) {
    return getChildId?.(record, childIndex, parent) || `${parent.id}_${childIndex}`
  }

  function rerenderNoop() {
    // Caller-owned state drives the row refresh; CellEditable only needs the callback signature.
  }

  const componentID = Env.getComponentID()

  $effect(() => {
    if (!onNodeClick && !onChildClick) { return }
    return Agent.register({
      id: componentID,
      type: "Table",
      label: "",
      select: (...ids) => {
        const targets = new Set(ids.map(String))
        for (let i = 0; i < data.length; i++) {
          const node = data[i]
          if (targets.has(String(node.id))) {
            toggleNode(node, i)
            continue
          }
          if (!onChildClick || !node.isOpen) { continue }
          for (let j = 0; j < node.children.length; j++) {
            const childRecord = node.children[j]
            if (targets.has(String(resolveChildId(childRecord, j, node)))) {
              onChildClick(childRecord, j, node)
            }
          }
        }
      },
    })
  })
</script>

<div data-id="Table:{componentID}" class="table-tree-shell {css}">
  <div class="table-tree-plain-scroll">
    <div class="table-tree-header table-tree-header-sticky">
      <div class="table-tree-header-row" role="row" style:grid-template-columns={gridTemplateColumns}>
        {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id || columnIndex)}
          {@const headerPaddingCss = /px-|pr-|pl-/.test(columnDefinition.headerCss || '') ? '' : 'px-6'}
          <div
            class="table-tree-header-cell {headerPaddingCss} {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
            role="columnheader"
          >
            {typeof columnDefinition.header === 'function' ? columnDefinition.header() : columnDefinition.header}
          </div>
        {/each}
      </div>
    </div>

    {#if data.length === 0}
      <div class="table-tree-empty">{emptyMessage}</div>
    {:else}
      <div class="table-tree-body">
        {#each data as node, nodeIndex(node.id)}
          <div class="table-tree-row-shell"
            data-id="TableRow:{node.id}"
            data-selected={selectedId === node.id ? "true" : undefined}>
            <div
              class="table-tree-row {rowCss}"
              class:table-tree-row-selected={selectedId === node.id}
              role="row"
              tabindex="0"
              style:grid-template-columns={gridTemplateColumns}
              onclick={() => toggleNode(node, nodeIndex)}
              onkeydown={(ev) => {
                if (ev.key === 'Enter' || ev.key === ' ') {
                  ev.preventDefault()
                  toggleNode(node, nodeIndex)
                }
              }}
            >
              {#each visibleColumns as colDef, columnIndex (`${node.id}_${colDef.id || columnIndex}`)}
                {@const defaultCellValue = getCellValue(node.record, colDef, nodeIndex)}
                {@const renderedCellContent = renderCell(node.record, colDef, nodeIndex)}
                {@const combinedCellCss = `${cellCss || ''} ${(colDef as any).cellCss || ''} ${colDef.setCellCss?.(node.record) || ''} ${colDef.css || ''}`}
                {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? '' : 'px-6'}
                {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || '') ? '' : 'px-6'}
                {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || '') ? '' : 'px-6'}
                <div
                  class="table-tree-cell {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                  role="cell"
                >
                  {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(node.record, nodeIndex)}
                    <CellEditable
                      saveOn={node.record}
                      getValue={() => defaultCellValue}
                      render={colDef.formatInputValue}
                      type={colDef.cellInputType || 'text'}
                      inputClass={`${inputPaddingCss} ${colDef.inputCss || ''}${colDef.align === 'right' ? ' text-right' : ''}`}
                      contentClass={`${contentPaddingCss} ${colDef.css || ''}${colDef.align === 'right' ? ' justify-end text-right' : ''}`}
                      onChange={(value) => colDef.onCellEdit?.(node.record, value, rerenderNoop)}
                    />
                  {:else if isHtmlContent(renderedCellContent)}
                    {@html renderedCellContent}
                  {:else if Array.isArray(renderedCellContent) || (typeof renderedCellContent === 'object' && renderedCellContent !== null)}
                    <Renderer elements={renderedCellContent as ElementAST | ElementAST[]} />
                  {:else}
                    {renderedCellContent}
                  {/if}
                  {#if columnIndex === 0}
                    <span class="table-tree-chevron" class:table-tree-chevron-open={node.isOpen}>
                      <i class="icon-down-open-1"></i>
                    </span>
                  {/if}
                </div>
              {/each}
            </div>
          </div>

          {#if node.isOpen}
            {#each node.children as childRecord, childIndex (resolveChildId(childRecord, childIndex, node))}
              {@const childId = resolveChildId(childRecord, childIndex, node)}
              <div class="table-tree-row-shell table-tree-child-row-shell"
                data-id="TableRow:{childId}"
                data-selected={selectedChildId === childId ? "true" : undefined}>
                <div
                  class="table-tree-row table-tree-child-row {rowCss}"
                  class:table-tree-row-selected={selectedChildId === childId}
                  role="row"
                  tabindex="0"
                  style:grid-template-columns={gridTemplateColumns}
                  onclick={() => {
                    console.debug('[table-tree] child click', { parentId: node.id, childId })
                    onChildClick?.(childRecord, childIndex, node)
                  }}
                  onkeydown={(ev) => {
                    if (ev.key === 'Enter' || ev.key === ' ') {
                      ev.preventDefault()
                      console.debug('[table-tree] child keydown', { parentId: node.id, childId })
                      onChildClick?.(childRecord, childIndex, node)
                    }
                  }}
                >
                  {#each visibleColumns as colDef, columnIndex (`${childId}_${colDef.id || columnIndex}`)}
                    {@const defaultCellValue = getCellValue(childRecord, colDef, childIndex)}
                    {@const renderedCellContent = renderCell(childRecord, colDef, childIndex)}
                    {@const combinedCellCss = `${cellCss || ''} ${(colDef as any).cellCss || ''} ${colDef.setCellCss?.(childRecord) || ''} ${colDef.css || ''}`}
                    {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? '' : 'px-6'}
                    {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || '') ? '' : 'px-6'}
                    {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || '') ? '' : 'px-6'}
                    <div
                      class="table-tree-cell {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                      role="cell"
                    >
                      {#if columnIndex === 0}
                        <i class="table-tree-child-indent icon-level-down"></i>
                      {/if}
                      {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(childRecord, childIndex)}
                        <CellEditable
                          saveOn={childRecord}
                          getValue={() => defaultCellValue}
                          render={colDef.formatInputValue}                          type={colDef.cellInputType || 'text'}
                          inputClass={`${inputPaddingCss} ${colDef.inputCss || ''}${colDef.align === 'right' ? ' text-right' : ''}`}
                          contentClass={`${contentPaddingCss} ${colDef.css || ''}${colDef.align === 'right' ? ' justify-end text-right' : ''}`}
                          onChange={(value) => colDef.onCellEdit?.(childRecord, value, rerenderNoop)}
                        />
                      {:else if isHtmlContent(renderedCellContent)}
                        {@html renderedCellContent}
                      {:else if Array.isArray(renderedCellContent) || (typeof renderedCellContent === 'object' && renderedCellContent !== null)}
                        <Renderer elements={renderedCellContent as ElementAST | ElementAST[]} />
                      {:else}
                        {renderedCellContent}
                      {/if}
                    </div>
                  {/each}
                </div>
              </div>
            {/each}
          {/if}
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .table-tree-shell {
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    font-size: 13px;
  }

  .table-tree-plain-scroll {
    min-height: 0;
    overflow: auto;
    position: relative;
    border: 1px solid #cfcedf;
  }

  .table-tree-header {
    background: #f8fafc;
    border-bottom: 1px solid #e5e7eb;
  }

  .table-tree-header-sticky {
    position: sticky;
    top: 0;
    z-index: 2;
  }

  .table-tree-header-row,
  .table-tree-row {
    display: grid;
    width: 100%;
    align-items: stretch;
  }

  .table-tree-header-row {
    min-height: 34px;
  }

  .table-tree-header-cell {
    min-width: 0;
    display: flex;
    align-items: center;
    overflow: hidden;
    color: #64748b;
    font-weight: 700;
    border-right: 1px solid #e5e7eb;
  }

  .table-tree-header-cell:last-child {
    border-right: 0;
  }

  .table-tree-body {
    min-height: 0;
  }

  .table-tree-row-shell {
    height: 34px;
  }

  .table-tree-child-row-shell {
    height: 32px;
    margin-left: 0;
  }

  .table-tree-row {
    height: 100%;
    border-bottom: 1px solid #e5e7eb;
    background: #f8fafc;
    color: #334155;
    cursor: pointer;
    overflow: hidden;
    box-shadow: inset 0em 0em 0 2px #ffffff;
  }

  .table-tree-body > .table-tree-row-shell:last-child .table-tree-row {
    border-bottom: 0;
  }

  .table-tree-row:hover {
  	background: #f1f0f9;
   	box-shadow: none;
  }

  .table-tree-row:focus-visible {
    outline: 2px solid #8b5cf6;
    outline-offset: 1px;
  }

  .table-tree-child-row {
    background: white;
    box-shadow: none;
  }

  .table-tree-row-selected,
  .table-tree-row-selected.table-tree-row:hover {
    background: #f5f3ff;
    color: #4c1d95;
  }

  .table-tree-cell {
    min-width: 0;
    display: flex;
    align-items: center;
    overflow: hidden;
    white-space: nowrap;
    border-right: 1px solid #e5e7eb;
    position: relative;
  }

  .table-tree-cell:last-child {
    border-right: 0;
  }

  .table-tree-chevron {
    position: absolute;
    right: 8px;
    width: 16px;
    color: #64748b;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.3s ease;
  }

  .table-tree-chevron-open {
    transform: rotate(180deg);
  }

  .table-tree-child-indent {
    width: 16px;
    margin-right: 8px;
    color: #8aa0bd;
    flex: 0 0 auto;
  }

  .table-tree-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 160px;
    color: #94a3b8;
  }

  .table-tree-plain-scroll::-webkit-scrollbar {
    width: 10px;
  }
</style>
