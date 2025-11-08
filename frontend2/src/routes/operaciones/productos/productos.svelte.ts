import { GetHandler } from "$lib/http";
import { browser } from '$app/environment';

export class ProductosService extends GetHandler {
  route = "productos"
  useCache = { min: 5, ver: 1 }

  productos = $state([])

  handler(result: any): void {
    
    console.log("productos result::", result)
  }

  constructor(){
    super()
    if (browser) {
      this.fetch()
    }
  }
}