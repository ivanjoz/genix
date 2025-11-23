import pkg from 'notiflix';
import { Env } from '../shared/env';
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

// Resolución en Mpx (máximo 2)
export const fileToImage = (blob: Blob | File, resolution: number, useJpeg?: boolean): Promise<string> => {

  if(resolution > 2){
    Notify.failure("2mpx is max resolution for image conversion.")
    return Promise.resolve("")
  }

  console.log('📸 fileToImage called with:', { 
    blobType: blob.type, 
    blobSize: blob.size, 
    resolution, 
    useJpeg,
    workerExists: !!Env.imageWorker 
  })

  if(!Env.imageWorker){
    console.error('❌ Image worker is not initialized!')
    return Promise.reject("Image worker is not initialized")
  }

  return new Promise((resolve, reject) => {
    let isResolved = false
    Env.imageWorker.onmessage = ({ data }) => {
      console.log('📨 Received message from worker:', data)
      isResolved = true
      resolve(data.dataUrl||"")
    }

    Env.imageWorker.onerror = (error) => {
      console.error('❌ Worker error:', error)
      isResolved = true
      reject("Worker error: " + error.message)
    }

    setTimeout(() => {
      if(!isResolved){
        console.error('⏱️ Worker timeout - no response after 8 seconds')
        reject("Error al procesar la imagen. (superó los 8 segundos.)")
      }
    },8000)

    console.log('📤 Posting message to worker...')
    Env.imageWorker.postMessage({ blob, resolution, useJpeg })
  })
}

// Resolución en Mpx (máximo 2)
export const bitmapToImage = (bitmap: ImageBitmap, resolution: number, useJpeg?: boolean): Promise<string> => {

  if(resolution > 2){
    Notify.failure("2mpx is max resolution for image conversion.")
    return Promise.resolve("")
  }

  console.log('📸 bitmapToImage called with:', { 
    bitmapWidth: bitmap.width,
    bitmapHeight: bitmap.height,
    resolution, 
    useJpeg,
    workerExists: !!Env.imageWorker 
  })

  if(!Env.imageWorker){
    console.error('❌ Image worker is not initialized!')
    return Promise.reject("Image worker is not initialized")
  }

  return new Promise((resolve, reject) => {
    let isResolved = false
    Env.imageWorker.onmessage = ({ data }) => {
      console.log('📨 Received message from worker:', data)
      isResolved = true
      resolve(data.dataUrl||"")
    }

    Env.imageWorker.onerror = (error) => {
      console.error('❌ Worker error:', error)
      isResolved = true
      reject("Worker error: " + error.message)
    }

    setTimeout(() => {
      if(!isResolved){
        console.error('⏱️ Worker timeout - no response after 8 seconds')
        reject("Error al procesar la imagen. (superó los 8 segundos.)")
      }
    },8000)

    console.log('📤 Posting message to worker with bitmap transfer...')
    Env.imageWorker.postMessage({ bitmap, resolution, useJpeg }, [bitmap])
  })
}

