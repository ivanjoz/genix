import { addWeeks, getISOWeek, getISOWeekYear, startOfISOWeekYear } from "date-fns";

export interface IFecSemana {
  fechaUnix: number,
  fechaInicio: Date,
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

export const dateToFechaUnix = (fecha: Date) => {
  // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
  const fechaT = (fecha.getTime() / 1000 / 60 - zoneOffset) / 60 / 24
  return Math.floor(fechaT)
}

export const semanaFromCode = (semanaCode: number): IFecSemana => {
  const sem = String(semanaCode)
  const yearString = sem.substring(0,4)
  const year = parseInt(yearString)
  // Le coloqué 10 días porque lo importante es que coincida con el año, si le coloco día 1 puede tomarlo como semana del año anterior.
  const yearDay1 = new Date(year,0,10,0,0,0)
  // Calcula el inicio del ISO week del año (no importa la fecha)
  const yearStartOfISOWeek = startOfISOWeekYear(yearDay1)
  const nroString = sem.substring(4,6)
  const nro = parseInt(nroString)
  // Se le resta una semana porque se le está agregando a la semana 1
  const fechaInicioT = addWeeks(yearStartOfISOWeek,nro - 1)
  // Agrega 12 horas para que la fecha sea el lunes a las 10:00
  const fechaInicio = new Date(fechaInicioT.getTime() + (10*60*60*1000))
  const fechaUnix = dateToFechaUnix(fechaInicio)
  const nro1 = (nro < 10 ? ('0'+String(nro)) : String(nro))
  const name = year +"-"+ nro1
  return { fechaUnix, fechaInicio, year, anio: year, nro, code: semanaCode, name } as IFecSemana
}

export class FechaHelper {
  fechaToSemanaMap: Map<number,IFecSemana> = new Map()
  semanaToFechaInicioMap: Map<number,any> = new Map()
  fechaToUnixMap: Map<string|number,number> = new Map()
  fechaUnixToIntMap: Map<number,any> = new Map()
  fechaUnixToStringMap: Map<number,any> = new Map()
  fechaUnixToYearMap: Map<number,number> = new Map()
  semanasMap: Map<number,IFecSemana> = new Map()
  anioMesToFechaUnixMap: Map<number,number> = new Map()
  fechaUnixToDayOfMonth: Map<number,number> = new Map()
  fechaUnixToDayOfWeek: Map<number,number> = new Map()
  fechaUnixToDate: Map<number,Date> = new Map()

	constructor() { }
  
	fechaUnixCurrent() {
		return this.dateToFechaUnix(new Date())
	}
	
	weekCurrent() {
		const fechaUnix = this.fechaUnixCurrent()
		return this.toWeek(fechaUnix)
	}

  toDate(fechaUnix: number): Date { // a las 12 del medio día
    if(!this.fechaUnixToDate.has(fechaUnix)){
      const fechaHoraUnix = (fechaUnix*24*60*60 + zoneOffset * 60) + (12*60*60)
      this.fechaUnixToDate.set(fechaUnix, new Date(fechaHoraUnix * 1000))
    }
    return this.fechaUnixToDate.get(fechaUnix) as Date
  }
  
  // Año de semana ISO
  toAnio(fechaUnix: number) {
    return getISOWeekYear(this.toDate(fechaUnix))
  }

  toYear(fechaUnix: number){
    if(!this.fechaUnixToYearMap.has(fechaUnix)){
      this.fechaUnixToYearMap.set(fechaUnix,  this.toDate(fechaUnix).getFullYear())
    }
    return this.fechaUnixToYearMap.get(fechaUnix)
  }

  toDayOfWeek(fechaUnix: number): IDayOfWeek {
    if(!this.fechaUnixToDayOfMonth.has(fechaUnix)){
      const day = this.toDate(fechaUnix).getDay()
      this.fechaUnixToDayOfMonth.set(fechaUnix, day)
    }
    const dayOfWeekID = this.fechaUnixToDayOfMonth.get(fechaUnix) as number
    return weekdaysMap.get(dayOfWeekID) || { short: String(dayOfWeekID), code: String(dayOfWeekID) } as IDayOfWeek
  }

  toDayOfMonth(fechaUnix: number): number {
    if(!this.fechaUnixToDayOfMonth.has(fechaUnix)){
      this.fechaUnixToDayOfMonth.set(fechaUnix, this.toDate(fechaUnix).getDate())
    }
    return this.fechaUnixToDayOfMonth.get(fechaUnix) as number
  }

  toFechaExcel(fechaUnix: number | Date): number{
    let dateX = 0
    if(fechaUnix instanceof Date){
      dateX = this.toFechaUnix(fechaUnix) || 0
    } else {
      dateX = fechaUnix
    }
    return dateX + 25569
  }

  dateToFechaUnix(fecha: Date){
    // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
    const fechaT = (fecha.getTime()/1000/60 - zoneOffset) /60/24
    return Math.floor(fechaT)
  }

