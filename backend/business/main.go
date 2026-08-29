package business

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.locations-warehouses":    GetLocationsWarehouses,
	"GET.country-cities":          GetCountryCities,
	"POST.sites":                  PostSite,
	"POST.warehouses":             PostWarehouse,
	"POST.initial-data":           PostInitialData,
	"GET.p-products-ecommerce":    GetProductsEcommerce,
	"GET.shared-lists":            GetSharedLists,
	"POST.shared-lists":           PostSharedLists,
	"GET.image-id-counter":        GetImageIdCounter,
	"GET.image-assets":            GetImageAssets,
	"GET.image-asset-text-search": GetImageAssetTextSearch,
	"POST.gallery-image":          PostGalleryImage,
	"GET.gallery-images":          GetGalleryImages,
}
