package exec

import (
	comercial "app/comercial/types"
	negocio "app/negocio/types"
	"app/core"
	"app/db"
	s "app/types"
	"fmt"
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
		makeDBController[s.Usuario](),
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
		makeDBController[DemoStruct](),
	}
}

// makeDBController creates a ScyllaController for db2 package types using generics.
// This function automatically handles queries for any db2 table type using the
// TableQueryInterface for clean, simple query building.
func makeDBController[T db.TableBaseInterface[E, T], E db.TableSchemaInterface[E]]() db.ScyllaControllerInterface {
	// Get the table struct instance
	schema := db.MakeSchema[T]()
	scyllaTable := db.MakeScyllaTable[T]()

	// Get table name and keyspace
	tableName := schema.Name
	keyspace := schema.Keyspace
	if keyspace == "" {
		keyspace = core.Env.DB_NAME
	}
	fullTableName := fmt.Sprintf("%s.%s", keyspace, tableName)

	contoller := db.ScyllaController[T, E]{
		TableName: fullTableName,
		Table:     db.ScyllaTable[T](scyllaTable),
		Schema:    schema,
	}
	return &contoller
}
