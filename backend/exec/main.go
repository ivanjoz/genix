package exec

import (
	"app/core"
)

type ExecRouterType map[string]func(args *core.ExecArgs) core.FuncResponse

var ExecHandlers = ExecRouterType{
	"fn-init":              ConfigInit,
	"fn-importar-ciudades": ImportCiudades,
	"fn-backup":            CreateBackupFile,
	"fn-homologate":        Homologate,
	"compress-image":       CompressImage,
}

var ExecHandlersCron = ExecRouterType{}

var ExecHandlersTesting = ExecRouterType{
	"fn10":  TestScyllaDBConnection,
	"fn11":  TestScyllaDBInsert,
	"fn12":  TestZstdCompression,
	"fn13":  TestDynamoCounter,
	"fn014": Test14,
	"fn015": Test15,
	"fn016": Test16,
	"fn018": Test18,
	"fn019": Test19,
	"fn020": Test20,
	"fn021": Test21,
	"fn022": Test22,
	"fn023": Test23,
}
