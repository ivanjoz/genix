export const recreateObject = (obj: any, keysMap: Map<string,string|number>): any => {
  if(Array.isArray(obj)){ return obj.map(x => recreateObject(x, keysMap)) }
  if(typeof obj !== 'object' || !obj || !obj._){ return obj }
  for(const [key, value] of Object.entries(obj)){
    if(keysMap.has(key)){
      const newKey = keysMap.get(key)
      if(newKey === key){ continue }
      obj[newKey] = value
      delete obj[key]
    }
  }

  for(let i = 0; i < obj._.length; i+=2){
    const key = keysMap.get(obj._[i])
    obj[key] = recreateObject(obj._[i+1], keysMap)
  }
  delete obj._
  return obj
}

export const simplifyObject = (obj: any, keysMap: Map<string,string|number>): any => {
  if(Array.isArray(obj)){ return obj.map(x => simplifyObject(x, keysMap)) }

  if(typeof obj !== 'object' || !obj){ return obj }

  const newObj = { _: [] as any[] } as {[key: string]: any  }
  for(const [key, value] of Object.entries(obj)){
    if(!keysMap.has(key)){
      const id = keysMap.get("__count__") as number || 0
      keysMap.set(key, id+1)
      keysMap.set("__count__", id+1) 
    }
    const id = keysMap.get(key)
    if(typeof id === 'string'){
      newObj[id] = value
    } else {
      newObj._.push(keysMap.get(key), simplifyObject(value, keysMap))
    }
  }
  return newObj
}

export const recreateArray = (records: any[]) => {
  const keysMap = (records.find(x => x._keysMap) || {})._keysMap
  if(keysMap){
    const newRecords: any[] = []
    for(const e of records){
      if(e._keysMap){ continue }
      newRecords.push(recreateObject(e, keysMap))
    }
    return newRecords
  } else {
    return records
  }
}
