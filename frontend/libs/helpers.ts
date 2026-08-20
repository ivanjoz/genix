import pkg from 'notiflix';
export const { Notify, Loading, Confirm } = pkg;

let throttleTimer: NodeJS.Timeout | null

if(typeof window !== 'undefined'){
  Loading.init({ zindex: 400 })
}

export const ConfirmWarn = (
  a: string, b: string, c: string, d?: string, e?: () => void, f?: () => void,
) =>{
  Confirm.init({
    fontFamily:'main',
    messageFontSize:'15px',
    titleColor:'#db3030',
    titleFontSize:'18px',
    messageColor:'#1e1e1e',
    okButtonColor:'#f8f8f8',
    okButtonBackground:'#f35c5c',
  })
  Confirm.show(a,b,c,d,e,f)
}

export const throttle = (func: (() => void), delay: number) => {
  if(throttleTimer){ clearTimeout(throttleTimer) }
  throttleTimer = setTimeout(() => {
    func()
    throttleTimer = null
  }, delay)
}

export const highlString = (
  phrase: string, words: string[]
): { text: string, highl?: boolean, isEnd?: boolean }[] => {
  // console.log("words 333:",phrase,words)

  if(typeof phrase !== 'string'){
    console.error("no es string")
    console.log(phrase)
    return [{ text: "!" }]
  }
  const arr: { text: string, highl?: boolean, isEnd?: boolean }[] = [{ text: phrase }]
  if (!words || words.length === 0){ return arr }
  // console.log("words 222:", arr.filter(x => x),"|",phrase,words)

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

        const splited = [
          { text: ini }, { text: middle, highl: true }, { text: fin }
        ].filter(x => x.text);

        arr.splice(i, 1, ...splited)
        if(arr.length > 40){
          // console.log("words 111:", arr.filter(x => x),"|",phrase,words)
          return arr.filter(x => x)
        }
        continue
      }
    }
  }
  // console.log("words 111:", arr.filter(x => x),"|",phrase,words)
  return arr.filter(x => x)
}

