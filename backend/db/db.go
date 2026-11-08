// Package db is this project's single entry point to the ORM. Application code
// imports only this package.
//
// Nothing in this file names a database. Every declaration is either an alias to
// genix-orm/db (the driver-agnostic layer) or a thin generic wrapper over it —
// wrappers exist only because Go cannot alias a generic function. The driver is
// chosen per build target in driver_mariadb.go (server) and driver_js.go (browser).
package db

import (
	"github.com/ivanjoz/genix-orm/dataframe"
	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/db/cache"
)

// Schema and Record are the constraints every wrapper in db.go is written against.
// Record does not name a driver: it takes the executor as a parameter, which Go
// infers from the record type. That is what keeps db.go driver-free.
type (
	Schema[TableT any]                     = orm.TableSchemaInterface[TableT]
	Record[TableT any, RecordT any, D any] = orm.RecordWithExecutor[TableT, RecordT, D]
)

// ─── Schema declaration ────────────────────────────────────────────────────────

type (
	TableSchema         = orm.TableSchema
	Index               = orm.Index
	Coln                = orm.Coln
	TableHandle         = orm.TableHandle
	ColumnStatement     = orm.ColumnStatement
	// Transaction is the transaction a RunTransaction body writes through.
	Transaction = orm.Transaction
	TableInfo           = orm.TableInfo
	GenericRecordSchema = orm.GenericRecordSchema
	GenericRecord       = orm.GenericRecord
	ColumnInfo          = orm.ColumnInfo
	IColInfo            = orm.IColInfo
	TextHit             = orm.TextHit
	TextSearch          = orm.TextSearch
	GroupIndexCache     = orm.GroupIndexCache
	KeyParser           = orm.KeyParser
	FixedValues         = orm.FixedValues
	DataFrame           = orm.DataFrame
	CompactOn           = orm.CompactOn
	ComputedCol         = orm.ComputedCol
)

type (
	Col[TableT orm.TableHandle, ValueT any]     = orm.Col[TableT, ValueT]
	ColSlice[TableT orm.TableHandle, ElemT any] = orm.ColSlice[TableT, ElemT]
	RecordGroup[RecordT any]                    = orm.RecordGroup[RecordT]
	Executor[TableT any, RecordT any]           = orm.Executor[TableT, RecordT]
)

const (
	TypeGlobalIndex       = orm.TypeGlobalIndex
	TypeLocalIndex        = orm.TypeLocalIndex
	TypeCacheGroupFromKey = orm.TypeCacheGroupFromKey
	TypeView              = orm.TypeView
	TypeViewTable         = orm.TypeViewTable
	TypeDelta             = orm.TypeDelta
	TypeCacheGroup        = orm.TypeCacheGroup
	TypeCacheGroupS3      = orm.TypeCacheGroupS3
	TypeCacheGroupNoIndex = orm.TypeCacheGroupNoIndex
	TypeUniqueIndex       = orm.TypeUniqueIndex
)

// Cols returns columns as the slice required by schema declarations.
var Cols = orm.Cols

// ─── Compiled tables, admin and the name registry ──────────────────────────────

type (
	Table      = orm.Table
	Controller = orm.Controller
)

var (
	RegisterTableFactory = orm.RegisterTableFactory
	ResolveTableByName   = orm.ResolveTableByName
	RegisteredTableNames = orm.RegisteredTableNames
)

// ─── Name-addressed access ─────────────────────────────────────────────────────
//
// The surface for code that identifies a table by a runtime string instead of by
// its Go type: admin tooling, scripts and the fn-db console. Reads and writes
// still go through the ORM, so nothing here bypasses a managed column.

type (
	FilterSpec        = orm.FilterSpec
	QuerySpec         = orm.QuerySpec
	TableDescription  = orm.TableDescription
	ColumnDescription = orm.ColumnDescription
)

var (
	RegisterControllerFactory = orm.RegisterControllerFactory
	ResolveControllerByName   = orm.ResolveControllerByName
	RegisteredControllerNames = orm.RegisteredControllerNames
)

// ─── Reads ─────────────────────────────────────────────────────────────────────
//
// Every wrapper forwards its driver type parameter, so the driver is still
// inferred from the record type and no call site needs explicit type arguments.

