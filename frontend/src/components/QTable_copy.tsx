import { VirtualItem, createVirtualizer } from "@tanstack/solid-virtual"
import { For, JSX, Show, createEffect, createMemo, createSignal, on } from "solid-js"
import { include } from "~/core/main"
import { highlString } from "./SearchSelect"
import { deviceType, viewType } from "~/app"
import { VList } from "../components/virtua/solid";

export interface ITableColumn<T> {
  id?: number
  header: (JSX.Element|string) | (() => (JSX.Element|string))
  headerCss?: string
  headerStyle?: JSX.CSSProperties
  cellStyle?: JSX.CSSProperties
  css?: string
  cardCss?: string
  field?: string
  subcols?: ITableColumn<T>[]
  cardColumn?: [number, (1|2|3)?]
  cardRender?: (e: T, idx: number, rerender: (ids?: number[]) => void) => (JSX.Element|string)
  getValue?: (e: T, idx: number) => (string|number)
  render?: (e: T, idx: number, rerender: (ids?: number[]) => void) => (JSX.Element|string)
  _colspan?: number
}

export interface IQTableBase<T> {
  columns: ITableColumn<T>[]
}

export interface IQTable<T> extends IQTableBase<T> {
  data: T[]
  maxHeight: string
  css?: string
  style?: JSX.CSSProperties
  tableStyle?: JSX.CSSProperties
  tableCss?: string
  styleMobile?: JSX.CSSProperties
  styleTable?: JSX.CSSProperties
  selected?: number | string | T
  isSelected?: (e: T, c: number | string | T) => boolean
  onRowCLick?: (e: T) => void
  filterText?: string
  makeFilter?:(e: T) => string
  filterKeys?: string[]
}

export interface ITableRow<T> {
  columns: ITableColumn<T>[]
  row: VirtualItem
  record: T
  isFinal?: boolean
  firstItemStart?: number
  tableSize?: number
  isSelected?: boolean
  onRowCLick?: (e: T) => void
  filterText?: string
  filterKeys?: string[]
}

export function QTable<T>(props: IQTable<T>) {

  let inputRef: HTMLDivElement = undefined as unknown as HTMLDivElement

  const makeCardColumns = (): (ITableColumn<T>[][]) => {
    let cardColumnsFlat: ITableColumn<T>[] = []
    for(let co of props.columns){
      if(co.subcols){
        for(let sc of co.subcols){ cardColumnsFlat.push(sc) }
      } else {
        cardColumnsFlat.push(co)
      }
    }
    cardColumnsFlat = cardColumnsFlat.filter(x => x.cardColumn?.length > 0)
    cardColumnsFlat.sort((a,b) => a.cardColumn[0] - b.cardColumn[0])

    const cardColumnsMap: Map<number,ITableColumn<T>[]> = new Map()
    for(let e of cardColumnsFlat){
      cardColumnsMap.has(e.cardColumn[0])
        ? cardColumnsMap.get(e.cardColumn[0]).push(e)
        : cardColumnsMap.set(e.cardColumn[0],[e])
    }

    const cardColumns: ITableColumn<T>[][] = []
    for(let columns of cardColumnsMap.values()){
      columns.sort((a,b) => (a.cardColumn[1]||1) - (b.cardColumn[1]||1))
      cardColumns.push(columns)
    }
    return cardColumns
  }

  let columnsMap: Map<string,ITableColumn<T>> = new Map()
  let cardColumns: ITableColumn<T>[][] = makeCardColumns()

  const isCardView = () => {
    console.log("is card view:: ", deviceType(), cardColumns.length)
    return deviceType() === 3 && cardColumns.length > 0
  }

  createEffect(on(() => [props.columns], 
    () => {
      columnsMap = new Map()
      for(let co of props.columns){
        if(co.field){ columnsMap.set(co.field, co)  }
      }
      cardColumns = makeCardColumns()
    }
  ))

  const makeVirtualizer = (props: IQTable<T>, length?: number) => {
    return createVirtualizer({
      count: (typeof length === 'number' ? length : props.data.length) || 1,
      estimateSize: () => 34,
      getScrollElement: () => { return inputRef },
    })
  }

  let Virtualizer = makeVirtualizer(props)
  const [recordRows, setRecordRows] = createSignal(Virtualizer.getVirtualItems())
  const [records, setRecords] = createSignal(props.data)

  const filter = (props: IQTable<T>, filterText?: string) => {

    let makeFilter = props.makeFilter
    if(!makeFilter && props.filterKeys?.length > 0){
      makeFilter = (e) => {
        let content = []
        for(let key of props.filterKeys){
          const key_ = key as keyof T
          if(e[key_] && (typeof e[key_] === 'string' || typeof e[key_] === 'number')){
            content.push(String(e[key_]))
          }
        }
        return content.join(" ").toLowerCase()
      }
    }

    if(!makeFilter || !filterText){ return props.data }

    let recordsFiltered: T[] = []
    const filterText_ = filterText.split(" ").map(x => x.toLowerCase())

    for(let e of props.data){
      const text = makeFilter(e)
      if(!text){ continue }
      if(include(text, filterText_)){
        recordsFiltered.push(e)
      }
    }

    return recordsFiltered
  }

  createEffect(on(() => [props.data, props.filterText||""], 
    () => {
      const recordsFiltered = filter(props, props.filterText)
      setRecords(recordsFiltered)
      Virtualizer = makeVirtualizer(props, recordsFiltered.length)
      setRecordRows(Virtualizer.getVirtualItems())
    }
  ))
  
  const tableSize = createMemo(() => {
    const _records = recordRows()
    console.log("registros render:: ", Virtualizer.getTotalSize() , Virtualizer.getVirtualItems().length, _records.length)
    if(_records.length === 0){ return 50 }
    return Virtualizer.getTotalSize() - _records[0].size * _records.length
  })

  const divClass = () => {
    if(isCardView()){ return "w100" }
    else { return "qtable-c" + (props.css  ? " " + props.css : "") }
  }

  console.log("style 11::", props.style)

  return <div class={divClass()} 
    ref={inputRef} style={{ 
      "max-height": props.maxHeight || "calc(100vh - 8rem - 12px)",
      ...(props.style||{}),
      ...(isCardView() ? props.styleMobile || {} : {})
    }}>
    <Show when={!isCardView()}>
      <table class={"qtable h100 fixed-header " + (props.tableCss || "")} 
        style={props.tableStyle}>
        <thead>
          <QTableHeaders columns={props.columns}/>
        </thead>
        <tbody>
          <VList data={props.data} style={{ height: "70vh" }}>
            {(record, i) => {
              return <QTableRow row={{ index: i }} record={record}
                firstItemStart={recordRows()[0].start}
                tableSize={tableSize()}
                columns={props.columns}
                onRowCLick={props.onRowCLick}
                filterText={props.filterText}
                filterKeys={props.filterKeys}
              />
            }}
          </VList>
        </tbody>
      </table>
    </Show>
    <Show when={isCardView()}>
      <For each={recordRows()}>
        {(row,i) => {
          const records_ = records()
          const record = records_[row.index]
          
          const isSelected = createMemo(() => {
            const selected = props.selected && props.isSelected
              ? props.isSelected(record, props.selected) 
              : false
            return selected
          })

          if(records_.length === 0 && i() === 0){
            return <tr>
              <td colSpan={99}>
                <div class="empty-message flex ai-center">
                  No se encontraron registros.
                </div>
              </td>
            </tr>
          }

          return <div class={"w100 flex-column card-ct mb-04" 
              + (isSelected() ? " selected" : "")}
            onClick={ev => {
              ev.stopPropagation()
              if(props.onRowCLick){ props.onRowCLick(record) }
            }}
          >
            <For each={cardColumns}>
              {cols => {
                return <div class="flex ai-center jc-between">
                  { cols.map(col => {
                      let content: (string|JSX.Element) = ""
                      if(col.cardRender){ 
                        content = col.cardRender(record, i(), null) 
                      } else if(col.render){ 
                        content = col.render(record, -1, null) 
                      } else if(col.getValue){ 
                        content = col.getValue(record, i()) 
                      }
                      return <div class={col.cardCss||""}>{content}</div>
                    })
                  }
                </div>
              }}
            </For>
          </div>
        }}
      </For>
    </Show>
  </div>
}

