package db

import (
	"fmt"
)

//Some docs
//https://medium.engineering/scylladb-implementation-lists-in-mediums-feature-store-part-2-905299c89392
//https://university.scylladb.com/courses/data-modeling/lessons/materialized-views-secondary-indexes-and-filtering/

type Perfil struct {
	Nombre string  `cbor:"1,keyasint,omitempty"`
	Param1 int32   `cbor:"2,keyasint,omitempty"`
	Param2 float32 `cbor:"3,keyasint,omitempty"`
}

type Permisos struct {
	Nombre string  `cbor:"1,keyasint,omitempty"`
	Param1 int32   `cbor:"2,keyasint,omitempty"`
	Param2 float32 `cbor:"3,keyasint,omitempty"`
}

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
	Perfil    Perfil
	Permisos  []Permisos
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
func (e _u) Perfil_() CoAny    { return CoAny{"perfil"} }
func (e _u) Permisos_() CoAny  { return CoAny{"permisos"} }

func (e Usuario) GetSchema() TableSchema {
	return TableSchema{
		Name:          "ztest_usuarios",
		Partition:     e.CompanyID_(),
		Keys:          []Coln{e.ID_()},
		GlobalIndexes: []Coln{e.Edad_(), e.GruposIDs_()},
		LocalIndexes:  []Coln{e.Nombre_(), e.GruposIDs_()},
		HashIndexes:   [][]Coln{{e.RolID_(), e.Edad_()}, {e.RolID_(), e.Accesos_()}},
		Views: []View{
			//{Cols: []Column{e.RolID_(), e.Accesos_()}},
			{Cols: []Coln{e.RolID_(), e.Updated_()}, ConcatI64: []int8{10}},
			{Cols: []Coln{e.RolID_(), e.Edad_(), e.Updated_()}, ConcatI64: []int8{4, 11}},
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

	fmt.Println(result.Records)
	/*
		fmt.Println("Query 2")
		result2 := Select(func(q *Query[Usuario], col Usuario) {
			q.Exclude(col.Apellido_()).
				// Where(col.RolID_().Equals(1)).
				Where(col.Accesos_().Contains(4))
		})

		core.Print(result2.Records)
	*/
}

func TestDeploy(params ConnParams) {

	MakeScyllaConnection(params)

	err1 := QueryExec(`DROP MATERIALIZED VIEW IF EXISTS genix.lista_compartida_registros__lista_id_view`)
	if err1 != nil {
		fmt.Println("error:", err1)
	}

	DeployScylla(Usuario{})

	usuarios := getUsuariosData()
	err := Insert(&usuarios)
	if err != nil {
		fmt.Println("Error al insertar::", err)
	}
}
