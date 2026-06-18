package business

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.locations-warehouses":   GetLocationsWarehouses,
	"GET.country-cities":         GetCountryCities,
	"POST.sites":                 PostSite,
	"POST.warehouses":            PostWarehouse,
	"GET.products":               GetProducts,
	"GET.p-products-ids":         GetProductsByIDs,
	"GET.p-product-text-search":  GetProductTextSearch,
	"GET.p-products-ecommerce":   GetProductsEcommerce,
	"POST.products":              PostProducts,
	"POST.product-image":         PostProductImage,
	"POST.product-category-image": PostProductCategoryImage,
	"GET.shared-lists":           GetSharedLists,
	"POST.shared-lists":          PostSharedLists,
	"GET.image-id-counter":       GetImageIdCounter,
	"GET.image-assets":           GetImageAssets,
	"POST.gallery-image":         PostGalleryImage,
	"GET.gallery-images":         GetGalleryImages,
	"GET.client-provider":        GetClientProviders,
	"GET.client-provider-ids":    GetClientProvidersByIDs,
	"POST.client-provider":       PostClientProviders,
}