  toFechaUnix(fecha: string | number | Date): number{
    if(typeof fecha === 'string'){
      fecha = fecha.substring(0,10)
			if (!this.fechaToUnixMap.has(fecha)) {
		
		    const regex1 = /[0-9]{1,2}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{4}/g
		    const regex2 = /[0-9]{4}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{1,2}/g
		    const r1 = regex1.test(fecha)
		    const r2 = r1 ? undefined : regex2.test(fecha)

				let fechaObject: Date | undefined = undefined
				
		    if (r1 || r2) {
		      let parsed
		      for (const s of ['/', '-', '.']) {
		        if (fecha.includes(s)) {
		          parsed = fecha.split(s)
		          if (r1) parsed.reverse()
							parsed = parsed.join('-') + 'T12:00:00'	
							// debugger
		          fechaObject = new Date(parsed)
		          if (!fechaObject.getTime) return 0
		        }
		      }
		    }
        // Si termina en "Z" entonces lo asume como una fecha UTC. Sin "Z" es fecha local
        // debugger
        if(!fechaObject){ fechaObject = new Date(`${fecha}T12:00:00`) }
        const fechaUnix = this.dateToFechaUnix(fechaObject)
        this.fechaToUnixMap.set(fecha,fechaUnix)
      }
      return this.fechaToUnixMap.get(fecha) as number
    } else if(fecha instanceof Date){
      return this.dateToFechaUnix(fecha)
    } else if(typeof fecha === 'number' && fecha > 20000000 && fecha < 30000000) {
      if(!this.fechaToUnixMap.has(fecha)){
        let f = String(fecha)
        const fechaObject = new Date(`${f.substring(0,4)}-${f.substring(4,6)}-${f.substring(6,8)}T12:00:00`)
        const fechaUnix = this.dateToFechaUnix(fechaObject)
        this.fechaToUnixMap.set(fecha,fechaUnix)
      }
      return this.fechaToUnixMap.get(fecha)  as number
    } else if(typeof fecha === 'number' && fecha > 30000000) {
      // window._zoneOffset es la cantidad de minutos que difieren de UTC, por ejemplo Perú es UTC-5 y posee un offset de 300 (5 * 60), es decir 300 min antes del UTC+0
      const fechaT = (fecha / 60 / 60 / 24) - (zoneOffset / 60 / 24)
      return Math.floor(fechaT)
    } else  {
      return 0
    }
  }

  toWeek(fechaUnix_: number): IFecSemana {
    if(!this.fechaToSemanaMap.has(fechaUnix_)){
      const fechaInicio = this.toDate(fechaUnix_)
      const anio = getISOWeekYear(fechaInicio)
      const nro = getISOWeek(fechaInicio)
      const semana = anio * 100 + nro
      const code = semana - 200000
      // Calcula la fecha de inicio
      const yearDay1 = new Date(anio,0,10,0,0,0)
      const yearStartOfISOWeek = startOfISOWeekYear(yearDay1)
      const fechaInicioDate = addWeeks(yearStartOfISOWeek,nro - 1)
      const fechaUnix = this.dateToFechaUnix(fechaInicioDate)
      
      const reg = { 
        id: semana, semana, anio, year: anio, nro, fechaUnix, fechaInicio, code,
        name: ""
      } as any
      this.fechaToSemanaMap.set(fechaUnix_, reg)
    }
    return this.fechaToSemanaMap.get(fechaUnix_) as IFecSemana
  }

  semanaFromCode(semana: number): IFecSemana | undefined {
    if(!this.semanasMap.has(semana)){
      this.semanasMap.set(semana, semanaFromCode(semana))
    }
    return this.semanasMap.get(semana)
  }

  addToSemana(semanaCode: number, cant: number): IFecSemana {
    const semanaInicio = this.semanaFromCode(semanaCode)
    const fechaUnix = (semanaInicio as any).fechaUnix + (cant * 7)
    const semanaFin = this.toWeek(fechaUnix)
    return semanaFin
  }

  semanaDiference(semanaInicio: number, semanaFin: number): number{
    const inicio = this.semanaFromCode(semanaInicio)
    const fin = this.semanaFromCode(semanaFin)
    return Math.round(((fin as any).fechaUnix - (inicio as any).fechaUnix)/7)
  }

  makeWeekRange(semanaInicioCode: number, semanaFinCode: number): IFecSemana[]{
    if(semanaFinCode < semanaInicioCode){
      [semanaInicioCode, semanaFinCode] = [semanaFinCode, semanaInicioCode]
    }
    const semanaInicio = this.semanaFromCode(semanaInicioCode)
    const semanaFin = this.semanaFromCode(semanaFinCode)
    const semanas = [semanaInicio] as IFecSemana[]

    const count = Math.floor(((semanaFin as any).fechaUnix - (semanaInicio as any).fechaUnix)/7)
    for(let i = 1; i < count; i++){
      const semanaFechaUnix = (semanaInicio as any).fechaUnix + (i * 7)
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

    semanas.sort((a,b) => (a as any).fechaUnix > (b as any).fechaUnix ? 1 : -1)
    return semanas
  }
}

export const fechaHelper = new FechaHelper()
