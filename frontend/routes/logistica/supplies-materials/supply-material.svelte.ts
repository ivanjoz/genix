import { GetHandler } from '$libs/http.svelte';

// Provider row attached to a supply — mirrors backend types.ProductSupplyProviderRow.
export interface ISupplyMaterialProviderRow {
  ProviderID: number
  Capacity: number
  DeliveryTime: number
  Price: number
}

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
  ProviderSupply: ISupplyMaterialProviderRow[]
  ss: number
  upd: number
  UpdatedBy: number
  Created: number
  CreatedBy: number
}

export class SupplyMaterialService extends GetHandler<ISupplyMaterial> {
  route = "supply-material"
  // Bump `ver` whenever the ISupplyMaterial shape changes so clients drop their stale snapshot.
  useCache = { min: 5, ver: 1 }
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
