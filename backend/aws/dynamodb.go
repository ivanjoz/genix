package aws

import (
	"app/core"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoTableRecords[T any] struct {
	PK             string
	TableName      string
	UseCompression bool
	Account        uint8
	GetIndexKeys   func(record T, idx uint8) string
}

type DynamoQueryParam struct {
	Index            string
	GreaterThan      string
	LesserThan       string
	BeginsWith       string
	Projection       string
	Equals           string
	BetweenStart     string
	BetweenEnd       string
	ScanIndexForward bool
	Limit            int32
}

func (params DynamoTableRecords[T]) MakeItem(record *T, accion uint8) (*map[string]types.AttributeValue, error) {

	item := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: params.PK},
	}

	sk := params.GetIndexKeys(*record, 0)
	if len(sk) == 0 {
		return nil, errors.New("no se ha especificado un [sk] que debe ser el index = 0 obligatorio")
	}
	item["sk"] = &types.AttributeValueMemberS{Value: sk}

	// Itera sobre los indices y si los encuentra los agrega
	for i := 1; i <= 4; i++ {
		indexValue := params.GetIndexKeys(*record, uint8(i))

		if len(indexValue) > 0 {
			item["ix"+strconv.Itoa(i)] = &types.AttributeValueMemberS{Value: indexValue}
		}
	}

	recordBytes, err := json.Marshal(record)
	if err != nil {
		return nil, errors.New("error al serializar el registro Dynamodb")
	}
	recordJson := string(recordBytes)

	if params.UseCompression && accion == 1 {
		compressed := core.CompressZstd(&recordJson)
		// Lo almacena como un array de bytes (binarios)
		item["data"] = &types.AttributeValueMemberB{Value: compressed}
	} else {
		item["json"] = &types.AttributeValueMemberS{Value: recordJson}
	}

	return &item, nil
}

// PUT ITEM
func (params DynamoTableRecords[T]) PutItem(record *T, accion uint8) error {

	Item, err := params.MakeItem(record, accion)
	if err != nil {
		return err
	}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	putRequest := dynamodb.PutItemInput{
		TableName:              aws.String(params.TableName),
		Item:                   *Item,
		ReturnConsumedCapacity: "TOTAL",
	}

	output, err := client.PutItem(context.TODO(), &putRequest)
	if err != nil {
		core.Log("error al enviar item", err)
		return err
	}
	core.Log("Put Item. Capacidad Consumida:: ", *output.ConsumedCapacity.CapacityUnits)
	return nil
}

// BASH PUT o BASH DELETE
func (params DynamoTableRecords[T]) DynamoAccionToItems(recordsToSend []T, accion uint8) error {
	core.Log("Enviando registros a Tabla DynamoDB:: ", params.TableName)
	dynamoItems := []*map[string]types.AttributeValue{}

	for i := range recordsToSend {
		item, err := params.MakeItem(&recordsToSend[i], accion)
		if err != nil {
			return err
		}
		dynamoItems = append(dynamoItems, item)
	}

	// Crea grupos de envío de 25 items
	group1 := []*map[string]types.AttributeValue{}
	recordsGroups := [][]*map[string]types.AttributeValue{group1}

	for _, record := range dynamoItems {
		group := &recordsGroups[len(recordsGroups)-1]
		if len(*group) >= 25 {
			recordsGroups = append(recordsGroups, []*map[string]types.AttributeValue{})
			group = &recordsGroups[len(recordsGroups)-1]
		}
		*group = append(*group, record)
	}

	client := dynamodb.NewFromConfig(core.GetAwsConfig())

	for _, records := range recordsGroups {

		writeRequests := []types.WriteRequest{}

		for _, record := range records {
			if accion == 1 { // Put items
				writeRequests = append(writeRequests, types.WriteRequest{
					PutRequest: &types.PutRequest{Item: *record},
				})
			} else if accion == 3 { // Delete items
				sk := (*record)["sk"]
				if sk == nil {
					core.Log("No se encontró el sk para el delete item")
					continue
				}
				writeRequests = append(writeRequests, types.WriteRequest{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							"pk": (*record)["pk"], "sk": sk,
						},
					},
				})
			}
		}

		// Itera sobre los grupos y realiza el envio
		requestItems := map[string][]types.WriteRequest{}
		requestItems[params.TableName] = writeRequests

		output, err := client.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
			RequestItems: requestItems,
		})

		if err != nil {
			return errors.New("Error en el guardado en Dynamodb: " + err.Error())
		}

		core.Print(output.ConsumedCapacity)
	}

	return nil
}

