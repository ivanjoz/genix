package cloud

import (
	"app/core"
	"context"
	"errors"
	"fmt"
	"github.com/bytedance/sonic"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoORM provides basic operations for DynamoDB (Single Table Design).
type DynamoORM[RecordT any] struct {
	meta *TableMeta
}

// Ensure DynamoORM implements ORM.
var _ ORM[any] = (*DynamoORM[any])(nil)

// NewDynamoORM creates a new ORM instance for a record type whose mirror metadata has
// already been resolved from its table schema.
func NewDynamoORM[RecordT any](meta *TableMeta) *DynamoORM[RecordT] {
	return &DynamoORM[RecordT]{meta: meta}
}

// Init verifica que exista la tabla única de DynamoDB. Deliberadamente NO la crea: la tabla
// la declara cloud/template.yml y la gobierna CloudFormation (acción 9 de deploy.sh).
//
// Antes había dos creadores. Ejecutar la acción 6 junto con la 9 hacía que fn-init ganase la
// carrera por segundos y CloudFormation abortase el stack entero con AlreadyExists, así que
// la creación vive en un solo sitio y aquí solo se comprueba.
func (o *DynamoORM[T]) Init() error {
	client := dynamodb.NewFromConfig(core.GetAwsConfig())

	_, err := client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: core.PtrString(core.Env.DYNAMO_TABLE),
	})
	if err != nil {
		return fmt.Errorf(
			"la tabla DynamoDB %q no existe o no es accesible. Despliega la infraestructura primero (./deploy.sh 9): %w",
			core.Env.DYNAMO_TABLE, err)
	}

	return nil
}

// Insert inserts multiple records into DynamoDB single table.
func (o *DynamoORM[RecordT]) Insert(records []RecordT) error {
	for i := range records {
		if err := o.insertOne(&records[i]); err != nil {
			return err
		}
	}
	return nil
}

// itemKey renders the pk/sk pair for one record. Both are derived from the table's
// partition and key columns, so an item lands on the same coordinates the primary
// database uses.
func (o *DynamoORM[RecordT]) itemKey(record reflect.Value) (string, string, error) {
	dynamoPK := o.meta.HashPrefix
	if partitionField, hasPartition := o.meta.PartitionField(); hasPartition {
		dynamoPK += stringify(record.FieldByName(partitionField))
	}

	keyField, err := o.meta.KeyField()
	if err != nil {
		return "", "", err
	}

	skValue := stringify(record.FieldByName(keyField))
	if skValue == "" {
		return "", "", errors.New("missing sort key (sk) in record")
	}

	return dynamoPK, skValue, nil
}

func (o *DynamoORM[RecordT]) insertOne(record *RecordT) error {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	dynamoPK, skValue, err := o.itemKey(v)
	if err != nil {
		return err
	}

	item := map[string]types.AttributeValue{}
	for _, index := range o.meta.Indexes {
		if indexValue := index.buildIndexValue(v); indexValue != "" {
			item[index.DynamoIndex] = &types.AttributeValueMemberS{Value: indexValue}
		}
	}

	item["pk"] = &types.AttributeValueMemberS{Value: dynamoPK}
	item["sk"] = &types.AttributeValueMemberS{Value: skValue}

	recordBytes, err := sonic.Marshal(record)
	if err != nil {
		return err
	}
	item["json"] = &types.AttributeValueMemberS{Value: string(recordBytes)}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: core.PtrString(core.Env.DYNAMO_TABLE),
		Item:      item,
	})

	return err
}

// GetByID retrieves a record from DynamoDB single table by passing a record populated with PK and SK.
func (o *DynamoORM[RecordT]) GetByID(record RecordT) (*RecordT, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	dynamoPK, skValue, err := o.itemKey(v)
	if err != nil {
		return nil, err
	}

	key := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: dynamoPK},
		"sk": &types.AttributeValueMemberS{Value: skValue},
	}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	output, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: core.PtrString(core.Env.DYNAMO_TABLE),
		Key:       key,
	})

	if err != nil {
		return nil, err
	}
	if len(output.Item) == 0 {
		return nil, nil // Not found
	}

	jsonAttr, ok := output.Item["json"]
	if !ok {
		return nil, errors.New("record found but no json data present")
	}

	jsonStr := jsonAttr.(*types.AttributeValueMemberS).Value
	var result RecordT
	err = sonic.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Select returns a QueryBuilder to query DynamoDB.
