package config

import (
	"app/core"
	coreTypes "app/core/types"
	"bytes"
	"testing"

	"github.com/ivanjoz/minijson"
)

// minijson no aplana campos anónimos: si la respuesta vuelve a embeber CompanyCreditBudget, el
// límite diario llega anidado y el panel lo lee como 0. Esta prueba fija el contrato de la forma
// más directa posible: los nombres que el cliente busca tienen que estar en la cabecera de claves.
func TestCompanyCreditBudgetResponseKeepsBudgetFieldsAtTopLevel(t *testing.T) {
	// Todos los campos van con valor: minijson omite los ceros, así que un cero borraría su clave
	// de la cabecera y la prueba no distinguiría "aplanado" de "omitido".
	response := companyCreditBudgetResponse{
		CompanyID: 1, DailyCPU: 1000, DailyInference: 1000, BudgetMonthStartDay: 20666,
		MonthlyCPUCeiling: 2233, MonthlyInferenceCeiling: 1000, Updated: 393538913,
		MonthCPUUsed: 1233, MonthInferenceUsed: 1, CurrentCPU: 1000, CurrentInference: 1000,
		IsCurrentMonth: true,
	}

	encoded, err := minijson.Marshal(&response)
	if err != nil {
		t.Fatalf("minijson.Marshal() error = %v", err)
	}
	for _, field := range []string{
		"CompanyID", "DailyCPU", "DailyInference", "BudgetMonthStartDay", "MonthlyCPUCeiling",
		"MonthlyInferenceCeiling", "Updated", "MonthCPUUsed", "MonthInferenceUsed", "CurrentCPU",
		"CurrentInference", "IsCurrentMonth",
	} {
		if !bytes.Contains(encoded, []byte(`"`+field+`"`)) {
			t.Fatalf("minijson.Marshal() = %s, want top-level key %q", encoded, field)
		}
	}
	if bytes.Contains(encoded, []byte(`"CompanyCreditBudget"`)) {
		t.Fatalf("minijson.Marshal() = %s, want no nested CompanyCreditBudget object", encoded)
	}
}

// El orden de aplicación es la razón de ser del parseo: "aumentar" antes de "reemplazar" es un
// rechazo del daemon cuando el mes todavía no tiene presupuesto, así que un guardado que manda las
// dos filas sólo funciona si el backend las reordena.
func TestParseCompanyCreditBudgetOperationsAppliesDailyThenCurrentThenIncrease(t *testing.T) {
	operations, err := parseCompanyCreditBudgetOperations([]companyCreditBudgetOperationRequest{
		{Operation: "increase-current", CPU: 50, Inference: 5},
		{Operation: "set-daily", CPU: 1000, Inference: 1000},
		{Operation: "set-current", CPU: 2000, Inference: 2000},
	})
	if err != nil {
		t.Fatalf("parseCompanyCreditBudgetOperations() error = %v", err)
	}
	wantCodes := []core.BudgetOperation{core.BudgetSetDaily, core.BudgetSetCurrent, core.BudgetIncreaseCurrent}
	for index, wantCode := range wantCodes {
		if operations[index].code != wantCode {
			t.Fatalf("operation %d = %d, want %d", index, operations[index].code, wantCode)
		}
	}
}

func TestParseCompanyCreditBudgetOperationsRejectsInvalidPayloads(t *testing.T) {
	cases := map[string][]companyCreditBudgetOperationRequest{
		"empty list":       {},
		"unknown code":     {{Operation: "set-whatever"}},
		"negative credits": {{Operation: "set-daily", CPU: -1}},
		"repeated code": {
			{Operation: "set-daily", CPU: 10},
			{Operation: "set-daily", CPU: 20},
		},
	}
	for name, requested := range cases {
		if _, err := parseCompanyCreditBudgetOperations(requested); err == nil {
			t.Fatalf("parseCompanyCreditBudgetOperations(%s) error = nil, want an error", name)
		}
	}
}

