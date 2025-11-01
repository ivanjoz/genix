import { goto } from '$app/navigation';
import type { IMenuRecord } from '../types/menu';
import type { IModule } from './modules';

export let Core = $state({
  module: { menus: [] as IMenuRecord[] } as IModule,
  openSearchLayer: 0 as number,
  deviceType: 1 as number,
  mobileMenuOpen: 0 as number,
  popoverShowID: 0 as number | string
})

export const WeakSearchRef: WeakMap<any,{ 
  idToRecord: Map<string|number, any>
  valueToRecord: Map<string,any>
}> = new WeakMap()