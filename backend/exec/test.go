package exec

import (
	"app/core"
	"app/db"
)

func makeConnParams() db.ConnParams {
	return db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	}
}

/*
func TestInsert(args *core.ExecArgs) core.FuncResponse {

	db.TestInsert(makeConnParams())

	return core.FuncResponse{}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	db.TestQuery(makeConnParams())

	return core.FuncResponse{}
}

func TestCBOR(args *core.ExecArgs) core.FuncResponse {

	db.TestCBOR()

	return core.FuncResponse{}
}

func TestDeploy(args *core.ExecArgs) core.FuncResponse {

	db.TestDeploy(makeConnParams())

	return core.FuncResponse{}
}
*/
