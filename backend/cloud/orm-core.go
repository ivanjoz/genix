package cloud

import (
	"app/core"
	"app/db"
	"errors"
)

// ORM defines a common interface for database operations across different providers (DynamoDB, SQLite/D1).
type ORM[RecordT any] interface {
	Init() error
	Insert(records []RecordT) error
	GetByID(record RecordT) (*RecordT, error)
	Select(dest *[]RecordT) QueryBuilder[RecordT]
}

// QueryBuilder provides a fluent interface to build and execute queries.
//
// Columns are named exactly as the table struct declares them: the mirror derives its
// keys and indexes from the same db.TableSchema the primary database uses, so a caller
// never names a mirror-only synthetic column.
type QueryBuilder[RecordT any] interface {
	Where(column string) QueryBuilder[RecordT]
	Equals(value interface{}) QueryBuilder[RecordT]
	Between(start interface{}, end interface{}) QueryBuilder[RecordT]
	Greater(value any) QueryBuilder[RecordT]
	Less(value any) QueryBuilder[RecordT]
	GreaterEqual(value any) QueryBuilder[RecordT]
	LessEqual(value any) QueryBuilder[RecordT]
	Exec() error
}

// getProviderORM initializes the cloud data mirror selected by BACKEND_PROVIDER. The
// table type parameters are inferred from the record type, exactly as in db.Query, which
// is what lets the mirror read the record's schema without any call site naming it.
func getProviderORM[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]]() (ORM[RecordT], error) {
	if core.Env == nil {
		core.PopulateVariables()
	}

	provider := core.Env.BACKEND_PROVIDER
	if provider != "aws" && provider != "cloudflare" {
		if provider == "none" {
			return nil, errors.New("cloud data mirror is disabled because BACKEND_PROVIDER is 'none'")
		}
		return nil, errors.New("BACKEND_PROVIDER in credentials.json is not set or invalid (must be 'aws', 'cloudflare', or 'none')")
	}

	tableMeta, err := buildTableMeta[RecordT, TableT, D]()
	if err != nil {
		return nil, err
	}

	if provider == "aws" {
		return NewDynamoORM[RecordT](tableMeta), nil
	}

	if core.Env.CLOUDFLARE_ACCOUNT == "" || core.Env.CLOUDFLARE_TOKEN == "" || core.Env.CLOUDFLARE_DATABASE_ID == "" {
		panic("CLOUDFLARE_ACCOUNT, CLOUDFLARE_TOKEN, and CLOUDFLARE_DATABASE_ID must be set in credentials.json when BACKEND_PROVIDER is 'cloudflare'")
	}
	return NewSqliteORM[RecordT](tableMeta), nil
}

// IsDataMirrorEnabled reports whether auth and tenant data must also use a cloud database.
func IsDataMirrorEnabled() bool {
	if core.Env == nil {
		core.PopulateVariables()
	}
	return core.Env.BACKEND_PROVIDER != "none"
}

// Init creates tables or checks if they exist through the selected backend provider.
func Init[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]]() error {
	// A self-hosted backend keeps these tables only in the primary database.
	if !IsDataMirrorEnabled() {
		core.Log("Cloud data mirror disabled; skipping table initialization")
		return nil
	}
	orm, err := getProviderORM[RecordT, TableT, D]()
	if err != nil {
		return err
	}
	return orm.Init()
}

// Insert inserts multiple records using the configured backend provider.
func Insert[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]](records []RecordT) error {
	// Callers already persist to the primary database first, so no second write is needed in self-hosted mode.
	if !IsDataMirrorEnabled() {
		core.Log("Cloud data mirror disabled; skipping mirror write", len(records))
		return nil
	}
	orm, err := getProviderORM[RecordT, TableT, D]()
	if err != nil {
		return err
	}
	return orm.Insert(records)
}

// GetByID retrieves a record using the configured backend provider.
func GetByID[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]](record RecordT) (*RecordT, error) {
	orm, err := getProviderORM[RecordT, TableT, D]()
	if err != nil {
		return nil, err
	}
	return orm.GetByID(record)
}

// Select returns a QueryBuilder using the configured backend provider.
func Select[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]](dest *[]RecordT) QueryBuilder[RecordT] {
	orm, err := getProviderORM[RecordT, TableT, D]()
	if err != nil {
		// Since Select returns a QueryBuilder and not an error directly,
		// we return a dummy builder that will return the error on Exec().
		return &errorQueryBuilder[RecordT]{err: err}
	}
	return orm.Select(dest)
}

// errorQueryBuilder is returned when an error occurs during the initialization of the QueryBuilder.
type errorQueryBuilder[RecordT any] struct {
	err error
}

func (b *errorQueryBuilder[RecordT]) Where(column string) QueryBuilder[RecordT]  { return b }
func (b *errorQueryBuilder[RecordT]) Equals(value interface{}) QueryBuilder[RecordT] { return b }
func (b *errorQueryBuilder[RecordT]) Between(start interface{}, end interface{}) QueryBuilder[RecordT] {
	return b
}
func (b *errorQueryBuilder[RecordT]) Greater(value any) QueryBuilder[RecordT]      { return b }
func (b *errorQueryBuilder[RecordT]) Less(value any) QueryBuilder[RecordT]         { return b }
func (b *errorQueryBuilder[RecordT]) GreaterEqual(value any) QueryBuilder[RecordT] { return b }
func (b *errorQueryBuilder[RecordT]) LessEqual(value any) QueryBuilder[RecordT]    { return b }
func (b *errorQueryBuilder[RecordT]) Exec() error                                  { return b.err }
