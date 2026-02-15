import * as ExcelJS from 'exceljs'
import { Alignment, Borders, Cell, CellFormulaValue, CellHyperlinkValue, Row, Workbook, Worksheet } from 'exceljs'

export { ExcelJS, Workbook }
export type { Alignment, Borders, Cell, CellFormulaValue, CellHyperlinkValue, Row, Worksheet }

export interface IExcelColumn {
  v: string
  key?: string
  subcols?: IExcelColumn[]
  format?: string
  width?: number
  maxWidth?: number
  field?: string
  idx?: number
  id?: number
  bgColor?: string
  colspan?: number
  fontColor?: string
  alignment?: Partial<Alignment>
  isFecha?: boolean
  divLine?: boolean
  type?: string
  isLink?: boolean
  getValue?: (e: any) => (number | string | Date)
  setCellProperties?: (record: any, cell: Cell, colN: number, rowN: number) => void
  formatter?: (cell: Cell, e?: any) => void
  getList?: () => string[]
  _validationListID?: number
  background?: string
  color?: string
  parse?: (value: any, saveError: (e: string) => void, obj: any) => any
  name?: string
  setValue?: (e: any) => any
  state?: string
}

declare global {
  interface Window {
    tt?: (msg: any, isHeader?: boolean) => string
    ttOriginal?: (msg: any) => string
    _zoneOffset: number
    _filialZonaHoraria: number
    excelWorker: any
  }
}

export interface IValidationList {
  id: number, values: string[], formulae?: string
}

export interface ExcelAccion {
  accion: number, sheets: ExcelSheet[], creator?: string,
  validationLists: IValidationList[]
}

export type excelParseHandler<T> = (
  e: string | number, saveError?: (e: string) => void, regRow?: T,
) => string | number | boolean | string[] | number[] | void

export interface columnExcelArgs<T = any> {
  name?: string, width?: number, maxWidth?: number, colspan?: number,
  format?: string, type?: string, field?: keyof T,
  decimals?: number, numFmt?: string | Date,
  bgColor?: string, fontColor?: string,
  alignment?: Partial<Alignment>,
  isLink?: boolean, // si es true, el valor se interpreta como un link
  setValue?: (e: T, i?: number) => string | number | Date,
  getList?: () => string[]
  setCellProperties?: (e: T, cell: Cell, colN: number, rowN: number) => void
  parse?: excelParseHandler<T>,
  formatter?: (cell: Cell, e: T) => void,
  // Se usan?
  render?: (e: T) => any,
  getValue?: (e: T) => any
  subcols?: any[]
}

export interface ExcelSheet {
  title: string
  name: string
  columns: IExcelColumn[]
  headerIdx: number
  values?: any[][]
  valuesParsed?: any[][]
  maxWidth?: number
  widthFactor?: number
  state?: string
}

export interface ICellLayout {
  col: IExcelColumn
  colStart: number
  colEnd?: number
  rowStart: number
  rowEnd?: number
  bgColor?: string
  fontColor?: string
  hasBorder?: boolean
}

const normalice = (e: string) => {
  return (e || "").replaceAll(" ", "_").toLowerCase()
}

export const makeColumnsFlat = (
  columns: IExcelColumn[], 
  sheetID: number, 
  columIdxWidthMap?: Map<number, number>,
  columIdxMaxWidthMap?: Map<number, number>
): IExcelColumn[] => {
  const columnsFlat: IExcelColumn[] = []
  if (columns) {
    let colIdx = 1
    for (const col of columns) {
      if (!col) { continue }
      if (col.subcols && col.subcols.length > 0) {
        for (let i = 0; i < col.subcols.length; i++) {
          const sub = col.subcols[i]
          if (!sub) { continue }
          if (sub.subcols && sub.subcols.length > 0) {
            for (let j = 0; j < sub.subcols.length; j++) {
              const sscol = sub.subcols[j]
              if (!sscol) { continue }
              sscol.key = normalice(col.v) + "_" + normalice(sub.v) + "_" + normalice(sscol.v)
              if (columIdxWidthMap && sscol.width) {
                columIdxWidthMap.set(colIdx, sscol.width)
              }
              if (columIdxMaxWidthMap && sscol.maxWidth) {
                columIdxMaxWidthMap.set(colIdx, sscol.maxWidth)
              }
              sscol.idx = colIdx; colIdx++
              columnsFlat.push(sscol)
            }
          } else {
            sub.key = normalice(col.v) + "_" + normalice(sub.v)

            if (col.divLine && i === 0) { sub.divLine = true }
            if (columIdxWidthMap && sub.width) { columIdxWidthMap.set(colIdx, sub.width) }
            sub.idx = colIdx; colIdx++
            columnsFlat.push(sub)
          }
        }
      } else {
        if (columIdxWidthMap && col.width) { columIdxWidthMap.set(colIdx, col.width) }
        col.idx = colIdx; colIdx++
        col.key = normalice(String(col.v || ""))
        columnsFlat.push(col)
      }
    }
  }
  if (sheetID) {
    for (const e of columnsFlat) {
      e.id = e.id || (sheetID * 1000 + (e.idx || 0))
    }
  }
  return columnsFlat
}

