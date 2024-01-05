package aws

import (
	"app/core"
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/gocarina/gocsv"
)

type QueryAthenaInput struct {
	QueryString    string
	Database       string // smartberry
	OutputLocation string // "s3//smartberry-datalake/output"
	ReuseMaxAge    int
}

func QueryAthena(params QueryAthenaInput) (*types.ResultSet, error) {

	client := athena.NewFromConfig(core.GetAwsConfig(3))

	MaxAgeInMinutes := int32(15)
	workgroup := "smartberry"

	input := athena.StartQueryExecutionInput{
		WorkGroup:   &workgroup,
		QueryString: &params.QueryString,
		QueryExecutionContext: &types.QueryExecutionContext{
			Database: &params.Database,
		},
		ResultConfiguration: &types.ResultConfiguration{
			OutputLocation: &params.OutputLocation,
		},
		ResultReuseConfiguration: &types.ResultReuseConfiguration{
			ResultReuseByAgeConfiguration: &types.ResultReuseByAgeConfiguration{
				Enabled:         true,
				MaxAgeInMinutes: &MaxAgeInMinutes,
			},
		},
	}

	newContext := context.TODO()

	result, err := client.StartQueryExecution(newContext, &input)
	if err != nil {
		core.Log(err)
		return nil, err
	}
	core.Log("StartQueryExecution result:")
	core.Log(result.QueryExecutionId)

	getQueryExecutionInput := athena.GetQueryExecutionInput{
		QueryExecutionId: result.QueryExecutionId,
	}

	var qrop *athena.GetQueryExecutionOutput
	duration := time.Duration(1) * time.Second // Pause for 2 seconds

	for {
		qrop, err = client.GetQueryExecution(context.TODO(), &getQueryExecutionInput)
		if err != nil {
			core.Log(err)
			return nil, err
		}
		if qrop.QueryExecution.Status.State != "RUNNING" && qrop.QueryExecution.Status.State != "QUEUED" {
			break
		}
		core.Log("waiting.")
		time.Sleep(duration)

	}
	if qrop.QueryExecution.Status.State == "SUCCEEDED" {

		var NextToken *string = nil
		var ResultSet *types.ResultSet = nil
		// MaxResults := int32(100000)

		for {
			input := athena.GetQueryResultsInput{
				QueryExecutionId: getQueryExecutionInput.QueryExecutionId,
				NextToken:        NextToken,
			}

			op, err := client.GetQueryResults(context.TODO(), &input)
			if err != nil {
				core.Log(err)
				return nil, err
			}

			core.Log("Rows athena obtenidos:: ", len(op.ResultSet.Rows))
			core.Log("Next result obtenido:: ", NextToken)

			if ResultSet == nil {
				ResultSet = op.ResultSet
			} else {
				ResultSet.Rows = append(ResultSet.Rows, op.ResultSet.Rows...)
			}

			if NextToken == nil || *NextToken == "" {
				break
			}
		}
		return ResultSet, nil
	} else {
		core.Print(qrop.QueryExecution.Status)
		return nil, err
	}
}

func QueryAthenaX[T any](params QueryAthenaInput, records *[]T) error {

	client := athena.NewFromConfig(core.GetAwsConfig(3))
	workgroup := "primary"

	input := athena.StartQueryExecutionInput{
		WorkGroup:   &workgroup,
		QueryString: &params.QueryString,
		QueryExecutionContext: &types.QueryExecutionContext{
			Database: &params.Database,
		},
		ResultConfiguration: &types.ResultConfiguration{
			OutputLocation: &params.OutputLocation,
		},
	}

	if params.ReuseMaxAge > 0 {
		maxAge := int32(params.ReuseMaxAge)

		input.ResultReuseConfiguration = &types.ResultReuseConfiguration{
			ResultReuseByAgeConfiguration: &types.ResultReuseByAgeConfiguration{
				Enabled:         true,
				MaxAgeInMinutes: &maxAge,
			},
		}
	}

	newContext := context.TODO()

	result, err := client.StartQueryExecution(newContext, &input)
	if err != nil {
		core.Log(err)
		return err
	}
	core.Log("StartQueryExecution result:")
	core.Log(result.QueryExecutionId)

	getQueryExecutionInput := athena.GetQueryExecutionInput{
		QueryExecutionId: result.QueryExecutionId,
	}

	var qrop *athena.GetQueryExecutionOutput
	duration := time.Duration(1) * time.Second // Pause for 2 seconds

	for {
		qrop, err = client.GetQueryExecution(context.TODO(), &getQueryExecutionInput)
		if err != nil {
			core.Log(err)
			return err
		}
		if qrop.QueryExecution.Status.State != "RUNNING" && qrop.QueryExecution.Status.State != "QUEUED" {
			break
		}
		core.Log("waiting.")
		time.Sleep(duration)

	}
	if qrop.QueryExecution.Status.State == "SUCCEEDED" {

		fileName := *result.QueryExecutionId + ".csv"
		filePath := core.Concat("/", params.OutputLocation, fileName)

		core.Log("retornando archivo athena:: ", filePath)
		location := strings.Split(params.OutputLocation, "/")
		core.Log("path file:: ", location[3])
		core.Log("name file:: ", location[2])

		output, err := GetFileFromS3(FileToS3Args{
			Account: 3,
			Bucket:  location[2],
			Path:    location[3],
			Name:    fileName,
		})

		if err != nil {
			return err
		}

		core.Log("output recibido::", len(output))

		if err := gocsv.UnmarshalBytes(output, records); err != nil {
			// Load clients from file
			panic(err)
		}
		core.Log("Registros parseados del .CSV:: ", len(*records))

		return nil
	} else {
		core.Print(qrop.QueryExecution.Status)
		return err
	}
}