// Query starts a read into refSlice and returns the table struct to build
// predicates on. Chain .Via(executor) to run this one query on another database.
func Query[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	refSlice *[]RecordT,
) *TableT {
	return orm.Query[RecordT, TableT, D](refSlice)
}

// QueryIndexGroup starts a grouped read, bucketed by index-group hash.
func QueryIndexGroup[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	refSlice *[]RecordGroup[RecordT],
) *TableT {
	return orm.QueryIndexGroup[RecordT, TableT, D](refSlice)
}

// TableOf returns a bound table struct for writes, with no read destination.
func TableOf[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]]() *TableT {
	return orm.TableOf[RecordT, TableT, D]()
}

// QueryCachedIDs returns, per partition, the requested records whose slot moved since
// the tick the client holds.
func QueryCachedIDs[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	request CachedIDsRequest,
) ([]CachedPartition[RecordT], error) {
	return orm.QueryCachedIDs[RecordT, TableT, D](request)
}

// ─── Client cache protocols (genix-orm/db/cache) ───────────────────────────────

type (
	// CachedIDsRequest is a decoded by-IDs request (the "cc" query parameter).
	CachedIDsRequest = cache.CachedIDsRequest
	// DeltaCursor is what a client holds of one delta response key: "C.V", or "C.V.r" to resend
	// the unsettled slice (genix-orm db/cache delta_window.go).
	DeltaCursor = cache.DeltaWindowCursor
)

// CachedPartition is the by-IDs response: the records of one partition that moved.
type CachedPartition[RecordT any] = cache.CachedPartition[RecordT]

var (
	// SetFileStore installs the file store holding the by-IDs slot files.
	SetFileStore = cache.SetFileStore
	// SetRecordSnapshotsEnabled turns the delta and cache-group snapshot files on or off.
	SetRecordSnapshotsEnabled = cache.SetRecordSnapshotsEnabled
	// IsRecordSnapshotKey says a file store key is a snapshot (a droppable cache of a read), not a
	// delta log or by-IDs file.
	IsRecordSnapshotKey = cache.IsRecordSnapshotKey
	// FileStore is the installed store. The agent keeps its turn checkpoints in it, under its own
	// prefix, so it needs no S3 client of its own.
	FileStore = cache.FileStore
	// DecodeCachedIDsRequest reads the "cc" query parameter of a by-IDs request.
	DecodeCachedIDsRequest = cache.DecodeCachedIDsRequest
	// ParseDeltaCursor reads a delta cursor query parameter.
	ParseDeltaCursor = cache.ParseDeltaWindowCursor
	// DeltaReplaceMetadata is the Metadata of a route not backed by a TypeDelta index: one record
	// carrying it makes the client replace its whole list with the response.
	DeltaReplaceMetadata = cache.DeltaReplaceMetadata
)

// ─── Writes ────────────────────────────────────────────────────────────────────

// Insert writes new records: the caller promises none is stored under its keys. PostgreSQL and
// MariaDB skip the read before writing and fail on a stored key.
func Insert[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToExclude ...Coln,
) error {
	return orm.Insert[RecordT, TableT, D](records, columnsToExclude...)
}

func InsertOne[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	record RecordT, columnsToExclude ...Coln,
) error {
	return orm.InsertOne[RecordT, TableT, D](record, columnsToExclude...)
}

// Upsert writes records whether or not they are stored: a stored record is overwritten.
func Upsert[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToExclude ...Coln,
) error {
	return orm.Upsert[RecordT, TableT, D](records, columnsToExclude...)
}

func UpsertOne[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	record RecordT, columnsToExclude ...Coln,
) error {
	return orm.UpsertOne[RecordT, TableT, D](record, columnsToExclude...)
}

func Update[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToInclude ...Coln,
) error {
	return orm.Update[RecordT, TableT, D](records, columnsToInclude...)
}

// ─── Transactions ──────────────────────────────────────────────────────────────

// RunTransaction runs body in one transaction on PostgreSQL and MariaDB: LockByKeys locks what it
// reads, InsertIn and UpdateIn commit together. A deadlock or a duplicate key reruns body, so body
// derives its writes from its own reads; a record that may not exist yet is created with InsertIn, so
// the body that loses the race to create it reruns and finds it. On a driver without transactions
// each call runs on its own.
func RunTransaction(body func(transaction *Transaction) error) error {
	return orm.RunTransaction(body)
}

