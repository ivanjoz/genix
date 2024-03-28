package handlers

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.p-hello":              HelloWorld,
	"POST.p-user-login":        PostLogin,
	"GET.empresas":             GetEmpresas,
	"GET.empresa-parametros":   GetEmpresaParametros,
	"POST.empresa-parametros":  PostEmpresaParametros,
	"POST.empresa":             PostEmpresa,
	"GET.usuarios":             GetUsuarios,
	"POST.usuarios":            PostUsuarios,
	"POST.seguridad-accesos":   PostAcceso,
	"GET.seguridad-accesos":    GetAccesos,
	"POST.perfiles":            PostPerfiles,
	"GET.perfiles":             GetPerfiles,
	"GET.sedes-almacenes":      GetSedesAlmacenes,
	"GET.pais-ciudades":        GetPaisCiudades,
	"POST.sedes":               PostSedes,
	"POST.almacenes":           PostAlmacen,
	"GET.productos":            GetProductos,
	"POST.productos":           PostProductos,
	"POST.producto-image":      PostProductoImage,
	"GET.listas-compartidas":   GetListasCompartidas,
	"POST.listas-compartidas":  PostListasCompartidas,
	"POST.images":              PostImage,
	"GET.productos-stock":      GetProductosStock,
	"GET.p-demo-serialization": Demo1,
}
