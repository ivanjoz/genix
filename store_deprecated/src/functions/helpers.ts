/*
  Y (año) = 2023, y = 23
  m (mes) = 01, M = JUL
  d (dia) = 01, D = ?
  h (horas)
  n (minutos) = 20
  s (segundos)
*/

const mesesMap = new Map([
  ['01', { es: 'ENE', en: 'JAN' } ],
  ['02', { es: 'FEB', en: 'FEB' } ],
  ['03', { es: 'MAR', en: 'MAR' } ],
  ['04', { es: 'ABR', en: 'APR' } ],
  ['05', { es: 'MAY', en: 'MAY' } ],
  ['06', { es: 'JUN', en: 'JUN' } ],
  ['07', { es: 'JUL', en: 'JUL' } ],
  ['08', { es: 'AGO', en: 'AGO' } ],
  ['09', { es: 'SEP', en: 'SEP' } ],
  ['10', { es: 'OCT', en: 'OCT' } ],
  ['11', { es: 'NOV', en: 'NOV' } ],
  ['12', { es: 'DIC', en: 'DEC' } ],
])

export const formatTime = (date: Date | number | string, layout?: string): (Date | string) => {
  
  let d // Objeto fecha a parsear
  if (!date) { d = new Date() }
  else if (typeof date === "number") {
    // Valida las fechas por dia, segundo o ms;
    if(date < 30000){
      // Si es por día, le agrega 10 horas por desface GTM Perú
      date = date * 1000 * 86400 + 36000000
    } else if (date < 800000000) { //SunixTime
      date = (10 ** 9) + (date * 2)
    }
    if (date < 180000000000) { date = date * 1000 }
    d = new Date(date)
  }
  else if (typeof date === 'object' && date.constructor === Date) {
    d = date
  }
  else if (typeof date === 'string' && date.length === 8) {
    const year = parseInt(date.substring(0, 4))
    const month = parseInt(date.substring(4, 6)) - 1
    const day = parseInt(date.substring(6, 8))
    d = new Date(year, month, day)
  }
  else if (typeof date === 'string') {
    if (date.includes('T')) date = date.replace('T', ' ')
    if (date.includes('Z') && date.includes('.')) {
      const idx1 = date.lastIndexOf('.')
      date = date.substring(0, idx1)
    }
    const portions = date.split(' ')
    let day = portions[0]

    const regex1 = /[0-9]{1,2}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{4}/g
    const regex2 = /[0-9]{4}(\.|-|\/)[0-9]{1,2}(\.|-|\/)[0-9]{1,2}/g
    const r1 = regex1.test(day)
    const r2 = r1 ? undefined : regex2.test(day)

    if (r1 || r2) {
      let parsed
      for (let s of ['/', '-', '.']) {
        if (day.includes(s)) {
          parsed = day.split(s)
          if (r1) parsed.reverse()
          parsed = parsed.join('-')
          parsed += 'T' + (portions[1] || '12:00:00')      
          d = new Date(parsed)
          if (!d.getTime) return
        }
      }
    }
    else { return }
  }

  // Revisa si es una fecha valida
  if (!d || !(d instanceof Date) || !d.getTime) return layout ? "" : null
  // Obtiene el Día
  const _dia = d.getDate()
  if (isNaN(_dia)) return !layout ? null : ""
  const dia = _dia < 10 ? "0" + _dia : String(_dia)
  // Obtiene el mes
  const _mes = d.getMonth() + 1
  const mes = _mes < 10 ? "0" + _mes : String(_mes)
  // Obtiene el año
  const year = String(d.getFullYear())
  
  // Si la opción es -1 devuelve la fecha
  if (!layout) { return d }

  let fechaStr = ""
  for(let sec of layout){
    switch(sec) {
      case "y":
        fechaStr += year.substring(2,4); break
      case "Y":
        fechaStr += year; break
      case "m":
        fechaStr += mes; break
      case "M":
        fechaStr += mesesMap.get(mes)?.es || "?"; break
      case "d":
        fechaStr += dia; break
      case "h":
        let hora: string | number = d.getHours()
        if (hora < 10) hora = "0" + hora
        fechaStr += hora; break
      case "n":
        let min: string | number = d.getMinutes()
        if (min < 10) min = "0" + min
        fechaStr += min; break
      default:
        fechaStr += sec
    }
  }

  return fechaStr
}

