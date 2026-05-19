export interface ICiudad {
  ID: number
  Name: string
  ParentID: number
  Hierarchy: number
}

export interface ICiudadesResult {
  departamentos: ICiudad[]
  provincias: ICiudad[]
  distritos: ICiudad[]
  ciudadHijosMap: Map<number,ICiudad[]>
  ciudadesMap: Map<number,ICiudad>
}

export const ciudadesService = $state({
  departamentos: [],
  provincias: [],
  distritos: [],
  ciudadHijosMap: new Map(),
  ciudadesMap: new Map()
} as ICiudadesResult)

const parseCiudades = (ciudades: ICiudad[]): ICiudadesResult => {
  const res: ICiudadesResult = {
    departamentos: [],
    provincias: [],
    distritos: [],
    ciudadHijosMap: new Map(),
    ciudadesMap: new Map()
  }

  for(const cd of ciudades){
    res.ciudadesMap.set(cd.ID, cd)
    // Numeric ubigeo IDs keep their hierarchy in Hierarchy instead of padded string length.
    if(cd.Hierarchy === 3){ res.distritos.push(cd) }
    else if(cd.Hierarchy === 2){ res.provincias.push(cd) }
    else if(cd.Hierarchy === 1){ res.departamentos.push(cd) }
    if(!cd.ParentID){ continue }

    if(!cd.ParentID){ continue }
    const hijos = res.ciudadHijosMap.get(cd.ParentID)
    if(hijos){
      hijos.push(cd)
    } else {
      res.ciudadHijosMap.set(cd.ParentID, [cd])
    }
  }
  return res
}

let ciudadesPromise: Promise<ICiudadesResult>

export const useCiudadesAPI = () => {

  if(!ciudadesPromise){
    ciudadesPromise = new Promise((resolve, reject) => {
      fetch("/files/peru_ciudades.json")
      .then(ciudades => {
        return ciudades.json()
      })
      .then(ciudades => {
        console.log("ciudades obtenidas::", ciudades)
        const ciudadesResult = parseCiudades(ciudades)
        resolve(ciudadesResult)
      })
      .catch(err => {
        reject(err)
      })
    })
  }

  ciudadesPromise.then(res => {
    ciudadesService.departamentos = res.departamentos
    ciudadesService.provincias = res.provincias
    ciudadesService.distritos = res.distritos
    ciudadesService.ciudadHijosMap = res.ciudadHijosMap
    ciudadesService.ciudadesMap = res.ciudadesMap
  })

  return ciudadesService
}
