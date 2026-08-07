package cloud

import (
	"app/core"
	"errors"
)

// ORM defines a common interface for database operations across different providers (DynamoDB, SQLite/D1).
type ORM[T any] interface {
	Init() error
	Insert(records []T) error
	GetByID(record T) (*T, error)
	Select(dest *[]T) QueryBuilder[T]
}

// QueryBuilder provides a fluent interface to build and execute queries.
type QueryBuilder[T any] interface {
	Where(column string) QueryBuilder[T]
	Equals(value interface{}) QueryBuilder[T]
	Between(start interface{}, end interface{}) QueryBuilder[T]
	Greater(value any) QueryBuilder[T]
	Less(value any) QueryBuilder[T]
	GreaterEqual(value any) QueryBuilder[T]
	LessEqual(value any) QueryBuilder[T]
	Exec() error
}

// getProviderORM initializes the cloud data mirror selected by BACKEND_PROVIDER.
func getProviderORM[T any]() (ORM[T], error) {
	if core.Env == nil {
		core.PopulateVariables()
	}

	provider := core.Env.BACKEND_PROVIDER
	switch provider {
	case "aws":
		return NewDynamoORM[T]()
	case "cloudflare":
		if core.Env.CLOUDFLARE_ACCOUNT == "" || core.Env.CLOUDFLARE_TOKEN == "" || core.Env.CLOUDFLARE_DATABASE_ID == "" {
			panic("CLOUDFLARE_ACCOUNT, CLOUDFLARE_TOKEN, and CLOUDFLARE_DATABASE_ID must be set in credentials.json when BACKEND_PROVIDER is 'cloudflare'")
		}
		return NewSqliteORM[T]()
	case "none":
		return nil, errors.New("cloud data mirror is disabled because BACKEND_PROVIDER is 'none'")
	default:
		return nil, errors.New("BACKEND_PROVIDER in credentials.json is not set or invalid (must be 'aws', 'cloudflare', or 'none')")
	}
}

// IsDataMirrorEnabled reports whether auth and tenant data must also use a cloud database.
func IsDataMirrorEnabled() bool {
	if core.Env == nil {
		core.PopulateVariables()
	}
	return core.Env.BACKEND_PROVIDER != "none"
}

// Init creates tables or checks if they exist through the selected backend provider.
func Init[T any]() error {
	// A self-hosted backend keeps these tables only in the primary Scylla database.
	if !IsDataMirrorEnabled() {
		core.Log("Cloud data mirror disabled; skipping table initialization")
		return nil
	}
	orm, err := getProviderORM[T]()
	if err != nil {
		return err
	}
	return orm.Init()
}

// Insert inserts multiple records using the configured backend provider.
func Insert[T any](records []T) error {
	// Callers already persist to Scylla first, so no second write is needed in self-hosted mode.
	if !IsDataMirrorEnabled() {
		core.Log("Cloud data mirror disabled; skipping mirror write", len(records))
		return nil
	}
	orm, err := getProviderORM[T]()
	if err != nil {
		return err
	}
	return orm.Insert(records)
}

// GetByID retrieves a record using the configured backend provider.
func GetByID[T any](record T) (*T, error) {
	orm, err := getProviderORM[T]()
	if err != nil {
		return nil, err
	}
	return orm.GetByID(record)
}

// Select returns a QueryBuilder using the configured backend provider.
func Select[T any](dest *[]T) QueryBuilder[T] {
	orm, err := getProviderORM[T]()
	if err != nil {
		// Since Select returns a QueryBuilder and not an error directly,
		// we return a dummy builder that will return the error on Exec().
		return &errorQueryBuilder[T]{err: err}
	}
	return orm.Select(dest)
}

// errorQueryBuilder is returned when an error occurs during the initialization of the QueryBuilder.
type errorQueryBuilder[T any] struct {
	err error
}

func (b *errorQueryBuilder[T]) Where(column string) QueryBuilder[T]                        { return b }
func (b *errorQueryBuilder[T]) Equals(value interface{}) QueryBuilder[T]                   { return b }
func (b *errorQueryBuilder[T]) Between(start interface{}, end interface{}) QueryBuilder[T] { return b }
func (b *errorQueryBuilder[T]) Greater(value any) QueryBuilder[T]                          { return b }
func (b *errorQueryBuilder[T]) Less(value any) QueryBuilder[T]                             { return b }
func (b *errorQueryBuilder[T]) GreaterEqual(value any) QueryBuilder[T]                     { return b }
func (b *errorQueryBuilder[T]) LessEqual(value any) QueryBuilder[T]                        { return b }
func (b *errorQueryBuilder[T]) Exec() error                                                { return b.err }
