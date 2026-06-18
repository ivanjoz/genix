<script lang="ts" module>
  export interface TableTreeNode<TRecord> {
    id: string | number
    record: TRecord
    children: TRecord[]
    isOpen: boolean
  }
</script>

<script lang="ts" generics="TRecord">
  import CellInput from '$components/vTable/CellInput.svelte'
  import Renderer, { type ElementAST } from '$components/misc/Renderer.svelte'
  import { Env } from '$core/env'
  import { tr } from '$core/store.svelte'
  import T from '$components/misc/T.svelte'
  import { Agent } from '$components/agent/registry'
  import {
    setVTableAgentContext,
    buildCellID,
    buildRowID,
    parseChildID,
    rowIndexFromRowID,
    type CellAgentMethods,
  } from '$components/vTable/agentContext'
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
    emptyMessage = 'No records found.|No se encontraron registros.',
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
    // Caller-owned state drives the row refresh; CellInput only needs the callback signature.
  }

  const componentID = Env.getComponentID()

  // Flat visible-row indices: node row, then child rows (when node.isOpen).
  // Row i in the flattened list has rowID = (i+1) * 100; cells inside use
  // rowID + columnIndex + 1 so the agent can address either via the same
  // composite scheme.
  const nodeFlatIndices = $derived.by(() => {
    const out: number[] = []
    let counter = 0
    for (let i = 0; i < data.length; i++) {
      out.push(counter)
      counter += 1
      if (data[i].isOpen) { counter += data[i].children.length }
    }
    return out
  })

  function flatChildIndex(nodeIndex: number, childIndex: number) {
    return (nodeFlatIndices[nodeIndex] ?? 0) + 1 + childIndex
  }

  const hasRowClick = $derived(Boolean(onNodeClick) || Boolean(onChildClick))

  // Cells (CellInput) hand their methods here keyed by cellID.
  const cellRegistry = new Map<number, CellAgentMethods>()

  setVTableAgentContext({
    tableID: componentID,
    registerCell: (cellID, methods) => {
      cellRegistry.set(cellID, methods)
      return () => {
        if (cellRegistry.get(cellID) === methods) { cellRegistry.delete(cellID) }
      }
    },
  })

  const hasInteractiveCell = $derived(
    columns.some((column) => column.onCellEdit || column.onCellSelect),
  )
  const shouldRegisterTable = $derived(
    Boolean(onNodeClick) || Boolean(onChildClick) || hasInteractiveCell,
  )

  // rowID → (nodeIndex, childIndex?) for select dispatch.
  function resolveFlatRow(rowID: number): { nodeIndex: number; childIndex?: number } | undefined {
    const rowIndex = rowIndexFromRowID(rowID)
    if (rowIndex < 0) { return undefined }
    for (let i = 0; i < data.length; i++) {
      const flat = nodeFlatIndices[i] ?? 0
      if (flat === rowIndex) { return { nodeIndex: i } }
      const childCount = data[i].isOpen ? data[i].children.length : 0
      if (rowIndex > flat && rowIndex <= flat + childCount) {
        return { nodeIndex: i, childIndex: rowIndex - flat - 1 }
      }
    }
    return undefined
  }

  const dispatchRowSelect = (rowID: number) => {
    const target = resolveFlatRow(rowID)
    if (!target) { return }
    const node = data[target.nodeIndex]
    if (target.childIndex === undefined) {
      toggleNode(node, target.nodeIndex)
      return
    }
    onChildClick?.(node.children[target.childIndex], target.childIndex, node)
  }

  $effect(() => {
    if (!shouldRegisterTable) { return }
    return Agent.register({
      id: componentID,
      type: "Table",
      label: "",
      select: (...ids) => {
        if (ids.length === 0) { return }
        const first = parseChildID(ids[0])
        if (Number.isFinite(first) && first % 100 === 0) {
          for (const rid of ids) { dispatchRowSelect(parseChildID(rid)) }
          return
        }
        cellRegistry.get(first)?.select?.(...ids.slice(1))
      },
      setValueChild: (cellID, value) => {
        cellRegistry.get(parseChildID(cellID))?.setValue?.(value)
      },
      searchChild: (cellID, text) => {
        cellRegistry.get(parseChildID(cellID))?.search?.(String(text ?? ''))
      },
      getOptionsChild: (cellID, max) => {
        return cellRegistry.get(parseChildID(cellID))?.getOptions?.(Number(max ?? 50)) ?? []
      },
    })
  })