func (params DynamoTableRecords[T]) DynamoPutItems(recordsToSend []T) error {
	err := params.DynamoAccionToItems(recordsToSend, 1)
	return err
}

func (params DynamoTableRecords[T]) DynamoDeleteItems(recordsToSend []T) error {
	err := params.DynamoAccionToItems(recordsToSend, 3)
	return err
}

func parseDynamoDBItem[T any](dynamoItem DynamoDBItem) (*T, error) {
	jsonContent := ""

	if len(dynamoItem.Json) > 0 {
		jsonContent = dynamoItem.Json
	} else if len(dynamoItem.Data) > 0 {
		jsonContent = core.DecompressZstd(&dynamoItem.Data)
	} else {
		jsonContentBytes, _ := json.Marshal(dynamoItem)
		jsonContent = string(jsonContentBytes)
	}

	record := new(T)
	err := json.Unmarshal([]byte(jsonContent), record)
	if err != nil {
		fmt.Println(dynamoItem.Json)
		core.Log("Error al deserializar Dynamodb: " + err.Error())
		return nil, err
	}

	return record, nil
}

// GET ITEM
func (params *DynamoTableRecords[T]) GetItem(sk string) (*T, error) {

	core.Log("Enviando registros a Tabla DynamoDB:: ", params.TableName)
	client := dynamodb.NewFromConfig(core.GetAwsConfig())

	getRequest := dynamodb.GetItemInput{
		TableName: aws.String(params.TableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: params.PK},
			"sk": &types.AttributeValueMemberS{Value: sk},
		},
	}

	output, err := client.GetItem(context.TODO(), &getRequest)
	if err != nil {
		core.Log("Error al obtener item", err)
		return nil, err
	}

	if output.Item == nil || len(output.Item) == 0 {
		return nil, nil
	}

	dynamoItem := DynamoDBItem{}
	err = attributevalue.UnmarshalMap(output.Item, &dynamoItem)
	if err != nil {
		core.Log("Error al interpretar item Dynamodb: " + err.Error())
		return nil, err
	}

	record, err := parseDynamoDBItem[T](dynamoItem)
	return record, err
}

func GetDynamoDBTables() error {

	client := dynamodb.NewFromConfig(core.GetAwsConfig())

	output, err := client.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		return errors.New("error al listar DynamoDB: " + err.Error())
	}

	core.Print(output.TableNames)

	return nil
}

type DynamoDBItem struct {
	PK   string `json:"pk" dynamodbav:"pk"`
	SK   string `json:"sk" dynamodbav:"sk"`
	Ix1  string `json:"ix1" dynamodbav:"ix1"`
	Ix2  string `json:"ix2" dynamodbav:"ix2"`
	Ix3  string `json:"ix3" dynamodbav:"ix3"`
	Ix4  string `json:"ix4" dynamodbav:"ix4"`
	Json string `json:"json" dynamodbav:"json"`
	Data []byte `json:"data" dynamodbav:"data"`
}

type dynamoQueryInput struct {
	input      dynamodb.QueryInput
	expression string
}

