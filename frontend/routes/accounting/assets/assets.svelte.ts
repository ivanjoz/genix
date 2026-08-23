import { GetHandler, GET, POST, PUT } from '$libs/ui-runtime.svelte'
import { AssetStatus, type IAsset } from './assets'

export type { IAsset }

export class AssetsService extends GetHandler<IAsset> {
  route = "assets"
  keyID = "ID"
  useCache = { min: 1, ver: 1 }
  prependOnSave = true

  records: IAsset[] = $state([])
  recordsMap: Map<number, IAsset> = $state(new Map())

  handler(result: IAsset[]): void {
    this.records = []
    this.recordsMap = new Map()
    // Disposed and fully depreciated assets stay on the register; only a removed row leaves it.
    this.addSavedRecords(...(result || []).filter(record => record.ss > AssetStatus.REMOVED))
    this.records.sort((a, b) => b.ID - a.ID)
  }

  constructor(init: boolean = false) {
    super()
    if (init) this.fetch()
  }
}

export interface IAssetAcquisition {
  ProductID: number
  SerialNumbers: string[]
  Quantity: number
  WarehouseID: number
  AcquisitionDate: number
  Name: string
  Description: string
  SupplierID: number
  CurrencyType: number
  AcquisitionValue: number
  // Cash owed per unit. 0 = donated: nothing is owed and the asset is never payable.
  PurchaseAmount: number
  DueDate: number
  DepreciationMonths: number
}

// Acquiring an asset writes no expense — only the asset rows and the stock movement.
export const postAssetAcquisition = (data: IAssetAcquisition): Promise<IAsset[]> => {
  return POST({ data, route: "asset", refreshRoutes: ["assets"] })
}

export interface IAssetPayment {
  AssetID: number
  CashBankID: number
  Amount: number   // Positive payment amount, in cents.
  Date: number
  IsFullyPaid: boolean
}

// The asset is paid on its own account, not through the expense register, so this refreshes
// the asset list and the cash banks whose balance it moved.
export const postAssetPayment = (data: IAssetPayment): Promise<IAsset> => {
  return POST({ data, route: "asset-payment", refreshRoutes: ["assets", "cash-banks"] })
}

export const putAssetDisposal = (data: { AssetID: number, DisposalDate: number }): Promise<IAsset> => {
  return PUT({ data, route: "asset-disposal", refreshRoutes: ["assets"] })
}

export const putAssetTransfer = (data: { AssetID: number, TargetWarehouseID: number }): Promise<IAsset> => {
  return PUT({ data, route: "asset-transfer", refreshRoutes: ["assets"] })
}

// Posts every depreciation period that has come due. Idempotent — the backend filters each
// period against the asset's own watermark, so calling it on every page load is safe.
export const runAssetDepreciation = (): Promise<{ PostedEntries: number, UpdatedAssets: number }> => {
  return POST({ data: {}, route: "asset-depreciation-run", refreshRoutes: ["assets"] })
}

// One asset's posted depreciation ledger — the Type-4 expense rows.
export interface IDepreciationEntry {
  ID: number
  Name: string
  Date: number
  PeriodDate: number
  Amount: number
}

export const getAssetDepreciation = async (assetID: number): Promise<IDepreciationEntry[]> => {
  const result = await GET({ route: `asset-depreciation?assetID=${assetID}` })
  return result || []
}