</script>

<div data-id={shouldRegisterTable ? `Table:${componentID}` : undefined} class="table-tree-shell {css}">
  <div class="table-tree-plain-scroll">
    <div class="table-tree-header table-tree-header-sticky">
      <div class="table-tree-header-row" role="row" style:grid-template-columns={gridTemplateColumns}>
        {#each visibleColumns as columnDefinition, columnIndex (columnDefinition.id || columnIndex)}
          {@const headerPaddingCss = /px-|pr-|pl-/.test(columnDefinition.headerCss || '') ? '' : 'px-6'}
          <div
            class="table-tree-header-cell {headerPaddingCss} {getAlignClassName(columnDefinition.align)} {columnDefinition.headerCss || ''}"
            role="columnheader"
          >
            <T text={typeof columnDefinition.header === 'function' ? columnDefinition.header() : columnDefinition.header} />
          </div>
        {/each}
      </div>
    </div>

    {#if data.length === 0}
      <div class="table-tree-empty"><T text={emptyMessage} /></div>
    {:else}
      <div class="table-tree-body">
        {#each data as node, nodeIndex(node.id)}
          {@const nodeFlatIdx = nodeFlatIndices[nodeIndex] ?? 0}
          <div class="table-tree-row-shell"
            data-id={hasRowClick ? `TableRow:${componentID}:${buildRowID(nodeFlatIdx)}` : undefined}
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
                {@const combinedCellCss = `${cellCss || ''} ${colDef.css || ''} ${colDef.setCellCss?.(node.record) || ''}`}
                {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? '' : 'px-6'}
                {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || '') ? '' : 'px-6'}
                {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || '') ? '' : 'px-6'}
                <div
                  class="table-tree-cell {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                  role="cell"
                >
                  {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(node.record, nodeIndex)}
                    <CellInput
                      saveOn={node.record}
                      cellID={buildCellID(nodeFlatIdx, columnIndex)}
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
                      <i class="icon-[fa--chevron-down]"></i>
                    </span>
                  {/if}
                </div>
              {/each}
            </div>
          </div>

          {#if node.isOpen}
            {#each node.children as childRecord, childIndex (resolveChildId(childRecord, childIndex, node))}
              {@const childId = resolveChildId(childRecord, childIndex, node)}
              {@const childFlatIdx = flatChildIndex(nodeIndex, childIndex)}
              <div class="table-tree-row-shell table-tree-child-row-shell"
                data-id={hasRowClick ? `TableRow:${componentID}:${buildRowID(childFlatIdx)}` : undefined}
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
                    {@const combinedCellCss = `${cellCss || ''} ${colDef.css || ''} ${colDef.setCellCss?.(childRecord) || ''}`}
                    {@const cellPaddingCss = /px-|pr-|pl-/.test(combinedCellCss) ? '' : 'px-6'}
                    {@const contentPaddingCss = /px-|pr-|pl-/.test(colDef.css || '') ? '' : 'px-6'}
                    {@const inputPaddingCss = /px-|pr-|pl-/.test(colDef.inputCss || '') ? '' : 'px-6'}
                    <div
                      class="table-tree-cell {cellPaddingCss} {getAlignClassName(colDef.align)} {combinedCellCss}"
                      role="cell"
                    >
                      {#if columnIndex === 0}
                        <i class="table-tree-child-indent icon-[fa--level-up]"></i>
                      {/if}
                      {#if colDef.onCellEdit && !colDef.disableCellInteractions?.(childRecord, childIndex)}
                        <CellInput
                          saveOn={childRecord}
                          cellID={buildCellID(childFlatIdx, columnIndex)}
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
