import pkg from 'notiflix';
import { Env } from './env';
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
  console.log("words 111:", arr.filter(x => x),"|",phrase,words)
  return arr.filter(x => x)
}

export const parseSVG = (svgContent: string)=> {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgContent)}`
}

export const cn = (...classNames: (string|boolean)[]) => {
  return classNames.filter(x => x).join(" ")
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

const pendingWorkerRequests = new Map<number, {
  resolve: (value: string) => void,
  reject: (reason?: any) => void,
  timeout: NodeJS.Timeout
}>();

interface WorkerInstance { worker: Worker; tasks: number; }

const workerPool = new Map<number|string, WorkerInstance>();
const MAX_WORKERS = 4;

const setWorkerCommunication = (wi: WorkerInstance) => {
  wi.worker.onmessage = ({ data }) => {
    wi.tasks = Math.max(0, wi.tasks - 1);
    const { id, dataUrl, error } = data;
    const request = pendingWorkerRequests.get(id);

    if (request) {
      clearTimeout(request.timeout);
      pendingWorkerRequests.delete(id);

      if (error) {
        console.error(`❌ Worker error for request ${id}:`, error);
        request.reject(error);
      } else {
        console.log(`📨 Received message from worker for request ${id} | len: ${dataUrl.length}`);
        request.resolve(dataUrl || "");
      }
    }
  };

  wi.worker.onerror = (error) => {
    wi.tasks = Math.max(0, wi.tasks - 1);
    console.error('❌ Worker error:', error);
  };
};

const getBestWorker = (): Worker => {
  if (Env.imageWorker && !workerPool.has('preloaded')) {
    const wi = { worker: Env.imageWorker, tasks: 0 }
    setWorkerCommunication(wi);
    workerPool.set('preloaded', wi);
  }

  let best: WorkerInstance | null = null;

  for (const wi of workerPool.values()) {
    if (wi.tasks === 0) { best = wi; break }
    if (!best || wi.tasks < best.tasks) { best = wi }
  }

  if ((!best || best.tasks > 0) && workerPool.size < MAX_WORKERS && Env.ImageWorkerClass) {
    console.log(`🚀 Spawning new worker (Pool size: ${workerPool.size + 1})`);
    const wi = { worker: new Env.ImageWorkerClass(), tasks: 0 };
    setWorkerCommunication(wi);
    const id = Math.random();
    workerPool.set(id, wi);
    best = wi;
  }

  if (best) {
    best.tasks++;
    return best.worker;
  }

  return Env.imageWorker;
};

// Resolución en Mpx (máximo 2)
export const fileToImage = (
  blob: Blob | File, resolution: number, fileType?: "webp" | "avif" | "jpg",
): Promise<string> => {

  if(resolution > 2000){
    Notify.failure("2mpx is max resolution for image conversion.")
    return Promise.resolve("")
  }

  const useJpeg = fileType === "jpg"
  const useAvif = fileType === "avif"

  const worker = getBestWorker();

  console.log('📸 fileToImage called with:', {
    blobType: blob.type,
    blobSize: blob.size,
    resolution, useJpeg, useAvif,
    poolSize: workerPool.size,
    workerExists: !!worker
  })

  if(!worker){
    console.error('❌ No image worker available!')
    return Promise.reject("Image worker not available")
  }

  return new Promise((resolve, reject) => {
    const id = Math.floor(Math.random() * 1000000);
    const timeout = setTimeout(() => {
      console.error(`⏱️ Worker timeout - no response after 8 seconds (id=${id}, r=${resolution})`)
      pendingWorkerRequests.delete(id);
      reject("Error al procesar la imagen. (superó los 8 segundos.)")
    }, 8000)

    pendingWorkerRequests.set(id, { resolve, reject, timeout });

    console.log(`📤 Posting message to worker (id=${id})...`)
    worker.postMessage({ id, blob, resolution, useJpeg, useAvif })
  })
}

// Resolución en Mpx (máximo 2)
export const bitmapToImage = (
  bitmap: ImageBitmap, resolution: number, fileType?: "webp" | "avif" | "jpg",
): Promise<string> => {

  if(resolution > 2000){
    Notify.failure("2mpx is max resolution for image conversion.")
    return Promise.resolve("")
  }

  const useJpeg = fileType === "jpg"
  const useAvif = fileType === "avif"

  const worker = getBestWorker();

  console.log('📸 bitmapToImage called with:', {
    bitmapWidth: bitmap.width,
    bitmapHeight: bitmap.height,
    resolution, useJpeg, useAvif,
    poolSize: workerPool.size,
    workerExists: !!worker
  })

  if(!worker){
    console.error('❌ No image worker available!')
    return Promise.reject("Image worker not available")
  }

  return new Promise((resolve, reject) => {
    const id = Math.floor(Math.random() * 1000000);
    const timeout = setTimeout(() => {
      console.error(`⏱️ Worker timeout - no response after 8 seconds (id=${id})`)
      pendingWorkerRequests.delete(id);
      reject("Error al procesar la imagen. (superó los 8 segundos.)")
    }, 8000)

    pendingWorkerRequests.set(id, { resolve, reject, timeout });

    console.log(`📤 Posting message to worker with bitmap transfer (id=${id})...`)
    worker.postMessage({ id, bitmap, resolution, useJpeg, useAvif }, [bitmap])
  })
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
    // Valida las fechas por dia, segundo o ms
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

  // Revisa si es una fecha válida
  if (!d || !(d instanceof Date) || !d.getTime) return layout ? "" : null

  const _dia = d.getDate()
  if (isNaN(_dia)) return !layout ? null : ""
  const dia = _dia < 10 ? "0" + _dia : String(_dia)

  const _mes = d.getMonth() + 1
  const mes = _mes < 10 ? "0" + _mes : String(_mes)

  const year = String(d.getFullYear())

  if (!layout) { return d }

  let fechaStr = ""
  for (const sec of layout) {
    switch (sec) {
      case "y":
        fechaStr += year.substring(2, 4)
        break
      case "Y":
        fechaStr += year
        break
      case "m":
        fechaStr += mes
        break
      case "M":
        fechaStr += mesesMap.get(mes)?.es || "?"
        break
      case "d":
        fechaStr += dia
        break
      case "h": {
        let hora: string | number = d.getHours()
        if (hora < 10) hora = "0" + hora
        fechaStr += hora
        break
      }
      case "n": {
        let min: string | number = d.getMinutes()
        if (min < 10) min = "0" + min
        fechaStr += min
        break
      }
      default:
        fechaStr += sec
    }
  }

  return fechaStr
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

export const decrypt = async (encryptedString: string, key: string) => {
  key = key.substring(0,32)
  // Decode the base64 string
  const encryptedData = Uint8Array.from(atob(encryptedString), c => c.charCodeAt(0))
  console.log("encrypted len::", encryptedData.length)

  // Convert the key to Uint8Array
  const keyBuffer = await crypto.subtle.importKey(
    "raw", new TextEncoder().encode(key), { name: "AES-GCM" }, false, ["decrypt"]
  )

  // Ensure the encrypted data is not too short
  if (encryptedData.length < 12) {
    throw new Error("Invalid encrypted data");
  }

  // Extract the nonce from the first 12 bytes
  const nonce = encryptedData.slice(0, 12)
  console.log("nonce:: " ,new TextDecoder().decode(nonce))

  const ciphertext = encryptedData.slice(12)

  console.log("desencriptando:: ", nonce.length, key.length, ciphertext.length)

  // Decrypt the data
  let decryptedData: ArrayBuffer
  try {
    decryptedData = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: nonce }, keyBuffer, ciphertext
    )
  } catch (error) {
    console.log("Error desencriptando:: ", error)
    return ""
  }

  console.log("decripted data:: ", decryptedData)
  // Convert the decrypted data to a string
  const decryptedString = new TextDecoder().decode(decryptedData)
  return decryptedString
}
