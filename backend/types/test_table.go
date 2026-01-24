package types

import "app/db"

type TestTable struct {
	db.TableStruct[TestTableTable, TestTable]
	EmpresaID int32    `json:"empresa_id,omitempty" db:"empresa_id"`
	Status    int8     `json:"ss,omitempty" db:"status"`
	Updated   int64    `json:"upd,omitempty" db:"updated"`
	UpdatedBy int32    `json:"updated_by,omitempty" db:"updated_by"`
	Name      string   `json:"name,omitempty" db:"name"`
	Age       int32    `json:"age,omitempty" db:"age,pk"`
	Data      []string `json:"data,omitempty" db:"data"`
}

type TestTableTable struct {
	db.TableStruct[TestTableTable, TestTable]
	EmpresaID db.Col[TestTableTable, int32]
	Status    db.Col[TestTableTable, int8]
	Updated   db.Col[TestTableTable, int64]
	UpdatedBy db.Col[TestTableTable, int32]
	Name      db.Col[TestTableTable, string]
	Age       db.Col[TestTableTable, int32]
	Data      db.ColSlice[TestTableTable, string]
}

func (e TestTableTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "test_table",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.Age},
	}
}
