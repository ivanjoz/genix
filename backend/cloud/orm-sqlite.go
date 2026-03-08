package cloud

import (
	"app/core"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/bytedance/sonic"
)

// d1Request maps the payload required by Cloudflare D1 HTTP API.
type d1Request struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params,omitempty"`
}

// d1Response maps the response format from Cloudflare D1 HTTP API.
type d1Response struct {
	Result  []d1Result `json:"result"`
	Success bool       `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

type d1Result struct {
	Results []map[string]interface{} `json:"results"`
	Success bool                     `json:"success"`
	Error   string                   `json:"error"`
}

func executeD1Queries(queries []d1Request) ([]d1Result, error) {
	if core.Env.CLOUDFLARE_DATABASE_ID == "" {
		return nil, errors.New("CLOUDFLARE_DATABASE_ID is missing from credentials")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query", core.Env.CLOUDFLARE_ACCOUNT, core.Env.CLOUDFLARE_DATABASE_ID)

	var allResults []d1Result

	for _, query := range queries {
		bodyBytes, err := sonic.Marshal(query)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
		req.Header.Set("Content-Type", "application/json")

		var response d1Response
		err = core.SendHttpRequest(req, &response)
		if err != nil {
			return nil, err
		}

		if !response.Success {
			var errMsgs []string
			for _, e := range response.Errors {
				errMsgs = append(errMsgs, e.Message)
			}
			return nil, fmt.Errorf("D1 API error: %s", strings.Join(errMsgs, ", "))
		}

		for _, res := range response.Result {
			if !res.Success {
				return nil, fmt.Errorf("D1 Query error: %s", res.Error)
			}
			allResults = append(allResults, res)
		}
	}

	return allResults, nil
}

func mapD1RowToStruct[T any](row map[string]interface{}, cols []ColumnMeta) (*T, error) {
	var result T
	v := reflect.ValueOf(&result).Elem()

	for _, col := range cols {
		val, ok := row[col.ColumnName]
		if !ok || val == nil {
			continue
		}

		fieldVal := v.FieldByName(col.FieldName)
		if !fieldVal.CanSet() {
			continue
		}

		kind := fieldVal.Kind()

		// Handle complex types previously marshaled to JSON strings
		if kind == reflect.Slice && fieldVal.Type().Elem().Kind() != reflect.Uint8 || kind == reflect.Struct || kind == reflect.Map || kind == reflect.Array {
			if strVal, isStr := val.(string); isStr {
				err := sonic.Unmarshal([]byte(strVal), fieldVal.Addr().Interface())
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal JSON column %s: %w", col.ColumnName, err)
				}
			}
			continue
		}

		// Cloudflare D1 returns numbers as float64 usually, due to generic JSON unmarshalling
		switch kind {
		case reflect.String:
			if s, ok := val.(string); ok {
				fieldVal.SetString(s)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f, ok := val.(float64); ok {
				fieldVal.SetInt(int64(f))
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f, ok := val.(float64); ok {
				fieldVal.SetUint(uint64(f))
			}
		case reflect.Float32, reflect.Float64:
			if f, ok := val.(float64); ok {
				fieldVal.SetFloat(f)
			}
		case reflect.Bool:
			if b, ok := val.(bool); ok {
				fieldVal.SetBool(b)
			} else if f, ok := val.(float64); ok {
				fieldVal.SetBool(f != 0)
			} else if s, ok := val.(string); ok {
				fieldVal.SetBool(s == "true" || s == "1")
			}
		case reflect.Slice:
			if fieldVal.Type().Elem().Kind() == reflect.Uint8 {
				// Handle BLOB/[]byte if D1 returns it as string or float array
				if s, ok := val.(string); ok {
					fieldVal.SetBytes([]byte(s))
				}
			}
		}
	}

	return &result, nil
}

// SqliteORM provides basic operations for Cloudflare D1 / SQLite.
type SqliteORM[T any] struct {
	tableName string
	columns   []ColumnMeta
}

// Ensure SqliteORM implements ORM.
var _ ORM[any] = (*SqliteORM[any])(nil)

// NewSqliteORM creates a new ORM instance for the given generic type T.
func NewSqliteORM[T any]() (*SqliteORM[T], error) {
	var model T

	cols, tableName, _ := parseColumns(model)
	if len(cols) == 0 {
		return nil, errors.New("no columns found in struct")
	}

	return &SqliteORM[T]{
		tableName: tableName,
		columns:   cols,
	}, nil
}

// Init constructs and executes the D1 CREATE TABLE statements via HTTP.
func (o *SqliteORM[T]) Init() error {
	queries := o.BuildInitQueries()
	
	var reqs []d1Request
	for _, q := range queries {
		reqs = append(reqs, d1Request{SQL: q})
	}

	_, err := executeD1Queries(reqs)
	return err
}

// BuildInitQueries constructs the CREATE TABLE and CREATE INDEX statements for D1.
func (o *SqliteORM[T]) BuildInitQueries() []string {
	var queries []string
	var colDefs []string
	primaryKeyColumns := []string{}

	for _, col := range o.columns {
		var sqlType string

		// Determine SQLite type from Go type
		switch col.FieldType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Bool:
			sqlType = "INTEGER"
		case reflect.Float32, reflect.Float64:
			sqlType = "REAL"
		case reflect.Slice:
			if col.FieldType.Elem().Kind() == reflect.Uint8 {
				sqlType = "BLOB"
			} else {
				sqlType = "TEXT" // fallback for JSON
			}
		case reflect.Struct, reflect.Map, reflect.Array:
			sqlType = "TEXT"
		default:
			sqlType = "TEXT"
		}

		def := fmt.Sprintf("%s %s", col.ColumnName, sqlType)
		if col.IsPK || col.IsSK {
			// Keep PK parts explicit so SQLite can build a composite primary key when both exist.
			primaryKeyColumns = append(primaryKeyColumns, col.ColumnName)
		}
		colDefs = append(colDefs, def)
	}

	if len(primaryKeyColumns) == 1 {
		primaryKeyColumnName := primaryKeyColumns[0]
		for columnIndex, columnDefinition := range colDefs {
			if strings.HasPrefix(columnDefinition, primaryKeyColumnName+" ") {
				colDefs[columnIndex] = columnDefinition + " PRIMARY KEY"
				break
			}
		}
	} else if len(primaryKeyColumns) > 1 {
		// Composite keys are required for entities like Usuario where id is scoped by empresa_id.
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeyColumns, ", ")))
	}

	createTableQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n\t%s\n);",
		o.tableName, strings.Join(colDefs, ",\n\t"))
	queries = append(queries, createTableQuery)

	// Create indexes
	for _, col := range o.columns {
		if col.IsIndex || col.IsSK {
			// For DynamoDB sk it's just another column, but in D1 we probably want an index.
			idxQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_%s_idx ON %s(%s);",
				o.tableName, col.ColumnName, o.tableName, col.ColumnName)
			queries = append(queries, idxQuery)
		}
	}

	return queries
}

// Insert constructs and executes an SQL INSERT statement for SQLite / D1.
func (o *SqliteORM[T]) Insert(records []T) error {
	var reqs []d1Request
	for i := range records {
		query, args := o.buildInsertQuery(&records[i])
		reqs = append(reqs, d1Request{SQL: query, Params: args})
	}

	_, err := executeD1Queries(reqs)
	return err
}

func (o *SqliteORM[T]) buildInsertQuery(record *T) (string, []interface{}) {
	var cols []string
	var placeholders []string
	var args []interface{}
	primaryKeyColumns := []string{}
	updateAssignments := []string{}

	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, col := range o.columns {
		cols = append(cols, col.ColumnName)
		placeholders = append(placeholders, "?")
		if col.IsPK || col.IsSK {
			primaryKeyColumns = append(primaryKeyColumns, col.ColumnName)
		} else {
			// Replace non-key columns on conflict so SQLite mirrors DynamoDB PutItem semantics.
			updateAssignments = append(updateAssignments, fmt.Sprintf("%s = excluded.%s", col.ColumnName, col.ColumnName))
		}

		fieldVal := v.FieldByName(col.FieldName)
		kind := fieldVal.Kind()
		
		if kind == reflect.Slice && fieldVal.Type().Elem().Kind() == reflect.Uint8 {
			args = append(args, fieldVal.Interface())
		} else if kind == reflect.Slice || kind == reflect.Struct || kind == reflect.Map || kind == reflect.Array {
			b, _ := sonic.Marshal(fieldVal.Interface())
			args = append(args, string(b))
		} else {
			args = append(args, fieldVal.Interface())
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		o.tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	if len(primaryKeyColumns) > 0 {
		if len(updateAssignments) > 0 {
			query += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s",
				strings.Join(primaryKeyColumns, ", "),
				strings.Join(updateAssignments, ", "))
		} else {
			query += fmt.Sprintf(" ON CONFLICT (%s) DO NOTHING", strings.Join(primaryKeyColumns, ", "))
		}
	}

	query += ";"

	return query, args
}

// GetByID constructs and executes an SQL SELECT statement by ID for SQLite / D1.
func (o *SqliteORM[T]) GetByID(record T) (*T, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	primaryKeyColumns := []string{}
	primaryKeyValues := []interface{}{}

	for _, col := range o.columns {
		if col.IsPK || col.IsSK {
			primaryKeyColumns = append(primaryKeyColumns, col.ColumnName)
			primaryKeyValues = append(primaryKeyValues, v.FieldByName(col.FieldName).Interface())
		}
	}

	if len(primaryKeyColumns) == 0 {
		return nil, errors.New("record must have at least one key column for GetByID")
	}

	whereClauses := make([]string, 0, len(primaryKeyColumns))
	for _, columnName := range primaryKeyColumns {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", columnName))
	}

	// Query by all key parts so D1 matches the same identity rules used by DynamoDB.
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1;", o.tableName, strings.Join(whereClauses, " AND "))

	reqs := []d1Request{{SQL: query, Params: primaryKeyValues}}
	results, err := executeD1Queries(reqs)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 || len(results[0].Results) == 0 {
		return nil, nil // Not found
	}

	return mapD1RowToStruct[T](results[0].Results[0], o.columns)
}

// Select returns a QueryBuilder to construct an SQL query.
func (o *SqliteORM[T]) Select(dest *[]T) QueryBuilder[T] {
	return &sqliteQueryBuilder[T]{
		orm:  o,
		dest: dest,
	}
}

// sqliteQueryBuilder implements the QueryBuilder interface for SQLite / D1.
type sqliteQueryBuilder[T any] struct {
	orm      *SqliteORM[T]
	dest     *[]T
	column   string
	operator string
	value    interface{}
	valueEnd interface{} // Used for Between
}

func (b *sqliteQueryBuilder[T]) Where(columnName string) QueryBuilder[T] {
	b.column = columnName
	return b
}

func (b *sqliteQueryBuilder[T]) Equals(value interface{}) QueryBuilder[T] {
	b.operator = "="
	b.value = value
	return b
}

func (b *sqliteQueryBuilder[T]) Between(start interface{}, end interface{}) QueryBuilder[T] {
	b.operator = "BETWEEN"
	b.value = start
	b.valueEnd = end
	return b
}

func (b *sqliteQueryBuilder[T]) Greater(value interface{}) QueryBuilder[T] {
	b.operator = ">"
	b.value = value
	return b
}

func (b *sqliteQueryBuilder[T]) Less(value interface{}) QueryBuilder[T] {
	b.operator = "<"
	b.value = value
	return b
}

func (b *sqliteQueryBuilder[T]) GreaterEqual(value interface{}) QueryBuilder[T] {
	b.operator = ">="
	b.value = value
	return b
}

func (b *sqliteQueryBuilder[T]) LessEqual(value interface{}) QueryBuilder[T] {
	b.operator = "<="
	b.value = value
	return b
}

func (b *sqliteQueryBuilder[T]) Exec() error {
	if b.column == "" {
		return errors.New("must specify a column using Where() before Exec()")
	}
	if b.operator == "" {
		return errors.New("must specify an operator (e.g. Equals, Greater) before Exec()")
	}

	var query string
	var args []interface{}

	if b.operator == "BETWEEN" {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s BETWEEN ? AND ?;", b.orm.tableName, b.column)
		args = append(args, b.value, b.valueEnd)
	} else {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s %s ?;", b.orm.tableName, b.column, b.operator)
		args = append(args, b.value)
	}

	reqs := []d1Request{{SQL: query, Params: args}}
	results, err := executeD1Queries(reqs)
	if err != nil {
		return err
	}

	var records []T
	if len(results) > 0 {
		for _, row := range results[0].Results {
			record, err := mapD1RowToStruct[T](row, b.orm.columns)
			if err != nil {
				return err
			}
			records = append(records, *record)
		}
	}

	*b.dest = records
	return nil
}
