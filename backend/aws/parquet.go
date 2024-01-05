package aws

import (
	"app/core"
	"log"
	"os"
	"strings"

	Parquet "github.com/parquet-go/parquet-go"
)

type recordGroup struct {
	records    []ParqReg
	parKeys    []string
	key        string
	folderPath string
}

type WriterConfig struct{}

func (e WriterConfig) ConfigureWriter(cf *Parquet.WriterConfig) {
	codec := Parquet.Zstd.CompressionCodec()
	cf.Compression = Parquet.LookupCompressionCodec(codec)
}

func SaveRecordsToParquet[T ParqReg](args *GlueTable[ParqReg], records []T) {

	// Primero revisa si el esquema está adecuado
	if len(args.Database) < 3 || len(args.Table) < 3 {
		core.Log("No se ha especificado un Glue Database, Table o S3Path")
		return
	}
	if len(args.PartitionKeys) < 1 {
		core.Log("No se han especificado las claves de particionado")
		return
	}

	// Agrupa los registros según su clave de particion
	recordsGroupMap := map[string]*recordGroup{}

	for _, record := range records {
		parKeys := record.GetParKeys()
		key := strings.Join(parKeys, "_")

		if len(parKeys) != len(args.PartitionKeys) {
			panic("El número de keys obtenidas es distinto al numero de particiones. Glue Table: " + args.Table)
		}

		recordsGroup, ok := recordsGroupMap[key]
		if !ok {
			folderPathArray := []string{}
			for idx, parValue := range parKeys {
				parKey := args.PartitionKeys[idx]
				folderPathArray = append(folderPathArray, parKey+"="+parValue)
			}
			group := recordGroup{
				records:    []ParqReg{record},
				key:        key,
				folderPath: strings.Join(folderPathArray, "/"),
				parKeys:    parKeys,
			}
			recordsGroupMap[key] = &group
			//	fmt.Println("folder path:: ", folderPathArray)
			//core.Print(record)
		} else {
			recordsGroup.records = append(recordsGroup.records, record)
			// core.Log("se agregaron 1 registro::", key, len(recordsGroup.records))
		}
	}

	// Crea las particiones
	gluePartitions := []GluePartition{}

	for _, group := range recordsGroupMap {
		gluePartitions = append(gluePartitions, GluePartition{
			Values:      group.parKeys,
			ObjectCount: 1,
			RecordCount: len(group.records),
		})
	}

	// Guardando las particiones
	// core.Print(gluePartitions)
	core.Log("Revisando si la Tabla Existe")
	CreateTable(args)
	core.Log("Guardando las particiones")

	SaveGluPartitions(args, gluePartitions)

	// Por cada uno de los grupos genera el archivo .parquet
	for parkey, group := range recordsGroupMap {
		localFilePath := "/tmp/" + parkey + ".parquet"

		var err error

		f, _ := os.Create(localFilePath)
		writer := Parquet.NewWriter(f, WriterConfig{})

		for _, record := range group.records {
			if err := writer.Write(record); err != nil {
				log.Fatal(err)
			}
		}
		_ = writer.Close()
		_ = f.Close()

		// Envía el archivo .parquet a s3
		S3Path := ""
		if len(S3Path) > 0 {
			S3Path = args.S3Path + "/"
		}

		err = SendFileToS3(FileToS3Args{
			Account:       args.AcountAWS,
			Bucket:        core.Env.S3_BUCKET,
			Path:          S3Path + args.Table + "/" + group.folderPath,
			Name:          "records.parquet",
			LocalFilePath: localFilePath,
		})
		if err != nil {
			core.Log("Error en el envío a S3", err)
		}
	}
}
