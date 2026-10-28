package types

import "app/db"

type CashBank struct {
	db.TableStruct[CashBankTable, CashBank]
	CompanyID            int32  `cb:"CompanyID,1"`
	ID                   int32  `cb:"ID,2"`
	Type                 int32  `cb:"Type,3"`
	SiteID               int32  `cb:"SiteID,4"`
	Name                 string `cb:"Name,5"`
	Description          string `cb:"Description,6"`
	CurrencyType         int8   `cb:"CurrencyType,7"`
	ReconciliationDate   int32  `cb:"ReconciliationDate,8"`
	ReconciliationAmount int32  `cb:"ReconciliationAmount,9"`
	CurrentAmount        int32  `cb:"CurrentAmount,10"`
	// General properties
	Status  int8  `json:"ss" cb:"ss,11"`
	Updated int32 `json:"upd" cb:"upd,12"`
	Version int16 `json:"ver" cb:"ver,13"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,14"`
	UpdatedBy int32   `cb:"UpdatedBy,15"`
	Created   int32   `cb:"Created,16"`
	CreatedBy int32   `cb:"CreatedBy,17"`
}

type CashBankTable struct {
	db.TableStruct[CashBankTable, CashBank]
	CompanyID            db.Col[*CashBankTable, int32]
	ID                   db.Col[*CashBankTable, int32]
	Type                 db.Col[*CashBankTable, int32]
	SiteID               db.Col[*CashBankTable, int32]
	Name                 db.Col[*CashBankTable, string]
	Description          db.Col[*CashBankTable, string]
	CurrencyType         db.Col[*CashBankTable, int8]
	ReconciliationDate   db.Col[*CashBankTable, int32]
	ReconciliationAmount db.Col[*CashBankTable, int32]
	CurrentAmount        db.Col[*CashBankTable, int32]
	Status               db.Col[*CashBankTable, int8]
	Updated              db.Col[*CashBankTable, int32]
	Version              db.Col[*CashBankTable, int16]
	UpdatedBy            db.Col[*CashBankTable, int32]
	Created              db.Col[*CashBankTable, int32]
	CreatedBy            db.Col[*CashBankTable, int32]
}

func (e CashBankTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           6,
		Name:         "cash_banks",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// CashMovementType is the CashBankMovement.Type vocabulary — one owned enum instead of the
// private copy each module used to keep. It is the discriminator that separates movements
// sharing a DocumentID: an asset payment and an expense payment can both carry the same
// document id, so the type is the only thing that tells them apart when summing what was paid.
//
// The ids are fixed by the frontend's cajaMovimientoTipos list
// (frontend/routes/finance/cash-banks/cajas.svelte.ts) and are persisted, so they must not be
// renumbered. Every value is listed even where the backend never writes it, so a new module
// cannot silently reuse a taken number.
type CashMovementType int8

const (
	CashMovementTypeUnspecified     CashMovementType = 1
	CashMovementTypePhysicalCount   CashMovementType = 2
	CashMovementTypeTransfer        CashMovementType = 3
	CashMovementTypeWithdrawal      CashMovementType = 4
	CashMovementTypeLoss            CashMovementType = 5
	CashMovementTypeSupplierPayment CashMovementType = 6
	CashMovementTypeCollection      CashMovementType = 7
	CashMovementTypeSaleCollection  CashMovementType = 8
	CashMovementTypeExpensePayment  CashMovementType = 9
	CashMovementTypeAssetPayment    CashMovementType = 10
	// CashMovementTypeSaleRefund gives back what a sale collected, when that sale is annulled.
	// It is its own type rather than a negative SaleCollection because the annulment nets the
	// two to decide what is still owed back, and a retry that could not tell them apart would
	// refund a second time.
	CashMovementTypeSaleRefund CashMovementType = 11
)

type CashBankMovement struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     int32            `cb:"CompanyID,1"`
	ID            int64            `cb:"ID,2"`
	CashBankID    int32            `cb:"CashBankID,3"`
	CashBankRefID int32            `cb:"CashBankRefID,4"`
	DocumentID    int64            `cb:"DocumentID,5"`  // Sale Order ID / ExpenseID
	ReferenceID   int32            `cb:"ReferenceID,6"` //
	Date          int16            `cb:"Date,7"`
	Type          CashMovementType `cb:"Type,8"`
	// AccountCode is the PCGE account on the other side of the cash, for the types the user
	// classifies (see CashMovementAccountOptions). 0 for every other type: its counter-account
	// follows from the type and the document it pays.
	AccountCode int16 `cb:"AccountCode,9"`
	FinalAmount int32 `cb:"FinalAmount,10"`
	Amount      int32 `cb:"Amount,11"`
	Created     int32 `cb:"Created,12"`
	CreatedBy   int32 `cb:"CreatedBy,13"`
}

