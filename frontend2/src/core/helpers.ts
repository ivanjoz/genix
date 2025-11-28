import pkg from 'notiflix';
export const { Notify } = pkg;

let throttleTimer: NodeJS.Timeout | null

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
  if(typeof phrase !== 'string'){
    console.error("no es string")
    console.log(phrase)
    return [{ text: "!" }]
  }
  const arr: { text: string, highl?: boolean, isEnd?: boolean }[] = [{ text: phrase }]
  if (!words || words.length === 0){ return arr }
  console.log("words 222:", arr.filter(x => x),"|",phrase,words)

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
        const parts = [{ text: ini }, { text: middle, highl: true }, { text: fin }].filter(x => x.text)
        arr.splice(i, 1, ...parts)
        if(arr.length > 40){ 
          console.log("words 333:", arr.filter(x => x),"|",phrase,words)
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