// LockByKeys replaces records (which carry their partition and keys) with their stored versions,
// locked until the transaction ends. A record with nothing stored is dropped.
func LockByKeys[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	transaction *Transaction, records *[]RecordT,
) error {
	return orm.LockByKeys[RecordT, TableT, D](transaction, records)
}

// InsertIn is Insert inside RunTransaction.
func InsertIn[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	transaction *Transaction, records *[]RecordT, columnsToExclude ...Coln,
) error {
	return orm.InsertIn[RecordT, TableT, D](transaction, records, columnsToExclude...)
}

// UpdateIn is Update inside RunTransaction.
func UpdateIn[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	transaction *Transaction, records *[]RecordT, columnsToInclude ...Coln,
) error {
	return orm.UpdateIn[RecordT, TableT, D](transaction, records, columnsToInclude...)
}

// UpsertIn is Upsert inside RunTransaction.
func UpsertIn[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	transaction *Transaction, records *[]RecordT, columnsToExclude ...Coln,
) error {
	return orm.UpsertIn[RecordT, TableT, D](transaction, records, columnsToExclude...)
}

// InsertUpdateIncludeIn is InsertUpdateInclude inside RunTransaction.
func InsertUpdateIncludeIn[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	transaction *Transaction, records *[]RecordT, isInsert func(record *RecordT) bool,
	columnsToIncludeUpdate []Coln, columnsToExcludeInsert ...Coln,
) error {
	return orm.InsertUpdateIncludeIn[RecordT, TableT, D](transaction, records, isInsert, columnsToIncludeUpdate, columnsToExcludeInsert...)
}

func UpdateOne[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	record RecordT, columnsToInclude ...Coln,
) error {
	return orm.UpdateOne[RecordT, TableT, D](record, columnsToInclude...)
}

func UpdateExclude[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToExclude ...Coln,
) error {
	return orm.UpdateExclude[RecordT, TableT, D](records, columnsToExclude...)
}

// InsertUpdate writes two already-split groups in one batch.
func InsertUpdate[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	recordsForInsert *[]RecordT, recordsForUpdate *[]RecordT,
	columnsToIncludeUpdate []Coln, columnsToExcludeInsert ...Coln,
) error {
	return orm.InsertUpdate[RecordT, TableT, D](
		recordsForInsert, recordsForUpdate, columnsToIncludeUpdate, columnsToExcludeInsert...)
}

// InsertUpdateInclude splits records with isInsert, listing updated columns explicitly.
func InsertUpdateInclude[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, isInsert func(record *RecordT) bool,
	columnsToIncludeUpdate []Coln, columnsToExcludeInsert ...Coln,
) error {
	return orm.InsertUpdateInclude[RecordT, TableT, D](
		records, isInsert, columnsToIncludeUpdate, columnsToExcludeInsert...)
}

// InsertUpdateExclude splits records with isInsert, listing updated columns by exclusion.
func InsertUpdateExclude[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, isInsert func(record *RecordT) bool,
	columnsToExcludeUpdate []Coln, columnsToExcludeInsert ...Coln,
) error {
	return orm.InsertUpdateExclude[RecordT, TableT, D](
		records, isInsert, columnsToExcludeUpdate, columnsToExcludeInsert...)
}

// Merge reads the existing rows first, so callers can decide per record whether an
// update is needed and mutate new records before insert.
func Merge[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToExcludeUpdate []Coln,
	onUpdate func(previous, current *RecordT) bool, onInsert func(record *RecordT),
) error {
	return orm.Merge[RecordT, TableT, D](records, columnsToExcludeUpdate, onUpdate, onInsert)
}

// ─── Text search ───────────────────────────────────────────────────────────────

// SetTextSearchBackend installs the engine that holds every table's text index.
var SetTextSearchBackend = orm.SetTextSearchBackend

// SearchTextIDs returns the active records of one partition that match query, best first.
func SearchTextIDs[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	partition int32, query string, limit, offset int,
) ([]TextHit, error) {
	return orm.SearchTextIDs[RecordT, TableT, D](partition, query, limit, offset)
}

// SearchText fills refSlice with the matching records, in the order of the returned hits.
func SearchText[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	refSlice *[]RecordT, partition int32, query string, limit, offset int,
) ([]TextHit, error) {
	return orm.SearchText[RecordT, TableT, D](refSlice, partition, query, limit, offset)
}