// El ledger existe para responder "cuánto se otorgó", así que el signo y el caso del mes nuevo son
// su contrato: una resta contra el techo del mes anterior daría un número inventado.
func TestCreditHistoryMovementReadsGrantsAndRevocations(t *testing.T) {
	previous := coreTypes.CompanyCreditBudget{
		BudgetMonthStartDay: 20666, MonthlyCPUCeiling: 2233, MonthlyInferenceCeiling: 1000,
	}
	cases := map[string]struct {
		current                coreTypes.CompanyCreditBudget
		wantCPU, wantInference int64
	}{
		"credit added": {
			current: coreTypes.CompanyCreditBudget{
				BudgetMonthStartDay: 20666, MonthlyCPUCeiling: 2445, MonthlyInferenceCeiling: 1000,
			},
			wantCPU: 212, wantInference: 0,
		},
		"credit taken away": {
			current: coreTypes.CompanyCreditBudget{
				BudgetMonthStartDay: 20666, MonthlyCPUCeiling: 2000, MonthlyInferenceCeiling: 900,
			},
			wantCPU: -233, wantInference: -100,
		},
		"daily limit only": {
			current: previous,
			wantCPU: 0, wantInference: 0,
		},
		"first budget of a new month": {
			current: coreTypes.CompanyCreditBudget{
				BudgetMonthStartDay: 20697, MonthlyCPUCeiling: 1500, MonthlyInferenceCeiling: 800,
				LastSetCPU: 1000, LastSetInference: 800,
			},
			wantCPU: 1000, wantInference: 800,
		},
	}
	for name, testCase := range cases {
		gotCPU, gotInference := creditHistoryMovement(previous, testCase.current)
		if gotCPU != testCase.wantCPU || gotInference != testCase.wantInference {
			t.Fatalf("creditHistoryMovement(%s) = (%d, %d), want (%d, %d)",
				name, gotCPU, gotInference, testCase.wantCPU, testCase.wantInference)
		}
	}
}

