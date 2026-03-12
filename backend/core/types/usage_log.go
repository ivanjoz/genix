package core

import "app/db"

type UsageLog struct {
	db.TableStruct[UsageLogTable, UsageLog]
	CompanyID              int32   `json:"companyID,omitempty" db:"company_id"`
	ID                     int32   `json:"id,omitempty" db:"id"`
	GetBandwith            int32   `json:"getBandwith,omitempty" db:"get_bandwith"`
	PostBandwith           int32   `json:"postBandwith,omitempty" db:"post_bandwith"`
	GetCpuTimeUsage        int32   `json:"getCpuTimeUsage,omitempty" db:"get_cpu_time_usage"`
	PostCpuTimeUsage       int32   `json:"postCpuTimeUsage,omitempty" db:"post_cpu_time_usage"`
	DetailUserID           []int32 `json:"detailUserID,omitempty" db:"detail_user_id"`
	DetailGetBandwith      []int32 `json:"detailGetBandwith,omitempty" db:"detail_get_bandwith"`
	DetailPostBandwith     []int32 `json:"detailPostBandwith,omitempty" db:"detail_post_bandwith"`
	DetailGetCpuTimeUsage  []int32 `json:"detailGetCpuTimeUsage,omitempty" db:"detail_get_cpu_time_usage"`
	DetailPostCpuTimeUsage []int32 `json:"detailPostCpuTimeUsage,omitempty" db:"detail_post_cpu_time_usage"`
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
	}
}
