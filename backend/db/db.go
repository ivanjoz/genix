// Package db is this project's single entry point to the ORM. Application code
// imports only this package.
//
// Nothing in this file names a database. Every declaration is either an alias to
// genix-orm/db (the driver-agnostic layer) or a thin generic wrapper over it —
// wrappers exist only because Go cannot alias a generic function. The driver is
// chosen in driver.go, which is the one file to edit to switch databases.
package db

import (
	orm "github.com/ivanjoz/genix-orm/db"
)

// ─── Schema declaration ────────────────────────────────────────────────────────

type (
	TableSchema         = orm.TableSchema
	Index               = orm.Index
	Coln                = orm.Coln
	ColumnStatement     = orm.ColumnStatement
	TableInfo           = orm.TableInfo
	GenericRecordSchema = orm.GenericRecordSchema
	GenericRecord       = orm.GenericRecord
	IDUpdatedVersion    = orm.IDUpdatedVersion
	ColumnInfo          = orm.ColumnInfo
	IColInfo            = orm.IColInfo
	IDWeight            = orm.IDWeight
	GroupIndexCache     = orm.GroupIndexCache
	KeyParser           = orm.KeyParser
	FixedValues         = orm.FixedValues
)

type (
	Col[TableT orm.TableInterface[TableT], ValueT any]     = orm.Col[TableT, ValueT]
	ColSlice[TableT orm.TableInterface[TableT], ElemT any] = orm.ColSlice[TableT, ElemT]
	RecordGroup[RecordT any]                               = orm.RecordGroup[RecordT]
	Executor[TableT any, RecordT any]                      = orm.Executor[TableT, RecordT]
)

const (
	TypeGlobalIndex    = orm.TypeGlobalIndex
	TypeLocalIndex     = orm.TypeLocalIndex
	TypeInheritFromKey = orm.TypeInheritFromKey
	TypeView           = orm.TypeView
	TypeViewTable      = orm.TypeViewTable
	TypeDelta          = orm.TypeDelta
)

// Cols returns columns as the slice required by schema declarations.
var Cols = orm.Cols

// ─── Compiled tables, admin and the name registry ──────────────────────────────

type (
	Table      = orm.Table
	Controller = orm.Controller
	CSVResult  = orm.CSVResult
)

var (
	RegisterTableFactory = orm.RegisterTableFactory
	ResolveTableByName   = orm.ResolveTableByName
	RegisteredTableNames = orm.RegisteredTableNames
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

// QueryCachedIDs resolves records by ID, skipping any whose client slot version
// still matches the server.
func QueryCachedIDs[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	refSlice *[]RecordT, cachedIDs []IDUpdatedVersion,
) error {
	return orm.QueryCachedIDs[RecordT, TableT, D](refSlice, cachedIDs)
}

// ─── Writes ────────────────────────────────────────────────────────────────────

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

func Update[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	records *[]RecordT, columnsToInclude ...Coln,
) error {
	return orm.Update[RecordT, TableT, D](records, columnsToInclude...)
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

// SearchTextIDs returns matching record IDs ranked by weight.
func SearchTextIDs[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	partition int32, query string, statusGroup int8, limit int,
) ([]IDWeight, error) {
	return orm.SearchTextIDs[RecordT, TableT, D](partition, query, statusGroup, limit)
}

// SearchText fills refSlice with the matching records and returns their weights.
func SearchText[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	refSlice *[]RecordT, partition int32, query string, statusGroup int8, limit int,
) ([]IDWeight, error) {
	return orm.SearchText[RecordT, TableT, D](refSlice, partition, query, statusGroup, limit)
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

var (
	// MakeKeyConcat joins values into the deterministic string a KeyConcatenated
	// column stores.
	MakeKeyConcat = orm.MakeKeyConcat
	// EncodeToBase62 / DecodeFromBase62 are the token encoding used inside keys.
	EncodeToBase62   = orm.EncodeToBase62
	DecodeFromBase62 = orm.DecodeFromBase62
	// GetAutoincrementID reserves consecutive IDs for an arbitrary counter key.
	GetAutoincrementID = orm.GetAutoincrementID
	// QueryCachedGenericByIDs resolves IDs to the flat GenericRecord shape for any
	// table that opted in through TableSchema.GenericRecord.
	QueryCachedGenericByIDs = orm.QueryCachedGenericByIDs
	// SetDebugLogging raises the ORM's log verbosity: 0 off, 1 statements, 2 full.
	SetDebugLogging = orm.SetDebugLogging
)
