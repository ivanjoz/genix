//go:build !js

package db

// ConnectParams is everything any server driver needs to reach its database. A driver reads only
// the fields it understands, so the entry points (main, exec scripts, the bench) build one value
// and never name a driver.
type ConnectParams struct {
	Host     string
	Port     int
	User     string
	Password string
	// Keyspace is the namespace every table lives in unless its schema names another.
	Keyspace string
	// MaxClusteringKey mirrors Scylla's max_clustering_key_restrictions_per_query.
	MaxClusteringKey int
	// FDBSocket is the unix socket of the fdbgate daemon (fdb driver only).
	FDBSocket string
	// YDBEndpoint is the gRPC address (host:port) and YDBDatabase the database path (ydb driver only).
	YDBEndpoint string
	YDBDatabase string
	// Scripting dials immediately and allows long queries, for the exec entry points that run
	// maintenance work. The server leaves it false: it connects on first use with tighter timeouts.
	Scripting bool
}

// connectedKeyspace is what Connect last received; MakeController qualifies table names with it.
var connectedKeyspace string
