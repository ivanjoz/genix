package exec

import (
	"app/core"
	"app/db2"
)

func makeConnParams() db2.ConnParams {
	return db2.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	}
}

/*
func TestInsert(args *core.ExecArgs) core.FuncResponse {

	db2.TestInsert(makeConnParams())

	return core.FuncResponse{}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	db2.TestQuery(makeConnParams())

	return core.FuncResponse{}
}

func TestCBOR(args *core.ExecArgs) core.FuncResponse {

	db2.TestCBOR()

	return core.FuncResponse{}
}

func TestDeploy(args *core.ExecArgs) core.FuncResponse {

	db2.TestDeploy(makeConnParams())

	return core.FuncResponse{}
}
*/
