package types

import "app/db"

type SystemParameters struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	ID        int32   `json:",omitempty" db:"parameter_id,pk" cb:"ID,1"`
	ValueText string  `json:",omitempty" db:"value_text" cb:"ValueText,2"`
	ValueInts []int32 `json:",omitempty" db:"value_ints" cb:"ValueInts,3"`
	Value     int32   `json:",omitempty" db:"value" cb:"Value,4"`
	CompanyID int32   `json:",omitempty" db:"empresa_id" cb:"CompanyID,5"`
	Updated   int32   `json:"upd," db:"updated" cb:"upd,6"`
	Version   int16   `json:"ver,omitempty" cb:"ver,7"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt,omitempty" cb:"mdt,8"`
	UpdatedBy int32   `json:",omitempty" db:"updated_by" cb:"UpdatedBy,9"`
}

type SystemParametersTable struct {
	db.TableStruct[SystemParametersTable, SystemParameters]
	CompanyID db.Col[*SystemParametersTable, int32]
	ID        db.Col[*SystemParametersTable, int32]
	ValueText db.Col[*SystemParametersTable, string]
	ValueInts db.ColSlice[*SystemParametersTable, int32]
	Value     db.Col[*SystemParametersTable, int32]
	Updated   db.Col[*SystemParametersTable, int32]
	Version   db.Col[*SystemParametersTable, int16]
	UpdatedBy db.Col[*SystemParametersTable, int32]
}

func (e SystemParametersTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        33,
		Name:      "system_parameters",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		Indexes: []db.Index{
			// Keyless: one log per partition and no sync filter. Read with Delta(cursor).
			{Type: db.TypeDelta},
		},
	}
}
