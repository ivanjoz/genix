import { GetHandler } from '#libs/ui-runtime.svelte.ts';

export class DemoService extends GetHandler {

  isTest = true
  message = $state("")
  route = "ruta de prueba"

  handler(e: any): void {
    this.message = e.message
  }

  constructor(){
    super()
  }
}
