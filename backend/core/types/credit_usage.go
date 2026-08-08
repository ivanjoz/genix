package types

import "app/db"

type CreditUsage struct {
	db.TableStruct[CreditUsageTable, CreditUsage]
	CompanyID   int32
	UserID      int32
	TimeFrame   int32
	UsedCredits []byte
}

type CreditUsageTable struct {
	db.TableStruct[CreditUsageTable, CreditUsage]
	CompanyID   db.Col[CreditUsageTable, int32]
	UserID      db.Col[CreditUsageTable, int32]
	TimeFrame   db.Col[CreditUsageTable, int32]
	UsedCredits db.Col[CreditUsageTable, []byte]
}

func (e CreditUsageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// This table stores externally aggregated absolute snapshots and needs no ORM-managed fields.
		ID:                    42,
		Name:                  "credit_usage",
		Partition:             e.CompanyID,
		Keys:                  db.Cols(e.UserID, e.TimeFrame),
		DisableDefaultColumns: true,
	}
}
