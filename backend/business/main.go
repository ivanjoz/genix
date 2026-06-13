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
	"POST.images":                  PostImage,
	"GET.image-id-counter":         GetImageIdCounter,
	"POST.galeria-image":           PostGaleriaImage,
	"GET.galeria-images":           GetGaleriaImages,
	"GET.client-provider":          GetClientProviders,
	"GET.client-provider-ids":      GetClientProvidersByIDs,
	"POST.client-provider":         PostClientProviders,
}
