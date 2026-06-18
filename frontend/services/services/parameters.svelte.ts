import { GetHandler } from '$libs/http.svelte';

export interface IParameter {

}

export class ParametersService extends GetHandler {
  route = "parametros"
  useCache = { min: 5, ver: 6 }

  Records: IParameter[] = $state([])
  RecordsMap: Map<number,IParameter>= $state(new Map())

  handler(result: {[k: string]: IParameter[]}): void {
    console.log("parametros getted:", result)
  }

  constructor(grupo: number){
    super()
    alert("parametros grupo: " + grupo)
    this.route = `parametros?grupo=${grupo}`
    this.fetch()
  }
}
