package types

import (
	"app/db"
)

type Profile struct {
	db.TableStruct[ProfileTable, Profile]
	CompanyID   int32   `db:"empresa_id,pk"`
	ID          int32   `db:"id,pk"`
	Name        string  `db:"nombre"`
	Description string  `db:"descripcion"`
	Modules     []int16 `db:"modulos_ids"`
	Accesos     []int32 `db:"accesos"`
	Status      int8    `json:"ss" db:"status"`
	Updated     int32   `json:"upd" db:"updated"`
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
		ID:           23,
		Name:         "profiles",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Declaration order is also the cloud mirror's index-slot order (ix1, ix2): the
		// plain delta read first, then the one restricted to active profiles.
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.Updated)},
			{Type: db.TypeView, Keys: db.Cols(e.Status, e.Updated.DecimalSize(10))},
		},
	}
}
