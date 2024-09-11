package exec

import (
	"app/core"
	. "app/db"
)

type Usuario struct {
	CompanyID int32
	ID        int32
	Nombre    string
	Apellido  string
	Direccion string
	Rol       string
	Edad      int32
	Updated   int64
	Accesos   []int32
	Proyectos []string
	Peso      *float32
}

type _u = Usuario

func (e _u) GetTableSchema() TableSchema {
	return TableSchema{
		Name:          "ztest_usuarios",
		Partition:     e.CompanyID_(),
		GlobalIndexes: []ColInfo{e.Edad_()},
		LocalIndexes:  []ColInfo{e.Nombre_()},
		HashIndexes:   [][]ColInfo{{e.Rol_(), e.Edad_()}},
		Views: []TableView{
			{Cols: []ColInfo{e.Nombre_(), e.Edad_()}},
			{Cols: []ColInfo{e.Edad_(), e.Updated_()}, Int64ConcatRadix: 9},
		},
	}
}

func (e _u) CompanyID_() CoI32 { return CoI32{"company_id", e.CompanyID} }
func (e _u) ID_() CoI32        { return CoI32{"usuario_id", e.ID} }
func (e _u) Nombre_() CoStr    { return CoStr{"nombre", e.Nombre} }
func (e _u) Rol_() CoStr       { return CoStr{"rol", e.Rol} }
func (e _u) Edad_() CoI32      { return CoI32{"edad", e.Edad} }
func (e _u) Apellidos_() CoStr { return CoStr{"apellidos", e.Apellido} }
func (e _u) Direccion_() CoStr { return CoStr{"direccion", e.Direccion} }
func (e _u) Updated_() CoI64   { return CoI64{"updated", e.Updated} }
func (e _u) Accesos_() CsI32   { return CsI32{"accesos", e.Accesos} }
func (e _u) Proyectos_() CsStr { return CsStr{"proyectos", e.Proyectos} }
func (e _u) Peso_() CpF32      { return CpF32{"peso", e.Peso} }

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	result := QuerySelect(func(query *Query[Usuario], t *Usuario) {
		query.
			Exclude(t.Peso_(), t.Updated_()).
			Where(t.Nombre_().Equals("hola")).
			Where(t.Updated_().Equals(1)).
			Where(t.Accesos_().Contains(4))
	})

	core.Print(result.Records)

	return core.FuncResponse{}
}
