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
	"time"
)

type companyCreditBudgetResponse struct {
	coreTypes.CompanyCreditBudget
	MonthCPUUsed       uint64
	MonthInferenceUsed uint64
	CurrentCPU         uint64
	CurrentInference   uint64
	IsCurrentMonth     bool
}

type companyCreditBudgetMutationRequest struct {
	CompanyID int32
	Operation string
	CPU       int64
	Inference int64
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
	if mutation.CPU < 0 || mutation.Inference < 0 {
		return req.MakeErr("Los créditos CPU e IA no pueden ser negativos.")
	}
	if _, err := requireCompany(mutation.CompanyID); err != nil {
		return req.MakeErr("No se pudo validar la empresa solicitada.", err)
	}

	operation, err := parseCompanyCreditBudgetOperation(mutation.Operation)
	if err != nil {
		return req.MakeErr("La operación del presupuesto de créditos no es válida.", err)
	}
	requestContext := context.Background()
	if req.ReqContext != nil {
		requestContext = req.ReqContext.Context()
	}
	core.Log("company credit budget mutation started::", " company::", mutation.CompanyID,
		" operation::", mutation.Operation, " cpu::", mutation.CPU, " inference::", mutation.Inference)
	err = core.MutateCompanyCreditBudget(
		requestContext,
		mutation.CompanyID,
		operation,
		uint64(mutation.CPU),
		uint64(mutation.Inference),
	)
	if errors.Is(err, core.ErrBudgetMonthNotConfigured) {
		return req.MakeErr("Primero debe establecer el presupuesto actual del mes.")
	}
	if errors.Is(err, core.ErrBudgetMutationOverflow) {
		return req.MakeErr("El presupuesto de créditos excede el máximo permitido.")
	}
	if err != nil {
		core.Log("company credit budget mutation unavailable::", " company::", mutation.CompanyID,
			" operation::", mutation.Operation, " err::", err)
		return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
	}

	budget, err := getCompanyCreditBudget(mutation.CompanyID)
	if err != nil {
		core.Log("company credit budget refresh failed::", " company::", mutation.CompanyID, " err::", err)
		return req.MakeErr("El presupuesto cambió, pero no se pudo volver a leer.", err)
	}
	core.Log("company credit budget mutation completed::", " company::", mutation.CompanyID,
		" operation::", mutation.Operation, " current_cpu::", budget.CurrentCPU,
		" current_inference::", budget.CurrentInference)
	return req.MakeResponse(budget)
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

	currentMonthStartDay := currentUTCMonthStartDay()
	response := companyCreditBudgetResponse{
		CompanyCreditBudget: record,
		IsCurrentMonth:      record.BudgetMonthStartDay == currentMonthStartDay,
	}
	if !response.IsCurrentMonth {
		return response, nil
	}
	rows := []coreTypes.CreditUsage{}
	usageQuery := db.Query(&rows)
	usageQuery.CompanyID.Equals(companyID).
		UserID.Equals(companyAggregateID).
		TimeFrame.Between(
		dailyTimeFramePrefix+int32(currentMonthStartDay),
		dailyTimeFramePrefix+int32(core.FechaUnix()),
	)
	if err := usageQuery.Exec(); err != nil {
		return companyCreditBudgetResponse{}, err
	}
	for _, row := range rows {
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			return companyCreditBudgetResponse{}, fmt.Errorf("invalid usage in time frame %d: %w", row.TimeFrame, err)
		}
		response.MonthCPUUsed += totals.CPU
		response.MonthInferenceUsed += totals.Inference
	}
	response.CurrentCPU = remainingCredits(record.MonthlyCPUCeiling, response.MonthCPUUsed)
	response.CurrentInference = remainingCredits(record.MonthlyInferenceCeiling, response.MonthInferenceUsed)
	return response, nil
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

func currentUTCMonthStartDay() int16 {
	now := core.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return int16(monthStart.Unix() / secondsPerUTCDate)
}

func remainingCredits(ceiling int64, used uint64) uint64 {
	if ceiling <= 0 || used >= uint64(ceiling) {
		return 0
	}
	return uint64(ceiling) - used
}
