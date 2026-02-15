
/* Excel builder con Go Excelize */
export class ExcelBuilder {

  constructor(e: ExcelHelperProps) {
    this.props = e
    const headerIdx = typeof this.props.headerIdx === 'number'
      ? this.props.headerIdx - 1
      : (this.props.headerIdx || [2])[0] - 1

    this.sheetMap.set(this.sheetCurrentIdx, {
      title: e.sheetTitle, name: e.sheetName, columns: e.columns, headerIdx
    } as ExcelSheet)
  }

  props: ExcelHelperProps
  workbook!: XWorkbook
  sheet!: XWorksheet
  columIdxWidthMap: Map<number, number> = new Map()
  columIdxNameMap: Map<number, string> = new Map()
  columnsFlat: IExcelColumn[] = []
  rowsStart!: number
  sheetMap: Map<number, ExcelSheet> = new Map()
  sheetCurrentIdx = 1

  createHeaders(columns: IExcelColumn[], title: string) {

    this.props.headerIdx = this.props.headerIdx || [2, 3]

    const headerIdx = typeof this.props.headerIdx === 'number'
      ? this.props.headerIdx - 1
      : this.props.headerIdx[0] - 1

    this.rowsStart = typeof this.props.headerIdx === 'number'
      ? this.props.headerIdx
      : this.props.headerIdx[this.props.headerIdx.length - 1]

    const layout = makeHeaderLayout(columns, 0, headerIdx)
    // console.log("layout_1:", layout)

    for (const cell of layout.layout) {
      const ff = new XFormat()
      ff.setTextWrap()
      ff.setAlign(XFormatAlign.VerticalCenter)
      ff.setAlign(XFormatAlign.Center)

      if (cell.bgColor) { ff.setBackgroundColor(Color.parse("#" + cell.bgColor)) }
      if (cell.fontColor) { ff.setFontColor(Color.parse("#" + cell.fontColor)) }

      if ((cell.rowEnd && cell.rowEnd > cell.rowStart) || (cell.colEnd && cell.colEnd > cell.colStart)) {
        this.sheet.mergeRange(
          cell.rowStart, cell.colStart,
          cell.rowEnd || cell.rowStart, cell.colEnd || cell.colStart,
          cell.col.v || "???", ff
        )
      } else {
        this.sheet.writeWithFormat(cell.rowStart, cell.colStart, cell.col.v || "", ff)
      }
    }

    // Hace el título
    const ff = new XFormat()
    ff.setBackgroundColor(Color.parse("#" + '1E6096'))
    ff.setFontColor(Color.parse("#" + 'FFFFFF'))
    ff.setAlign(XFormatAlign.Center)
    ff.setAlign(XFormatAlign.VerticalCenter)
    this.sheet.mergeRange(0, 0, 0, layout.cols, title, ff)
  }

  createNewSheet(e: IWorkProps) {
    const clone = { ...this.sheetMap.get(1) }
    this.sheetCurrentIdx++
    clone.name = e.name
    clone.title = e.title
    clone.columns = e.columns || clone.columns
    this.sheetMap.set(this.sheetCurrentIdx, clone as ExcelSheet)
  }

  fillRows(records: any[], maxWidth?: number, widthFactor?: number) {
    const sheet = this.sheetMap.get(this.sheetCurrentIdx)!
    const columnsFlat = makeColumnsFlat(sheet.columns, 1, new Map())

    sheet.values = []
    sheet.valuesParsed = []
    sheet.maxWidth = maxWidth
    sheet.widthFactor = widthFactor

    const showTitle = !(this.props.headerIdxExport == 1)
    let rowN = showTitle ? 4 : 3

    for (const record of records) {
      const values: any[] = []
      for (const col of columnsFlat) {
        const value = extractValue(col, record, true)
        if (this.props.useWasm) {
          values.push(value)
        } else {
          const valueObject = { v: value } as { v: (number | string), f: string }
          if (col.setCellProperties) {
            const cell = {} as Cell
            col.setCellProperties(record, cell, col.idx || 0, rowN)
            const cellValue = cell.value as CellFormulaValue
            if (cellValue.formula) {
              valueObject.v = cellValue.result as (number | string)
              valueObject.f = cellValue.formula
            }
          }
          values.push(valueObject)
        }
      }
      if (this.props.useWasm) {
        sheet.values?.push(values)
      } else {
        sheet.valuesParsed?.push(values)
      }
      rowN++
    }
  }

