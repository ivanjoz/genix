package types

import "app/db"

type CustomerProfile struct {
	db.TableStruct[CustomerProfileTable, CustomerProfile]
	EmpresaID  int32    `json:"empresa_id,omitempty" db:"empresa_id"`
	Status     int8     `json:"ss,omitempty" db:"status"`
	Updated    int64    `json:"upd,omitempty" db:"updated"`
	UpdatedBy  int32    `json:"updated_by,omitempty" db:"updated_by"`
	FirstName  string   `json:"first_name,omitempty" db:"first_name"`
	LastName   string   `json:"last_name,omitempty" db:"last_name"`
	Age        int32    `json:"age,omitempty" db:"age"`
	Tags       []string `json:"tags,omitempty" db:"tags"`
	DemoNumber int64    `json:"demo_number,omitempty" db:"demo_number,pk"`
}

type CustomerProfileTable struct {
	db.TableStruct[CustomerProfileTable, CustomerProfile]
	EmpresaID  db.Col[CustomerProfileTable, int32]
	Status     db.Col[CustomerProfileTable, int8]
	Updated    db.Col[CustomerProfileTable, int64]
	UpdatedBy  db.Col[CustomerProfileTable, int32]
	FirstName  db.Col[CustomerProfileTable, string]
	LastName   db.Col[CustomerProfileTable, string]
	Age        db.Col[CustomerProfileTable, int32]
	Tags       db.ColSlice[CustomerProfileTable, string]
	DemoNumber db.Col[CustomerProfileTable, int64]
}

func (e CustomerProfileTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "customer_profile",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.DemoNumber},
	}
}
