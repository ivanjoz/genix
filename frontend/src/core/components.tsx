import { Accessor, createEffect, createSignal } from "solid-js"
import { SearchSelect } from "~/components/SearchSelect"

export interface ICiudadSelector<T> {
  css: string[]
  saveOn?: T
  save?: keyof T
}

let ciudadesPromise: Promise<ICiudadesResult>

interface ICiudad {
  ID: string
  Nombre: string
  PadreID: string
  Nivel?: number
}

interface ICiudadesResult {
  departamentos: ICiudad[]
  provincias: ICiudad[]
  distritos: ICiudad[]
  ciudadHijosMap: Map<string,ICiudad[]>
  ciudadesMap: Map<string,ICiudad>
}

export const useCiudadesAPI = () => {

  const [fetchedRecords, setFetchedRecords] = createSignal(null)

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

      res.ciudadHijosMap.has(cd.PadreID)
        ? res.ciudadHijosMap.get(cd.PadreID).push(cd)
        : res.ciudadHijosMap.set(cd.PadreID, [cd])
    }
    return res
  }
  
  const route = "./assets/peru_ciudades.json"
  if(!ciudadesPromise){
    ciudadesPromise = new Promise((resolve, reject) => {
      fetch(route)
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

  ciudadesPromise.then(results => {
    setFetchedRecords(results)
  })

  return [fetchedRecords as Accessor<ICiudadesResult>]
}

export const CiudadSelector = <T,>(props: ICiudadSelector<T>) => {

  const [ciudades] = useCiudadesAPI()
  const [form, setForm] = createSignal({
    departamentoID: "", distritoID: "", provinciaID: ""
  })
  const [isLoading, setIsLoading] = createSignal(true)
  const [departamentos, setDepartamentos] = createSignal([] as ICiudad[])
  const [provincias, setProvincias] = createSignal([] as ICiudad[])
  const [distritos, setDistritos] = createSignal([] as ICiudad[])

  createEffect(() => {
    if(!ciudades()){ return }
    setIsLoading(false)
    setDepartamentos(ciudades().departamentos)

    if(props.save && props.saveOn){
      const ciudad = ciudades().ciudadesMap.get(props.saveOn[props.save] as string)
      console.log("ciudad::", ciudad)
      if(ciudad){
        const form_ = form()
        if(ciudad.ID.length === 2){
          form_.departamentoID = ciudad.ID
        } else if(ciudad.ID.length === 4){
          form_.departamentoID = ciudad.PadreID
          form_.provinciaID = ciudad.ID
        } else {
          const provincia = ciudades().ciudadesMap.get(ciudad.PadreID)
          form_.departamentoID = provincia.PadreID
          form_.provinciaID = ciudad.PadreID
          form_.distritoID = ciudad.ID
        }
        if(form_.departamentoID){
          setProvincias(ciudades().ciudadHijosMap.get(form_.departamentoID))
        }
        if(form_.provinciaID){
          setDistritos(ciudades().ciudadHijosMap.get(form_.provinciaID))
        }
      }
      console.log("ciudades save on::", props.saveOn, form())
    }
  })

  const save = () => {
    if(!props.saveOn || !props.save){ return }
    const ciudadID = form().distritoID || form().provinciaID || form().departamentoID || ""
    props.saveOn[props.save] = ciudadID as never
  }

  return <>
    <SearchSelect label="Departamento" keys="ID.Nombre" 
      options={departamentos()} required={true} saveOn={form()} save="departamentoID"
      css={(props.css||[])[0]||""} showLoading={isLoading()}
      onChange={e => {
        console.log("departamento::", e)
        form().distritoID = null
        form().provinciaID = null
        setProvincias(ciudades().ciudadHijosMap.get(e.ID))
        setDistritos([])
        save()
      }}
    />
    <SearchSelect label="Provincia" keys="ID.Nombre"  required={true}
      options={provincias()} disabled={provincias().length === 0}
      saveOn={form()} save="provinciaID"
      css={(props.css||[])[0]||""} showLoading={isLoading()}
      onChange={e => {
        form().distritoID = null
        setDistritos(ciudades().ciudadHijosMap.get(e.ID))
        save()
      }}
    />
    <SearchSelect label="Distrito" keys="ID.Nombre"  required={true}
      saveOn={form()} save="distritoID"
      options={distritos()} disabled={distritos().length === 0}
      css={(props.css||[])[0]||""} showLoading={isLoading()}
      onChange={() => { save() }}
    />
  </>

}