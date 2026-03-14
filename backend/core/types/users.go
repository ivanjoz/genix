package core

import (
	"app/db"
	"fmt"
)

type Usuario struct { // DynamoDB + ScyllaDB
	db.TableStruct[UsuarioTable, Usuario]
	EmpresaID   int32   `json:",omitempty" col:"empresa_id,pk"`
	ID          int32   `json:",omitempty" col:"id,pk,sk"`
	Usuario     string  `json:",omitempty" col:"usuario,index"`
	Apellidos   string  `json:",omitempty" col:"apellidos"`
	Nombres     string  `json:",omitempty" col:"nombres"`
	PerfilesIDs []int32 `json:",omitempty" col:"perfiles_ids"`
	// AccesoID * 10 + Nivel
	AccesosNivelIDs    []int32  `json:",omitempty" col:"accesos_nivel_ids"`
	AccesosComputed    []uint16 `json:",omitempty" col:"accesos_computed"`
	Email              string   `json:",omitempty" col:"email,index"`
	Cargo              string   `json:",omitempty" col:"cargo"`
	DocumentoNro       string   `json:",omitempty" col:"documento_nro"`
	PasswordHash       string   `json:",omitempty" col:"password_hash"`
	Password           string   `json:",omitempty" col:"-"`
	Created            int32    `json:",omitempty" col:"created"`
	CreatedBy          int32    `json:",omitempty"  col:"created_by"`
	Updated            int32    `json:"upd,omitempty" col:"updated"`
	UpdatedBy          int32    `json:",omitempty" col:"updated_by"`
	Status             int8     `json:"ss,omitempty" col:"status"`
	CompanyUserIndex   string   `json:"-" col:"company_usuario,index"`
	CompanyStatusIndex string   `json:"-" col:"company_status_updated,index"`
	// CacheVersion is returned in delta-by-IDs endpoints to let clients track per-record cache freshness.
	CacheVersion uint8 `json:",omitempty" col:"-"`
}

func (usuario *Usuario) PrepareCloudSync() {
	// Company + status + padded updated keeps delta queries lexicographically sortable across providers.
	usuario.CompanyUserIndex = fmt.Sprintf("%d_%s", usuario.EmpresaID, usuario.Usuario)
	usuario.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", usuario.EmpresaID, usuario.Status, usuario.Updated)
}

type UsuarioTable struct {
	db.TableStruct[UsuarioTable, Usuario]
	ID              db.Col[UsuarioTable, int32]
	EmpresaID       db.Col[UsuarioTable, int32]
	Usuario         db.Col[UsuarioTable, string]
	Apellidos       db.Col[UsuarioTable, string]
	Nombres         db.Col[UsuarioTable, string]
	PerfilesIDs     db.ColSlice[UsuarioTable, int32] `db:"perfiles_ids"`
	AccesosNivelIDs db.Col[UsuarioTable, []int32]    `db:"accesos_nivel_ids"`
	AccesosComputed db.Col[UsuarioTable, []uint16]
	Email           db.Col[UsuarioTable, string]
	Cargo           db.Col[UsuarioTable, string]
	DocumentoNro    db.Col[UsuarioTable, string]
	Created         db.Col[UsuarioTable, int32]
	CreatedBy       db.Col[UsuarioTable, int32]
	Updated         db.Col[UsuarioTable, int32]
	UpdatedBy       db.Col[UsuarioTable, int32]
	Status          db.Col[UsuarioTable, int8]
}

func (usuarioTable UsuarioTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "usuarios",
		Partition:        usuarioTable.EmpresaID,
		UseSequences:     true,
		SaveCacheVersion: true,
		Keys:             []db.Coln{usuarioTable.ID.Autoincrement(0)},
	}
}
