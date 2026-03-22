package types

import "app/db"

type UsageLog struct {
	db.TableStruct[UsageLogTable, UsageLog]
	CompanyID              int32   `json:",omitempty"`
	ID                     int32   `json:",omitempty"`
	GetBandwith            int32   `json:",omitempty"`
	PostBandwith           int32   `json:",omitempty"`
	GetCpuTimeUsage        int32   `json:",omitempty"`
	PostCpuTimeUsage       int32   `json:",omitempty"`
	DetailUserID           []int32 `json:",omitempty" db:",list"`
	DetailGetBandwith      []int32 `json:",omitempty" db:",list"`
	DetailPostBandwith     []int32 `json:",omitempty" db:",list"`
	DetailGetCpuTimeUsage  []int32 `json:",omitempty" db:",list"`
	DetailPostCpuTimeUsage []int32 `json:",omitempty" db:",list"`
}

type UsageLogTable struct {
	db.TableStruct[UsageLogTable, UsageLog]
	CompanyID              db.Col[UsageLogTable, int32]
	ID                     db.Col[UsageLogTable, int32]
	GetBandwith            db.Col[UsageLogTable, int32]
	PostBandwith           db.Col[UsageLogTable, int32]
	GetCpuTimeUsage        db.Col[UsageLogTable, int32]
	PostCpuTimeUsage       db.Col[UsageLogTable, int32]
	DetailUserID           db.Col[UsageLogTable, []int32]
	DetailGetBandwith      db.Col[UsageLogTable, []int32]
	DetailPostBandwith     db.Col[UsageLogTable, []int32]
	DetailGetCpuTimeUsage  db.Col[UsageLogTable, []int32]
	DetailPostCpuTimeUsage db.Col[UsageLogTable, []int32]
}

func (usageLogTable UsageLogTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "usage_log",
		Partition: usageLogTable.CompanyID,
		Keys:      []db.Coln{usageLogTable.ID},
		Views: []db.View{
			{Cols: []db.Coln{usageLogTable.ID}, KeepPart: false},
		},
	}
}
