package exec

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
)

func makeConnParams() scylla.ConnParams {
	return scylla.ConnParams{
		Host:             core.Env.DB_HOST,
		Port:             int(core.Env.DB_PORT),
		User:             core.Env.DB_USER,
		Password:         core.Env.DB_PASSWORD,
		Keyspace:         core.Env.DB_NAME,
		MaxClusteringKey: int(core.Env.MAX_CLUSTERING_KEY),
	}
}

/*
func TestInsert(args *core.ExecArgs) core.FuncResponse {

	scylla.TestInsert(makeConnParams())

	return core.FuncResponse{}
}

func TestQuery(args *core.ExecArgs) core.FuncResponse {

	scylla.TestQuery(makeConnParams())

	return core.FuncResponse{}
}

func TestCBOR(args *core.ExecArgs) core.FuncResponse {

	scylla.TestCBOR()

	return core.FuncResponse{}
}

func TestDeploy(args *core.ExecArgs) core.FuncResponse {

	scylla.TestDeploy(makeConnParams())

	return core.FuncResponse{}
}
*/
