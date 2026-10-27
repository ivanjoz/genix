package types

import (
	"app/db"
)

type User struct {
	db.TableStruct[UserTable, User]
	CompanyID  int32   `json:",omitempty" cb:"CompanyID,1"`
	ID         int32   `json:",omitempty" cb:"ID,2"`
	User       string  `json:",omitempty" cb:"User,3"`
	LastName   string  `json:",omitempty" cb:"LastName,4"`
	FirstName  string  `json:",omitempty" cb:"FirstName,5"`
	ProfileIDs []int32 `json:",omitempty" cb:"ProfileIDs,6"`
	// AccesosGrants are the accesses granted to this user directly, on top of their profiles.
	// Same shape a profile stores, so both are edited and merged by the same code.
	AccesosGrants []AccesoGrantRecord `json:",omitempty" cb:"AccesosGrants,7"`
	// The two grant blobs, encoded by core.EncodeAccesosGrants and read byte for byte by
	// fareward and by genix-ui. Which blob an access lands in is itself the "has sub-accesses"
	// flag: accesos_computed is fixed-stride grant words, accesos_sub_computed appends each
	// grant's sub bytes. See core/accesos-blob.go.
	AccesosComputed    []byte `json:",omitempty" cb:"AccesosComputed,8"`
	AccesosSubComputed []byte `json:",omitempty" cb:"AccesosSubComputed,9"`
	Email              string `json:",omitempty" cb:"Email,10"`
	JobTitle           string `json:",omitempty" cb:"JobTitle,11"`
	DocumentNumber     string `json:",omitempty" cb:"DocumentNumber,12"`
	PasswordHash       string `json:",omitempty" cb:"PasswordHash,13"`
	// Password is write-only input from the client. It is absent from UserTable, which is
	// what keeps it out of every database: the column set comes from the table struct.
	Password  string `json:",omitempty" cb:"Password,14"`
	Created   int32  `json:",omitempty" cb:"Created,15"`
	CreatedBy int32  `json:",omitempty" cb:"CreatedBy,16"`
	Updated   int32  `json:"upd,omitempty" cb:"upd,17"`
	UpdatedBy int32  `json:",omitempty" cb:"UpdatedBy,18"`
	Status    int8   `json:"ss,omitempty" cb:"ss,19"`
}

type UserTable struct {
	db.TableStruct[UserTable, User]
	ID                 db.Col[*UserTable, int32]
	CompanyID          db.Col[*UserTable, int32]
	User               db.Col[*UserTable, string]
	LastName           db.Col[*UserTable, string]
	FirstName          db.Col[*UserTable, string]
	ProfileIDs         db.ColSlice[*UserTable, int32]          `db:"profile_ids"`
	AccesosGrants      db.Col[*UserTable, []AccesoGrantRecord] `db:"accesos_grants"`
	AccesosComputed    db.Col[*UserTable, []byte]
	AccesosSubComputed db.Col[*UserTable, []byte]
	Email              db.Col[*UserTable, string]
	JobTitle           db.Col[*UserTable, string]
	DocumentNumber     db.Col[*UserTable, string]
	PasswordHash       db.Col[*UserTable, string]
	Created            db.Col[*UserTable, int32]
	CreatedBy          db.Col[*UserTable, int32]
	Updated            db.Col[*UserTable, int32]
	UpdatedBy          db.Col[*UserTable, int32]
	Status             db.Col[*UserTable, int8]
}

func (usuarioTable UserTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           35,
		Name:         "users",
		Partition:    usuarioTable.CompanyID,
		UseSequences: true,
		CacheByIDs:   true,
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
				usuarioTable.Status, usuarioTable.Updated.Size(31))},
		},
	}
}
