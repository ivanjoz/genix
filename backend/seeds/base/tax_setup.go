package base

import (
	"app/accounting"
	accountingTypes "app/accounting/types"
	"app/config"
	configTypes "app/config/types"
	"app/core"
	"app/db"
	hr "app/hr/types"
	"app/seeds/runner"
	"time"
)

// sunatDueDays2026 is SUNAT's schedule for the monthly obligations of the 2026 tax periods, as
// published on https://www.sunat.gob.pe/orientacion/cronogramas/2026/cObligacionMensual2026.html
// (legal basis: Anexo I of RS 281-2022/SUNAT, which covers 2023 onwards). Transcribed 2026-11-21.
// Every deadline falls in the month after the period, so each row holds days of that month: RUC last
// digit 0–9, then buenos contribuyentes / UESP. The 2027 page was not published yet (404); its rows
// are added in System → Calendario Tributario once it is. Never compute these: decreed
// non-working days move them.
var sunatDueDays2026 = map[int16][accountingTypes.TaxDueColumns]int{
	2601: {16, 17, 18, 18, 19, 19, 20, 20, 23, 23, 24},
	2602: {16, 17, 18, 18, 19, 19, 20, 20, 23, 23, 24},
	2603: {17, 20, 21, 21, 22, 22, 23, 23, 24, 24, 27},
	2604: {18, 19, 20, 20, 21, 21, 22, 22, 25, 25, 26},
	2605: {15, 16, 17, 17, 18, 18, 19, 19, 22, 22, 23},
	2606: {15, 16, 17, 17, 20, 20, 21, 21, 22, 22, 24},
	2607: {18, 19, 20, 20, 21, 21, 24, 24, 25, 25, 26},
	2608: {15, 16, 17, 17, 18, 18, 21, 21, 22, 22, 23},
	2609: {16, 19, 20, 20, 21, 21, 22, 22, 23, 23, 26},
	2610: {16, 17, 18, 18, 19, 19, 20, 20, 23, 23, 24},
	2611: {17, 18, 21, 21, 22, 22, 23, 23, 24, 24, 28},
	2612: {18, 19, 20, 20, 21, 21, 22, 22, 25, 25, 26},
}

// seedTaxDueCalendar stores the transcribed SUNAT schedule through the production handler. Periods
// already stored are left alone: the SaaS admin may have corrected them.
func seedTaxDueCalendar(run *runner.Run) error {
	stored := []accountingTypes.TaxDueCalendar{}
	query := db.Query(&stored)
	query.CountryID.Equals(hr.CountryIDPeru)
	if err := query.Exec(); err != nil {
		return core.Err("No se pudo leer el calendario tributario:", err)
	}
	storedPeriods := core.SliceSet[int16]{}
	for _, calendar := range stored {
		storedPeriods.Add(calendar.Period)
	}

	missing := []accountingTypes.TaxDueCalendar{}
	for period, daysOfMonth := range sunatDueDays2026 {
		if storedPeriods.Include(period) {
			continue
		}
		// The deadlines fall in the month after the period: December's in January of the next year.
		dueYear, dueMonth := 2000+int(period/100), time.Month(period%100)+1
		if dueMonth > time.December {
			dueYear, dueMonth = dueYear+1, time.January
		}
		calendar := accountingTypes.TaxDueCalendar{Period: period, Status: 1}
		for _, dayOfMonth := range daysOfMonth {
			dueDate := time.Date(dueYear, dueMonth, dayOfMonth, 0, 0, 0, 0, time.UTC)
			calendar.DueDays = append(calendar.DueDays, int16(dueDate.Unix()/86400))
		}
		missing = append(missing, calendar)
	}
	if len(missing) > 0 {
		if err := runner.CallAs(run.UserToken, "POST.tax-due-calendar", accounting.PostTaxDueCalendar, nil, missing, nil); err != nil {
			return err
		}
	}
	core.Log("Seed:: calendario tributario", "periodos nuevos", len(missing))
	return nil
}

// seedCompanyTaxSetup gives a company with no tax regime the demo minimarket's: Régimen General
// with the 1.5 % renta prepayment. A regime already set is kept.
func seedCompanyTaxSetup(run *runner.Run) error {
	companies := []configTypes.Company{}
	query := db.Query(&companies)
	query.ID.Equals(run.UserToken.CompanyID)
	if err := query.Exec(); err != nil {
		return core.Err("No se pudo leer la empresa:", err)
	}
	if len(companies) == 0 {
		return core.Err("No existe la empresa", run.UserToken.CompanyID)
	}
	company := companies[0]
	if company.TaxRegime != 0 {
		return nil
	}
	company.TaxRegime, company.IncomeTaxPrepaymentRate = configTypes.TaxRegimeGeneral, 150
	if err := runner.CallAs(run.UserToken, "POST.company-parametros", config.PostEmpresaParametros, nil, company, nil); err != nil {
		return err
	}
	core.Log("Seed:: régimen tributario", "régimen", company.TaxRegime, "pago a cuenta bp", company.IncomeTaxPrepaymentRate)
	return nil
}
