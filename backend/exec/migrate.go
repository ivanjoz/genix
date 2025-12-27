package exec

import (
	"app/core"
	"app/db2"
	s "app/types"
	"fmt"
)

func MakeScyllaControllers() []db2.ScyllaControllerInterface {
	return []db2.ScyllaControllerInterface{
		//v2 controllers
		makeDBController[s.Almacen](),
		makeDBController[s.Sede](),
		makeDBController[s.AlmacenProducto](),
		makeDBController[s.PaisCiudad](),
		makeDBController[s.GaleriaImagen](),
		makeDBController[s.ListaCompartidaRegistro](),
		makeDBController[s.AlmacenMovimiento](),
		makeDBController[s.Usuario](),
		makeDBController[s.Caja](),
		makeDBController[s.CajaMovimiento](),
		makeDBController[s.CajaCuadre](),
		makeDBController[s.Producto](),
		makeDBController[s.Increment](),
		makeDBController[DemoStruct](),
	}
}

// makeDBController creates a ScyllaController for db2 package types using generics.
// This function automatically handles queries for any db2 table type using the
// TableQueryInterface for clean, simple query building.
func makeDBController[T db2.TableBaseInterface[E, T], E db2.TableSchemaInterface[E]]() db2.ScyllaControllerInterface {
	// Get the table struct instance
	schema := db2.MakeSchema[T]()
	scyllaTable := db2.MakeScyllaTable[T]()

	// Get table name and keyspace
	tableName := schema.Name
	keyspace := schema.Keyspace
	if keyspace == "" {
		keyspace = core.Env.DB_NAME
	}
	fullTableName := fmt.Sprintf("%s.%s", keyspace, tableName)

	contoller := db2.ScyllaController[T, E]{
		TableName: fullTableName,
		Table:     db2.ScyllaTable[T](scyllaTable),
		Schema:    schema,
	}
	return &contoller
}
