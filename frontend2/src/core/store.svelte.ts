import { goto } from '$app/navigation';
import { Env } from '$lib/security';
import type { IMenuRecord } from '../types/menu';
import type { IModule } from './modules';
import { SvelteMap } from 'svelte/reactivity';

export const Core = $state({
  module: { menus: [] as IMenuRecord[] } as IModule,
  openSearchLayer: 0 as number,
  deviceType: 1 as number,
  mobileMenuOpen: 0 as number,
  popoverShowID: 0 as number | string,
  showSideLayer: 0 as number,
  isLoading: 1
})

export const WeakSearchRef: WeakMap<any,{ 
  idToRecord: Map<string|number, any>
  valueToRecord: Map<string,any>
}> = new WeakMap()


export interface IFetchEvent {
  url: string 
}

export const fetchOnCourse = $state<Map<number,IFetchEvent>>(new SvelteMap())

export const fetchEvent = (fetchID: number, props: IFetchEvent | 0) => {
  if (fetchID === 0) { // Para obtener un nuevo ID de FetchEvent
    Env.fetchID++
    return Env.fetchID
  }

  if (props === 0) { // Para eliminar el proceso del Map
    fetchOnCourse.delete(fetchID)
  }
  else { // Para setear el proceso
    fetchOnCourse.set(fetchID, props)
  }
}