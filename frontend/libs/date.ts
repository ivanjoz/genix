import { addWeeks, getISOWeek, getISOWeekYear, startOfISOWeekYear } from "date-fns";

export interface IFecSemana {
  dateUnix: number,
  dateInicio: Date,
  year: number,
  anio: number,
  nro: number,
  code: number,
  name: string,
  i: number,
}

export interface IDayOfWeek {
  long: string, short: string, code: string
}

export const weekdaysMap: Map<number,IDayOfWeek> = new Map([
  [0,{ long: "Domingo", short: 'DO', code: "D" }],
  [1,{ long: "Lunes", short: 'LU', code: "L" }],     
  [2,{ long: "Martes", short: 'MA', code: "M" }],
  [3,{ long: "Miércoles", short: 'MI', code: "X" }], 
  [4,{ long: "Jueves", short: 'JU', code: "J" }],
  [5,{ long: "Viernes", short: 'VI', code: "V" }],   
  [6,{ long: "Sabado", short: 'SA', code: "S" }],
  [7,{ long: "Domingo", short: 'DO', code: "D" }],
])

export const zoneOffset = (new Date()).getTimezoneOffset()

export const dateToFechaUnix = (date: Date) => {
  // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
  const dateT = (date.getTime() / 1000 / 60 - zoneOffset) / 60 / 24
  return Math.floor(dateT)
}

export const semanaFromCode = (semanaCode: number): IFecSemana => {
  const sem = String(semanaCode)
  const yearString = sem.substring(0,4)
  const year = parseInt(yearString)
  // Le coloqué 10 días porque lo importante es que coincida con el año, si le coloco día 1 puede tomarlo como semana del año anterior.
  const yearDay1 = new Date(year,0,10,0,0,0)
  // Calcula el inicio del ISO week del año (no importa la date)
  const yearStartOfISOWeek = startOfISOWeekYear(yearDay1)
  const nroString = sem.substring(4,6)
  const nro = parseInt(nroString)
  // Se le resta una semana porque se le está agregando a la semana 1
  const dateInicioT = addWeeks(yearStartOfISOWeek,nro - 1)
  // Agrega 12 horas para que la date sea el lunes a las 10:00
  const dateInicio = new Date(dateInicioT.getTime() + (10*60*60*1000))
  const dateUnix = dateToFechaUnix(dateInicio)
  const nro1 = (nro < 10 ? ('0'+String(nro)) : String(nro))
  const name = year +"-"+ nro1
  return { dateUnix, dateInicio, year, anio: year, nro, code: semanaCode, name } as IFecSemana
}

export class DateHelper {
  dateToSemanaMap: Map<number,IFecSemana> = new Map()
  semanaToFechaInicioMap: Map<number,any> = new Map()
  dateToUnixMap: Map<string|number,number> = new Map()
  dateUnixToIntMap: Map<number,any> = new Map()
  dateUnixToStringMap: Map<number,any> = new Map()
  dateUnixToYearMap: Map<number,number> = new Map()
  semanasMap: Map<number,IFecSemana> = new Map()
  anioMesToFechaUnixMap: Map<number,number> = new Map()
  dateUnixToDayOfMonth: Map<number,number> = new Map()
  dateUnixToDayOfWeek: Map<number,number> = new Map()
  dateUnixToDate: Map<number,Date> = new Map()

	constructor() { }
  
	dateUnixCurrent() {
		return this.dateToFechaUnix(new Date())
	}
	
	weekCurrent() {
		const dateUnix = this.dateUnixCurrent()
		return this.toWeek(dateUnix)
	}

  toDate(dateUnix: number): Date { // a las 12 del medio día
    if(!this.dateUnixToDate.has(dateUnix)){
      const dateTimeUnix = (dateUnix*24*60*60 + zoneOffset * 60) + (12*60*60)
      this.dateUnixToDate.set(dateUnix, new Date(dateTimeUnix * 1000))
    }
    return this.dateUnixToDate.get(dateUnix) as Date
  }
  
  // Año de semana ISO
  toAnio(dateUnix: number) {
    return getISOWeekYear(this.toDate(dateUnix))
  }

  toYear(dateUnix: number){
    if(!this.dateUnixToYearMap.has(dateUnix)){
      this.dateUnixToYearMap.set(dateUnix,  this.toDate(dateUnix).getFullYear())
    }
    return this.dateUnixToYearMap.get(dateUnix)
  }

  toDayOfWeek(dateUnix: number): IDayOfWeek {
    if(!this.dateUnixToDayOfMonth.has(dateUnix)){
      const day = this.toDate(dateUnix).getDay()
      this.dateUnixToDayOfMonth.set(dateUnix, day)
    }
    const dayOfWeekID = this.dateUnixToDayOfMonth.get(dateUnix) as number
    return weekdaysMap.get(dayOfWeekID) || { short: String(dayOfWeekID), code: String(dayOfWeekID) } as IDayOfWeek
  }

  toDayOfMonth(dateUnix: number): number {
    if(!this.dateUnixToDayOfMonth.has(dateUnix)){
      this.dateUnixToDayOfMonth.set(dateUnix, this.toDate(dateUnix).getDate())
    }
    return this.dateUnixToDayOfMonth.get(dateUnix) as number
  }

  toFechaExcel(dateUnix: number | Date): number{
    let dateX = 0
    if(dateUnix instanceof Date){
      dateX = this.toFechaUnix(dateUnix) || 0
    } else {
      dateX = dateUnix
    }
    return dateX + 25569
  }