// ─── DataFrames (genix-orm/dataframe) ──────────────────────────────────────────

type (
	// FrameRow is one row of a DataFrame file: Key holds the frame's Keys and Rows columns.
	FrameRow[RecordT any] = orm.FrameRow[RecordT]
	// FrameQuery selects the files of one partition of a DataFrame.
	FrameQuery[TableT any, RecordT any] = orm.FrameQuery[TableT, RecordT]
)

// QueryFrame reads the rows of one partition of a DataFrame, as of its last compaction unless Fresh.
func QueryFrame[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	partition int32, frameName string,
) *FrameQuery[TableT, RecordT] {
	return orm.QueryFrame[RecordT, TableT, D](partition, frameName)
}

// ToFrames declares a DataFrame by a method of the record (DataFrame.ToFrames): it appends the
// record's lines to the slice it is given and returns it, as structs whose fields are tagged
// frame:"key", frame:"row" or frame:"sum". It must be pure: it is called on the record before and
// after each write. A rebuild hands it one reused buffer, so it must not keep the slice.
func ToFrames[R any, F any](method func(*R, []F) []F) *dataframe.Projection {
	return orm.ToFrames(method)
}

// Compute binds a computed column (TableSchema.Computed) to a record method:
// db.Compute((*SaleOrder).DetailProductIDs). It must be pure: it is called on every write.
func Compute[RecordT any, ValueT any](method func(*RecordT) ValueT) *orm.ComputedValue {
	return orm.Compute(method)
}

// ─── Table compilation and helpers ─────────────────────────────────────────────

// MakeTable compiles a table's metadata, panicking on an invalid schema. Generated
// registry code and deploy tooling use it to force validation early.
func MakeTable[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]]() Table {
	return orm.MakeTable[RecordT, TableT, D]()
}

// MakeSchema returns a table's declared schema without compiling it.
func MakeSchema[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]]() TableSchema {
	return orm.MakeSchema[RecordT, TableT]()
}

// PackedKeyComponent is one decoded slot of a KeyIntPacking ID.
type PackedKeyComponent = orm.PackedKeyComponent

// DecodePackedKey splits a KeyIntPacking ID into its column values, for debugging:
// db.DecodePackedKey[types.CashBankMovement](movement.ID).
func DecodePackedKey[RecordT orm.TableBaseInterface[TableT, RecordT], TableT Schema[TableT]](packedKey int64) []PackedKeyComponent {
	return orm.DecodeKeyIntPacking(MakeSchema[RecordT, TableT](), packedKey)
}

var (
	// MakeKeyConcat joins values into the deterministic string a KeyConcatenated
	// column stores.
	MakeKeyConcat = orm.MakeKeyConcat
	// GetAutoincrementID reserves consecutive IDs for an arbitrary counter key.
	GetAutoincrementID = orm.GetAutoincrementID
	// GetBlockAutoincrementID reserves from the server's block of the counter: one write per
	// block, gaps on restart. Only for counters that may skip values.
	GetBlockAutoincrementID = orm.GetBlockAutoincrementID
	// QueryCachedGenericByIDs resolves IDs to the flat GenericRecord shape for any
	// table that opted in through TableSchema.GenericRecord.
	QueryCachedGenericByIDs = orm.QueryCachedGenericByIDs
	// SetDebugLogging raises the ORM's log verbosity: 0 off, 1 statements, 2 full.
	SetDebugLogging = orm.SetDebugLogging
)

// ─── Effective clock ───────────────────────────────────────────────────────────

// The ORM stamps created/updated itself, so the clock has to live below both the ORM and the
// application for a backdated write to come out consistent. core wraps these as core.Now() and
// friends; application code should use those instead of reaching in here.
var (
	// Now is the effective wall clock: the real one, or the frozen instant set by
	// GENIX_HISTORICAL_UNIX / SetHistoricalUnix.
	Now = orm.Now
	// SetHistoricalUnix freezes the process clock at a unix second; 0 restores the real clock.
	SetHistoricalUnix = orm.SetHistoricalUnix
	// HistoricalUnix reports the active override, or 0 when the real clock is in use.
	HistoricalUnix = orm.HistoricalUnix
)

// HistoricalUnixEnvVar names the environment variable that freezes the clock at boot.
const HistoricalUnixEnvVar = orm.HistoricalUnixEnvVar
