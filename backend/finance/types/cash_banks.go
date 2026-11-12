package types

import "app/db"

type CashBank struct {
	db.TableStruct[CashBankTable, CashBank]
	CompanyID            int32  `cb:"1"`
	ID                   int32  `cb:"2"`
	Type                 int32  `cb:"3"`
	SiteID               int32  `cb:"4"`
	Name                 string `cb:"5"`
	Description          string `cb:"6"`
	CurrencyType         int8   `cb:"7"`
	ReconciliationDate   int32  `cb:"8"`
	ReconciliationAmount int32  `cb:"9"`
	CurrentAmount        int32  `cb:"10"`
	// General properties
	Status  int8  `json:"ss" cb:"ss,11"`
	Updated int32 `json:"upd" cb:"upd,12"`
	Version int16 `json:"ver" cb:"ver,13"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,14"`
	UpdatedBy int32   `cb:"15"`
	Created   int32   `cb:"16"`
	CreatedBy int32   `cb:"17"`
	// ReceivesCulqiPayments marks the account the web store's card/Yape charges are collected
	// into. At most one per company (PostCashBanks enforces it).
	ReceivesCulqiPayments bool `cb:"18"`
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
	ReceivesCulqiPayments db.Col[*CashBankTable, bool]
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
	// CashMovementTypePayrollPayment pays the staff (app/hr): a run's net pay (4111), a weekly
	// advance (1412) or a loan disbursement (1411). DocumentID is the run, advance or loan; the
	// counter-account tells which.
	CashMovementTypePayrollPayment CashMovementType = 12
	// CashMovementTypePayrollContributions pays what a payroll withheld or owes to third parties:
	// EsSalud, ONP, 5ta, the AFPs, a judicial withholding. DocumentID is the run.
	CashMovementTypePayrollContributions CashMovementType = 13
)

type CashBankMovement struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID     int32            `cb:"1"`
	ID            int64            `cb:"2"`
	CashBankID    int32            `cb:"3"`
	CashBankRefID int32            `cb:"4"`
	DocumentID    int64            `cb:"5"` // Sale Order ID / ExpenseID
	ReferenceID   int32            `cb:"6"` //
	Date          int16            `cb:"7"`
	Type          CashMovementType `cb:"8"`
	// CounterAccountCode is the PCGE account on the other side of the cash, for the types the
	// user classifies (see CashMovementAccountOptions). 0 for every other type: its
	// counter-account follows from the type and the document it pays.
	CounterAccountCode int32 `cb:"9"`
	FinalAmount        int32 `cb:"10"`
	Amount             int32 `cb:"11"`
	Created            int32 `cb:"12"`
	CreatedBy          int32 `cb:"13"`
	// Period is the YYMM of Date (core.PeriodOfDay): the pinned key of the period delta index the
	// accounting operations sync a month through.
	Period int16 `cb:"14"`
	// SameDayMovement is 1 when this movement alone settled its whole document on the document's
	// accounting date, at the cash bank the document records (SameDaySettlement); the document
	// mirrors it in its SameDayCashBankID. A pinned key of the period delta index, so the accounting
	// operations never download these rows: they infer them from the document.
	SameDayMovement int8  `cb:"15"`
	Version         int16 `json:"ver" cb:"ver,16"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,17"`
	// Updated is the ORM-stamped write tick: the period delta index packs it into its key and a
	// Delta() read hands it to the client as the cursor.
	Updated int32 `json:"upd" cb:"upd,18"`
}

type CashBankMovementTable struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID          db.Col[*CashBankMovementTable, int32]
	ID                 db.Col[*CashBankMovementTable, int64]
	CashBankID         db.Col[*CashBankMovementTable, int32]
	CashBankRefID      db.Col[*CashBankMovementTable, int32]
	DocumentID         db.Col[*CashBankMovementTable, int64]
	ReferenceID        db.Col[*CashBankMovementTable, int32]
	Date               db.Col[*CashBankMovementTable, int16]
	Type               db.Col[*CashBankMovementTable, CashMovementType]
	CounterAccountCode db.Col[*CashBankMovementTable, int32]
	FinalAmount        db.Col[*CashBankMovementTable, int32]
	Amount             db.Col[*CashBankMovementTable, int32]
	Created            db.Col[*CashBankMovementTable, int32]
	CreatedBy          db.Col[*CashBankMovementTable, int32]
	Period             db.Col[*CashBankMovementTable, int16]
	SameDayMovement    db.Col[*CashBankMovementTable, int8]
	Version            db.Col[*CashBankMovementTable, int16]
	Updated            db.Col[*CashBankMovementTable, int32]
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
		// Size the slots of the period delta index. Type leaves room for new movement types.
		FixedValues: []db.FixedValues{
			{Col: e.Period, Min: 0, Max: 9912},
			{Col: e.SameDayMovement, Values: []int64{0, 1}},
			{Col: e.Type, Min: 0, Max: 31},
		},
		Indexes: []db.Index{
			// The payments of one document (an expense's payment tab): one cache group per document,
			// downloaded again only after a new payment of it. Also the local index the payment
			// handlers sum a document's payments through.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.DocumentID)},
			// A cash bank's movements of a date range: one group per (cash bank, day), a key range.
			{Type: db.TypeCacheGroupFromKey, Keys: db.Cols(e.CashBankID, e.Date)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.CreatedBy)},
			// One log per accounting period and SameDayMovement value: GET.accounting-operations
			// syncs a month's SameDayMovement = 0 log only. Movements are never deleted, so Type is
			// just the sync filter a first sync lists.
			{Type: db.TypeDelta, Keys: db.Cols(e.Period, e.SameDayMovement, e.Type)},
		},
	}
}

type CashReconciliation struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID        int32 `cb:"1"`
	ID               int64 `cb:"2"`
	CashBankID       int32 `cb:"3"`
	Type             int8  `cb:"4"`
	MovementID       int64 `cb:"5"`
	SystemAmount     int32 `cb:"6"`
	ActualAmount     int32 `cb:"7"`
	DifferenceAmount int32 `cb:"8"`
	Created          int32 `cb:"9"`
	CreatedBy        int32 `cb:"10"`
	Updated          int32 `json:"upd" cb:"upd,11"`
	Version          int16 `json:"ver" cb:"ver,12"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,13"`
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
	Updated          db.Col[*CashReconciliationTable, int32]
	Version          db.Col[*CashReconciliationTable, int16]
}

func (e CashReconciliationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        7,
		Name:      "cash_reconciliations",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		Indexes: []db.Index{
			// One log per cash bank: the Cuadres tab syncs its cash bank's counts. Counts are manual
			// and never deleted, so the list stays short and only new counts travel.
			{Type: db.TypeDelta, Keys: db.Cols(e.CashBankID)},
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
	// Required for the types in CashMovementAccountOptions and for a document's payment
	// (DocumentPaymentAccounts).
	CounterAccountCode int32
	Amount             int32
	FinalAmount        int32 // Optional: calculated if 0
	// SameDayMovement is 1 when the document's process decided this payment alone settles it on its
	// accounting date (SameDaySettlement).
	SameDayMovement int8
}

type SaleProduct struct {
	ProductID int32
	Quantity  int32
	Amount    int32
}
