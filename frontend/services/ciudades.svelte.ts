export interface ICiudad {
  ID: string
  Nombre: string
  PadreID: string
  Nivel?: number
}

export interface ICiudadesResult {
  departamentos: ICiudad[]
  provincias: ICiudad[]
  distritos: ICiudad[]
  ciudadHijosMap: Map<string,ICiudad[]>
  ciudadesMap: Map<string,ICiudad>
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
    if(cd.ID.length === 6){ res.distritos.push(cd) }
    else if(cd.ID.length === 4){ res.provincias.push(cd) }
    else if(cd.ID.length === 2){ res.departamentos.push(cd) }
    if(!cd.PadreID){ continue }

    if(!cd.PadreID){ continue }
    const hijos = res.ciudadHijosMap.get(cd.PadreID)
    if(hijos){
      hijos.push(cd)
    } else {
      res.ciudadHijosMap.set(cd.PadreID, [cd])
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