package types

import (
	"app/db"
	"fmt"
)

type User struct { // DynamoDB + ScyllaDB
	db.TableStruct[UserTable, User]
	CompanyID          int32    `json:",omitempty" col:"empresa_id,pk"`
	ID                 int32    `json:",omitempty" col:"id,pk,sk"`
	User               string   `json:",omitempty" col:"user,index"`
	LastName           string   `json:",omitempty" col:"last_name"`
	FirstName          string   `json:",omitempty" col:"first_name"`
	ProfileIDs         []int32  `json:",omitempty" col:"profile_ids"`
	// AccesoID * 10 + Nivel
	AccessLevelIDs     []int32  `json:",omitempty" col:"access_level_ids"`
	AccesosComputed    []uint16 `json:",omitempty" col:"accesos_computed"`
	Email              string   `json:",omitempty" col:"email,index"`
	JobTitle           string   `json:",omitempty" col:"job_title"`
	DocumentNumber     string   `json:",omitempty" col:"document_number"`
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
	CacheVersion uint8 `json:"ccv,omitempty" col:"-"`
}

func (user *User) PrepareCloudSync() {
	// Company + status + padded updated keeps delta queries lexicographically sortable across providers.
	user.CompanyUserIndex = fmt.Sprintf("%d_%s", user.CompanyID, user.User)
	user.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", user.CompanyID, user.Status, user.Updated)
}

type UserTable struct {
	db.TableStruct[UserTable, User]
	ID              db.Col[UserTable, int32]
	CompanyID       db.Col[UserTable, int32]
	User            db.Col[UserTable, string]
	LastName        db.Col[UserTable, string]
	FirstName       db.Col[UserTable, string]
	ProfileIDs      db.ColSlice[UserTable, int32] `db:"profile_ids"`
	AccessLevelIDs  db.Col[UserTable, []int32]    `db:"access_level_ids"`
	AccesosComputed db.Col[UserTable, []uint16]
	Email           db.Col[UserTable, string]
	JobTitle        db.Col[UserTable, string]
	DocumentNumber  db.Col[UserTable, string]
	Created         db.Col[UserTable, int32]
	CreatedBy       db.Col[UserTable, int32]
	Updated         db.Col[UserTable, int32]
	UpdatedBy       db.Col[UserTable, int32]
	Status          db.Col[UserTable, int8]
}

func (usuarioTable UserTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "users",
		Partition:        usuarioTable.CompanyID,
		UseSequences:     true,
		SaveCacheVersion: true,
		Keys:             []db.Coln{usuarioTable.ID.Autoincrement(0)},
	}
}
