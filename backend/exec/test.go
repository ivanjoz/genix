package exec

import (
	"app/core"
	"app/db"
)

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

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	db.TestQuery(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	return core.FuncResponse{}
}

func TestCBOR(args *core.ExecArgs) core.FuncResponse {

	db.TestCBOR()

	return core.FuncResponse{}
}
