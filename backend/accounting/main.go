package accounting

import "app/core"

// depreciationCategoryID is the expense category depreciation entries are filed under.
// 6 = Maintenance is the closest of the existing static categories; it is only a label,
// since what actually keeps these entries off the P&L is Type 4 plus Status 3.
const depreciationCategoryID int8 = 6

var ModuleHandlers = core.AppRouterType{
	"GET.assets":                  GetAssets,
	"POST.asset":                  PostAsset,
	"PUT.asset":                   PutAssetEdit,
	"PUT.asset-disposal":          PutAssetDisposal,
	"PUT.asset-transfer":          PutAssetTransfer,
	"POST.asset-depreciation-run": PostAssetDepreciationRun,
	"GET.asset-depreciation":      GetAssetDepreciation,
	"POST.asset-payment":          PostAssetPayment,
	"POST.expense-inventory":      PostInventoryExpense,
}
