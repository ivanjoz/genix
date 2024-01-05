package aws

import (
	"app/core"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type QueryLogsArgs struct {
	Account        uint8
	Query          string
	Start          int64
	End            int64
	LogLikeMessage string
	Filter         string
	LogGroups      []string
	Limit          int32
}

type CloudLogRecord struct {
	Time      int64
	Order     int
	Message   string
	RequestId string
	Type      string
}

func parseLogsResults(cloudWachRecords [][]types.ResultField) []CloudLogRecord {

	logRecords := []CloudLogRecord{}

	core.Log("Nº de registros:: ", len(cloudWachRecords))

	for i, cloudWachRecord := range cloudWachRecords {
		logRecord := CloudLogRecord{Order: i + 1}

		for _, value := range cloudWachRecord {
			field := *value.Field
			if field == "@timestamp" {
				date, err := time.Parse("2006-01-02 15:04:05.000", *value.Value)
				if err != nil {
					core.Log("No se reconoció el la fecha::", err)
				}
				logRecord.Time = date.Unix()
			} else if field == "@message" {
				messages := strings.Split(*value.Value, "\t")
				// Mensajes tipo REPORT y END
				if strings.Contains(messages[0], " RequestId: ") {
					msg1 := strings.Split(messages[0], " RequestId: ")
					logRecord.Type = strings.TrimSpace(msg1[0])
					logRecord.RequestId = strings.TrimSpace(msg1[1])
					if len(messages) > 1 {
						logRecord.Message = strings.TrimSpace(strings.Join(messages[1:], "\t"))
					}
					// Mensajes comununes de console.log comienzan con un TimeStamp
				} else if len(messages) > 1 && messages[0][0:2] == "20" {
					logRecord.Type = strings.TrimSpace(messages[2])
					logRecord.RequestId = strings.TrimSpace(messages[1])
					if len(messages) > 3 {
						logRecord.Message = strings.TrimSpace(strings.Join(messages[3:], "\t"))
					}
				} else {
					// core.Log(messages)
					logRecord.Message = strings.TrimSpace(strings.Join(messages, "\t"))
				}
			} /* else {
				core.Log("field:: ", field, *value.Value)
			} */
		}
		logRecords = append(logRecords, logRecord)
	}
	return logRecords
}

func QueryLogsInsights(args QueryLogsArgs) ([]CloudLogRecord, error) {

	client := cloudwatchlogs.NewFromConfig(core.GetAwsConfig(args.Account))

	startQueryInput := cloudwatchlogs.StartQueryInput{
		QueryString:   &args.Query,
		StartTime:     &args.Start,
		EndTime:       &args.End,
		LogGroupNames: args.LogGroups,
		Limit:         &args.Limit,
	}

	startOutput, err := client.StartQuery(context.TODO(), &startQueryInput)
	if err != nil {
		core.Log("error al enviar item", err)
		return nil, err
	}

	queryId := startOutput.QueryId

	// obtiene los resultados del query enviado (pueden ser parciales y debe esperarse a obtenerlos todos)
	getQueryResultsInput := cloudwatchlogs.GetQueryResultsInput{
		QueryId: queryId,
	}

	records := [][]types.ResultField{}

	tryCount := 0
	for tryCount < 10 {
		time.Sleep(1 * time.Second)

		core.Log("recibiendo resultados::", tryCount)
		queryOutput, err := client.GetQueryResults(context.TODO(), &getQueryResultsInput)
		if err != nil {
			core.Log("error al enviar item", err)
			return nil, err
		}

		tryCount++

		status := queryOutput.Status
		core.Log("Query Status...", status.Values())
		if queryOutput.Results != nil {
			core.Log("result quantity::", len(queryOutput.Results))
			core.Print(queryOutput.Statistics)
			// core.Print(queryOutput.Results[0])
		}

		switch status {
		case types.QueryStatusScheduled:
			core.Log("Query is scheduled...")
			continue
		case types.QueryStatusRunning:
			core.Log("Query is running...")
			continue
		case types.QueryStatusComplete:
			core.Log("Query Completed...")
			return parseLogsResults(queryOutput.Results), nil
		case types.QueryStatusFailed:
			return parseLogsResults(records), nil
		default:
			continue
		}
	}

	return parseLogsResults(records), nil
}

type LogRecordMessage1 struct {
	Type    string        `json:"g"`
	Accion  int16         `json:"a"`
	Message []interface{} `json:"m"`
}

type LogRecordInner1 struct {
	Type      string        `json:"g"`
	Ip        string        `json:"i"`
	Params    string        `json:"p"`
	Device    interface{}   `json:"d"`
	UsuarioID int           `json:"u"`
	Accion    int16         `json:"a"`
	FilialID  int16         `json:"f"`
	Message   []interface{} `json:"m"`
}

type LogRecord1 struct {
	Nro       int                 `json:"n"`
	Time      int64               `json:"t"`
	RequestId string              `json:"r"`
	Ip        string              `json:"i"`
	Params    string              `json:"p"`
	Device    interface{}         `json:"d"`
	UsuarioID int                 `json:"u"`
	FilialID  int16               `json:"f"`
	Logs      []LogRecordMessage1 `json:"m"`
}

func QueryLogsParsed(args QueryLogsArgs) ([]*LogRecord1, error) {

	args2 := args
	queryArgs := []string{"fields @timestamp, @message", "sort @timestamp desc"}
	if len(args.LogLikeMessage) > 0 {
		queryArgs = append(queryArgs, "filter @message like '"+args.LogLikeMessage+"'")
	}
	queryArgs = append(queryArgs, "limit "+strconv.Itoa(int(args.Limit)))

	args2.Query = strings.Join(queryArgs, " | ")

	core.Print(args2)

	records, err := QueryLogsInsights(args2)
	recordsParsedMap := map[string]*LogRecord1{}

	for _, e := range records {
		var rec LogRecordInner1
		if len(e.Message) > 8 && e.Message[0:3] == "LOG" {
			err = json.Unmarshal([]byte(e.Message[6:]), &rec)
			if err != nil {
				core.Log("No se pudo parsear el JSON", err)
				core.Log(e.Message[6:])
			}

			logMessage := LogRecordMessage1{
				Accion:  rec.Accion,
				Message: rec.Message,
				Type:    rec.Type,
			}

			if logRecord, ok := recordsParsedMap[e.RequestId]; ok {
				logRecord.Logs = append(logRecord.Logs, logMessage)
			} else {
				recordsParsedMap[e.RequestId] = &LogRecord1{
					Nro:       e.Order,
					Time:      e.Time,
					RequestId: e.RequestId,
					Ip:        rec.Ip,
					Params:    rec.Params,
					Device:    rec.Device,
					UsuarioID: rec.UsuarioID,
					FilialID:  rec.FilialID,
					Logs:      []LogRecordMessage1{logMessage},
				}
			}
		} else {
			core.Log("No se reconoció el mensaje: ", e.Message)
		}
	}

	recordsParsed := []*LogRecord1{}
	for _, e := range recordsParsedMap {
		recordsParsed = append(recordsParsed, e)
	}

	return recordsParsed, err
}

func QueryLogsCloudWatch(args QueryLogsArgs) ([]CloudLogRecord, error) {

	args2 := args
	queryArgs := []string{"fields @timestamp, @message", "sort @timestamp desc"}
	if len(args.LogLikeMessage) > 0 {
		queryArgs = append(queryArgs, "filter @message like '"+args.LogLikeMessage+"'")
	}
	if len(args.Filter) > 0 {
		queryArgs = append(queryArgs, "filter ("+args.Filter+")")
	}
	queryArgs = append(queryArgs, "limit "+strconv.Itoa(int(args.Limit)))

	args2.Query = strings.Join(queryArgs, " | ")

	core.Print(args2)

	records, err := QueryLogsInsights(args2)
	return records, err
}

type GetDBMetricsArgs struct {
	Account      uint8
	EndTime      time.Time
	StartTime    time.Time
	MetricName   string
	Period       int32
	DatabaseName string
}

type DataPoint struct {
	Average     float64
	Maximum     float64
	Minimum     float64
	Sum         float64
	SampleCount int
	Timestamp   int64
	Unit        string
}

func GetDBMetrics(args GetDBMetricsArgs) ([]DataPoint, error) {

	input := cloudwatch.GetMetricStatisticsInput{
		EndTime:    &args.EndTime,
		StartTime:  &args.StartTime,
		MetricName: &args.MetricName,
		Period:     &args.Period,
		Statistics: []cloudwatchTypes.Statistic{
			"Average", "Maximum",
		},
	}

	if len(args.DatabaseName) > 0 {
		name := "DBInstanceIdentifier"
		nameSpace := "AWS/RDS"
		input.Namespace = &nameSpace
		input.Dimensions = append(input.Dimensions, cloudwatchTypes.Dimension{
			Name: &name, Value: &args.DatabaseName,
		})
	}

	core.Print(args)

	client := cloudwatch.NewFromConfig(core.GetAwsConfig(args.Account))
	result, err := client.GetMetricStatistics(context.TODO(), &input)
	if err != nil {
		core.Log("Error en cloudwatch::")
		core.Log(err)
		return nil, err
	}

	datapoints := []DataPoint{}
	for _, e := range result.Datapoints {
		reg := DataPoint{}
		if e.Minimum != nil {
			reg.Minimum = *e.Minimum
		}
		if e.Maximum != nil {
			reg.Maximum = *e.Maximum
		}
		if e.Average != nil {
			reg.Average = *e.Average
		}
		if e.Sum != nil {
			reg.Sum = *e.Sum
		}
		if e.SampleCount != nil {
			reg.SampleCount = int(*e.SampleCount)
		}
		reg.Unit = string(e.Unit)
		reg.Timestamp = e.Timestamp.Unix()

		datapoints = append(datapoints, reg)
	}

	return datapoints, nil
}
