//go:build !js

package db

// ConnectParams is everything any server driver needs to reach its database. A driver reads only
// the fields it understands, so the entry points (main, exec scripts, the bench) build one value
// and never name a driver.
type ConnectParams struct {
	// Keyspace is the namespace every table lives in unless its schema names another: the MariaDB
	// database, a PostgreSQL schema.
	Keyspace string
	// FDBSocket is the unix socket of the fdbgate daemon (fdb driver only).
	FDBSocket string
	// YDBEndpoint is the gRPC address (host:port) and YDBDatabase the database path (ydb driver only).
	YDBEndpoint string
	YDBDatabase string
	// OceanBase is the MySQL-protocol server of the oceanbase driver; its database is Keyspace.
	OceanBase OceanBaseParams
	// Postgres is the server of the postgres driver; Keyspace is a schema in its Database.
	Postgres PostgresParams
	// MariaDB is the server of the mariadb driver; Keyspace is its database.
	MariaDB MariaDBParams
	// Scripting dials immediately and allows long queries, for the exec entry points that run
	// maintenance work. The server leaves it false: it connects on first use with tighter timeouts.
	Scripting bool
}

// OceanBaseParams is read only by a `-tags oceanbase` build (docs/OCEANBASE_PLAN.md §13). User carries
// the tenant (`root@test`). Partitions is the KEY(company_id) partition count of every table, 1 for
// none; MaxConns caps the connection pool, 0 for the driver default. Engine "seekdb" deploys seekdb's
// subset (obdriver.EngineSeekDB).
type OceanBaseParams struct {
	Host       string
	Port       int
	User       string
	Password   string
	Partitions int
	MaxConns   int
	Engine     string
}

// PostgresParams is read only by a `-tags postgres` build, the sale-flow benchmarks
// (seeds/loadtest/plan.md). Database is the PostgreSQL database ("postgres" when empty); MaxConns caps
// the connection pool, 0 for the driver default.
type PostgresParams struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	MaxConns int
}

// MariaDBParams is read by the default (untagged) build: MariaDB is the database. MaxConns caps the connection pool, 0 for the driver default.
// Socket, when set, is dialed instead of Host:Port.
type MariaDBParams struct {
	Host     string
	Port     int
	User     string
	Password string
	MaxConns int
	Socket   string
}

// connectedKeyspace is what Connect last received; MakeController qualifies table names with it.
var connectedKeyspace string
