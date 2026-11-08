package exec

import (
	"app/core"
	"app/db"
)

// makeConnParams is the connection every exec entry point uses; Scripting dials immediately and
// allows the long queries maintenance work runs.
func makeConnParams() db.ConnectParams {
	return db.ConnectParams{
		Host:             core.Env.DB_HOST,
		Port:             int(core.Env.DB_PORT),
		User:             core.Env.DB_USER,
		Password:         core.Env.DB_PASSWORD,
		Keyspace:         core.Env.DB_NAME,
		MaxClusteringKey: int(core.Env.MAX_CLUSTERING_KEY),
		FDBSocket:        core.Env.FDB_SOCKET,
		YDBEndpoint:      core.Env.YDB_ENDPOINT,
		YDBDatabase:      core.Env.YDB_DATABASE,
		OceanBase: db.OceanBaseParams{
			Host: core.Env.OCEANBASE_HOST, Port: core.Env.OCEANBASE_PORT,
			User: core.Env.OCEANBASE_USER, Password: core.Env.OCEANBASE_PASSWORD,
			Partitions: core.Env.OCEANBASE_PARTITIONS, MaxConns: core.Env.OCEANBASE_MAX_CONNS,
			Engine: core.Env.OCEANBASE_ENGINE,
		},
		Postgres: db.PostgresParams{
			Host: core.Env.POSTGRES_HOST, Port: core.Env.POSTGRES_PORT,
			User: core.Env.POSTGRES_USER, Password: core.Env.POSTGRES_PASSWORD,
			Database: core.Env.POSTGRES_DATABASE, MaxConns: core.Env.POSTGRES_MAX_CONNS,
		},
		MariaDB: db.MariaDBParams{
			Host: core.Env.MARIADB_HOST, Port: core.Env.MARIADB_PORT,
			User: core.Env.MARIADB_USER, Password: core.Env.MARIADB_PASSWORD, MaxConns: core.Env.MARIADB_MAX_CONNS,
			Socket: core.Env.MARIADB_SOCKET,
		},
		Scripting: true,
	}
}
