package types

import "app/db"

type UsageLog struct {
	db.TableStruct[UsageLogTable, UsageLog]
	CompanyID              int32   `cb:"1"`
	ID                     int32   `cb:"2"`
	GetBandwith            int32   `cb:"3"`
	PostBandwith           int32   `cb:"4"`
	GetCpuTimeUsage        int32   `cb:"5"`
	PostCpuTimeUsage       int32   `cb:"6"`
	DetailUserID           []int32 `db:",list" cb:"7"`
	DetailGetBandwith      []int32 `db:",list" cb:"8"`
	DetailPostBandwith     []int32 `db:",list" cb:"9"`
	DetailGetCpuTimeUsage  []int32 `db:",list" cb:"10"`
	DetailPostCpuTimeUsage []int32 `db:",list" cb:"11"`
}

type UsageLogTable struct {
	db.TableStruct[UsageLogTable, UsageLog]
	CompanyID              db.Col[*UsageLogTable, int32]
	ID                     db.Col[*UsageLogTable, int32]
	GetBandwith            db.Col[*UsageLogTable, int32]
	PostBandwith           db.Col[*UsageLogTable, int32]
	GetCpuTimeUsage        db.Col[*UsageLogTable, int32]
	PostCpuTimeUsage       db.Col[*UsageLogTable, int32]
	DetailUserID           db.Col[*UsageLogTable, []int32]
	DetailGetBandwith      db.Col[*UsageLogTable, []int32]
	DetailPostBandwith     db.Col[*UsageLogTable, []int32]
	DetailGetCpuTimeUsage  db.Col[*UsageLogTable, []int32]
	DetailPostCpuTimeUsage db.Col[*UsageLogTable, []int32]
}

func (usageLogTable UsageLogTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        34,
		Name:      "usage_log",
		Partition: usageLogTable.CompanyID,
		Keys:      db.Cols(usageLogTable.ID),
		Indexes: []db.Index{
			// Partitioned by ID so a single log row is readable without knowing the company.
			{Type: db.TypeView, Keys: db.Cols(usageLogTable.ID), Partition: usageLogTable.ID},
		},
	}
}