export function include(e: string, h: string | string[]) {
  if (h && typeof h === 'string') {
    h = h.split(' ').filter(x => x.length > 0)
  }

  if (!h || h === "undefined" || h.length === 0) {
    return true;
  } else if (h.length === 1) {
    return e.includes(h[0])
  } else if (h.length === 2) {
    return e.includes(h[0]) && e.includes(h[1])
  } else if (h.length === 3) {
    return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2])
  } else if (h.length === 4) {
    return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2])
      && e.includes(h[3])
  } else if (h.length === 5) {
    return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2])
      && e.includes(h[3]) && e.includes(h[4])
  } else {
    return e.includes(h[0]) && e.includes(h[1])
      && e.includes(h[2]) && e.includes(h[3]) && e.includes(h[4])
      && e.includes(h[5])
  }
}

export const downloadFile = (url?: string) =>{
  const aElement = document.createElement("a")
  const names = url.split("/").filter(x => x)
  const name = names[names.length - 1]
  aElement.setAttribute("download", name)
  /*
  const href = URL.createObjectURL(res);
  console.log(href);
  */
  aElement.href = url // href;
  aElement.setAttribute("target", "_blank")
  aElement.click()
  aElement.remove()
  // URL.revokeObjectURL(href);
}

export const parseSVG = (svgContent: string)=> {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgContent)}`
}

export const cn = (...classNames: (string|boolean)[]) => {
  return classNames.filter(x => x).join(" ")
}

export const arrayToMapS = <T>(array: T[], keys?: keyof T | (keyof T)[]):
  Map<string, T> => {
  const map = new Map()
  if (typeof keys === 'string') { 
    for (const e of array) { map.set(e[keys as keyof T], e) } 
  } else if (Array.isArray(keys)) {
    for (const e of array) {
      const keyGrouped = keys.map(key => (e[key as keyof T] || "0")).join("_")
      map.set(keyGrouped, e)
    }
  }
  else { console.warn('No es un array::', array) }
  return map
}

export const arrayToMapN = <T>(array: T[], keys?: keyof T | (keyof T)[]):
  Map<number, T> => {
  const map = new Map()
  if (typeof keys === 'string') { 
    for (const e of array) { map.set(e[keys  as keyof T], e) } 
  } else if (Array.isArray(keys)) {
    for (const e of array) {
      const keyGrouped = keys.map(key => (e[key as keyof T] || "")).join("_")
      map.set(keyGrouped, e)
    }
  }
  else { console.warn('No es un array::', array) }
  return map
}

let throttleTimer: NodeJS.Timeout

export const throttle = (func: (() => void), delay: number) => {
  if(throttleTimer){ clearTimeout(throttleTimer) }
  throttleTimer = setTimeout(() => {
    func()
    throttleTimer = null
  }, delay)
}

export const highlString = (phrase: string, words: string[]): { text: string, highl?: boolean }[] => {
  if(typeof phrase !== 'string'){
    console.error("no es string")
    console.log(phrase)
    return [{ text: "!" }]
  }
  const arr: { text: string, highl?: boolean }[] = [{ text: phrase }]
  if (!words || words.length === 0) return arr

  for (let word of words) {
    if (word.length < 2) continue
    for (let i = 0; i < arr.length; i++) {
      const str = arr[i].text
      if (typeof str !== 'string') continue
      const idx = str.toLowerCase().indexOf(word)
      if (idx !== -1) {
        const ini = str.slice(0, idx)
        const middle = str.slice(idx, idx + word.length)
        const fin = str.slice(idx + word.length)
        arr.splice(i, 1, { text: ini }, { text: middle, highl: true }, { text: fin })
        if(arr.length > 40){ return arr.filter(x => x) }
        continue
      }
    }
  }
  return arr.filter(x => x)
}

export function formatN(
  x: number, decimal?: number, fixedLen?: number, charF?: string
){
  decimal = decimal || 0
  if (typeof x !== 'number') return x ? '-' : ''
  
  if(decimal === -1){
    if(x < 1) x = Math.round(x*10000)/10000  
    else if(x < 10) x = Math.round(x*1000)/1000
    else if(x >= 10) x = Math.round(x*100)/ 100
  }
  
  let xString
  if(typeof decimal === 'number' && decimal >= 0){
    if(decimal === 0){
      xString = Math.round(x).toString()
    } else {
      const pow = Math.pow(10, decimal)
      xString = (Math.round(x * pow) / pow).toFixed(decimal)
    }
  }
  else xString = x.toString()
  if(x >= 100) xString = xString.replace(/\B(?=(\d{3})+(?!\d))/g, ",")
  if(fixedLen){
    charF = charF || ' '
    while (xString.length < fixedLen) { xString = charF + xString }
  }
  return xString
}