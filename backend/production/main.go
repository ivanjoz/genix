package production

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.products":                GetProducts,
	"GET.products-ids":            GetProductsByIDs,
	"GET.p-products-ids":          GetPublicProductsByIDs,
	"GET.p-product-text-search":   GetProductTextSearch,
	"GET.p-product-detail":        GetProductDetail,
	"POST.products":               PostProducts,
	"POST.product-image":          PostProductImage,
	"POST.product-category-image": PostProductCategoryImage,
	"GET.supply-material":         GetSupplyMaterials,
	"POST.supply-material":        PostSupplyMaterial,
	"GET.production-close-pending": GetDailyClosePending,
	"GET.production-close-preview": GetDailyClosePreview,
	"POST.production-close":        PostDailyClose,
	"GET.production-closes":        GetProductionCloses,
}
