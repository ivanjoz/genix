import { GetHandler } from '$libs/http.svelte';

export interface IParametro {

}

export class ParametrosService extends GetHandler {
  route = "parametros"
  useCache = { min: 5, ver: 6 }

  Records: IParametro[] = $state([])
  RecordsMap: Map<number,IParametro>= $state(new Map())

  handler(result: {[k: string]: IParametro[]}): void {
    console.log("parametros getted:", result)
  }

  constructor(grupo: number){
    super()
    alert("parametros grupo: " + grupo)
    this.route = `parametros?grupo=${grupo}`
    this.fetch()
  }
}
