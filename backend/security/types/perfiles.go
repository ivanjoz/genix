package types

import (
	"app/db"
	"fmt"
)

type Profile struct {
	db.TableStruct[ProfileTable, Profile]
	CompanyID           int32   `db:"empresa_id,pk" col:"empresa_id,pk"`
	ID                  int32   `db:"id,pk" col:"id,pk,sk"`
	Name                string  `db:"nombre" col:"nombre"`
	Description         string  `db:"descripcion" col:"descripcion"`
	Modules             []int16 `db:"modulos_ids" col:"modulos_ids"`
	Accesos             []int32 `db:"accesos" col:"accesos"`
	Status              int8    `json:"ss" db:"status" col:"status"`
	Updated             int32   `json:"upd" db:"updated" col:"updated"`
	CompanyUpdatedIndex string  `json:"-" col:"company_updated,index"`
	CompanyStatusIndex  string  `json:"-" col:"company_status_updated,index"`
}

func (e *Profile) PrepareCloudSync() {
	// Synthetic keys keep profile lookups and delta queries scoped by company across providers.
	e.CompanyUpdatedIndex = fmt.Sprintf("%d_%020d", e.CompanyID, e.Updated)
	e.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", e.CompanyID, e.Status, e.Updated)
}

type ProfileTable struct {
	db.TableStruct[ProfileTable, Profile]
	ID          db.Col[ProfileTable, int32]
	CompanyID   db.Col[ProfileTable, int32]
	Name        db.Col[ProfileTable, string]
	Description db.Col[ProfileTable, string]
	Modules     db.ColSlice[ProfileTable, int16] `db:"modulos_ids"`
	Accesos     db.ColSlice[ProfileTable, int32] `db:"accesos"`
	Status      db.Col[ProfileTable, int8]
	Updated     db.Col[ProfileTable, int32]
}

func (e ProfileTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "profiles",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
	}
}
