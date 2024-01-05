package aws

import (
	"app/core"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AccionLog struct {
	UsuarioID  int32  `json:"usr"`
	Created    string `json:"crd"`
	Semana     int    `json:"sm,omitempty"`
	FilialID   int16  `json:"fl"`
	Accion     int16  `json:"ac"`
	IP         string `json:"ip,omitempty"`
	DeviceInfo string `json:"dvi,omitempty"`
	Mensaje    string `json:"msg"`
	BodyLen    int    `json:"bl,omitempty"`
	// 4 = Error con cronjobs, 3 = Error, 2 = Log de CronJob, 1 = Log de Accion
	Estado uint8 `json:"ss"`
}

func makeTableLogs() DynamoTableRecords[AccionLog] {
	return DynamoTableRecords[AccionLog]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             "logAc",
		UseCompression: true,
		GetIndexKeys: func(e AccionLog, idx uint8) string {
			switch idx {
			case 0: // este es el sk (sort key)
				return core.Concatn(e.FilialID, e.Estado, e.Created)
			case 1: // ix1
				return core.Concatn(e.UsuarioID, e.Created)
			case 2: // ix2
				return core.Concatn(e.FilialID, e.Accion, e.Created)
			}
			return ""
		},
	}
}

func MakeTableLogs2() DynamoTableRecords[core.ReqLog] {
	pk := "logs"
	if !core.Env.IS_PROD {
		pk = pk + "QAS"
	}

	return DynamoTableRecords[core.ReqLog]{
		TableName:      "smartberry_logs",
		PK:             pk,
		Account:        3,
		UseCompression: true,
		GetIndexKeys: func(e core.ReqLog, idx uint8) string {
			switch idx {
			case 0: // este es el sk (sort key)
				return core.Concatn(e.Created)
			case 1: // ix1
				return core.Concatn(e.UserID, e.Type, e.Created)
			}
			return ""
		},
	}
}

func SaveLog(filialID int16, accion int16, mensaje string, bodyLen int) error {
	core.Log("guardando log...")

	usuarioID := core.Env.USUARIO_ID

	reg := AccionLog{UsuarioID: usuarioID, Mensaje: mensaje, Accion: accion,
		BodyLen: bodyLen, Estado: 1, FilialID: filialID, Created: core.ToBase36(0)}

	core.Log("guardando con ID:: ", reg.Created)

	dynamoTable := makeTableLogs()
	err := dynamoTable.PutItem(&reg, 2)

	return err
}

func SaveLogCron(filialID int16, accion int16, mensaje string) error {
	core.Log("guardando log...")

	reg := AccionLog{UsuarioID: 2, Mensaje: mensaje, Accion: accion,
		BodyLen: 0, Estado: 2, FilialID: filialID, Created: core.ToBase36(0)}

	dynamoTable := makeTableLogs()
	err := dynamoTable.PutItem(&reg, 2)

	return err
}

type ReqCache struct {
	SK      string
	Created int64
	Content string
	Param0  int
	Param1  string
	Param2  string
	Param3  string
	Param4  string
	Param5  string
}

func MakeTableCache() DynamoTableRecords[ReqCache] {
	pk := "cache"
	if !core.Env.IS_PROD {
		pk = pk + "QAS"
	}

	return DynamoTableRecords[ReqCache]{
		TableName:      "smartberry_logs", // "hf-smartberry-db",
		PK:             pk,
		Account:        3,
		UseCompression: false,
		GetIndexKeys: func(e ReqCache, idx uint8) string {
			switch idx {
			case 0: // este es el sk (sort key)
				return e.SK
			}
			return ""
		},
	}
}

type FuncInvokation struct {
	Created  int64    `json:"crd"`
	Duration int      `json:"dur"`
	Message  string   `json:"msg"`
	IsLocal  bool     `json:"isl"`
	Logs     []string `json:"logs"`
}

type FuncLog struct {
	SK          string           `json:"sk"`
	Updated     int64            `json:"updated"`
	Invokations []FuncInvokation `json:"invokations"`
}

