package exec

import (
	"app/core"
)

type ExecRouterType map[string]func(args *core.ExecArgs) core.FuncResponse

var ModuleHandlers = core.AppRouterType{
	"GET.backups":         GetBackups,
	"POST.backup-restore": RestoreBackup,
	"POST.backup-create":  CreateBackup,
}

var ExecHandlers = ExecRouterType{
	"fn-init":              ConfigInit,
	"fn-importar-ciudades": ImportCiudades,
	"fn-exportar-ciudades": ExportCiudades,
	"fn-backup":            DoSaveBackup,
	"fn-homologate":        Homologate,
	"fn-recalc":            RecalcVirtualColumnsValues,
	"compress-image":       CompressImage,
}

var ExecHandlersCron = ExecRouterType{}

var ExecHandlersTesting = ExecRouterType{
	// "fn10":  TestScyllaDBConnection,
	"fn11":  TestScyllaDBInsert,
	"fn12":  TestZstdCompression,
	"fn13":  TestDynamoCounter,
	"fn15":  TestInsert,
	"fn16":  TestQuery,
	"fn17":  TestCBOR,
	"fn18":  TestDeploy,
	"fn014": Test14,
	"fn015": Test15,
	"fn016": Test16,
	"fn018": Test18,
	"fn019": Test19,
	"fn020": Test20,
	"fn021": Test21,
	"fn022": Test22,
	"fn023": Test23,
	"fn024": Test24,
	"fn025": Test25,
	"fn026": Test26,
	"fn027": Test27,
	"fn028": Test28,
	"fn029": Test29,
	"fn030": Test30,
	"fn032": Test32,
	"fn033": Test33,
	"fn034": Test34,
	"fn035": Test35,
	"fn036": Test36,
}
