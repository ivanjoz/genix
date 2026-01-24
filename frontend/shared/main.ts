export const makeRamdomString = (length: number) => {
  let str = ""
  while(str.length < length){
    str += (Math.random() + 1).toString(36).substring(2)
  }
  return str.substring(0,length)
}

export const formatN = (
  x: number, decimal?: number, fixedLen?: number, charF?: string
) => {
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
