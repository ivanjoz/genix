import { GetSignal, POST, makeApiGetHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

export interface IListaRegistro {
  ID: number
  ListaID: number
  Nombre: string
  Images?: string[]
  Descripcion?: string
  UpdatedBy?: number
  ss: number
  upd: number
}

export interface IListas {
  Records: IListaRegistro[]
  RecordsMap: Map<number,IListaRegistro>
}

export const listasCompartidas = [
  { id: 1, name: "Categoría" },
  { id: 2, name: "Marca" }
]

export const useListasCompartidasAPI = (ids: number[]): GetSignal<IListas> => {
  const pk = ids.sort().join(",")

  return  makeApiGetHandler(
    { route: "listas-compartidas",
      errorMessage: 'Hubo un error al obtener las listas compartidas.',
      cacheSyncTime: 1, mergeRequest: true,
      partition: { key: 'pk', value: pk, param: 'ids' },
      useIndexDBCache: 'listas_compartidas',
      makeTransform: e => { e.pk = pk }
    },
    (result_) => {
      const result = result_ as IListas
      result.RecordsMap = arrayToMapN(result.Records, 'ID')
      // console.log("listas compartidas API::",result)
      result.Records = result.Records.filter(x => x.ss)

      // Custom Convert
      for(const e of result.Records){
        if(e.ListaID === 1){ // Productos Categorias
          const imagesMap = new Map((e.Images||[]).map(x => (
            [parseInt(x.split("-")[1]),x])))
          
          e.Images = []
          for(const order of [1,2,3]){
            e.Images.push(imagesMap.get(order)||"")
          }
        }
      }

      return result
    }
  )
}

export interface INewIDToID {
  NewID:  number
	TempID: number
}

export const postListaRegistros = (data: IListaRegistro[]) => {
  return POST({
    data,
    route: "listas-compartidas",
    refreshIndexDBCache: "listas_compartidas"
  })
}
