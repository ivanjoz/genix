import pkg from "notiflix";
export const { Notify, Loading, Confirm } = pkg;

export const throttle = (func: () => void, delay: number) => {
  if(window._throttleTimer){ clearTimeout(window._throttleTimer) }
  window._throttleTimer = setTimeout(() => {
    func()
    delete window._throttleTimer
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
export const formatTime = (date: Date | number | string, layout?: string): (Date | string) => {
  
  let d // Objeto fecha a parsear
  if (!date) { d = new Date() }
  else if (typeof date === "number") {
    // Valida las fechas por dia, segundo o ms;
    if(date < 30000){
      // Si es por día, le agrega 10 horas por desface GTM Perú
      date = date * 1000 * 86400 + 36000000
    }
    else if (date < 180000000000) { date = date * 1000 }
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
  a: string, b: string, c: string, d?: string,
  e?: () => void, f?: () => void,
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