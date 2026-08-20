package config

import (
	"app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Las columnas de la entitlement se escriben una por una en vez de embeber CompanyCreditBudget:
// minijson (el encoder de las respuestas) no aplana los campos anónimos, así que un struct embebido
// llega al cliente anidado bajo el nombre de su tipo y allí cada campo promovido queda undefined.
type companyCreditBudgetResponse struct {
	CompanyID               int32
	DailyCPU                int64
	DailyInference          int64
	BudgetMonthStartDay     int16
	MonthlyCPUCeiling       int64
	MonthlyInferenceCeiling int64
	LastSetCPU              int64
	LastSetInference        int64
	Updated                 int32
	MonthCPUUsed            uint64
	MonthInferenceUsed      uint64
	CurrentCPU              uint64
	CurrentInference        uint64
	// El gasto del día y lo que queda de la asignación diaria. El límite diario se rechaza por
	// separado del techo mensual, así que una empresa con crédito mensual de sobra puede estar
	// bloqueada hasta mañana y sólo estas dos cifras lo explican.
	DayCPUUsed              uint64
	DayInferenceUsed        uint64
	DailyRemainingCPU       uint64
	DailyRemainingInference uint64
	// El pool diario de créditos extra: lo que la company puede gastar en lecturas DESPUÉS de que su
	// cuota la haya rechazado. Viaja aunque no haya presupuesto del mes en curso, y eso no es un
	// descuido: esa es precisamente la company que vive del pool, así que esconderlo dejaría el modo
	// de sólo lectura invisible justo cuando está en uso.
	ExtraCPU          int64
	DayExtraCPUUsed   uint64
	ExtraRemainingCPU uint64
	IsCurrentMonth    bool
}

// El panel guarda con un solo botón, así que la petición es la lista de filas que el usuario tocó
// y no una operación suelta: un único viaje deja el presupuesto consistente y devuelve un solo
// estado para repintar.
type companyCreditBudgetMutationRequest struct {
	CompanyID  int32
	Operations []companyCreditBudgetOperationRequest
}

type companyCreditBudgetOperationRequest struct {
	Operation string
	CPU       int64
	Inference int64
}

// La operación ya validada y traducida al código que entiende el daemon.
type companyCreditBudgetOperation struct {
	code      core.BudgetOperation
	cpu       uint64
	inference uint64
}

func GetCompanyCreditBudget(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.GetQueryInt("target-company-id")
	if companyID <= 0 {
		return req.MakeErr("Debe enviar una empresa válida.")
	}
	if _, err := requireCompany(companyID); err != nil {
		return req.MakeErr("No se pudo validar la empresa solicitada.", err)
	}

	budget, err := getCompanyCreditBudget(companyID)
	if err != nil {
		core.Log("company credit budget read failed::", " company::", companyID, " err::", err)
		return req.MakeErr("No se pudo obtener el presupuesto de créditos.", err)
	}
	core.Log("company credit budget read::", " company::", companyID,
		" current_month::", budget.IsCurrentMonth, " current_cpu::", budget.CurrentCPU,
		" current_inference::", budget.CurrentInference)
	return req.MakeResponse(budget)
}

func PostCompanyCreditBudget(req *core.HandlerArgs) core.HandlerResponse {
	mutation := companyCreditBudgetMutationRequest{}
	if req.Body == nil || json.Unmarshal([]byte(*req.Body), &mutation) != nil {
		return req.MakeErr("El presupuesto de créditos enviado no es válido.")
	}
	if mutation.CompanyID <= 0 {
		return req.MakeErr("Debe enviar una empresa válida.")
	}
	operations, validationError := parseCompanyCreditBudgetOperations(mutation.Operations)
	if validationError != nil {
		return req.MakeErr("Las operaciones del presupuesto de créditos no son válidas.", validationError)
	}
	if _, err := requireCompany(mutation.CompanyID); err != nil {
		return req.MakeErr("No se pudo validar la empresa solicitada.", err)
	}

	requestContext := context.Background()
	if req.ReqContext != nil {
		requestContext = req.ReqContext.Context()
	}
	previousRecord, recordError := getCompanyCreditBudgetRecord(mutation.CompanyID)
	if recordError != nil {
		core.Log("company credit budget read failed::", " company::", mutation.CompanyID, " err::", recordError)
		return req.MakeErr("No se pudo leer el presupuesto de créditos.", recordError)
	}
	core.Log("company credit budget mutation started::", " company::", mutation.CompanyID,
		" operations::", len(operations))
	// El daemon sólo acepta una operación por llamada, así que un guardado con varias filas no es
	// atómico: si la segunda falla, la primera ya quedó escrita. El cliente vuelve a leer el
	// presupuesto cuando esto devuelve error, que es lo que resuelve la ambigüedad.
	for _, operation := range operations {
		err := core.MutateCompanyCreditBudget(
			requestContext, mutation.CompanyID, operation.code, operation.cpu, operation.inference,
		)
		if errors.Is(err, core.ErrBudgetMonthNotConfigured) {
			return req.MakeErr("Primero debe establecer el presupuesto actual del mes.")
		}
		if errors.Is(err, core.ErrBudgetMutationOverflow) {
			return req.MakeErr("El presupuesto de créditos excede el máximo permitido.")
		}
		if err != nil {
			core.Log("company credit budget mutation unavailable::", " company::", mutation.CompanyID,
				" operation::", operation.code, " err::", err)
			return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
		}
		previousRecord = registerCreditHistory(req, previousRecord, operation)
	}

	budget, err := getCompanyCreditBudget(mutation.CompanyID)
	if err != nil {
		core.Log("company credit budget refresh failed::", " company::", mutation.CompanyID, " err::", err)
		return req.MakeErr("El presupuesto cambió, pero no se pudo volver a leer.", err)
	}
	core.Log("company credit budget mutation completed::", " company::", mutation.CompanyID,
		" operations::", len(operations), " current_cpu::", budget.CurrentCPU,
		" current_inference::", budget.CurrentInference)
	return req.MakeResponse(budget)
}

// registerCreditHistory anota en el ledger el movimiento que dejó una mutación que el daemon ya
// confirmó, y devuelve el estado recién leído para que la siguiente operación del mismo guardado
// compare contra él. Un fallo aquí se registra pero no aborta la respuesta: los créditos ya se
// movieron, así que devolver error diría que no pasó nada, y eso es peor que perder una fila de
// auditoría.
func registerCreditHistory(
	req *core.HandlerArgs,
	previous coreTypes.CompanyCreditBudget,
	operation companyCreditBudgetOperation,
) coreTypes.CompanyCreditBudget {
	current, err := getCompanyCreditBudgetRecord(previous.CompanyID)
	if err != nil {
		core.Log("credit history read failed::", " company::", previous.CompanyID, " err::", err)
		return previous
	}
	cpuMoved, inferenceMoved := creditHistoryMovement(previous, current)
	// El límite diario no otorga créditos, así que no deja fila: sin movimiento no hay nada que anotar.
	if cpuMoved == 0 && inferenceMoved == 0 {
		return current
	}
	entry := coreTypes.CreditHistory{
		CompanyID:        previous.CompanyID,
		Day:              core.FechaUnix(),
		Created:          core.SUnixTime(),
		CreatedBy:        req.User.ID,
		Operation:        int8(operation.code),
		CPUCredits:       cpuMoved,
		InferenceCredits: inferenceMoved,
		CPUCeiling:       current.MonthlyCPUCeiling,
		InferenceCeiling: current.MonthlyInferenceCeiling,
	}
	if insertError := db.InsertOne(entry); insertError != nil {
		core.Log("credit history insert failed::", " company::", previous.CompanyID,
			" operation::", operation.code, " cpu::", cpuMoved, " inference::", inferenceMoved,
			" err::", insertError)
		return current
	}
	core.Log("credit history saved::", " company::", previous.CompanyID, " operation::", operation.code,
		" cpu::", cpuMoved, " inference::", inferenceMoved, " cpu_ceiling::", current.MonthlyCPUCeiling)
	return current
}

// creditHistoryMovement traduce dos estados del presupuesto al crédito que se otorgó (positivo) o se
// quitó (negativo). Es la diferencia entre techos, salvo cuando la mutación estrenó un mes: ahí el
// techo anterior pertenece a un presupuesto que ya caducó, así que lo otorgado es la cifra que se
// acaba de fijar y restar contra el mes viejo daría un número sin sentido.
func creditHistoryMovement(previous, current coreTypes.CompanyCreditBudget) (int64, int64) {
	if current.BudgetMonthStartDay != previous.BudgetMonthStartDay {
		return current.LastSetCPU, current.LastSetInference
	}
	return current.MonthlyCPUCeiling - previous.MonthlyCPUCeiling,
		current.MonthlyInferenceCeiling - previous.MonthlyInferenceCeiling
}

// Las operaciones se aplican ordenadas por su código —diario, reemplazar, aumentar— y no en el
// orden en que llegaron: "aumentar" sobre un mes sin presupuesto es un rechazo del daemon, así que
// reemplazar primero es lo que permite mandar ambas filas en el mismo guardado. Un código repetido
// se rechaza porque dos valores para la misma operación no tienen un resultado único.
func parseCompanyCreditBudgetOperations(
	requested []companyCreditBudgetOperationRequest,
) ([]companyCreditBudgetOperation, error) {
	if len(requested) == 0 {
		return nil, errors.New("no operations were sent")
	}
	operations := make([]companyCreditBudgetOperation, 0, len(requested))
	for _, request := range requested {
		code, err := parseCompanyCreditBudgetOperation(request.Operation)
		if err != nil {
			return nil, err
		}
		if request.CPU < 0 || request.Inference < 0 {
			return nil, fmt.Errorf("operation %q carries negative credits", request.Operation)
		}
		if slices.ContainsFunc(operations, func(seen companyCreditBudgetOperation) bool {
			return seen.code == code
		}) {
			return nil, fmt.Errorf("operation %q was sent twice", request.Operation)
		}
		operations = append(operations, companyCreditBudgetOperation{
			code: code, cpu: uint64(request.CPU), inference: uint64(request.Inference),
		})
	}
	slices.SortFunc(operations, func(left, right companyCreditBudgetOperation) int {
		return int(left.code) - int(right.code)
	})
	return operations, nil
}

func requireCompany(companyID int32) (*types.Company, error) {
	company, err := getCompanyByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, fmt.Errorf("company %d does not exist", companyID)
	}
	return company, nil
}

