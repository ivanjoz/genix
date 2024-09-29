package db

import (
	"app/core"
)

//Some docs
//https://medium.engineering/scylladb-implementation-lists-in-mediums-feature-store-part-2-905299c89392

type Usuario struct {
	CompanyID int32
	ID        int32
	Nombre    string
	Apellido  string
	Direccion string
	Rol       string
	Edad      int32
	Updated   int64
	GruposIDs []int32
	Accesos   []int32
	Proyectos []string
	Peso      *float32
}

type _u = Usuario

func (e _u) CompanyID_() CoI32 { return CoI32{"company_id"} }
func (e _u) ID_() CoI32        { return CoI32{"id"} }
func (e _u) Nombre_() CoStr    { return CoStr{"nombre"} }
func (e _u) Apellido_() CoStr  { return CoStr{"apellido"} }
func (e _u) Direccion_() CoStr { return CoStr{"direccion"} }
func (e _u) Rol_() CoStr       { return CoStr{"rol"} }
func (e _u) Updated_() CoF64   { return CoF64{"updated"} }
func (e _u) Accesos_() CsI32   { return CsI32{"accesos"} }
func (e _u) Edad_() CsI32      { return CsI32{"edad"} }
func (e _u) Proyectos_() CsStr { return CsStr{"proyectos"} }
func (e _u) Peso_() CoF32      { return CoF32{"peso"} }
func (e _u) GruposIDs_() CsI32 { return CsI32{"grupos_ids"} }

func (e Usuario) GetSchema() TableSchema {
	return TableSchema{
		Name:          "ztest_usuarios",
		Partition:     e.CompanyID_(),
		Keys:          []Column{e.ID_()},
		GlobalIndexes: []Column{e.Edad_(), e.GruposIDs_()},
		LocalIndexes:  []Column{e.Nombre_()},
		HashIndexes:   [][]Column{{e.Rol_(), e.Edad_()}},
		Views: []TableView{
			{Cols: []Column{e.Rol_(), e.Accesos_()}},
			{Cols: []Column{e.Edad_(), e.Updated_()}, Int64ConcatRadix: 9},
		},
	}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	result := Select(func(q *Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			Where(col.Nombre_().Equals("hola")).
			Where(col.Updated_().Equals(1)).
			Where(col.Accesos_().Contains(4))
	})

	core.Print(result.Records)

	result2 := Select(func(q *Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			With([]Usuario{}...).Join(col.Nombre_(), col.Apellido_())
	})

	core.Print(result2.Records)

	return core.FuncResponse{}
}

func TestDeploy(params ConnParams) {

	MakeScyllaConnection(params)

	DeployScylla(Usuario{Nombre: "dadasd"})
}
