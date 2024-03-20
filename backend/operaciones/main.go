package operaciones

import (
	"app/core"
)

type ExecRouterType map[string]func(args *core.ExecArgs) core.FuncResponse

var ModuleHandlers = core.AppRouterType{
	"POST.productos-stock":    PostAlmacenStock,
	"GET.almacen-movimientos": GetAlmacenMovimientos,
}
