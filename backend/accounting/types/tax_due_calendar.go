package types

import (
	"app/core"
	"app/db"
)

// TaxDueCalendar is SUNAT's due-date schedule for the monthly obligations (IGV, renta pagos a
// cuenta, PLAME, NRUS cuota) of one tax period. RS 281-2022/SUNAT sets every deadline as a business
// day of the following month by the RUC's last digit; SUNAT turns that into calendar dates once a
// year (sunat.gob.pe/orientacion/cronogramas/<year>/cObligacionMensual<year>.html). The dates are
// transcribed from that page, never computed: decreed non-working days move them.
//
// Shared by every company of a country, so partitioned by CountryID like the HR legal tables, and
// written only by the SaaS company (System → Calendario Tributario).
type TaxDueCalendar struct {
	db.TableStruct[TaxDueCalendarTable, TaxDueCalendar]
	CountryID int32 `cb:"1"`
	// Period is the YYMM of the tax period being declared (2601 = January 2026), not of the month
	// it falls due in.
	Period int16 `cb:"2"`
	// DueDays are TaxDueColumns UnixDays: one per RUC last digit 0–9, then the buenos
	// contribuyentes / UESP column.
	DueDays   []int16 `cb:"3"`
	Status    int8    `json:"ss" cb:"ss,4"`
	Updated   int32   `json:"upd" cb:"upd,5"`
	Version   int16   `json:"ver" cb:"ver,6"`
	Metadata  *string `json:"mdt" cb:"mdt,7"`
	UpdatedBy int32   `cb:"8"`
}

type TaxDueCalendarTable struct {
	db.TableStruct[TaxDueCalendarTable, TaxDueCalendar]
	CountryID db.Col[*TaxDueCalendarTable, int32]
	Period    db.Col[*TaxDueCalendarTable, int16]
	DueDays   db.ColSlice[*TaxDueCalendarTable, int16]
	Status    db.Col[*TaxDueCalendarTable, int8]
	Updated   db.Col[*TaxDueCalendarTable, int32]
	Version   db.Col[*TaxDueCalendarTable, int16]
	UpdatedBy db.Col[*TaxDueCalendarTable, int32]
}

func (e TaxDueCalendarTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        129,
		Name:      "tax_due_calendar",
		Partition: e.CountryID,
		// One schedule per period: the period is the key, so saving a period again overwrites it.
		Keys:    db.Cols(e.Period),
		Indexes: []db.Index{{Type: db.TypeDelta}},
	}
}

// TaxDueColumns is the number of DueDays: RUC last digits 0–9 plus the buenos contribuyentes column.
const TaxDueColumns = 11

// taxDueGoodTaxpayerColumn is the DueDays index of the buenos contribuyentes / UESP column.
const taxDueGoodTaxpayerColumn = 10

// DueDayFor is the deadline of a taxpayer: a buen contribuyente uses its own column, everyone else
// the column of the RUC's last digit. 0 when the RUC does not end in a digit or the row is short.
func (e TaxDueCalendar) DueDayFor(ruc string, isGoodTaxpayer bool) int16 {
	column := taxDueGoodTaxpayerColumn
	if !isGoodTaxpayer {
		if ruc == "" || ruc[len(ruc)-1] < '0' || ruc[len(ruc)-1] > '9' {
			return 0
		}
		column = int(ruc[len(ruc)-1] - '0')
	}
	if column >= len(e.DueDays) {
		return 0
	}
	return e.DueDays[column]
}

// ValidateTaxDueCalendar checks a schedule about to be saved active: a real period and every
// deadline after the period's last day, since a tax is declared once its month has closed.
func ValidateTaxDueCalendar(calendar TaxDueCalendar) error {
	month := calendar.Period % 100
	if calendar.Period < 2001 || month < 1 || month > 12 {
		return core.Err("El periodo", calendar.Period, "no es válido (formato AAMM).")
	}
	if len(calendar.DueDays) != TaxDueColumns {
		return core.Err("El periodo", calendar.Period, "debe tener", TaxDueColumns,
			"vencimientos (dígitos 0 al 9 y buenos contribuyentes).")
	}
	nextPeriod := calendar.Period + 1
	if month == 12 {
		nextPeriod = calendar.Period + 100 - 11
	}
	firstDayAfterPeriod := core.PeriodFirstDay(nextPeriod)
	for column, dueDay := range calendar.DueDays {
		if dueDay < firstDayAfterPeriod {
			return core.Err("El vencimiento", column+1, "del periodo", calendar.Period,
				"debe caer después del cierre del periodo.")
		}
	}
	return nil
}
