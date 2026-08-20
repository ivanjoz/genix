package types

import "app/db"

// CreditHistory is the append-only ledger of credit grants and revocations. The backend writes one
// row only after the credit daemon confirms the mutation, so a row here means the credits really
// moved — company_credit_budget keeps just the resulting balance, which cannot say who granted what
// or when.
//
// Credits carries the signed movement: positive when credit was added, negative when it was taken
// away. Ceiling* is the absolute monthly ceiling the mutation left behind, so a row is readable on
// its own without replaying the whole ledger.
type CreditHistory struct {
	db.TableStruct[CreditHistoryTable, CreditHistory]
	CompanyID int32
	// ID packs Day(5)+Autoincrement(3) into an int64. Budget changes are rare administrative acts,
	// so 999 of them in one day is far beyond anything real.
	ID               int64
	Day              int16
	Created          int32
	CreatedBy        int32
	Operation        int8
	CPUCredits       int64
	InferenceCredits int64
	CPUCeiling       int64
	InferenceCeiling int64
}

type CreditHistoryTable struct {
	db.TableStruct[CreditHistoryTable, CreditHistory]
	CompanyID        db.Col[CreditHistoryTable, int32]
	ID               db.Col[CreditHistoryTable, int64]
	Day              db.Col[CreditHistoryTable, int16]
	Created          db.Col[CreditHistoryTable, int32]
	CreatedBy        db.Col[CreditHistoryTable, int32]
	Operation        db.Col[CreditHistoryTable, int8]
	CPUCredits       db.Col[CreditHistoryTable, int64]
	InferenceCredits db.Col[CreditHistoryTable, int64]
	CPUCeiling       db.Col[CreditHistoryTable, int64]
	InferenceCeiling db.Col[CreditHistoryTable, int64]
}

func (e CreditHistoryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        49,
		Name:      "credit_history",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		// The counter runs per day, which is what keeps the packed ID dense instead of leaving a
		// company's whole history in one ever-growing sequence.
		KeyIntPacking:     db.Cols(e.Day.DecimalSize(5), e.Autoincrement(3)),
		AutoincrementPart: e.Day,
		// Rows are written once and never touched, and Created already carries the write time.
		DisableDefaultColumns: true,
		Indexes: []db.Index{
			// Free prefix scan over the packed key: "the grants between these two days".
			{Type: db.TypeInheritFromKey, Keys: db.Cols(e.Day), UseIndexGroup: true},
		},
	}
}
