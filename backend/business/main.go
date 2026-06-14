package business

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.sedes-almacenes":          GetSedesAlmacenes,
	"GET.pais-ciudades":            GetPaisCiudades,
	"POST.sedes":                   PostSedes,
	"POST.almacenes":               PostAlmacen,
	"GET.productos":                GetProductos,
	"GET.p-productos-ids":          GetProductosByIDs,
	"GET.p-product-text-search":    GetProductTextSearch,
	"GET.p-productos-ecommerce":    GetProductsEcommerce,
	"POST.productos":               PostProducts,
	"POST.product-image":           PostProductoImage,
	"POST.product-categoria-image": PostProductoCategoriaImage,
	"GET.listas-compartidas":       GetListasCompartidas,
	"POST.listas-compartidas":      PostListasCompartidas,
	"GET.image-id-counter":         GetImageIdCounter,
	"GET.image-assets":             GetImageAssets,
	"POST.gallery-image":           PostGalleryImage,
	"GET.gallery-images":           GetGalleryImages,
	"GET.client-provider":          GetClientProviders,
	"GET.client-provider-ids":      GetClientProvidersByIDs,
	"POST.client-provider":         PostClientProviders,
}
