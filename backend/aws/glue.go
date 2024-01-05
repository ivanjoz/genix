package aws

import (
	"app/core"
	"context"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"
)

type GlueTableSchemaParsed struct {
	Order    int32
	Key      string
	ParqKey  string
	Type     string
	ParqType string
	GlueType string
}

type ParqReg interface {
	GetParKeys() []string
}

type ParqHash interface {
	GetHashProps() []any
}

type GlueTable[T ParqReg] struct {
	Table            string
	Database         string
	S3Path           string
	PartitionKeys    []string
	PartitionIndexes [][]string
	AcountAWS        uint8
	S3Bucket         string
	Schema           T
	SchemaParsed     []GlueTableSchemaParsed
	SchemaParsedMap  map[string]GlueTableSchemaParsed
	MakeHash         func(ParqHash) []any
	UseNewParquet    bool
}

var ParquetTypeToGlue = map[string]string{
	"INT32":            "int",
	"INT_32":           "int",
	"INT_16":           "smallint",
	"INT_8":            "tinyint",
	"UINT_8":           "tinyint",
	"INT_64":           "bigint",
	"DATE":             "date",
	"FLOAT":            "float",
	"DOUBLE":           "double",
	"TIMESTAMP_MILLIS": "timestamp",
	"TIME_MILLIS":      "timestamp",
	"UTF8":             "string",
	"BOOLEAN":          "boolean",
}

var GoTypeToGlue = map[string]string{
	"int":     "int",
	"int16":   "smallint",
	"string":  "string",
	"float32": "float",
	"float64": "double",
}

func makeGlueClient(acount uint8) *glue.Client {
	if acount == 0 {
		acount = 1
	}
	client := glue.NewFromConfig(core.GetAwsConfig(acount))
	return client
}

// Crea el Storage descriptor para la Glue Table
func MakeStorageDescriptor(args *GlueTable[ParqReg]) *types.StorageDescriptor {
	ParseGlueTableSchema(args)

	InputFormat := "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat"
	OutputFormat := "org.apache.hadoop.hive.ql.io.parquet.MapredParquetOutputFormat"
	SerDeLibrary := "org.apache.hadoop.hive.ql.io.parquet.serde.ParquetHiveSerDe"

	S3Path := ""
	if len(S3Path) > 0 {
		S3Path = args.S3Path + "/"
	}

	s3Bucket := core.Env.S3_BUCKET

	Location := "s3://" + s3Bucket + "/" + S3Path + args.Table
	columnsMap := map[string]bool{}
	columns := []types.Column{}

	for i := range args.SchemaParsed {
		column := args.SchemaParsed[i]

		if len(column.ParqKey) == 0 || len(column.GlueType) == 0 {
			panic("La columna " + column.Key + " no contiene un ParquetKey o un GlueType")
		}
		// Revisa si la columna se encuentra ya agregada dentro de las particiones
		isIncluded := false
		for _, value := range args.PartitionKeys {
			if value == column.ParqKey {
				isIncluded = true
			}
		}
		if isIncluded {
			continue
		}

		if _, ok := columnsMap[column.ParqKey]; ok {
			panic("Hay una columna repetida en la definición del .parquet: " + column.ParqKey)
		} else {
			columnsMap[column.ParqKey] = true
			columns = append(columns, types.Column{
				Name: &column.ParqKey,
				Type: &column.GlueType,
			})
		}
	}

	storageDescriptor := types.StorageDescriptor{
		Columns:      columns,
		InputFormat:  &InputFormat,
		OutputFormat: &OutputFormat,
		Location:     &Location,
		SerdeInfo: &types.SerDeInfo{
			SerializationLibrary: &SerDeLibrary,
			Parameters:           map[string]string{"serialization.format": "1"},
		},
		Parameters: map[string]string{"classification": "parquet"},
		Compressed: false,
		// NumberOfBuckets: 1,
	}
	return &storageDescriptor
}