func (e DynamoTableRecords[T]) QueryBatch(querys []DynamoQueryParam) ([]T, error) {

	queryInputs := []dynamoQueryInput{}
	TableName := aws.String(core.Env.DYNAMO_TABLE)

	for _, query := range querys {
		// Crea la query según el tipo de lógica
		queryInput := &dynamodb.QueryInput{
			TableName:              TableName,
			ScanIndexForward:       &query.ScanIndexForward,
			ReturnConsumedCapacity: "TOTAL",
		}

		if query.Limit > 0 {
			queryInput.Limit = &query.Limit
		}

		if query.Index != "sk" {
			queryInput.IndexName = aws.String(query.Index)
		}

		expression := "pk = :pk and "
		attributes := map[string]string{":pk": e.PK}

		if len(query.BetweenStart) > 0 {
			expression += "$index BETWEEN :value1 AND :value2"
			attributes[":value1"] = query.BetweenStart
			attributes[":value2"] = query.BetweenEnd
		} else if len(query.BeginsWith) > 0 {
			expression += "begins_with($index, :value)"
			attributes[":value"] = query.BeginsWith
		} else if len(query.GreaterThan) > 0 {
			expression += "$index >= :value"
			attributes[":value"] = query.GreaterThan
		} else if len(query.LesserThan) > 0 {
			expression += "$index <= :value"
			attributes[":value"] = query.LesserThan
		} else if len(query.Equals) > 0 {
			expression += "$index = :value"
			attributes[":value"] = query.Equals
		}

		expression = strings.Replace(expression, "$index", query.Index, 1)
		queryInput.KeyConditionExpression = &expression

		expressionAttributes, _ := attributevalue.MarshalMap(attributes)
		expressionParsed := expression
		for attKey, attValue := range attributes {
			expressionParsed = strings.Replace(expressionParsed, attKey, attValue, 1)
		}

		queryInput.ExpressionAttributeValues = expressionAttributes
		if len(query.Projection) > 0 {
			queryInput.ProjectionExpression = &query.Projection
		}

		dynamoInput := dynamoQueryInput{input: *queryInput, expression: expressionParsed}
		queryInputs = append(queryInputs, dynamoInput)
	}

	core.Log("estamo aqui...")

	client := dynamodb.NewFromConfig(core.GetAwsConfig())
	records := []T{}

	// Itera por cada query input
	for _, dynamoQuery := range queryInputs {

		queryInput := dynamoQuery.input
		queryCount := 0
		lastEvaluatedKey := map[string]types.AttributeValue{}

		for {
			queryCount++
			if queryCount > 100 {
				break
			}

			if len(lastEvaluatedKey) > 0 {
				queryInput.ExclusiveStartKey = lastEvaluatedKey
			}

			output, err := client.Query(context.TODO(), &queryInput)
			if err != nil {
				core.Log("Error al ejecutar la DynamoDB")
				panic(err)
			}

			items := []DynamoDBItem{}
			err = attributevalue.UnmarshalListOfMaps(output.Items, &items)
			if err != nil {
				core.Log("Error al deserializar Dynamodb: " + err.Error())
				return nil, err
			}

			for _, dynamoItem := range items {
				record, _ := parseDynamoDBItem[T](dynamoItem)
				if record != nil {
					records = append(records, *record)
				}
			}

			capacity := float64(0)
			if output.ConsumedCapacity.CapacityUnits != nil {
				capacity = *output.ConsumedCapacity.CapacityUnits
			}

			core.Log("Recibidos ", len(items), " registros:: ", dynamoQuery.expression)
			core.Log("Consumed Capacity::", capacity, " | Last key: ", output.LastEvaluatedKey)

			if len(output.LastEvaluatedKey) > 0 {
				lastEvaluatedKey = output.LastEvaluatedKey
			} else {
				break
			}
		}
	}

	return records, nil
}

type CounterRecord struct {
	SK      string `json:"sk"`
	Counter int64  `json:"counter"`
	Updated int64  `json:"updated"`
}

func MakeCounterTable() DynamoTableRecords[CounterRecord] {
	return DynamoTableRecords[CounterRecord]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             "counter",
		UseCompression: false,
		GetIndexKeys: func(e CounterRecord, idx uint8) string {
			switch idx {
			case 0:
				return core.Concatn(e.SK)
			}
			return ""
		},
	}
}

func GetDynamoCounter(sk string, minValue_ ...int64) (int64, error) {
	minValue := int64(0)
	if len(minValue_) == 1 {
		minValue = minValue_[0]
	}

	dynamoTable := MakeCounterTable()
	record, err := dynamoTable.GetItem(sk)

	if err != nil {
		return 0, errors.New("Error al obtener el counter: " + err.Error())
	}

	counter := int64(0)
	if record == nil {
		counter = 1
	} else {
		counter = record.Counter + 1
	}

	if counter < minValue {
		counter = minValue
	}

	newRecord := CounterRecord{
		Counter: counter,
		SK:      sk,
		Updated: time.Now().Unix(),
	}

	err = dynamoTable.PutItem(&newRecord, 1)
	if err != nil {
		return 0, errors.New("Error al actualizar el counter: " + err.Error())
	}

	return counter, nil
}
