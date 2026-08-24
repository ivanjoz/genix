// Pure asset accounting. No Svelte, no fetch — unit-testable in isolation.
// Mirrors backend/accounting/types/asset.go and backend/accounting/depreciation.go.

export interface IAsset {
  ID: number
  ProductID: number
  SerialNumber: string
  Quantity: number
  WarehouseID: number
  Name: string
  Description: string
  SupplierID: number
  // The acquisition's own money. An asset is not an expense, so what is owed for it lives
  // here rather than in the expense register; only depreciation reaches that table.
  PurchaseAmount: number
  PaidAmount: number
  PaymentStatus: number
  DueDate: number
  AcquisitionDate: number
  AcquisitionValue: number
  DepreciationMonths: number
  CurrencyType: number
  AccumulatedDepreciation: number
  LastDepreciationDate: number
  DisposalDate: number
  ss: number
  upd: number
}

export const AssetStatus = {
  REMOVED: 0,
  ACTIVE: 1,
  FULLY_DEPRECIATED: 2,
  DISPOSED: 3,
}

export const AssetPayment = {
  NONE: 0,    // Donated or contributed — nothing was ever owed.
  PENDING: 1,
  PAID: 2,
}

export const assetPaymentLabels: Record<number, string> = {
  [AssetPayment.NONE]: "Donated|Donado",
  [AssetPayment.PENDING]: "Pending|Pendiente",
  [AssetPayment.PAID]: "Paid|Pagado",
}

export const assetStatusLabels: Record<number, string> = {
  [AssetStatus.ACTIVE]: "Active|Activo",
  [AssetStatus.FULLY_DEPRECIATED]: "Fully depreciated|Totalmente depreciado",
  [AssetStatus.DISPOSED]: "Disposed|Dado de baja",
}

// What the asset is still worth. Never negative: the backend sizes the final period to
// land exactly on zero, and this clamps anything a stale cache might disagree about.
export const assetBookValue = (asset: IAsset) =>
  Math.max((asset.AcquisitionValue || 0) - (asset.AccumulatedDepreciation || 0), 0)

// AccumulatedDepreciation is `omitempty` on the wire, so a brand-new asset arrives without
// the field at all — hence the `|| 0` rather than trusting it to be present.
export const assetDepreciatedPercent = (asset: IAsset) => {
  if (!asset.AcquisitionValue) return 0
  return Math.min(
    Math.round(((asset.AccumulatedDepreciation || 0) / asset.AcquisitionValue) * 100), 100,
  )
}

// Months still to post, derived from what has accumulated rather than from the calendar,
// so the figure always agrees with the ledger even if a run has not happened yet today.
export const assetRemainingMonths = (asset: IAsset) => {
  if (!asset.DepreciationMonths || !asset.AcquisitionValue) return 0
  const monthlyAmount = asset.AcquisitionValue / asset.DepreciationMonths
  if (monthlyAmount <= 0) return 0
  return Math.max(Math.ceil(assetBookValue(asset) / monthlyAmount), 0)
}

// What is still owed on the acquisition. Never negative: a write-off can settle more than
// the balance, and that is a paid asset, not a negative debt.
export const assetPendingAmount = (asset: IAsset) =>
  Math.max((asset.PurchaseAmount || 0) - (asset.PaidAmount || 0), 0)

// A donation is not payable at all, and a settled asset has nothing left to pay.
export const canPayAsset = (asset: IAsset) =>
  (asset.PurchaseAmount || 0) > 0 && assetPendingAmount(asset) > 0

// An asset is only a candidate for disposal while it is still held, whether or not it has
// finished depreciating — a fully depreciated laptop is still a laptop the business owns.
export const canDisposeAsset = (asset: IAsset) =>
  asset.ss === AssetStatus.ACTIVE || asset.ss === AssetStatus.FULLY_DEPRECIATED

// Serial-tracked assets are one unit each; a grouped acquisition carries its lot size.
export const assetUnitLabel = (asset: IAsset) =>
  asset.SerialNumber || `x${asset.Quantity || 1}`

// One asset's posted depreciation ledger — the Type-4 expense rows. The backend stores
// neither a Name nor a PeriodDate on them, so nothing here mirrors those: Date is the
// first of the month the period covers, and the label is composed below.
export interface IDepreciationEntry {
  ID: number
  Date: number
  Amount: number
}

// The schedule's running order. The ledger is queried by asset, so the rows arrive in
// insertion order; sorting on Date makes the numbering hold even if a re-run ever
// interleaved them, and it is the period — not the row's age — the number refers to.
export const depreciationSchedule = (entries: IDepreciationEntry[], totalMonths: number) =>
  [...entries]
    .sort((a, b) => a.Date - b.Date)
    .map((entry, index) => ({
      entry,
      // Composed here rather than persisted: a stored label cannot be translated, and it
      // freezes DepreciationMonths at the value it had the day the row was written.
      label: `Depreciation ${index + 1}/${totalMonths}|Depreciación ${index + 1}/${totalMonths}`,
    }))
