package exec

import (
	"app/core"
	. "app/db"
)

type Usuario struct {
	ID        int32
	Nombre    string
	Apellido  string
	Direccion string
	Edad      int32
	Updated   int64
	Accesos   []int32
	Proyectos []string
	Peso      *float32
}

type _U = Usuario

func (e _U) GetTableSchema() TableSchema {
	return TableSchema{
		Name:        "ztest_usuarios",
		Indexes:     []ColInfo{e.Edad_()},
		HashIndexes: [][]ColInfo{{e.Nombre_(), e.Apellidos_()}},
		Views: []TableView{
			{Cols: []ColInfo{e.Nombre_(), e.Edad_()}},
			{Cols: []ColInfo{e.Edad_(), e.Updated_()}, Int64ConcatRadix: 9},
		},
	}
}
func (e _U) ID_() CoI32        { return CoI32{"usuario_id", e.ID} }
func (e _U) Nombre_() CoStr    { return CoStr{"nombre", e.Nombre} }
func (e _U) Edad_() CoI32      { return CoI32{"edad", e.Edad} }
func (e _U) Apellidos_() CoStr { return CoStr{"apellidos", e.Apellido} }
func (e _U) Direccion_() CoStr { return CoStr{"direccion", e.Direccion} }
func (e _U) Updated_() CoI64   { return CoI64{"updated", e.Updated} }
func (e _U) Accesos_() CsI32   { return CsI32{"accesos", e.Accesos} }
func (e _U) Proyectos_() CsStr { return CsStr{"proyectos", e.Proyectos} }
func (e _U) Peso_() CpF32      { return CpF32{"peso", e.Peso} }

func TestQuery(args *core.ExecArgs) core.FuncResponse {
	query := Query[Usuario]{}
	records, err := query.
		Where(query.T.Nombre_().Equals("hola")).
		Where(query.T.Updated_().Equals(1)).
		Where(query.T.Accesos_().Contains(4)).Exec()

	if err != nil {
		core.Log(err)
	}
	core.Log(records)

	return core.FuncResponse{}
}
