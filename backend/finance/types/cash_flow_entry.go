package types

import (
	"app/core"
	"app/db"
	"strings"
)

// CashFlowEntry is a planned one-off movement the forecast cannot derive from any record or rule:
// a capital expenditure, a dividend, a capital contribution (plan/CASH_FLOW_FORECAST_PLAN.md §10).
// Edited in the cash flow page's Planned tab. An entry whose date has passed stops counting.
type CashFlowEntry struct {
	db.TableStruct[CashFlowEntryTable, CashFlowEntry]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	Date      int16 `cb:"3"`
	// SiteID is the site the money belongs to, 0 for the company lane.
	SiteID int16 `cb:"4"`
	// Amount is signed cents: + money in, − money out. The sign must match the Kind.
	Amount       int32  `cb:"5"`
	CurrencyType int8   `cb:"6"`
	Kind         int8   `cb:"7"`
	Name         string `cb:"8"`
	Status       int8   `json:"ss" cb:"ss,9"`
	Updated      int32  `json:"upd" cb:"upd,10"`
	Version      int16  `json:"ver" cb:"ver,11"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,12"`
	Created   int32   `cb:"13"`
	CreatedBy int32   `cb:"14"`
	UpdatedBy int32   `cb:"15"`
}

type CashFlowEntryTable struct {
	db.TableStruct[CashFlowEntryTable, CashFlowEntry]
	CompanyID    db.Col[*CashFlowEntryTable, int32]
	ID           db.Col[*CashFlowEntryTable, int32]
	Date         db.Col[*CashFlowEntryTable, int16]
	SiteID       db.Col[*CashFlowEntryTable, int16]
	Amount       db.Col[*CashFlowEntryTable, int32]
	CurrencyType db.Col[*CashFlowEntryTable, int8]
	Kind         db.Col[*CashFlowEntryTable, int8]
	Name         db.Col[*CashFlowEntryTable, string]
	Status       db.Col[*CashFlowEntryTable, int8]
	Updated      db.Col[*CashFlowEntryTable, int32]
	Version      db.Col[*CashFlowEntryTable, int16]
	Created      db.Col[*CashFlowEntryTable, int32]
	CreatedBy    db.Col[*CashFlowEntryTable, int32]
	UpdatedBy    db.Col[*CashFlowEntryTable, int32]
}

func (e CashFlowEntryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        131,
		Name:      "cash_flow_entry",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// Kinds of planned entry, and whether each brings money in.
const (
	CashFlowEntryCapex        int8 = 1
	CashFlowEntryDividend     int8 = 2
	CashFlowEntryCapital      int8 = 3
	CashFlowEntryOtherIncome  int8 = 4
	CashFlowEntryOtherOutflow int8 = 5
)

var cashFlowEntryIsInflow = map[int8]bool{
	CashFlowEntryCapex: false, CashFlowEntryDividend: false, CashFlowEntryCapital: true,
	CashFlowEntryOtherIncome: true, CashFlowEntryOtherOutflow: false,
}

// ValidateCashFlowEntry checks an entry about to be saved active, trimming its name in place.
func ValidateCashFlowEntry(entry *CashFlowEntry) error {
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		return core.Err("La entrada planificada debe tener un nombre.")
	}
	isInflow, isKind := cashFlowEntryIsInflow[entry.Kind]
	if !isKind {
		return core.Err("El tipo de la entrada planificada no es válido:", entry.Kind)
	}
	if entry.Date <= 0 {
		return core.Err("Indique la fecha de la entrada planificada.")
	}
	if entry.CurrencyType != CurrencyPEN && entry.CurrencyType != CurrencyUSD {
		return core.Err("La moneda de la entrada planificada no es válida.")
	}
	if entry.Amount == 0 || (entry.Amount > 0) != isInflow {
		return core.Err("El monto debe ser positivo para un ingreso y negativo para una salida.")
	}
	return nil
}
