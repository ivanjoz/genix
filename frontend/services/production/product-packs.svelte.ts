import { GetHandler } from '#libs/ui-runtime.svelte.ts'
import type { IProductPack } from '#core/product-packs.ts'

// The company's active packs, by name. A deleted pack leaves the delta with ss = 0.
export class ProductPacksService extends GetHandler<IProductPack> {
  route = 'product-packs'
  useCache = { min: 5, ver: 1 }

  handler(result: IProductPack[]): void {
    const packs = (result || []).filter((pack) => pack.ss === 1)
    packs.sort((leftPack, rightPack) => leftPack.Name.localeCompare(rightPack.Name))
    this.records = packs
    this.recordsMap = new Map(packs.map((pack) => [pack.ID, pack]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

// Saves one pack; ss = 0 deletes it.
export const postProductPack = (packs: ProductPacksService, pack: IProductPack) =>
  packs.writeAndSync<IProductPack>('product-pack', pack)
