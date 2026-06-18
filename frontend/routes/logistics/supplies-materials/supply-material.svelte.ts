import { GetHandler } from '$libs/http.svelte';
import type { IProductSupplyProviderRow } from '../purchase-management/supply-management.svelte';

// Mirror of backend logisticaTypes.SupplyMaterial. Field names/casing must match
// the Go struct so the JSON round-trips cleanly through GetHandler.
export interface ISupplyMaterial {
  ID: number
  CompanyID: number
  Name: string
  Description: string
  BrandID: number
  Price: number
  CurrencyID: number
  SKU: string
  MinimunStock: number
  ProviderSupply: IProductSupplyProviderRow[]
  ss: number
  upd: number
  UpdatedBy: number
  Created: number
  CreatedBy: number
}

export class SupplyMaterialService extends GetHandler<ISupplyMaterial> {
  route = "supply-material"
  // Bump `ver` whenever the ISupplyMaterial shape changes so clients drop their stale snapshot.
  useCache = { min: 5, ver: 2 }
  inferRemoveFromStatus = true
  prependOnSave = true

  makeName(record: Partial<ISupplyMaterial>) {
    return record.Name || ""
  }

  handler(result: ISupplyMaterial[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
    this.addSavedRecords(...result)
    this.records.sort((a, b) => b.ID - a.ID)
  }

  constructor(init: boolean = false) {
    super()
    if (init) {
      this.fetch()
    }
  }
}
