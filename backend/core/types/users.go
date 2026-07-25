package types

import (
	"github.com/ivanjoz/genix-orm/scylla"
	"fmt"
)

type User struct { // DynamoDB + ScyllaDB
	scylla.TableStruct[UserTable, User]
	CompanyID  int32   `json:",omitempty" col:"empresa_id,pk"`
	ID         int32   `json:",omitempty" col:"id,pk,sk"`
	User       string  `json:",omitempty" col:"user,index"`
	LastName   string  `json:",omitempty" col:"last_name"`
	FirstName  string  `json:",omitempty" col:"first_name"`
	ProfileIDs []int32 `json:",omitempty" col:"profile_ids"`
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
	scylla.TableStruct[UserTable, User]
	ID              scylla.Col[UserTable, int32]
	CompanyID       scylla.Col[UserTable, int32]
	User            scylla.Col[UserTable, string]
	LastName        scylla.Col[UserTable, string]
	FirstName       scylla.Col[UserTable, string]
	ProfileIDs      scylla.ColSlice[UserTable, int32] `db:"profile_ids"`
	AccessLevelIDs  scylla.Col[UserTable, []int32]    `db:"access_level_ids"`
	AccesosComputed scylla.Col[UserTable, []uint16]
	Email           scylla.Col[UserTable, string]
	JobTitle        scylla.Col[UserTable, string]
	DocumentNumber  scylla.Col[UserTable, string]
	Created         scylla.Col[UserTable, int32]
	CreatedBy       scylla.Col[UserTable, int32]
	Updated         scylla.Col[UserTable, int32]
	UpdatedBy       scylla.Col[UserTable, int32]
	Status          scylla.Col[UserTable, int8]
}

func (usuarioTable UserTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:             "users",
		Partition:        usuarioTable.CompanyID,
		UseSequences:     true,
		SaveCacheVersion: true,
		// Users have no single display column, so the login handle is the label and the client
		// composes the full name from S1/S2. Email and DocumentNumber stay out: a label doesn't need them.
		GenericRecord: scylla.GenericRecordSchema{
			Name: usuarioTable.User, S1: usuarioTable.FirstName, S2: usuarioTable.LastName,
		},
		Keys: scylla.Cols(usuarioTable.ID.Autoincrement(0)),
	}
}