// Parsea el esquema del GlueTable
func ParseGlueTableSchema(args *GlueTable[ParqReg]) {
	// Revisa si ya está parseado para no volverlo a parsear
	if args.SchemaParsed != nil && len(args.SchemaParsed) > 0 {
		return
	}

	args.SchemaParsed = []GlueTableSchemaParsed{}

	schemaReflect := reflect.ValueOf(args.Schema)
	schemaTypes := schemaReflect.Type()

	// Itera sobre el struct reflejado
	for i := 0; i < schemaTypes.NumField(); i++ {
		f := schemaTypes.Field(i)

		order := int32(i + 1)
		orderTag := f.Tag.Get("i")
		if len(orderTag) > 0 {
			order = core.SrtToInt(orderTag)
		}

		// Nuevo parquet
		if args.UseNewParquet {
			fieldType := strings.ReplaceAll(f.Type.Name(), "*", "")
			// Revisa si posee el Tag:parquet
			if _, ok := f.Tag.Lookup("parquet"); ok {
				tagValues := f.Tag.Get("parquet")
				// Revisa el type de GO
				if glueType, ok := GoTypeToGlue[fieldType]; ok {
					args.SchemaParsed = append(args.SchemaParsed, GlueTableSchemaParsed{
						Order:    order,
						ParqKey:  strings.Split(tagValues, ",")[0],
						ParqType: fieldType,
						GlueType: glueType,
					})
				}
			}
			continue
		}

		if _, ok := f.Tag.Lookup("parquet"); ok {
			tagValues := f.Tag.Get("parquet")
			tagValuesMap := map[string]string{}
			// Itera sobre los valores del Tag: ejemplo "name=bool_type, type=BOOLEAN"
			for _, value := range strings.Split(tagValues, ",") {
				value = strings.TrimSpace(value)
				valuePairs := strings.Split(value, "=")
				tagValuesMap[valuePairs[0]] = valuePairs[1]
			}
			parqKey, ok_1 := tagValuesMap["name"]
			pType, ok_2 := tagValuesMap["type"]

			if !ok_1 || !ok_2 {
				core.Log(f)
				panic("No se encontró el nombre o type de la columna parquet")
			}
			// core.Log("Parquet Name: ", parqKey, " | Type: ", pType)

			// Aquí revisa el nombre y el tipo de la propiedad en glue y parquet
			parqType := tagValuesMap["convertedtype"]
			if parqType == "*" {
				if pType == "INT64" {
					parqType = "TIMESTAMP_MILLIS"
				}
			} else if len(parqType) == 0 {
				parqType = tagValuesMap["type"]
			}

			// Revisa si el tipo de dato parquet corresponde con un tipo de dato en glue
			glueType, ok_1 := ParquetTypeToGlue[parqType]
			if !ok_1 {
				core.Log(tagValuesMap)
				panic("No se encontró al GlueType del ParquetType : " + parqType)
			}

			args.SchemaParsed = append(args.SchemaParsed, GlueTableSchemaParsed{
				Order:    order,
				ParqKey:  parqKey,
				ParqType: parqType,
				GlueType: glueType,
			})
		}
	}

	sort.Slice(args.SchemaParsed, func(i, j int) bool {
		return args.SchemaParsed[i].Order < args.SchemaParsed[j].Order
	})

	args.SchemaParsedMap = core.SliceToMapE(args.SchemaParsed,
		func(e GlueTableSchemaParsed) string { return e.ParqKey })

	// Revisa si los partition keys están contenidos dentro del esquema
	for _, value := range args.PartitionKeys {
		_, ok := args.SchemaParsedMap[value]
		if !ok {
			panic("El valor del índice no se encontró en el esquema : " + value)
		}
	}
}

// Crea la Glue Table en base a los argumentos para crear un archivo .parquet
func CreateTable(args *GlueTable[ParqReg]) {
	// core.Log("Creando Tabla Glue...")
	client := makeGlueClient(args.AcountAWS)

	// Revisa si la tabla ya existe
	glueTableCurrent, err := client.GetTable(context.TODO(), &glue.GetTableInput{
		DatabaseName: &args.Database,
		Name:         &args.Table,
	})

	// Crea el storage descriptor y parsea el squema
	StorageDescriptor := MakeStorageDescriptor(args)

	columnsTypesNames := []string{}
	schemaParsed := core.MapToSliceT(args.SchemaParsedMap)
	sort.Slice(schemaParsed,
		func(i, j int) bool { return schemaParsed[i].ParqKey < schemaParsed[j].ParqKey })

	for _, e := range schemaParsed {
		columnsTypesNames = append(columnsTypesNames, core.Concatn(e.ParqKey, e.GlueType))
	}

	columnsHashValues := strings.Join(columnsTypesNames, "|")
	columnsHash := strconv.Itoa(int(core.BasicHash(columnsHashValues)))

	needTableUpdate := 0

	if err == nil { // Significa que la tabla existe
		core.Log("La Tabla", args.Table, "existe en: ", args.Database)
		// Revisa si es nacesario actualizar la tabla
		hash, ok := glueTableCurrent.Table.Parameters["columnsHash"]
		core.Log("Hash previo::", hash, " | hash actual:: ", columnsHash)
		if !ok || hash != columnsHash {
			core.Log("La Tabla", args.Table, "necesita ser actualizada...")
			needTableUpdate = 1
		} else {
			return
		}
	}

	if err != nil {
		errString := err.Error()
		core.Log("Estamos aqui..", errString)
		if strings.Contains(errString, "EntityNotFoundException") {
			core.Log("No se encontró la Tabla:: ", args.Table)
			needTableUpdate = 2
		} else {
			log.Fatalf("failed to list tables, %v", err)
		}
	}

	PartitionColumns := []types.Column{}

	for _, columName := range args.PartitionKeys {
		col := args.SchemaParsedMap[columName]
		if len(col.ParqKey) == 0 {
			core.Print(args.SchemaParsed)
			log.Panicln("No se encontró la columna:: ", columName, "en las particiones::", args.PartitionKeys)
		}
		PartitionColumns = append(PartitionColumns, types.Column{
			Name: &col.ParqKey,
			Type: &col.GlueType,
		})
	}

	PartitionIndexes := []types.PartitionIndex{}
	for i := range args.PartitionIndexes {
		p := args.PartitionIndexes[i]
		PartitionIndexes = append(PartitionIndexes, types.PartitionIndex{
			IndexName: &p[0],
			Keys:      p,
		})
	}

	TableType := "EXTERNAL_TABLE"
	SkipArchive := true

	core.Print(PartitionColumns)

	TableInput := types.TableInput{
		Name:              &args.Table,
		Description:       &args.Table,
		StorageDescriptor: StorageDescriptor,
		PartitionKeys:     PartitionColumns,
		TableType:         &TableType,
		Parameters: map[string]string{
			"typeOfData":      "file",
			"classification":  "parquet",
			"compressionType": "none",
			"sizeKey":         "1000",
			"objectCount":     "1000",
			"recordCount":     "10000",
			"columnsHash":     columnsHash,
		},
	}

	core.Log("Tabla a crear o actualizar:: | update?:", needTableUpdate)
	// core.Print(TableInput)

	// Actualiza la Tabla
	if needTableUpdate == 1 {
		// Actualiza la Tabla si no existe
		output, err := client.UpdateTable(context.TODO(), &glue.UpdateTableInput{
			DatabaseName: &args.Database,
			TableInput:   &TableInput,
			SkipArchive:  &SkipArchive,
		})

		if err != nil {
			core.Log("Hubo un error al actualizar la tabla en Glue:: ", args.Table)
			core.Log(err)
		} else {
			core.Log("Tabla actualizada::", output)
		}
	} else if needTableUpdate == 2 {
		// Crea la Tabla si no existe
		output, err := client.CreateTable(context.TODO(), &glue.CreateTableInput{
			DatabaseName:     &args.Database,
			TableInput:       &TableInput,
			PartitionIndexes: PartitionIndexes,
		})

		if err != nil {
			core.Log("Hubo un error al crear la tabla en Glue:: ", args.Table)
			core.Log(err)
		} else {
			core.Log("Tabla creada::", output)
		}
	}
}

