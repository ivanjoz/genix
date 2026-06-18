export interface ICity {
  ID: number
  Name: string
  ParentID: number
  Hierarchy: number
}

export interface ICitiesResult {
  departments: ICity[]
  provinces: ICity[]
  districts: ICity[]
  cityChildrenMap: Map<number,ICity[]>
  citiesMap: Map<number,ICity>
}

export const citiesService = $state({
  departments: [],
  provinces: [],
  districts: [],
  cityChildrenMap: new Map(),
  citiesMap: new Map()
} as ICitiesResult)

const parseCities = (ciudades: ICity[]): ICitiesResult => {
  const res: ICitiesResult = {
    departments: [],
    provinces: [],
    districts: [],
    cityChildrenMap: new Map(),
    citiesMap: new Map()
  }

  for(const cd of ciudades){
    res.citiesMap.set(cd.ID, cd)
    // Numeric ubigeo IDs keep their hierarchy in Hierarchy instead of padded string length.
    if(cd.Hierarchy === 3){ res.districts.push(cd) }
    else if(cd.Hierarchy === 2){ res.provinces.push(cd) }
    else if(cd.Hierarchy === 1){ res.departments.push(cd) }
    if(!cd.ParentID){ continue }

    if(!cd.ParentID){ continue }
    const hijos = res.cityChildrenMap.get(cd.ParentID)
    if(hijos){
      hijos.push(cd)
    } else {
      res.cityChildrenMap.set(cd.ParentID, [cd])
    }
  }
  return res
}

let citiesPromise: Promise<ICitiesResult>

export const useCitiesAPI = () => {

  if(!citiesPromise){
    citiesPromise = new Promise((resolve, reject) => {
      fetch("/files/peru_ciudades.json")
      .then(ciudades => {
        return ciudades.json()
      })
      .then(ciudades => {
        console.log("ciudades obtenidas::", ciudades)
        const ciudadesResult = parseCities(ciudades)
        resolve(ciudadesResult)
      })
      .catch(err => {
        reject(err)
      })
    })
  }

  citiesPromise.then(res => {
    citiesService.departments = res.departments
    citiesService.provinces = res.provinces
    citiesService.districts = res.districts
    citiesService.cityChildrenMap = res.cityChildrenMap
    citiesService.citiesMap = res.citiesMap
  })

  return citiesService
}
