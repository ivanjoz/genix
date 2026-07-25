package types

import (
	"github.com/ivanjoz/genix-orm/scylla"
	"fmt"
)

type Profile struct {
	scylla.TableStruct[ProfileTable, Profile]
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
	scylla.TableStruct[ProfileTable, Profile]
	ID          scylla.Col[ProfileTable, int32]
	CompanyID   scylla.Col[ProfileTable, int32]
	Name        scylla.Col[ProfileTable, string]
	Description scylla.Col[ProfileTable, string]
	Modules     scylla.ColSlice[ProfileTable, int16] `db:"modulos_ids"`
	Accesos     scylla.ColSlice[ProfileTable, int32] `db:"accesos"`
	Status      scylla.Col[ProfileTable, int8]
	Updated     scylla.Col[ProfileTable, int32]
}

func (e ProfileTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "profiles",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
	}
}