func (o *DynamoORM[T]) Select(dest *[]T) QueryBuilder[T] {
	return &dynamoQueryBuilder[T]{
		orm:  o,
		dest: dest,
	}
}

// dynamoQueryBuilder implements the QueryBuilder interface for DynamoDB.
type dynamoQueryBuilder[T any] struct {
	orm           *DynamoORM[T]
	dest          *[]T
	pendingColumn string
	conditions    []queryCondition
}

func (b *dynamoQueryBuilder[T]) Where(columnName string) QueryBuilder[T] {
	b.pendingColumn = columnName
	return b
}

func (b *dynamoQueryBuilder[T]) Equals(value interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, "=", value, nil)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) Between(start interface{}, end interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, "BETWEEN", start, end)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) Greater(value interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, ">", value, nil)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) Less(value interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, "<", value, nil)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) GreaterEqual(value interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, ">=", value, nil)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) LessEqual(value interface{}) QueryBuilder[T] {
	b.conditions = appendCondition(b.conditions, b.pendingColumn, "<=", value, nil)
	b.pendingColumn = ""
	return b
}

func (b *dynamoQueryBuilder[T]) Exec() error {
	if b.pendingColumn != "" {
		return fmt.Errorf("column %s is missing an operator before Exec()", b.pendingColumn)
	}
	if len(b.conditions) == 0 {
		return errors.New("must specify at least one condition using Where() before Exec()")
	}

	partitionColumn, hasLogicalPartition := findLogicalPartitionColumn(b.orm.meta.Columns)
	partitionCondition, indexedConditions, validationError := splitQueryConditions(b.conditions, partitionColumn, hasLogicalPartition)
	if validationError != nil {
		return validationError
	}

	index, matchError := matchIndex(b.orm.meta.Indexes, indexedConditions)
	if matchError != nil {
		return matchError
	}

	partitionValue := ""
	if partitionCondition != nil {
		partitionValue = fmt.Sprintf("%v", partitionCondition.Value)
	}
	dynamoPartitionKey := b.orm.meta.HashPrefix + partitionValue

	// The GSI holds one composite string per index, so every query — equality or range —
	// becomes a comparison against the bounds that composite can take.
	indexRange, rangeError := buildCompositeRange(index, partitionValue, indexedConditions)
	if rangeError != nil {
		return rangeError
	}

	var expr string
	attrValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: dynamoPartitionKey},
	}

	if indexRange.IsExact {
		expr = fmt.Sprintf("pk = :pk AND %s = :val", index.DynamoIndex)
		attrValues[":val"] = &types.AttributeValueMemberS{Value: indexRange.Lower}
	} else {
		expr = fmt.Sprintf("pk = :pk AND %s BETWEEN :val1 AND :val2", index.DynamoIndex)
		attrValues[":val1"] = &types.AttributeValueMemberS{Value: indexRange.Lower}
		attrValues[":val2"] = &types.AttributeValueMemberS{Value: indexRange.Upper}
	}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	queryInput := &dynamodb.QueryInput{
		TableName:                 core.PtrString(core.Env.DYNAMO_TABLE),
		IndexName:                 core.PtrString(index.DynamoIndex),
		KeyConditionExpression:    core.PtrString(expr),
		ExpressionAttributeValues: attrValues,
	}

	output, err := client.Query(context.TODO(), queryInput)
	if err != nil {
		return err
	}

	var records []T
	for _, item := range output.Items {
		jsonAttr, ok := item["json"]
		if !ok {
			continue
		}
		jsonStr := jsonAttr.(*types.AttributeValueMemberS).Value
		var record T
		if err := sonic.Unmarshal([]byte(jsonStr), &record); err != nil {
			return err
		}
		records = append(records, record)
	}

	*b.dest = records
	return nil
}
