package exec

import (
	"app/core"
	"app/db"
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

func (e _u) CompanyID_() db.CoI32 { return db.CoI32{"company_id"} }
func (e _u) ID_() db.CoI32        { return db.CoI32{"id"} }
func (e _u) Nombre_() db.CoStr    { return db.CoStr{"nombre"} }
func (e _u) Apellido_() db.CoStr  { return db.CoStr{"apellido"} }
func (e _u) Direccion_() db.CoStr { return db.CoStr{"direccion"} }
func (e _u) Rol_() db.CoStr       { return db.CoStr{"rol"} }
func (e _u) Updated_() db.CoF64   { return db.CoF64{"updated"} }
func (e _u) Accesos_() db.CsI32   { return db.CsI32{"accesos"} }
func (e _u) Edad_() db.CsI32      { return db.CsI32{"edad"} }
func (e _u) Proyectos_() db.CsI32 { return db.CsI32{"proyectos"} }
func (e _u) Peso_() db.CoF32      { return db.CoF32{"peso"} }

func (e Usuario) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:          "ztest_usuarios",
		Partition:     e.CompanyID_(),
		Keys:          []db.Column{e.ID_()},
		GlobalIndexes: []db.Column{e.Edad_()},
		LocalIndexes:  []db.Column{e.Nombre_()},
		HashIndexes:   [][]db.Column{{e.Rol_(), e.Edad_()}},
		Views: []db.TableView{
			{Cols: []db.Column{e.Rol_(), e.Accesos_()}},
			{Cols: []db.Column{e.Edad_(), e.Updated_()}, Int64ConcatRadix: 9},
		},
	}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	result := db.Select(func(q *db.Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			Where(col.Nombre_().Equals("hola")).
			Where(col.Updated_().Equals(1)).
			Where(col.Accesos_().Contains(4))
	})

	core.Print(result.Records)

	result2 := db.Select(func(q *db.Query[Usuario], col Usuario) {
		q.Exclude(col.Apellido_()).
			With([]Usuario{}...).Join(col.Nombre_(), col.Apellido_())
	})

	core.Print(result2.Records)

	return core.FuncResponse{}
}

func TestDeploy(args *core.ExecArgs) core.FuncResponse {

	db.TestDeploy(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	return core.FuncResponse{}
}
