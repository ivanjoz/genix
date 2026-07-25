package types

import "github.com/ivanjoz/genix-orm/scylla"

type CashBank struct {
	scylla.TableStruct[CashBankTable, CashBank]
	CompanyID            int32
	ID                   int32
	Type                 int32
	SiteID               int32
	Name                 string
	Description          string
	CurrencyType         int8
	ReconciliationDate   int32
	ReconciliationAmount int32
	CurrentAmount        int32
	// General properties
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
}

type CashBankTable struct {
	scylla.TableStruct[CashBankTable, CashBank]
	CompanyID            scylla.Col[CashBankTable, int32]
	ID                   scylla.Col[CashBankTable, int32]
	Type                 scylla.Col[CashBankTable, int32]
	SiteID               scylla.Col[CashBankTable, int32]
	Name                 scylla.Col[CashBankTable, string]
	Description          scylla.Col[CashBankTable, string]
	CurrencyType         scylla.Col[CashBankTable, int8]
	ReconciliationDate   scylla.Col[CashBankTable, int32]
	ReconciliationAmount scylla.Col[CashBankTable, int32]
	CurrentAmount        scylla.Col[CashBankTable, int32]
	Status               scylla.Col[CashBankTable, int8]
	Updated              scylla.Col[CashBankTable, int32]
	UpdatedBy            scylla.Col[CashBankTable, int32]
	Created              scylla.Col[CashBankTable, int32]
	CreatedBy            scylla.Col[CashBankTable, int32]
}

func (e CashBankTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "cash_banks",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}

type CashBankMovement struct {
	scylla.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     int32 `json:",omitempty"`
	ID            int64
	CashBankID    int32
	CashBankRefID int32
	DocumentID    int64 `json:",omitempty"` // Sale Order ID / ExpenseID
	ReferenceID   int32 `json:",omitempty"` //
	Date          int16
	Type          int8 `json:",omitempty"`
	FinalAmount   int32
	Amount        int32
	Created       int32 `json:",omitempty"`
	CreatedBy     int32 `json:",omitempty"`
}

type CashBankMovementTable struct {
	scylla.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     scylla.Col[CashBankMovementTable, int32]
	ID            scylla.Col[CashBankMovementTable, int64]
	CashBankID    scylla.Col[CashBankMovementTable, int32]
	CashBankRefID scylla.Col[CashBankMovementTable, int32]
	DocumentID    scylla.Col[CashBankMovementTable, int64]
	ReferenceID   scylla.Col[CashBankMovementTable, int32]
	Date          scylla.Col[CashBankMovementTable, int16]
	Type          scylla.Col[CashBankMovementTable, int8]
	FinalAmount   scylla.Col[CashBankMovementTable, int32]
	Amount        scylla.Col[CashBankMovementTable, int32]
	Created       scylla.Col[CashBankMovementTable, int32]
	CreatedBy     scylla.Col[CashBankMovementTable, int32]
}

func (e CashBankMovementTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "cash_bank_movements",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID),
		KeyIntPacking: scylla.Cols(
			//TODO: decrease to e.Autoincrement(2) in the future
			e.CashBankID.DecimalSize(5), e.Date.DecimalSize(5), e.Autoincrement(3),
		),
		AutoincrementPart: e.Date,
		Indexes: []scylla.Index{
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.DocumentID)},
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.ReferenceID)},
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.CreatedBy)},
		},
	}
}

type CashReconciliation struct {
	scylla.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        int32 `json:",omitempty"`
	ID               int64 `json:",omitempty"`
	CashBankID       int32 `json:",omitempty"`
	Type             int8  `json:",omitempty"`
	MovementID       int64 `json:",omitempty"`
	SystemAmount     int32 `json:",omitempty"`
	ActualAmount     int32 `json:",omitempty"`
	DifferenceAmount int32 `json:",omitempty"`
	Created          int32 `json:",omitempty"`
	CreatedBy        int32 `json:",omitempty"`
}

type CashReconciliationTable struct {
	scylla.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        scylla.Col[CashReconciliationTable, int32]
	ID               scylla.Col[CashReconciliationTable, int64]
	CashBankID       scylla.Col[CashReconciliationTable, int32]
	Type             scylla.Col[CashReconciliationTable, int8]
	MovementID       scylla.Col[CashReconciliationTable, int64]
	SystemAmount     scylla.Col[CashReconciliationTable, int32]
	ActualAmount     scylla.Col[CashReconciliationTable, int32]
	DifferenceAmount scylla.Col[CashReconciliationTable, int32]
	Created          scylla.Col[CashReconciliationTable, int32]
	CreatedBy        scylla.Col[CashReconciliationTable, int32]
}

func (e CashReconciliationTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "cash_reconciliations",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.CreatedBy)},
		},
	}
}

type InternalCashMovement struct {
	CashBankID    int32
	CashBankRefID int32
	DocumentID    int64
	ReferenceID   int32 // Optional: e.g. the originating ExpenseScheduled.ID for expense payments.
	Date          int16 // Optional: movement date; falls back to the request's effective date if 0.
	Type          int8
	Amount        int32
	FinalAmount   int32 // Optional: calculated if 0
}

type SaleProduct struct {
	ProductID int32
	Quantity  int32
	Amount    int32
}
