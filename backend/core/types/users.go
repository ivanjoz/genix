package types

import (
	"app/db"
)

type User struct {
	db.TableStruct[UserTable, User]
	CompanyID  int32   `json:",omitempty"`
	ID         int32   `json:",omitempty"`
	User       string  `json:",omitempty"`
	LastName   string  `json:",omitempty"`
	FirstName  string  `json:",omitempty"`
	ProfileIDs []int32 `json:",omitempty"`
	// AccesoID * 10 + Nivel
	AccessLevelIDs  []int32  `json:",omitempty"`
	AccesosComputed []uint16 `json:",omitempty"`
	Email           string   `json:",omitempty"`
	JobTitle        string   `json:",omitempty"`
	DocumentNumber  string   `json:",omitempty"`
	PasswordHash    string   `json:",omitempty"`
	// Password is write-only input from the client. It is absent from UserTable, which is
	// what keeps it out of every database: the column set comes from the table struct.
	Password  string `json:",omitempty"`
	Created   int32  `json:",omitempty"`
	CreatedBy int32  `json:",omitempty"`
	Updated   int32  `json:"upd,omitempty"`
	UpdatedBy int32  `json:",omitempty"`
	Status    int8   `json:"ss,omitempty"`
	// UpdatedVersion is the write sequence number. By-IDs endpoints overwrite it with the record's
	// slot version, which is what the client sends back to prove its copy is still current.
	UpdatedVersion int32 `json:"upv,omitempty"`
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
	PasswordHash    db.Col[UserTable, string]
	Created         db.Col[UserTable, int32]
	CreatedBy       db.Col[UserTable, int32]
	Updated         db.Col[UserTable, int32]
	UpdatedBy       db.Col[UserTable, int32]
	Status          db.Col[UserTable, int8]
	UpdatedVersion  db.Col[UserTable, int32]
}

func (usuarioTable UserTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 35,
		Name:               "users",
		Partition:          usuarioTable.CompanyID,
		UseSequences:       true,
		SaveUpdatedVersion: true,
		// Users have no single display column, so the login handle is the label and the client
		// composes the full name from S1/S2. Email and DocumentNumber stay out: a label doesn't need them.
		GenericRecord: db.GenericRecordSchema{
			Name: usuarioTable.User, S1: usuarioTable.FirstName, S2: usuarioTable.LastName,
		},
		Keys: db.Cols(usuarioTable.ID.Autoincrement(0)),
		// Declaration order is also the cloud mirror's index-slot order (ix1, ix2, ix3).
		// The login handle and the email are looked up directly; status+updated is the
		// delta read, padded so the range compares in numeric order.
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(usuarioTable.User)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(usuarioTable.Email)},
			{Type: db.TypeView, Keys: db.Cols(
				usuarioTable.Status, usuarioTable.Updated.DecimalSize(10))},
		},
	}
}
