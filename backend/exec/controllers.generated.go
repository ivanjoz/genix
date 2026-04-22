package exec

import (
	comercial "app/comercial/types"
	configuracionTypes "app/configuracion/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	finanzasTypes "app/finanzas/types"
	logisticaTypes "app/logistica/types"
	negocio "app/negocio/types"
	negocioTypes "app/negocio/types"
	seguridadTypes "app/seguridad/types"
)

func MakeScyllaControllers() []db.ScyllaControllerInterface {
	return []db.ScyllaControllerInterface{

		makeDBController[negocioTypes.Product](),
		makeDBController[negocioTypes.Almacen](),
		makeDBController[negocioTypes.Sede](),
		makeDBController[logisticaTypes.ProductStock](),
		makeDBController[logisticaTypes.ProductStockV2](),
		makeDBController[logisticaTypes.ProductStockDetail](),
		makeDBController[logisticaTypes.ProductStockLot](),
		makeDBController[logisticaTypes.DeliveryOrderNote](),
		makeDBController[logisticaTypes.ProductSupply](),
		makeDBController[negocioTypes.PaisCiudad](),
		makeDBController[negocioTypes.GaleriaImagen](),
		makeDBController[negocioTypes.ListaCompartidaRegistro](),
		makeDBController[logisticaTypes.WarehouseProductMovement](),
		makeDBController[coreTypes.Usuario](),
		makeDBController[configuracionTypes.Empresa](),
		makeDBController[seguridadTypes.Perfil](),
		makeDBController[finanzasTypes.Caja](),
		makeDBController[finanzasTypes.CajaMovimiento](),
		makeDBController[finanzasTypes.CajaCuadre](),
		makeDBController[configuracionTypes.Parametros](),
		makeDBController[configuracionTypes.SystemParameters](),
		makeDBController[core.Cache](),
		makeDBController[comercial.SaleOrder](),
		makeDBController[comercial.SaleSummary](),
		makeDBController[db.CacheVersion](),
		makeDBController[negocio.ClientProvider](),
		makeDBController[core.CronAction](),
		makeDBController[coreTypes.UsageLog](),
		makeDBController[comercial.ProductSaleSummary](),
		makeDBController[DemoStruct](),
	}
}