// El panel resta los contadores que el daemon publica en cada flush, y esas columnas nombran la
// ventana a la que pertenecen: nadie reescribe la fila cuando cambia el día o el mes sin tráfico, así
// que un período que ya pasó tiene que leerse como "esa ventana no gastó nada" y no como gasto vivo.
// Al revés sería peor que un número feo: mostraría crédito consumido que el limiter ya devolvió.
func TestMakeCompanyCreditBudgetResponseSubtractsOnlyTheCurrentWindows(t *testing.T) {
	defer core.SetHistoricalUnix(0)
	// 2026-08-16 12:00 Lima: día de negocio 20681, mes que empieza el 20666.
	const businessDay = int16(20_681)
	const monthStartDay = int16(20_666)
	core.SetHistoricalUnix(int64(businessDay)*int64(secondsPerUTCDate) - creditDayZoneOffsetSeconds + 12*3_600)

	baseRecord := coreTypes.CompanyCreditBudget{
		CompanyID: 7, DailyCPU: 1_000, DailyInference: 200,
		BudgetMonthStartDay: monthStartDay, MonthlyCPUCeiling: 40_000, MonthlyInferenceCeiling: 5_000,
		UsageUpdated: 393_538_913,
	}
	withUsage := func(dayPeriod, usageMonthStartDay int16) coreTypes.CompanyCreditBudget {
		record := baseRecord
		record.UsageDayPeriod = dayPeriod
		record.DayCPUUsed, record.DayInferenceUsed = 300, 50
		record.UsageMonthStartDay = usageMonthStartDay
		record.MonthCPUUsed, record.MonthInferenceUsed = 12_000, 900
		return record
	}

	for name, testCase := range map[string]struct {
		record                                   coreTypes.CompanyCreditBudget
		wantDailyCPU, wantDailyInference         uint64
		wantRemainingCPU, wantRemainingInference uint64
	}{
		"both windows current": {
			record:       withUsage(businessDay, monthStartDay),
			wantDailyCPU: 700, wantDailyInference: 150,
			wantRemainingCPU: 28_000, wantRemainingInference: 4_100,
		},
		"the day rolled over without traffic": {
			record:       withUsage(businessDay-1, monthStartDay),
			wantDailyCPU: 1_000, wantDailyInference: 200,
			wantRemainingCPU: 28_000, wantRemainingInference: 4_100,
		},
		"the counters belong to the previous month": {
			record:       withUsage(businessDay, monthStartDay-31),
			wantDailyCPU: 700, wantDailyInference: 150,
			wantRemainingCPU: 40_000, wantRemainingInference: 5_000,
		},
		"the day spent its whole allowance": {
			record: func() coreTypes.CompanyCreditBudget {
				record := withUsage(businessDay, monthStartDay)
				record.DayCPUUsed = 1_500
				return record
			}(),
			wantDailyCPU: 0, wantDailyInference: 150,
			wantRemainingCPU: 28_000, wantRemainingInference: 4_100,
		},
	} {
		t.Run(name, func(t *testing.T) {
			response := makeCompanyCreditBudgetResponse(testCase.record, 0)
			if !response.IsCurrentMonth {
				t.Fatalf("IsCurrentMonth = false, want true")
			}
			if response.DailyRemainingCPU != testCase.wantDailyCPU ||
				response.DailyRemainingInference != testCase.wantDailyInference {
				t.Fatalf("daily remaining = (%d, %d), want (%d, %d)",
					response.DailyRemainingCPU, response.DailyRemainingInference,
					testCase.wantDailyCPU, testCase.wantDailyInference)
			}
			if response.CurrentCPU != testCase.wantRemainingCPU ||
				response.CurrentInference != testCase.wantRemainingInference {
				t.Fatalf("monthly remaining = (%d, %d), want (%d, %d)",
					response.CurrentCPU, response.CurrentInference,
					testCase.wantRemainingCPU, testCase.wantRemainingInference)
			}
		})
	}
}

// Un presupuesto de otro mes deja a la empresa bloqueada: el daemon rechaza todo cargo, así que no
// hay crédito que anunciar ni gasto vigente que restar, aunque la fila traiga contadores.
func TestMakeCompanyCreditBudgetResponseReportsNoBudgetOutsideTheCurrentMonth(t *testing.T) {
	defer core.SetHistoricalUnix(0)
	core.SetHistoricalUnix(int64(20_681)*int64(secondsPerUTCDate) - creditDayZoneOffsetSeconds + 12*3_600)

	response := makeCompanyCreditBudgetResponse(coreTypes.CompanyCreditBudget{
		CompanyID: 7, DailyCPU: 1_000, BudgetMonthStartDay: 20_635, MonthlyCPUCeiling: 40_000,
		UsageDayPeriod: 20_681, DayCPUUsed: 300, UsageMonthStartDay: 20_635, MonthCPUUsed: 12_000,
		UsageUpdated: 393_538_913, DayExtraCPUUsed: 400,
	}, 50_000)
	if response.IsCurrentMonth {
		t.Fatalf("IsCurrentMonth = true, want false")
	}
	if response.CurrentCPU != 0 || response.DailyRemainingCPU != 0 || response.MonthCPUUsed != 0 {
		t.Fatalf("response = %+v, want every credit figure at zero", response)
	}
	// El pool extra es la excepción, y es a propósito: una company sin presupuesto del mes en curso
	// es precisamente la que vive de él, así que esconderlo dejaría invisible el modo de sólo lectura
	// justo cuando está en uso.
	if response.ExtraCPU != 50_000 || response.DayExtraCPUUsed != 400 ||
		response.ExtraRemainingCPU != 49_600 {
		t.Fatalf("el pool extra no sobrevivió a un mes sin presupuesto: %+v", response)
	}
}
