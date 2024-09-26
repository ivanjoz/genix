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

type UsuarioSchema struct {
	CompanyID db.Col[int32]      `db:"company_id"`
	ID        db.Col[int32]      `db:"id"`
	Edad      db.Col[int32]      `db:"edad"`
	Nombre    db.Col[string]     `db:"nombre"`
	Apellido  db.Col[string]     `db:"apellido"`
	Accesos   db.ColSlice[int32] `db:"accesos"`
	Updated   db.Col[int64]      `db:"updated"`
	Rol       db.Col[string]     `db:"rol"`
}

func (e UsuarioSchema) GetSchema() db.TableSchema[Usuario] {
	/*
		e1 := UsuarioSchema{}
		hola := &e1.Edad
		hola.SetName("dasda")
	*/

	return db.TableSchema[Usuario]{
		StructType:    Usuario{},
		Name:          "ztest_usuarios",
		Partition:     e.CompanyID,
		PrimaryKey:    e.ID,
		GlobalIndexes: []db.Column{e.Edad},
		LocalIndexes:  []db.Column{e.Nombre},
		HashIndexes:   [][]db.Column{{e.Rol, e.Edad}},
		Views: []db.TableView{
			{Cols: []db.Column{e.Rol, e.Accesos}},
			{Cols: []db.Column{e.Edad, e.Updated}, Int64ConcatRadix: 9},
		},
	}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	result := db.QuerySelect(func(q *db.Query[Usuario], col UsuarioSchema) {
		q.Exclude(col.Apellido).
			Where(col.Nombre.Equals("hola")).
			Where(col.Updated.Equals(1)).
			Where(col.Accesos.Contains(4))
	})

	core.Print(result.Records)

	result2 := db.QuerySelect(func(q *db.Query[Usuario], col UsuarioSchema) {
		q.Exclude(col.Apellido).
			With([]Usuario{}...).Join(col.Nombre, col.Apellido)
	})

	core.Print(result2.Records)

	return core.FuncResponse{}
}

func TestDeploy(args *core.ExecArgs) core.FuncResponse {

	db.MakeScyllaConnection(db.DBConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	db.DeployScylla(UsuarioSchema{})

	return core.FuncResponse{}
}