func parseCompanyCreditBudgetOperation(operation string) (core.BudgetOperation, error) {
	switch operation {
	case "set-daily":
		return core.BudgetSetDaily, nil
	case "set-current":
		return core.BudgetSetCurrent, nil
	case "increase-current":
		return core.BudgetIncreaseCurrent, nil
	default:
		return 0, fmt.Errorf("unknown operation %q", operation)
	}
}

func getCompanyCreditBudget(companyID int32) (companyCreditBudgetResponse, error) {
	record, err := getCompanyCreditBudgetRecord(companyID)
	if err != nil {
		return companyCreditBudgetResponse{}, err
	}
	response := makeCompanyCreditBudgetResponse(record, core.Env.COMPANY_EXTRA_CREDITS_24H)
	// El daemon nunca escribió los contadores de esta empresa, así que no hay nada que restar y las
	// cifras se reconstruyen igual que él las reconstruye al arrancar: sumando los días del mes. Sólo
	// pasa con filas escritas antes de que el flush existiera, y se corrige solo en el primer cargo.
	if response.IsCurrentMonth && record.UsageUpdated == 0 {
		if err := fillCompanyCreditUsageFromRows(&response, record); err != nil {
			return companyCreditBudgetResponse{}, err
		}
	}
	return response, nil
}

