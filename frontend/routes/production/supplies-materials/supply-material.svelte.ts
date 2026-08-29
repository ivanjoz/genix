import { GetHandler } from '$libs/ui-runtime.svelte';
import type { IProductSupplyProviderRow } from '$routes/logistics/purchase-management/supply-management.svelte';

// A supply/material is a Product row with ss = 2. The catalog fields below mirror
// businessTypes.Product; MinimunStock and ProviderSupply live in the separate
// product_supply table and are merged in by the page from ProductSupplyService.
export const SUPPLY_PRODUCT_STATUS = 2

export interface ISupplyMaterial {
  ID: number
  Name: string
  Description: string
  BrandID: number
  Price: number
  CurrencyID: number
  UnitID: number
  SKU: string
  // Straight-line depreciation term. > 0 makes this supply a fixed asset, and the
  // Activos page can then acquire instances of it.
  DepreciationMonths: number
  // Merged from product_supply for the form; not part of the Product row itself.
  MinimunStock: number
  ProviderSupply: IProductSupplyProviderRow[]
  ss: number
  upd: number
}

export class SupplyMaterialService extends GetHandler<ISupplyMaterial> {
  route = "supply-material"
  // Bump `ver` whenever the ISupplyMaterial shape changes so clients drop their stale snapshot.
  // ver 4: supplies became Product rows (ss=2) and gained DepreciationMonths.
  useCache = { min: 5, ver: 4 }
  prependOnSave = true

  makeName(record: Partial<ISupplyMaterial>) {
    return record.Name || ""
  }

  handler(result: ISupplyMaterial[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
    // ss=2 is the live state for a supply; the delta also carries rows that left the
    // bucket (deleted, or converted to a product) so they can be evicted locally.
    this.addSavedRecords(...(result || []).filter(record => record.ss === SUPPLY_PRODUCT_STATUS))
    this.records.sort((a, b) => b.ID - a.ID)
  }

  constructor(init: boolean = false) {
    super()
    if (init) {
      this.fetch()
    }
  }
}

// Depreciation terms offered on the supply form. 0 leaves the supply a plain consumable;
// anything else makes it an asset. The terms match the usual Peruvian tax rates
// (computers 25%/yr, vehicles 20%/yr, machinery 10%/yr, buildings 3%/yr).
export const depreciationTerms = [
  { id: 0, label: "Not depreciable|No depreciable" },
  { id: 48, label: "4 years · computers|4 años · equipos de cómputo" },
  { id: 60, label: "5 years · vehicles|5 años · vehículos" },
  { id: 120, label: "10 years · machinery|10 años · maquinaria" },
  { id: 240, label: "20 years · furniture|20 años · muebles" },
  { id: 400, label: "33 years · buildings|33 años · edificios" },
]
