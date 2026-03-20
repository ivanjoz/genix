package types

import (
	"app/db"
	"fmt"
)

type Perfil struct {
	db.TableStruct[PerfilTable, Perfil]
	EmpresaID           int32   `db:"empresa_id,pk" col:"empresa_id,pk"`
	ID                  int32   `db:"id,pk" col:"id,pk,sk"`
	Nombre              string  `db:"nombre" col:"nombre"`
	Descripcion         string  `db:"descripcion" col:"descripcion"`
	Modulos             []int16 `db:"modulos_ids" col:"modulos_ids"`
	Accesos             []int32 `db:"accesos" col:"accesos"`
	Status              int8    `json:"ss" db:"status" col:"status"`
	Updated             int32   `json:"upd" db:"updated" col:"updated"`
	CompanyUpdatedIndex string  `json:"-" col:"company_updated,index"`
	CompanyStatusIndex  string  `json:"-" col:"company_status_updated,index"`
}

func (e *Perfil) PrepareCloudSync() {
	// Synthetic keys keep profile lookups and delta queries scoped by company across providers.
	e.CompanyUpdatedIndex = fmt.Sprintf("%d_%020d", e.EmpresaID, e.Updated)
	e.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", e.EmpresaID, e.Status, e.Updated)
}

type PerfilTable struct {
	db.TableStruct[PerfilTable, Perfil]
	ID          db.Col[PerfilTable, int32]
	EmpresaID   db.Col[PerfilTable, int32]
	Nombre      db.Col[PerfilTable, string]
	Descripcion db.Col[PerfilTable, string]
	Modulos     db.ColSlice[PerfilTable, int16] `db:"modulos_ids"`
	Accesos     db.ColSlice[PerfilTable, int32] `db:"accesos"`
	Status      db.Col[PerfilTable, int8]
	Updated     db.Col[PerfilTable, int32]
}

func (e PerfilTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "seguridad_perfiles",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
	}
}