// makeCompanyCreditBudgetResponse traduce una fila almacenada a la vista del panel.
//
// Los contadores los publica el propio daemon en cada flush (quota.rs: flush_dirty_budget_usage),
// que son los mismos con los que decide admitir o rechazar un cargo. Por eso se restan tal cual en
// vez de volver a sumar las filas de uso: sumarlas otra vez aquí sería una segunda implementación de
// la misma cuenta, y en cuanto las dos divergieran el panel mostraría un crédito que el limiter no
// concede. La contrapartida es un retraso de como máximo un intervalo de flush.
//
// Las columnas de período dicen a qué ventana pertenece cada contador: nadie reescribe la fila
// cuando cambia el día o el mes sin tráfico, así que un período que no es el actual significa que
// esa ventana todavía no gastó nada.
// extraPoolCeiling es un parámetro y no una lectura de core.Env aquí dentro: esta función es pura y
// los tests la llaman directamente, donde Env es un puntero nil. Un cero significa "no se sabe" y el
// panel esconde el tramo, que es exactamente el caso Lambda.
func makeCompanyCreditBudgetResponse(
	record coreTypes.CompanyCreditBudget, extraPoolCeiling int64,
) companyCreditBudgetResponse {
	response := companyCreditBudgetResponse{
		CompanyID:               record.CompanyID,
		DailyCPU:                record.DailyCPU,
		DailyInference:          record.DailyInference,
		BudgetMonthStartDay:     record.BudgetMonthStartDay,
		MonthlyCPUCeiling:       record.MonthlyCPUCeiling,
		MonthlyInferenceCeiling: record.MonthlyInferenceCeiling,
		LastSetCPU:              record.LastSetCPU,
		LastSetInference:        record.LastSetInference,
		Updated:                 record.Updated,
		IsCurrentMonth:          record.BudgetMonthStartDay == currentMonthStartDay(),
		ExtraCPU:                extraPoolCeiling,
	}
	// El pool se resuelve antes del retorno anticipado de abajo, no después, porque no depende del
	// presupuesto mensual: lo concede la configuración de la plataforma y sólo lo acota el día.
	if record.UsageDayPeriod == currentCreditUnixDay() {
		response.DayExtraCPUUsed = uint64(max(record.DayExtraCPUUsed, 0))
	}
	response.ExtraRemainingCPU = remainingCredits(response.ExtraCPU, response.DayExtraCPUUsed)
	// Sin presupuesto del mes en curso el daemon rechaza todo cargo, así que no hay crédito que
	// mostrar y el gasto acumulado pertenece a un presupuesto que ya caducó.
	if !response.IsCurrentMonth {
		return response
	}
	if record.UsageMonthStartDay == record.BudgetMonthStartDay {
		response.MonthCPUUsed = uint64(max(record.MonthCPUUsed, 0))
		response.MonthInferenceUsed = uint64(max(record.MonthInferenceUsed, 0))
	}
	if record.UsageDayPeriod == currentCreditUnixDay() {
		response.DayCPUUsed = uint64(max(record.DayCPUUsed, 0))
		response.DayInferenceUsed = uint64(max(record.DayInferenceUsed, 0))
	}
	response.CurrentCPU = remainingCredits(record.MonthlyCPUCeiling, response.MonthCPUUsed)
	response.CurrentInference = remainingCredits(record.MonthlyInferenceCeiling, response.MonthInferenceUsed)
	response.DailyRemainingCPU = remainingCredits(record.DailyCPU, response.DayCPUUsed)
	response.DailyRemainingInference = remainingCredits(record.DailyInference, response.DayInferenceUsed)
	return response
}

