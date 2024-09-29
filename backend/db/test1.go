package db

import (
	"app/core"
	"fmt"
)

//Some docs
//https://medium.engineering/scylladb-implementation-lists-in-mediums-feature-store-part-2-905299c89392
//https://university.scylladb.com/courses/data-modeling/lessons/materialized-views-secondary-indexes-and-filtering/

type Usuario struct {
	CompanyID int32
	ID        int32
	Nombre    string
	Apellido  string
	Direccion string
	RolID     int16
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
func (e _u) RolID_() CoI16     { return CoI16{"rol_id"} }
func (e _u) Updated_() CoF64   { return CoF64{"updated"} }
func (e _u) Accesos_() CsI32   { return CsI32{"accesos_ids"} }
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
		HashIndexes:   [][]Column{{e.RolID_(), e.Edad_()}, {e.RolID_(), e.Accesos_()}},
		Views: []View{
			//{Cols: []Column{e.RolID_(), e.Accesos_()}},
			{Cols: []Column{e.RolID_(), e.Updated_()}, IntConcatRadix: []int8{10}},
			{Cols: []Column{e.RolID_(), e.Edad_(), e.Updated_()}, IntConcatRadix: []int8{4, 11}},
		},
	}
}

func TestQuery(params ConnParams) {
	MakeScyllaConnection(params)

	fmt.Println("Query 1")
	result := Select(func(q *Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			Where(col.CompanyID_().Equals(1)).
			Where(col.Nombre_().Equals("Carlos"))
	})

	core.Print(result.Records)

	fmt.Println("Query 2")
	result2 := Select(func(q *Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			// Where(col.RolID_().Equals(1)).
			Where(col.Accesos_().Contains(4))
	})

	core.Print(result2.Records)
}

func TestDeploy(params ConnParams) {

	MakeScyllaConnection(params)

	err1 := QueryExec(`DROP MATERIALIZED VIEW IF EXISTS genix.ztest_usuarios__rol_id_accesos_view`)
	if err1 != nil {
		fmt.Println("error:", err1)
	}

	DeployScylla(Usuario{})

	usuarios := getUsuariosData()
	err := InsertExclude(&usuarios)
	if err != nil {
		fmt.Println("Error al insertar::", err)
	}
}