export function QTableRow<T>(props: ITableRow<T>) {

  let cn = props.row.index % 2 === 0 ? "tr-even" : "tr-odd"
  if(props.isFinal){ cn += " tr-final" }

  const [columns] = createSignal(props.columns)

  const renderMap: Map<number,(() => void)> = new Map()

  const makeRerender = (columnIDs?: number[]) => {
    const idSet = columnIDs ? new Set(columnIDs) : null
    for(let [id, rerender] of renderMap){
      if(idSet && !idSet.has(id)){ continue }
      rerender()
    }
  }

  const makeContent = (column: ITableColumn<T>) => {

    let content: (string|number|JSX.Element) = ""
    if(column.render){ 
      content = column.render(props.record, 1, makeRerender) 
    } else if(column.getValue){
      content = column.getValue(props.record, 1)
    }

    if(typeof content === 'string' && column.field && props.filterText){
      if(props.filterKeys && props.filterKeys.includes(column.field)){
        return <span class="_highlight">
          {highlString(content, props.filterText.split(" "))}
        </span>
      }
    } else {
      return content
    }
  }

  return <tr class={cn + (props.isSelected ? " selected" : "")} 
      onClick={ev => {
        ev.stopPropagation()
        if(props.onRowCLick){ props.onRowCLick(props.record) }
      }}  
    >
      <For each={columns()}>
        {(column,i) => {
          const [content, setContent] = createSignal(makeContent(column))
          const rerender = () => { setContent(makeContent(column)) }
          renderMap.set(column.id || i(), rerender)

          return <td class={column.css || undefined} 
          style={column.cellStyle || undefined}>
            {content()}
          </td>
        }}
      </For>
    </tr>
}

export function QTableHeaders<T>(props: IQTableBase<T>) {

  const columns = () => {
    const columns1: ITableColumn<T>[] = []
    const columns2: ITableColumn<T>[] = []

    for(let column of props.columns){
      column._colspan = column?.subcols?.length || 0
      if(column._colspan){
        for(let sc of column.subcols){ columns2.push(sc) }
      }
      columns1.push(column)
    }

    return [columns1, columns2]
  }

  return <>
    <tr>
      <For each={columns()[0]}>
        {(column) => {
            return <QTableHeader column={column} />
          }
        }
      </For>
    </tr>
    <Show when={columns()[1].length > 0}>
      <For each={columns()[1]}>
        {(column) => {
            return <QTableHeader column={column} />
          }
        }
      </For>        
    </Show>    
  </>
}

interface IQTableHeader<T> {
  column: ITableColumn<T>
}

export function QTableHeader<T>(props: IQTableHeader<T>){
  const column = props.column
  let header = column.header as (string|JSX.Element)
  if(typeof column.header === 'function'){ header = column.header() }
  return <th style={column.headerStyle}>
    {header}
  </th>
}