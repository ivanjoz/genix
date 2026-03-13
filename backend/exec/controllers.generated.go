package exec

import (
	comercial "app/comercial/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	negocio "app/negocio/types"
	s "app/types"
)

func MakeScyllaControllers() []db.ScyllaControllerInterface {
	return []db.ScyllaControllerInterface{

		makeDBController[s.Producto](),
		makeDBController[s.Almacen](),
		makeDBController[s.Sede](),
		makeDBController[s.AlmacenProducto](),
		makeDBController[s.PaisCiudad](),
		makeDBController[s.GaleriaImagen](),
		makeDBController[s.ListaCompartidaRegistro](),
		makeDBController[s.AlmacenMovimiento](),
		makeDBController[coreTypes.Usuario](),
		makeDBController[s.Empresa](),
		makeDBController[s.Perfil](),
		makeDBController[s.Caja](),
		makeDBController[s.CajaMovimiento](),
		makeDBController[s.CajaCuadre](),
		makeDBController[s.Parametros](),
		makeDBController[s.SystemParameters](),
		makeDBController[core.Cache](),
		makeDBController[comercial.SaleOrder](),
		makeDBController[comercial.SaleSummary](),
		makeDBController[db.CacheVersion](),
		makeDBController[negocio.Entity](),
		makeDBController[coreTypes.UsageLog](),
		makeDBController[DemoStruct](),
	}
}
