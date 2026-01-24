export const makeRamdomString = (length: number) => {
  let str = ""
  while(str.length < length){
    str += (Math.random() + 1).toString(36).substring(2)
  }
  return str.substring(0,length)
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

