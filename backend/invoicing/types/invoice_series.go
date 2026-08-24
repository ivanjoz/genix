package types

import (
	"app/db"
	"strconv"
)

// InvoiceSeries is a series a company is allowed to issue under.
//
// The counter that hands out numbers lives in the ORM, not here — this table
// answers the questions the counter cannot: which series exist, what each one is
// called, which document type it serves and which establishment it belongs to.
//
// SeriesID is a small local number, and SeriesCode is what SUNAT sees. They are
// separate because a series code is alphanumeric — F001, B001, FC01 — and cannot
// be packed into a numeric key, while the key needs to be numeric to partition
// the counter.
type InvoiceSeries struct {
	db.TableStruct[InvoiceSeriesTable, InvoiceSeries]
	CompanyID int32 `json:",omitempty"`
	ID        int32 `json:",omitempty"`

	DocType    int8   `json:",omitempty"`
	SeriesID   int16  `json:",omitempty"`
	SeriesCode string `json:",omitempty"`

	// Where documents of this series are issued from. The establishment code
	// SUNAT assigned to the site travels in the XML, so a company with several
	// branches keeps one series per branch.
	SiteID      int32 `json:",omitempty"`
	WarehouseID int32 `json:",omitempty"`

	// IsDefault marks the series a sale falls back to when the caller does not
	// name one. At most one per document type, which the handler enforces.
	IsDefault int8 `json:",omitempty"`

	Status         int8  `json:"ss,omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	CreatedBy      int32 `json:",omitempty"`
}

// SelfParse keeps the key and the identity it encodes in agreement.
func (e *InvoiceSeries) SelfParse() {
	if e.ID == 0 {
		e.ID = PackDocTypeSeries(e.DocType, e.SeriesID)
	}
}

type InvoiceSeriesTable struct {
	db.TableStruct[InvoiceSeriesTable, InvoiceSeries]
	CompanyID      db.Col[*InvoiceSeriesTable, int32]
	ID             db.Col[*InvoiceSeriesTable, int32]
	DocType        db.Col[*InvoiceSeriesTable, int8]
	SeriesID       db.Col[*InvoiceSeriesTable, int16]
	SeriesCode     db.Col[*InvoiceSeriesTable, string]
	SiteID         db.Col[*InvoiceSeriesTable, int32]
	WarehouseID    db.Col[*InvoiceSeriesTable, int32]
	IsDefault      db.Col[*InvoiceSeriesTable, int8]
	Status         db.Col[*InvoiceSeriesTable, int8]
	Updated        db.Col[*InvoiceSeriesTable, int32]
	UpdatedVersion db.Col[*InvoiceSeriesTable, int32]
	UpdatedBy      db.Col[*InvoiceSeriesTable, int32]
	Created        db.Col[*InvoiceSeriesTable, int32]
	CreatedBy      db.Col[*InvoiceSeriesTable, int32]
}

func (e InvoiceSeriesTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 53,
		Name:               "invoice_series",
		Partition:          e.CompanyID,
		Keys:               db.Cols(e.ID),
		SaveUpdatedVersion: true,
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// itoa is the one number-to-string this package needs, kept here so the
// document type does not import strconv for a single call.
func itoa(value int64) string { return strconv.FormatInt(value, 10) }
