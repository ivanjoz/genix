package finance

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.cajas":            GetCajas,
	"POST.cajas":           PostCajas,
	"GET.cashBank-movimientos": GetCajaMovimientos,
	"GET.cashBank-cuadres":     GetCajaCuadres,
	"POST.cashBank-cuadre":     PostCajaCuadre,
	"POST.cashBank-movimiento": PostMovimientoCaja,
}