type CashBankMovementTable struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     db.Col[*CashBankMovementTable, int32]
	ID            db.Col[*CashBankMovementTable, int64]
	CashBankID    db.Col[*CashBankMovementTable, int32]
	CashBankRefID db.Col[*CashBankMovementTable, int32]
	DocumentID    db.Col[*CashBankMovementTable, int64]
	ReferenceID   db.Col[*CashBankMovementTable, int32]
	Date          db.Col[*CashBankMovementTable, int16]
	Type          db.Col[*CashBankMovementTable, CashMovementType]
	AccountCode   db.Col[*CashBankMovementTable, int16]
	FinalAmount   db.Col[*CashBankMovementTable, int32]
	Amount        db.Col[*CashBankMovementTable, int32]
	Created       db.Col[*CashBankMovementTable, int32]
	CreatedBy     db.Col[*CashBankMovementTable, int32]
}

func (e CashBankMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        5,
		Name:      "cash_bank_movements",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		KeyIntPacking: db.Cols(
			//TODO: decrease to e.Autoincrement(7) in the future
			e.CashBankID.Size(17), e.Date.Size(15), e.Autoincrement(8),
		),
		AutoincrementPart: e.Date,
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.DocumentID)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.ReferenceID)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.CreatedBy)},
		},
	}
}

type CashReconciliation struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        int32 `cb:"CompanyID,1"`
	ID               int64 `cb:"ID,2"`
	CashBankID       int32 `cb:"CashBankID,3"`
	Type             int8  `cb:"Type,4"`
	MovementID       int64 `cb:"MovementID,5"`
	SystemAmount     int32 `cb:"SystemAmount,6"`
	ActualAmount     int32 `cb:"ActualAmount,7"`
	DifferenceAmount int32 `cb:"DifferenceAmount,8"`
	Created          int32 `cb:"Created,9"`
	CreatedBy        int32 `cb:"CreatedBy,10"`
}

type CashReconciliationTable struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        db.Col[*CashReconciliationTable, int32]
	ID               db.Col[*CashReconciliationTable, int64]
	CashBankID       db.Col[*CashReconciliationTable, int32]
	Type             db.Col[*CashReconciliationTable, int8]
	MovementID       db.Col[*CashReconciliationTable, int64]
	SystemAmount     db.Col[*CashReconciliationTable, int32]
	ActualAmount     db.Col[*CashReconciliationTable, int32]
	DifferenceAmount db.Col[*CashReconciliationTable, int32]
	Created          db.Col[*CashReconciliationTable, int32]
	CreatedBy        db.Col[*CashReconciliationTable, int32]
}

func (e CashReconciliationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        7,
		Name:      "cash_reconciliations",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.CreatedBy)},
		},
	}
}

type InternalCashMovement struct {
	CashBankID    int32
	CashBankRefID int32
	DocumentID    int64
	ReferenceID   int32 // Optional: e.g. the originating ExpenseScheduled.ID for expense payments.
	Date          int16 // Optional: movement date; falls back to the request's effective date if 0.
	Type          CashMovementType
	AccountCode   int16 // Required only for the types in CashMovementAccountOptions.
	Amount        int32
	FinalAmount   int32 // Optional: calculated if 0
}

type SaleProduct struct {
	ProductID int32
	Quantity  int32
	Amount    int32
}