export const parseSVG = (svgContent: string)=> {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgContent)}`
}

export const cn = (...classNames: (string|boolean)[]) => {
  return classNames.filter(x => x).join(" ")
}

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const base62IndexesByCharCode = new Int16Array(123)

export const decodeFromBase62 = (token: string): number => {
	if (base62IndexesByCharCode[0] === 0) {
		base62IndexesByCharCode.fill(-1)
		
		for (let currentIndex = 0; currentIndex < base62Alphabet.length; currentIndex++) {
		  // Precompute base62 indexes by ASCII code to avoid scanning the alphabet on every decode.
		  const currentCharCode = base62Alphabet.charCodeAt(currentIndex)
		  base62IndexesByCharCode[currentCharCode] = currentIndex
		}
	}
	
  // Keep this logic aligned with backend/db/helpers.go::decodeFromBase62.
  let decodedNumber = 0
  const alphabetLength = base62Alphabet.length

  for (let currentIndex = 0; currentIndex < token.length; currentIndex++) {
    const currentCharCode = token.charCodeAt(currentIndex)
    const charIndex = base62IndexesByCharCode[currentCharCode]
    decodedNumber = decodedNumber * alphabetLength + charIndex
  }

  return decodedNumber
}

export function wordInclude(e: string, h: string | string[]) {
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

const mesesMap = new Map([
  ['01', { es: 'ENE', en: 'JAN' }],
  ['02', { es: 'FEB', en: 'FEB' }],
  ['03', { es: 'MAR', en: 'MAR' }],
  ['04', { es: 'ABR', en: 'APR' }],
  ['05', { es: 'MAY', en: 'MAY' }],
  ['06', { es: 'JUN', en: 'JUN' }],
  ['07', { es: 'JUL', en: 'JUL' }],
  ['08', { es: 'AGO', en: 'AUG' }],
  ['09', { es: 'SEP', en: 'SEP' }],
  ['10', { es: 'OCT', en: 'OCT' }],
  ['11', { es: 'NOV', en: 'NOV' }],
  ['12', { es: 'DIC', en: 'DEC' }],
])

export const formatTime = (date: Date | number | string, layout?: string): (Date | string | null) => {
  let d: Date | undefined

  if (!date) {
    d = new Date()
  }
  else if (typeof date === "number") {
    // Valida las dates por dia, segundo o ms
    if (date < 30000) {
      // Si es por día, le agrega 10 horas por desfase GTM Perú
      date = date * 1000 * 86400 + 36000000
    } else if (date < 800000000) {
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
      for (const s of ['/', '-', '.']) {
        if (day.includes(s)) {
          let parsed = day.split(s)
          if (r1) parsed.reverse()
          const parsedStr = parsed.join('-') + 'T' + (portions[1] || '12:00:00')
          d = new Date(parsedStr)
          if (!d.getTime) return null
        }
      }
    } else {
      return null
    }
  }

  // Revisa si es una date válida
  if (!d || !(d instanceof Date) || !d.getTime) return layout ? "" : null

  const _dia = d.getDate()
  if (isNaN(_dia)) return !layout ? null : ""
  const dia = _dia < 10 ? "0" + _dia : String(_dia)

  const _mes = d.getMonth() + 1
  const mes = _mes < 10 ? "0" + _mes : String(_mes)

  const year = String(d.getFullYear())

  if (!layout) { return d }

  let dateStr = ""
  for (const sec of layout) {
    switch (sec) {
      case "y":
        dateStr += year.substring(2, 4)
        break
      case "Y":
        dateStr += year
        break
      case "m":
        dateStr += mes
        break
      case "M":
        dateStr += mesesMap.get(mes)?.es || "?"
        break
      case "d":
        dateStr += dia
        break
      case "h": {
        let hora: string | number = d.getHours()
        if (hora < 10) hora = "0" + hora
        dateStr += hora
        break
      }
      case "n": {
        let min: string | number = d.getMinutes()
        if (min < 10) min = "0" + min
        dateStr += min
        break
      }
      default:
        dateStr += sec
    }
  }

  return dateStr
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

// Compact form for cells too narrow for a grouped number: past 5 digits the value is shown in
// thousands (100,000 -> 100K), past 8 in millions. Below that formatN is used unchanged, so the
// exact figure stays readable while it still fits.
export function numberToK(x: number, decimal?: number): string {
  if (typeof x !== 'number' || !isFinite(x)) return ''
  const magnitude = Math.abs(x)
  if (magnitude > 99999999) return `${formatN(x / 1000000, 0)}M`
  if (magnitude > 99999) return `${formatN(x / 1000, 0)}K`
  return String(formatN(x, decimal))
}

export const normalizeComparableValue = (value: unknown): unknown => {
  if (!value || (Array.isArray(value) && value.length === 0)) return 0

  if (Array.isArray(value)) {
    const normalizedValues = value
      .filter(entry => !!entry)
      .map(entry => normalizeComparableValue(entry))
      .filter((entry) => entry !== 0)
      .map((entry) => String(entry))
      .filter((entry) => entry.length > 0)
      .sort()

    return normalizedValues.length > 0 ? normalizedValues.join('|') : 0
  }

  if (typeof value === 'object') {
    const objectValue = value as Record<string, unknown>
    const keys = Object.keys(objectValue).sort()
    const normalizedPairs: string[] = []
    for (const key of keys) {
      const normalizedEntry = normalizeComparableValue(objectValue[key])
      if (normalizedEntry === 0) continue
      normalizedPairs.push(`${key}:${String(normalizedEntry)}`)
    }
    return normalizedPairs.length > 0 ? normalizedPairs.join(',') : 0
  }

  if (typeof value === 'string') return value.trim() || 0
  return value
}

export const splitTwoStrings = (str: string, maxLen?: number): [string,string] => {
  if(!str){ return ["",""] }
  if(maxLen && str?.length <= maxLen){ return [str,""] }
  let bestDiff = Infinity;
  let bestIndex = -1;

  for (let i = 0; i < str.length; i++) {
    if (str[i] === " ") {
      let leftLen = i;                  // length of left part
      let rightLen = str.length - i - 1; // minus the space
      let diff = Math.abs(leftLen - rightLen);

      if (diff < bestDiff) {
        bestDiff = diff;
        bestIndex = i;
      }
    }
  }
  
  return [str.slice(0, bestIndex), str.slice(bestIndex + 1)]
}