// fillCompanyCreditUsageFromRows suma los días del mes de una empresa, que es la vía de recuperación
// del daemon (quota.rs: ensure_budget) y aquí el respaldo para una fila sin contadores. La fila del
// día en curso sale de la misma lectura, así que la parte diaria no cuesta una consulta extra.
func fillCompanyCreditUsageFromRows(
	response *companyCreditBudgetResponse,
	record coreTypes.CompanyCreditBudget,
) error {
	rows := []coreTypes.CreditUsageCompany{}
	currentFrame := currentDailyTimeFrame()
	usageQuery := db.Query(&rows)
	usageQuery.CompanyID.Equals(record.CompanyID).
		TimeFrame.Between(dailyTimeFramePrefix+int32(record.BudgetMonthStartDay), currentFrame)
	if err := usageQuery.Exec(); err != nil {
		return err
	}
	response.MonthCPUUsed, response.MonthInferenceUsed = 0, 0
	response.DayCPUUsed, response.DayInferenceUsed = 0, 0
	for _, row := range rows {
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			return fmt.Errorf("invalid usage in time frame %d: %w", row.TimeFrame, err)
		}
		response.MonthCPUUsed += totals.CPU
		response.MonthInferenceUsed += totals.Inference
		if row.TimeFrame == currentFrame {
			response.DayCPUUsed = totals.CPU
			response.DayInferenceUsed = totals.Inference
		}
	}
	response.CurrentCPU = remainingCredits(record.MonthlyCPUCeiling, response.MonthCPUUsed)
	response.CurrentInference = remainingCredits(record.MonthlyInferenceCeiling, response.MonthInferenceUsed)
	response.DailyRemainingCPU = remainingCredits(record.DailyCPU, response.DayCPUUsed)
	response.DailyRemainingInference = remainingCredits(record.DailyInference, response.DayInferenceUsed)
	return nil
}

