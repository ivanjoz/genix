package negocio

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.sedes-almacenes":           GetSedesAlmacenes,
	"GET.pais-ciudades":             GetPaisCiudades,
	"POST.sedes":                    PostSedes,
	"POST.almacenes":                PostAlmacen,
	"GET.productos":                 GetProductos,
	"GET.p-productos-ids":           GetProductosByIDs,
	"GET.p-productos-cms":           GetProductosCMS,
	"GET.p-productos-index":         GetProductsIndex,
	"GET.p-productos-index-delta":   GetProductsIndexDelta,
	"POST.productos":                PostProductos,
	"POST.producto-image":           PostProductoImage,
	"POST.producto-categoria-image": PostProductoCategoriaImage,
	"GET.listas-compartidas":        GetListasCompartidas,
	"POST.listas-compartidas":       PostListasCompartidas,
	"POST.images":                   PostImage,
	"POST.galeria-image":            PostGaleriaImage,
	"GET.galeria-images":            GetGaleriaImages,
	"GET.client-provider":           GetClientProviders,
	"POST.client-provider":          PostClientProviders,
}
