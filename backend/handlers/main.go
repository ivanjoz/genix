package handlers

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.p-hello":            HelloWorld,
	"POST.p-user-login":      PostLogin,
	"GET.empresas":           GetEmpresas,
	"GET.usuarios":           GetUsuarios,
	"POST.usuarios":          PostUsuarios,
	"POST.seguridad-accesos": PostAcceso,
	"GET.seguridad-accesos":  GetAccesos,
	"POST.perfiles":          PostPerfiles,
	"GET.perfiles":           GetPerfiles,
	"GET.sedes-almacenes":    GetSedesAlmacenes,
	"GET.pais-ciudades":      GetPaisCiudades,
	"POST.sedes":             PostSedes,
	"POST.almacenes":         PostAlmacen,
	"GET.productos":          GetProductos,
	"POST.productos":         PostProductos,
	"GET.listas-compartidas": GetListasCompartidas,
	"POST.images":            PostImage,
}