// getCompanyCreditBudgets lee la entitlement de todas las empresas de una vez, para el panel de
// tarjetas. Una fila corta por empresa y sin partición que filtrar: es la misma forma de lectura que
// getCompaniesUpdatedSince hace sobre el catálogo de empresas. A diferencia del endpoint de una
// empresa no hay respaldo por sumas: serían tantos escaneos de mes como empresas, y una empresa que
// nunca gastó un crédito los pagaría en cada refresco del panel para leer ceros.
func getCompanyCreditBudgets() ([]companyCreditBudgetResponse, error) {
	records := []coreTypes.CompanyCreditBudget{}
	if err := db.Query(&records).Exec(); err != nil {
		return nil, err
	}
	budgets := make([]companyCreditBudgetResponse, 0, len(records))
	for _, record := range records {
		if record.CompanyID <= 0 {
			continue
		}
		budgets = append(budgets,
			makeCompanyCreditBudgetResponse(record, core.Env.COMPANY_EXTRA_CREDITS_24H))
	}
	return budgets, nil
}

func getCompanyCreditBudgetRecord(companyID int32) (coreTypes.CompanyCreditBudget, error) {
	records := []coreTypes.CompanyCreditBudget{}
	query := db.Query(&records)
	query.CompanyID.Equals(companyID).Limit(1)
	if err := query.Exec(); err != nil {
		return coreTypes.CompanyCreditBudget{}, err
	}
	record := coreTypes.CompanyCreditBudget{CompanyID: companyID}
	if len(records) > 0 {
		record = records[0]
	}
	return record, nil
}

// currentMonthStartDay is the first day of the current month, indexed the same way the credit
// daemon indexes it (time_frame::month_start_day). The two are compared to decide whether a stored
// budget still belongs to this month, so the calendar has to be read in the same business day: on
// a UTC reading they would disagree for the last hours of the first and last day of every month.
func currentMonthStartDay() int16 {
	localNow := core.Now().Add(time.Duration(creditDayZoneOffsetSeconds) * time.Second).UTC()
	monthStart := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.UTC)
	return int16(monthStart.Unix() / secondsPerUTCDate)
}

func remainingCredits(ceiling int64, used uint64) uint64 {
	if ceiling <= 0 || used >= uint64(ceiling) {
		return 0
	}
	return uint64(ceiling) - used
}
