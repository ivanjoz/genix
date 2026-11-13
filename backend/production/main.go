package production

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.products":                GetProducts,
	"GET.p-products-ids":          GetProductsByIDs,
	"GET.p-product-text-search":   GetProductTextSearch,
	"GET.p-product-detail":        GetProductDetail,
	"POST.products":               PostProducts,
	"POST.product-image":          PostProductImage,
	"POST.product-category-image": PostProductCategoryImage,
	"GET.supply-material":         GetSupplyMaterials,
	"POST.supply-material":        PostSupplyMaterial,
}
