import pkg from "notiflix";
import { Env } from "~/env";
export const { Notify, Loading, Confirm } = pkg;

export const fechaUnixToSunix = (fechaUnix: number) => {
  if(!fechaUnix){ return 0 }
  const fechaHoraUnix = fechaUnix*24*60*60 + (Env.zoneOffset||0)
  const fechaSunix = Math.floor((fechaHoraUnix - (10**9)) / 2)
  return fechaSunix
}

export const throttle = (func: (() => void), delay: number) => {
  if(Env.throttleTimer){ clearTimeout(Env.throttleTimer) }
  Env.throttleTimer = setTimeout(() => {
    func()
    Env.throttleTimer = null
  }, delay)
}

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

export const ConfirmWarn = (
  a: string, b: string, c: string, d?: string, e?: () => void, f?: () => void,
) =>{
  Confirm.init({
    fontFamily:'main',
    messageFontSize:'0.96rem',
    titleColor:'#db3030',
    titleFontSize:'1.06rem',
    messageColor:'#1e1e1e',
    okButtonColor:'#f8f8f8',
    okButtonBackground:'#f35c5c',
  })
  Confirm.show(a,b,c,d,e,f)
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