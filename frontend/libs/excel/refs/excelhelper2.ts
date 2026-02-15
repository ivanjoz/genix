// Utility functions
export const charLetters = [
  '', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z']

export interface IValidationList {
  id: number, values: string[], formulae?: string
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

export interface ExcelAccion {
  accion: number, sheets: ExcelSheet[], creator?: string,
  validationLists: IValidationList[]
}

export interface Alignment {
	horizontal: 'left' | 'center' | 'right' | 'fill' | 'justify' | 'centerContinuous' | 'distributed';
	vertical: 'top' | 'middle' | 'bottom' | 'distributed' | 'justify';
	wrapText: boolean;
	shrinkToFit: boolean;
	indent: number;
	readingOrder: 'rtl' | 'ltr';
	textRotation: number | 'vertical';
}

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
  setCellProperties?: (record: any, cell: any, colN: number, rowN: number) => void
  formatter?: (cell: any, e?: any) => void
  getList?: () => string[]
  _validationListID?: number
  background?: string
  color?: string
  parse?: (value: any, saveError: (e: string) => void, obj: any) => any
  name?: string
  setValue?: (e: any) => any
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