export function fillCell(cell: Cell, bgColor?: string, fontColor?: string) {
  if (bgColor) {
    if (bgColor[0] === '#') bgColor = bgColor.substring(1, bgColor.length)
    cell.fill = {
      type: 'pattern', pattern: 'solid',
      fgColor: { argb: 'FF' + bgColor }
    }
  }
  if (fontColor) {
    if (fontColor[0] === '#') {
      fontColor = fontColor.substring(1, fontColor.length)
    }
    cell.font = { color: { argb: 'FF' + fontColor } }
  }
}

export function monedaFormat(cell: Cell, value: number | string, moneda: string) {
  cell.value = value
  cell.numFmt = '_-[$S/-es-PE]* #,##0.00_-'
}

export function porcentFormat(cell: Cell, value: number | string) {
  cell.value = value
  cell.numFmt = '0.00%'
}

export function numFormat(cell: Cell, value: number | string, numFmt?: string) {
  cell.value = value
  cell.numFmt = numFmt || '#,##0.00'
}

const colorHeader1 = 'D1FD89'
const colorHeader2 = 'F0F0F0'

export const makeHeaderLayout = (columns: IExcelColumn[], rowBaseIdx: number, headerIdxStart: number = 1, simpleHeader?: boolean) => {
  const layout: ICellLayout[] = []
  let row1ColCount = rowBaseIdx
  let row2ColCount = rowBaseIdx
  let row3ColCount = rowBaseIdx
  let headerSize = simpleHeader ? 1 : 2

  if (!simpleHeader) {
    for (const col of columns) {
      for (const sc of col.subcols || []) {
        if (sc.subcols && sc.subcols?.length > 0) { headerSize = 3; break }
      }
    }
  }

  if (simpleHeader) {
    let colIdx = rowBaseIdx
    for (const col of columns) {
      if (!col) continue
      layout.push({
        col: col,
        colStart: colIdx,
        rowStart: headerIdxStart,
        bgColor: col.bgColor || colorHeader2,
        fontColor: col.fontColor
      })
      colIdx++
    }
    return { layout, cols: colIdx }
  }

  for (let i = 0; i < columns.length; i++) {
    const e = columns[i]
    if (!e) { continue }

    if (e.subcols && e.subcols.length > 0) {
      for (const sc of e.subcols) {
        if (!sc) { continue }
        const cell: ICellLayout = {
          col: sc, colStart: row2ColCount, rowStart: headerIdxStart + 1,
        }

        if (sc.subcols && sc.subcols.length > 0) {
          for (const ssc of sc.subcols) {
            const subcell: ICellLayout = {
              col: ssc, colStart: row3ColCount, rowStart: headerIdxStart + 2,
            }
            const colspan = ssc.colspan || 1
            row3ColCount += colspan
            if (colspan > 1) { subcell.colEnd = subcell.colStart + colspan - 1 }

            subcell.bgColor = ssc.bgColor || colorHeader2
            subcell.fontColor = ssc.fontColor
            subcell.hasBorder = ssc.divLine || false

            layout.push(subcell)
          }
          cell.colEnd = (row3ColCount - 1)
          row2ColCount = row3ColCount
        } else {
          const colspan = sc.colspan || 1
          row2ColCount += colspan
          row3ColCount = row2ColCount
          if (colspan > 1) { cell.colEnd = cell.colStart + colspan - 1 }
          if (headerSize > 2) {
            cell.rowEnd = headerSize + headerIdxStart - 1
          }
        }

        cell.bgColor = sc.bgColor || colorHeader2
        cell.fontColor = sc.fontColor
        cell.hasBorder = sc.divLine || false
        layout.push(cell)
      }

      layout.push({
        col: e, colStart: row1ColCount, rowStart: headerIdxStart,
        colEnd: row2ColCount - 1, bgColor: e.bgColor || colorHeader1,
        fontColor: e.fontColor,
        hasBorder: e.subcols[0].divLine
      })
      row1ColCount = row2ColCount
    } else {
      layout.push({
        col: e, colStart: row1ColCount, rowStart: headerIdxStart,
        rowEnd: headerSize + headerIdxStart - 1,
        bgColor: e.bgColor || colorHeader2, fontColor: e.fontColor
      })
      row1ColCount++
      row2ColCount++
      row3ColCount++
    }
  }

  return { layout, cols: row2ColCount }
}

export const align1 = { vertical: 'middle', horizontal: 'center', wrapText: true } as Alignment

export const createExcelHeader = (
  sheet: Worksheet, columns: IExcelColumn[], divisions?: number[], headerIdxExport?: number, simpleHeader?: boolean
): number[] => {
  const rowIdx = headerIdxExport || 2
  const layout = makeHeaderLayout(columns, 1, rowIdx, simpleHeader)
  console.log("layout::", layout)

  for (const cellProps of layout.layout) {

    const row = sheet.getRow(cellProps.rowStart)
    const cell = row.getCell(cellProps.colStart)
    cell.value = cellProps.col.v

    fillCell(cell, cellProps.bgColor || undefined, cellProps.fontColor || undefined)
    if (cellProps.col.formatter) { cellProps.col.formatter(cell) }

    // console.log("cellProps", cellProps)
    if (cellProps.rowEnd > cellProps.rowStart || cellProps.colEnd > cellProps.colStart) {
      sheet.mergeCells(
        cellProps.rowStart, cellProps.colStart,
        cellProps.rowEnd || cellProps.rowStart, cellProps.colEnd || cellProps.colStart
      )
    }

    cell.alignment = align1
    if (divisions?.includes(cellProps.colStart)) {
      cell.border = { left: { style: 'double', color: { argb: 'FF486786' } } }
    }
  }
  return []
}
