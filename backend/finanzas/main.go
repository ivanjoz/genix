package finanzas

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.cajas":            GetCajas,
	"POST.cajas":           PostCajas,
	"GET.caja-movimientos": GetCajaMovimientos,
	"GET.caja-cuadres":     GetCajaCuadres,
	"POST.caja-cuadre":     PostCajaCuadre,
	"POST.caja-movimiento": PostMovimientoCaja,
}
