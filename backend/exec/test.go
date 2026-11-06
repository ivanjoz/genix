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
		Scripting:        true,
	}
}