  generateExcelBuffer(): Promise<Uint8Array> {
    const sheets = [...this.sheetMap.values()]
    console.log("sheets to create::", sheets)

    const valListMap: Map<string, { id: number, values: string[] }> = new Map()

    for (const e of sheets) {
      for (const col of makeColumnsFlat(e.columns, 0)) {
        if (!col.getList) { continue }
        if (!valListMap.has(col.key || '')) {
          const values = col.getList()!.filter(x => typeof x === 'string')
          const id = valListMap.size + 1
          valListMap.set(col.key || '', { values, id })
        }
        col._validationListID = valListMap.get(col.key || '')!.id
      }
    }

    for (const e of sheets) {
      e.columns = JSON.parse(JSON.stringify(e.columns))
    }

    return new Promise((resolve, reject) => {
      const accion = this.props.useWasm ? 1 : 2
      const excelWorker = window.excelWorker
      
      console.log("---------------------------------------------------------");
      console.log("DEBUG: generateExcelBuffer started");
      console.log("DEBUG: Checking window.excelWorker:", excelWorker);
      
      if (!excelWorker) {
        console.error("DEBUG: Excel worker not found in window object!")
        reject("Worker not found")
        return
      }

      console.log("DEBUG: excelWorker found. Preparing content...");

      const excelContent = {
        accion, sheets, validationLists: [...valListMap.values()],
        creator: this.props.creator || ""
      }
      console.log("DEBUG: excelContent prepared:", excelContent);
      console.log("DEBUG: Posting message to worker...");

      try {
          excelWorker.postMessage(excelContent)
          console.log("DEBUG: Message posted successfully.");
      } catch (e) {
          console.error("DEBUG: Failed to post message to worker:", e);
          reject(e);
          return;
      }

      console.log("DEBUG: Setting up onmessage handler...");
      excelWorker.onmessage = (event: MessageEvent<any>) => {
        console.log("DEBUG: Worker message received:", event.data);
        if (event.data.fileBuffer) {          
          console.log("DEBUG: fileBuffer received, resolving promise.");
          resolve(event.data.fileBuffer)
        } else if (event.data.advance) {
          console.log("DEBUG: Progress update:", event.data.advance);
          if (event.data.advance === 1) {
            Loading.change(`Generando archivo a descarga...`)
          } else {
            Loading.change(`Procesando ${Math.round(event.data.advance * 85)}%...`)
          }
        } else if (event.data.message) {
          console.log("DEBUG: Message update:", event.data.message);
          Loading.change(event.data.message)
        } else {
          console.log("DEBUG: Unrecognized message:", event.data)
        }
      }

      console.log("DEBUG: Setting up onerror handler...");
      excelWorker.onerror = (error: any) => {
        console.error('DEBUG: Worker error:', error)
        reject(error)
      }
    })
  }

  async download(fileName: string) {
    const buf = await this.generateExcelBuffer()
    const blob = new Blob([buf as any], { type: blobType })
    // console.log('Guardando archivo::')
    saveAs(blob, fileName)
  }

  drawDivisions() {
    // No hace nada, sólo para compatibilidad con ExcelHelper
  }
}

export class Formula {
  static ceroBlack(formula: string, mayorQue?: number) {
    if (mayorQue) { return `IF(${formula}>${mayorQue},"",${formula})` }
    return `IF(${formula}=0,"",${formula})`
  }
  static safeNumber(colN: number, rowN: number) {
    return `IF(ISNUMBER(${getLetter(colN)}${rowN}),${getLetter(colN)}${rowN},0)`
  }
  static row(colN: number) {
    const colL = getLetter(colN)
    return `INDEX(${colL}:${colL}; ROW())`
  }
  static number(colN: number) {
    const colL = getLetter(colN)
    return `N(INDEX(${colL}:${colL}, ROW()))`
  }
  // The coln must be number, anything alse is string
  static make(...statemnts: (string|number)[]) {
    const formula = []
    for(const e of statemnts){
      if(typeof e === "string"){
        formula.push(e)
      } else {
        formula.push(Formula.number(e))
      }
    }
    return "=" + formula.join(" ")
  }
}

// EIT-41899: Parse DD-MM-YYYY or DD/MM/YYYY format strings
function parseDateDMY(dateStr: string): Date | null {
  // Match DD-MM-YYYY or DD/MM/YYYY (with optional time)
  const match = dateStr.match(/^(\d{1,2})[-/](\d{1,2})[-/](\d{4})(?:\s+(\d{1,2}):(\d{2})(?::(\d{2}))?)?$/)
  if (match) {
    const [, day, month, year, hours, minutes, seconds] = match
    const d = new Date(
      parseInt(year),
      parseInt(month) - 1, // Month is 0-indexed
      parseInt(day),
      hours ? parseInt(hours) : 0,
      minutes ? parseInt(minutes) : 0,
      seconds ? parseInt(seconds) : 0
    )
    // Validate the date is valid (e.g., not 31-02-2026)
    if (!isNaN(d.getTime()) && d.getDate() === parseInt(day)) {
      return d
    }
  }
  return null
}

export function fechaExcelToUnix(value: any): [number, string | null] {
  if (!value) return [0, null]

  // Si es número (días Excel o timestamp Unix)
  if (typeof value === 'number') {
    // Heurística: si es mayor a 1.000.000 asumimos que ya es un timestamp unix
    const fechaUnix = value > 1000000 ? value : value - 25569
    return [fechaUnix, null]
  }

  // Si es string: "DD-MM-YYYY", "DD/MM/YYYY", ISO, etc.
  if (typeof value === 'string') {
    // Intentar convertir a número primero (por si viene como "45678")
    const valN = Number(value)
    if (!isNaN(valN)) {
      const fechaUnix = valN > 1000000 ? valN : valN - 25569
      return [fechaUnix, null]
    }

    // EIT-41899: Try DD-MM-YYYY or DD/MM/YYYY format first
    const parsedDMY = parseDateDMY(value)
    if (parsedDMY) {
      return [Math.floor(parsedDMY.getTime() / 1000 / 86400), null]
    }

    // Si no es número, intentar parsear como fecha string (ISO, etc.)
    const fechaD = formatTime(value, -1) as Date
    if (!fechaD?.getTime || isNaN(fechaD.getTime())) {
      return [0, `El valor "${value}" no se pudo interpretar como fecha.`]
    }
    // Convertir Date a Unix days (fecha sin hora)
    return [Math.floor(fechaD.getTime() / 1000 / 86400), null]
  }

  // Si es Date object
  if (value instanceof Date) {
    return [Math.floor(value.getTime() / 1000 / 86400), null]
  }

  return [0, `Tipo de valor no soportado: ${typeof value}`]
}
