package types

import "app/db"

type CashBank struct {
	db.TableStruct[CashBankTable, CashBank]
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
	db.TableStruct[CashBankTable, CashBank]
	CompanyID            db.Col[CashBankTable, int32]
	ID                   db.Col[CashBankTable, int32]
	Type                 db.Col[CashBankTable, int32]
	SiteID               db.Col[CashBankTable, int32]
	Name                 db.Col[CashBankTable, string]
	Description          db.Col[CashBankTable, string]
	CurrencyType         db.Col[CashBankTable, int8]
	ReconciliationDate   db.Col[CashBankTable, int32]
	ReconciliationAmount db.Col[CashBankTable, int32]
	CurrentAmount        db.Col[CashBankTable, int32]
	Status               db.Col[CashBankTable, int8]
	Updated              db.Col[CashBankTable, int32]
	UpdatedBy            db.Col[CashBankTable, int32]
	Created              db.Col[CashBankTable, int32]
	CreatedBy            db.Col[CashBankTable, int32]
}

func (e CashBankTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "cash_banks",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type CashBankMovement struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
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
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     db.Col[CashBankMovementTable, int32]
	ID            db.Col[CashBankMovementTable, int64]
	CashBankID    db.Col[CashBankMovementTable, int32]
	CashBankRefID db.Col[CashBankMovementTable, int32]
	DocumentID    db.Col[CashBankMovementTable, int64]
	ReferenceID   db.Col[CashBankMovementTable, int32]
	Date          db.Col[CashBankMovementTable, int16]
	Type          db.Col[CashBankMovementTable, int8]
	FinalAmount   db.Col[CashBankMovementTable, int32]
	Amount        db.Col[CashBankMovementTable, int32]
	Created       db.Col[CashBankMovementTable, int32]
	CreatedBy     db.Col[CashBankMovementTable, int32]
}

func (e CashBankMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "cash_bank_movements",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			//TODO: decrease to e.Autoincrement(2) in the future
			e.CashBankID.DecimalSize(5), e.Date.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Date,
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.DocumentID}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.ReferenceID}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.CreatedBy}},
		},
	}
}

type CashReconciliation struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
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
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        db.Col[CashReconciliationTable, int32]
	ID               db.Col[CashReconciliationTable, int64]
	CashBankID       db.Col[CashReconciliationTable, int32]
	Type             db.Col[CashReconciliationTable, int8]
	MovementID       db.Col[CashReconciliationTable, int64]
	SystemAmount     db.Col[CashReconciliationTable, int32]
	ActualAmount     db.Col[CashReconciliationTable, int32]
	DifferenceAmount db.Col[CashReconciliationTable, int32]
	Created          db.Col[CashReconciliationTable, int32]
	CreatedBy        db.Col[CashReconciliationTable, int32]
}

func (e CashReconciliationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "cash_reconciliations",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.CreatedBy}, KeepPart: true},
		},
	}
}

type InternalCashMovement struct {
	CashBankID    int32
	CashBankRefID int32
	DocumentID    int64
	Type          int8
	Amount        int32
	FinalAmount   int32 // Optional: calculated if 0
}

type SaleProduct struct {
	ProductID int32 `cbor:"1,keyasint"`
	Quantity  int32 `cbor:"2,keyasint"`
	Amount    int32 `cbor:"3,keyasint"`
}
