package types

import "github.com/ivanjoz/genix-orm/scylla"

type UsageLog struct {
	scylla.TableStruct[UsageLogTable, UsageLog]
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
	scylla.TableStruct[UsageLogTable, UsageLog]
	CompanyID              scylla.Col[UsageLogTable, int32]
	ID                     scylla.Col[UsageLogTable, int32]
	GetBandwith            scylla.Col[UsageLogTable, int32]
	PostBandwith           scylla.Col[UsageLogTable, int32]
	GetCpuTimeUsage        scylla.Col[UsageLogTable, int32]
	PostCpuTimeUsage       scylla.Col[UsageLogTable, int32]
	DetailUserID           scylla.Col[UsageLogTable, []int32]
	DetailGetBandwith      scylla.Col[UsageLogTable, []int32]
	DetailPostBandwith     scylla.Col[UsageLogTable, []int32]
	DetailGetCpuTimeUsage  scylla.Col[UsageLogTable, []int32]
	DetailPostCpuTimeUsage scylla.Col[UsageLogTable, []int32]
}

func (usageLogTable UsageLogTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:                 "usage_log",
		Partition:            usageLogTable.CompanyID,
		Keys:                 scylla.Cols(usageLogTable.ID),
		DisableUpdateCounter: true,
		Indexes: []scylla.Index{
			// Partitioned by ID so a single log row is readable without knowing the company.
			{Type: scylla.TypeView, Keys: scylla.Cols(usageLogTable.ID), Partition: usageLogTable.ID},
		},
	}
}