  dateToFechaUnix(date: Date){
    // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
    const dateT = (date.getTime()/1000/60 - zoneOffset) /60/24
    return Math.floor(dateT)
  }

  toFechaUnix(date: string | number | Date): number{
    if(typeof date === 'string'){
      date = date.substring(0,10)
			if (!this.dateToUnixMap.has(date)) {
		
		    const regex1 = /[0-9]{1,2}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{4}/g
		    const regex2 = /[0-9]{4}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{1,2}/g
		    const r1 = regex1.test(date)
		    const r2 = r1 ? undefined : regex2.test(date)

				let dateObject: Date | undefined = undefined
				
		    if (r1 || r2) {
		      let parsed
		      for (const s of ['/', '-', '.']) {
		        if (date.includes(s)) {
		          parsed = date.split(s)
		          if (r1) parsed.reverse()
							parsed = parsed.join('-') + 'T12:00:00'	
							// debugger
		          dateObject = new Date(parsed)
		          if (!dateObject.getTime) return 0
		        }
		      }
		    }
        // Si termina en "Z" entonces lo asume como una date UTC. Sin "Z" es date local
        // debugger
        if(!dateObject){ dateObject = new Date(`${date}T12:00:00`) }
        const dateUnix = this.dateToFechaUnix(dateObject)
        this.dateToUnixMap.set(date,dateUnix)
      }
      return this.dateToUnixMap.get(date) as number
    } else if(date instanceof Date){
      return this.dateToFechaUnix(date)
    } else if(typeof date === 'number' && date > 20000000 && date < 30000000) {
      if(!this.dateToUnixMap.has(date)){
        let f = String(date)
        const dateObject = new Date(`${f.substring(0,4)}-${f.substring(4,6)}-${f.substring(6,8)}T12:00:00`)
        const dateUnix = this.dateToFechaUnix(dateObject)
        this.dateToUnixMap.set(date,dateUnix)
      }
      return this.dateToUnixMap.get(date)  as number
    } else if(typeof date === 'number' && date > 30000000) {
      // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
      const dateT = (date / 60 / 60 / 24) - (zoneOffset / 60 / 24)
      return Math.floor(dateT)
    } else  {
      return 0
    }
  }

  toWeek(dateUnix_: number): IFecSemana {
    if(!this.dateToSemanaMap.has(dateUnix_)){
      const dateInicio = this.toDate(dateUnix_)
      const anio = getISOWeekYear(dateInicio)
      const nro = getISOWeek(dateInicio)
      const semana = anio * 100 + nro
      const code = semana - 200000
      // Calcula la date de inicio
      const yearDay1 = new Date(anio,0,10,0,0,0)
      const yearStartOfISOWeek = startOfISOWeekYear(yearDay1)
      const dateInicioDate = addWeeks(yearStartOfISOWeek,nro - 1)
      const dateUnix = this.dateToFechaUnix(dateInicioDate)
      
      const reg = { 
        id: semana, semana, anio, year: anio, nro, dateUnix, dateInicio, code,
        name: ""
      } as any
      this.dateToSemanaMap.set(dateUnix_, reg)
    }
    return this.dateToSemanaMap.get(dateUnix_) as IFecSemana
  }

  semanaFromCode(semana: number): IFecSemana | undefined {
    if(!this.semanasMap.has(semana)){
      this.semanasMap.set(semana, semanaFromCode(semana))
    }
    return this.semanasMap.get(semana)
  }

  addToSemana(semanaCode: number, cant: number): IFecSemana {
    const semanaInicio = this.semanaFromCode(semanaCode)
    const dateUnix = (semanaInicio as any).dateUnix + (cant * 7)
    const semanaFin = this.toWeek(dateUnix)
    return semanaFin
  }

  semanaDiference(semanaInicio: number, semanaFin: number): number{
    const inicio = this.semanaFromCode(semanaInicio)
    const fin = this.semanaFromCode(semanaFin)
    return Math.round(((fin as any).dateUnix - (inicio as any).dateUnix)/7)
  }

  makeWeekRange(semanaInicioCode: number, semanaFinCode: number): IFecSemana[]{
    if(semanaFinCode < semanaInicioCode){
      [semanaInicioCode, semanaFinCode] = [semanaFinCode, semanaInicioCode]
    }
    const semanaInicio = this.semanaFromCode(semanaInicioCode)
    const semanaFin = this.semanaFromCode(semanaFinCode)
    const semanas = [semanaInicio] as IFecSemana[]

    const count = Math.floor(((semanaFin as any).dateUnix - (semanaInicio as any).dateUnix)/7)
    for(let i = 1; i < count; i++){
      const semanaFechaUnix = (semanaInicio as any).dateUnix + (i * 7)
      semanas.push(this.toWeek(semanaFechaUnix))
    }
    semanas.push(semanaFin as any)
    return semanas
  }

  makeWeekRangeFromWeek(semanaBase: number, increment: number, decrement?: number): IFecSemana[]{
    if(!semanaBase){
      semanaBase = this.toWeek(this.dateToFechaUnix(new Date())).code
    }
    increment = Math.abs(increment||0)
    decrement = Math.abs(decrement||0)

    const semanas = [] as IFecSemana[]
    if(decrement > 0){
      for(let i = 0; i < decrement; i++){
        semanas.push(this.addToSemana(semanaBase, i*-1))
      }
    }
    if(increment > 0){
      for(let i = 0; i < increment; i++){
        semanas.push(this.addToSemana(semanaBase, i))
      }
    }

    semanas.sort((a,b) => (a as any).dateUnix > (b as any).dateUnix ? 1 : -1)
    return semanas
  }
}

export const dateHelper = new DateHelper()