type GluePartition struct {
	Values      []string
	RecordCount int
	ObjectCount int
}

func SaveGluPartitions(args *GlueTable[ParqReg], partitionsAll []GluePartition) error {
	client := makeGlueClient(args.AcountAWS)

	// Agrupa las particiones en grupos de 20 (máximo 25)
	partitionsGroups := [][]GluePartition{{}}
	for _, e := range partitionsAll {
		ln := len(partitionsGroups) - 1
		if len(partitionsGroups[ln]) >= 20 {
			partitionsGroups = append(partitionsGroups, []GluePartition{})
			ln = len(partitionsGroups) - 1
		}
		partitionsGroups[ln] = append(partitionsGroups[ln], e)
	}

	for _, partitions := range partitionsGroups {
		partitionList := []types.PartitionInput{}
		partitionToDeleteList := []types.PartitionValueList{}

		for _, partition := range partitions {

			// Crea el path de la partición
			locationArray := []string{}
			values := []string{}
			for idx, value := range partition.Values {
				locationArray = append(locationArray, args.PartitionKeys[idx]+"="+value)
				values = append(values, value)
			}
			partitionToDeleteList = append(partitionToDeleteList, types.PartitionValueList{
				Values: values,
			})

			storageDescriptor := MakeStorageDescriptor(args)
			location := *storageDescriptor.Location + "/" + strings.Join(locationArray, "/")
			storageDescriptor.Location = &location
			storageDescriptor.Parameters = map[string]string{
				"objectCount":     strconv.Itoa(partition.ObjectCount),
				"recordCount":     strconv.Itoa(partition.RecordCount),
				"compressionType": "none",
				"classification":  "parquet",
				"typeOfData":      "file",
			}

			partInput := types.PartitionInput{
				StorageDescriptor: storageDescriptor,
				Values:            values,
			}
			partitionList = append(partitionList, partInput)
		}

		// Elimina las particiones
		deletedOutput, err := client.BatchDeletePartition(context.TODO(),
			&glue.BatchDeletePartitionInput{
				DatabaseName:       &args.Database,
				TableName:          &args.Table,
				PartitionsToDelete: partitionToDeleteList,
			})

		if err != nil {
			core.Log("Hubo un error al eliminar las particiones")
			core.Log(err.Error())
		}
		if deletedOutput != nil {
			core.Log("Se eliminaron las particiones:: ", len(partitionToDeleteList))
		}

		// Crea las particiones
		createdOutput, err := client.BatchCreatePartition(context.TODO(),
			&glue.BatchCreatePartitionInput{
				DatabaseName:       &args.Database,
				TableName:          &args.Table,
				PartitionInputList: partitionList,
			})

		if err != nil {
			core.Log("Hubo un error al crear las particiones")
			core.Log(err.Error())
		}

		if createdOutput != nil {
			for _, e := range partitionList {
				core.Log(args.Table+". Particion creada: ", core.Concat("/", e.Values))
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
