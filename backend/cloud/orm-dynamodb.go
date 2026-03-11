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
type DynamoORM[T any] struct {
	hashPrefix string
	columns    []ColumnMeta
}

// Ensure DynamoORM implements ORM.
var _ ORM[any] = (*DynamoORM[any])(nil)

// NewDynamoORM creates a new ORM instance for the given generic type T.
func NewDynamoORM[T any]() (*DynamoORM[T], error) {
	var model T

	cols, _, hashPrefix := parseColumns(model)
	if len(cols) == 0 {
		return nil, errors.New("no columns found in struct")
	}

	return &DynamoORM[T]{
		hashPrefix: hashPrefix,
		columns:    cols,
	}, nil
}

// Init creates the DynamoDB single table if it does not already exist.
func (o *DynamoORM[T]) Init() error {
	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	tableName := core.PtrString(core.Env.DYNAMO_TABLE)

	_, err := client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: tableName,
	})

	if err == nil {
		// Table already exists
		return nil
	}

	// Assuming the error is because the table doesn't exist, try to create it.
	// We create pk, sk, and ix1 through ix4 as strings.
	_, err = client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName: tableName,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: core.PtrString("pk"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: core.PtrString("sk"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: core.PtrString("ix1"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: core.PtrString("ix2"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: core.PtrString("ix3"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: core.PtrString("ix4"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: core.PtrString("pk"), KeyType: types.KeyTypeHash},
			{AttributeName: core.PtrString("sk"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: core.PtrString("ix1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: core.PtrString("pk"), KeyType: types.KeyTypeHash},
					{AttributeName: core.PtrString("ix1"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: core.PtrString("ix2"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: core.PtrString("pk"), KeyType: types.KeyTypeHash},
					{AttributeName: core.PtrString("ix2"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: core.PtrString("ix3"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: core.PtrString("pk"), KeyType: types.KeyTypeHash},
					{AttributeName: core.PtrString("ix3"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: core.PtrString("ix4"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: core.PtrString("pk"), KeyType: types.KeyTypeHash},
					{AttributeName: core.PtrString("ix4"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// Insert inserts multiple records into DynamoDB single table.
func (o *DynamoORM[T]) Insert(records []T) error {
	for i := range records {
		if err := o.insertOne(&records[i]); err != nil {
			return err
		}
	}
	return nil
}

func (o *DynamoORM[T]) insertOne(record *T) error {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	item := map[string]types.AttributeValue{}
	var dynamoPartitionValue string
	var skValue string

	for _, col := range o.columns {
		fieldVal := v.FieldByName(col.FieldName)
		strVal := stringify(fieldVal)

		if col.IsPK && dynamoPartitionValue == "" {
			dynamoPartitionValue = strVal
		}

		if col.IsSK {
			skValue = strVal
		}

		if col.IsIndex && strVal != "" {
			item[col.DynamoIndex] = &types.AttributeValueMemberS{Value: strVal}
		}
	}

	dynamoPK := o.hashPrefix
	if dynamoPartitionValue != "" {
		dynamoPK += dynamoPartitionValue
	}

	item["pk"] = &types.AttributeValueMemberS{Value: dynamoPK}

	if skValue == "" {
		return errors.New("missing sort key (sk) in record")
	}
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
func (o *DynamoORM[T]) GetByID(record T) (*T, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var dynamoPartitionValue, skValue string

	for _, col := range o.columns {
		fieldVal := v.FieldByName(col.FieldName)
		strVal := stringify(fieldVal)

		if col.IsPK && dynamoPartitionValue == "" {
			dynamoPartitionValue = strVal
		}
		if col.IsSK {
			skValue = strVal
		}
	}

	if skValue == "" {
		return nil, errors.New("record must have sk populated for GetByID")
	}

	dynamoPK := o.hashPrefix
	if dynamoPartitionValue != "" {
		dynamoPK += dynamoPartitionValue
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
	var result T
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

	partitionColumn, hasLogicalPartition := findLogicalPartitionColumn(b.orm.columns)
	partitionCondition, indexedConditions, validationError := splitQueryConditions(b.conditions, partitionColumn, hasLogicalPartition)
	if validationError != nil {
		return validationError
	}
	if len(indexedConditions) != 1 {
		return errors.New("dynamo queries require exactly one indexed Where() in addition to the partition Where()")
	}

	indexedCondition := indexedConditions[0]
	var dynIndex string
	for _, col := range b.orm.columns {
		if col.ColumnName == indexedCondition.ColumnName && col.IsIndex {
			dynIndex = col.DynamoIndex
			break
		}
	}

	if dynIndex == "" {
		return fmt.Errorf("column %s is not marked as an index", indexedCondition.ColumnName)
	}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	dynamoPartitionKey := b.orm.hashPrefix
	if partitionCondition != nil {
		dynamoPartitionKey += fmt.Sprintf("%v", partitionCondition.Value)
	}

	var expr string
	attrValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: dynamoPartitionKey},
	}

	if indexedCondition.Operator == "BETWEEN" {
		expr = fmt.Sprintf("pk = :pk AND %s BETWEEN :val1 AND :val2", dynIndex)
		attrValues[":val1"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", indexedCondition.Value)}
		attrValues[":val2"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", indexedCondition.ValueEnd)}
	} else {
		expr = fmt.Sprintf("pk = :pk AND %s %s :val", dynIndex, indexedCondition.Operator)
		attrValues[":val"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", indexedCondition.Value)}
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 core.PtrString(core.Env.DYNAMO_TABLE),
		IndexName:                 core.PtrString(dynIndex),
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
