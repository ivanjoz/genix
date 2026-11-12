package base

import (
	"app/core"
	"app/db"
	"app/hr"
	"app/hr/types"
	"app/seeds/runner"
	"time"
)

// seedHRLegalParameters loads the Peruvian legal payroll values of 2025–2026 and the 2026 AFP rates
// through the production handlers. It can be run again: values already stored (same key and date)
// are skipped, AFP rates are overwritten.
//
// Sources: plan/HR_PAYROLL_PLAN.md §1.3 and the SBS commission table. The SaaS admin corrects them
// in System → Parámetros Laborales; the ones marked "verify" were not confirmed against the source.
func seedHRLegalParameters(run *runner.Run) error {
	admin := run.UserToken
	day := func(year int, month time.Month, dayOfMonth int) int16 {
		return int16(time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	}
	parameters := []types.LegalParameter{
		{Key: types.LegalParamRMV, ValidFrom: day(2025, 1, 1), Value: 113000},
		{Key: types.LegalParamRMV, ValidFrom: day(2026, 10, 1), Value: 123000},
		{Key: types.LegalParamUIT, ValidFrom: day(2025, 1, 1), Value: 535000},
		{Key: types.LegalParamUIT, ValidFrom: day(2026, 1, 1), Value: 550000},
		{Key: types.LegalParamEssaludRate, ValidFrom: day(2025, 1, 1), Value: 900},
		{Key: types.LegalParamEssaludEpsRate, ValidFrom: day(2025, 1, 1), Value: 675},
		{Key: types.LegalParamOnpRate, ValidFrom: day(2025, 1, 1), Value: 1300},
		{Key: types.LegalParamFamilyAllowanceRate, ValidFrom: day(2025, 1, 1), Value: 1000},
		{Key: types.LegalParamGratiBonusRate, ValidFrom: day(2025, 1, 1), Value: 900},
		{Key: types.LegalParamGratiBonusEpsRate, ValidFrom: day(2025, 1, 1), Value: 675},
		// Remuneración máxima asegurable, quarterly. Q3 2026 also stands in for Q1–Q2 (verify).
		{Key: types.LegalParamMaxInsurableSalary, ValidFrom: day(2026, 1, 1), Value: 1267265},
		{Key: types.LegalParamMaxInsurableSalary, ValidFrom: day(2026, 10, 1), Value: 1273270},
	}

	stored := []types.LegalParameter{}
	query := db.Query(&stored)
	query.CountryID.Equals(types.CountryIDPeru)
	if err := query.Exec(); err != nil {
		return core.Err("No se pudieron leer los parámetros laborales:", err)
	}
	created := 0
	for _, parameter := range parameters {
		alreadyStored := false
		for _, existing := range stored {
			alreadyStored = alreadyStored || (existing.Status == types.StatusActive && existing.Key == parameter.Key &&
				existing.ValidFrom == parameter.ValidFrom)
		}
		if alreadyStored {
			continue
		}
		if err := runner.CallAs(admin, "POST.hr-legal-parameters", hr.PostLegalParameter, nil, parameter, nil); err != nil {
			return err
		}
		created++
	}

	// SBS rates of October 2026, applied to every month of 2026 (verify the earlier months): fund 10 %,
	// premium 1.37 %, commission on flow, and on balance for the mixed scheme.
	afpCommissions := map[int8][2]int32{
		types.PensionSystemHabitat:   {147, 125},
		types.PensionSystemIntegra:   {155, 78},
		types.PensionSystemPrima:     {160, 125},
		types.PensionSystemProfuturo: {169, 68},
	}
	afpRates := []types.AfpRate{}
	for period := int16(2601); period <= 2612; period++ {
		for pensionSystem, commissions := range afpCommissions {
			afpRates = append(afpRates, types.AfpRate{Period: period, PensionSystem: pensionSystem, FundRate: 1000,
				// Status 0 would be read as a deletion.
				PremiumRate: 137, FlowCommission: commissions[0], MixedCommission: commissions[1], Status: types.StatusActive})
		}
	}
	if err := runner.CallAs(admin, "POST.hr-afp-rates", hr.PostAfpRates, nil, afpRates, nil); err != nil {
		return err
	}
	core.Log("Seed:: parámetros laborales", "nuevos", created, "tasas AFP", len(afpRates))
	return nil
}