func MakeTableFuncLogs() DynamoTableRecords[FuncLog] {
	pk := "eLogs"
	if !core.Env.IS_PROD {
		pk = pk + "QAS"
	}

	return DynamoTableRecords[FuncLog]{
		TableName:      "hf-smartberry-db",
		PK:             pk,
		Account:        3,
		UseCompression: true,
		GetIndexKeys: func(e FuncLog, idx uint8) string {
			switch idx {
			case 0: // este es el SK (Sort Key)
				return e.SK
			case 1: // ix1
				return core.Concat("", e.Updated)
			}
			return ""
		},
	}
}

type FuncLogDetail struct {
	SK      string   `json:"sk"`
	Updated int64    `json:"updated"`
	Logs    []string `json:"logs"`
}

func MakeTableFuncLogsDetail() DynamoTableRecords[FuncLogDetail] {
	pk := "funcLogDe"
	if !core.Env.IS_PROD {
		pk = pk + "QAS"
	}

	return DynamoTableRecords[FuncLogDetail]{
		TableName:      "hf-smartberry-db",
		PK:             pk,
		Account:        3,
		UseCompression: true,
		GetIndexKeys: func(e FuncLogDetail, idx uint8) string {
			switch idx {
			case 0: // este es el SK (Sort Key)
				return e.SK
			}
			return ""
		},
	}
}

func PutFuncLog(funcName, message string, duration int) {
	dynamoTable := MakeTableFuncLogs()
	// Obtiene el registro donde están las últimas invocaciones
	fmt.Println("Obteniendo invocacion previa...")
	rec, err := dynamoTable.GetItem(funcName)
	if err != nil {
		fmt.Println("Hubo un error: ", err.Error())
	}

	var funcLog FuncLog
	updated := time.Now().Unix()

	if rec == nil {
		fmt.Println("Se encontró invocación:: NO = ", funcName)

		funcLog = FuncLog{
			SK:          funcName,
			Updated:     updated,
			Invokations: []FuncInvokation{},
		}
	} else {
		fmt.Println("Se encontró invocación:: SI = ", funcName)
		funcLog = *rec
	}

	fmt.Println("Nro de invocaciones:: ", len(funcLog.Invokations))

	sort.Slice(funcLog.Invokations, func(i, j int) bool {
		return funcLog.Invokations[i].Created > funcLog.Invokations[j].Created
	})
	// Si hay más o igual a 5 invocaciones borra 1
	if len(funcLog.Invokations) >= 5 {
		funcLog.Invokations = funcLog.Invokations[0:4]
	}

	logsMin := []string{}
	logsDetailed := []string{}
	logsDetailedLen := 0

	for _, msg := range core.LogsSaved {
		if len(msg) > 1 && msg[0:1] == "*" {
			logsMin = append(logsMin, msg)
		} else if strings.Contains(strings.ToLower(msg), "error") {
			logsMin = append(logsMin, msg)
		}
		if logsDetailedLen < 500000 /* Máximo 500 kb */ {
			logsDetailedLen += len(msg)
			logsDetailed = append(logsDetailed, msg)
		}
	}

	invokation := FuncInvokation{
		Created:  updated,
		Duration: duration,
		Message:  message,
		IsLocal:  false,
		Logs:     logsMin,
	}

	// Agrega la invocaciones
	funcLog.Invokations = append(funcLog.Invokations, invokation)
	funcLog.Updated = updated

	fmt.Println("Nro de invocaciones (nuevo):: ", len(funcLog.Invokations))

	fmt.Println("Guardando invocación....")
	err = dynamoTable.PutItem(&funcLog, 1)
	if err != nil {
		fmt.Println("Error en PutItem DynamoDB: ", err)
	}

	// Agrega el detalle de la invocación
	MakeTableFuncLogsDetail().PutItem(&FuncLogDetail{
		SK:      core.Concat("_", funcName, updated),
		Updated: updated,
		Logs:    logsDetailed,
	}, 1)
}
