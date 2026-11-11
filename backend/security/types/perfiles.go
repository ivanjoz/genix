package types

import (
	coreTypes "app/core/types"
	"app/db"
)

type Profile struct {
	db.TableStruct[ProfileTable, Profile]
	CompanyID   int32  `db:",pk" cb:"1"`
	ID          int32  `db:",pk" cb:"2"`
	Name        string `cb:"3"`
	Description string `cb:"4"`
	// AccesosGrants is what the profile grants: one record per access, carrying its level and its
	// sub-accesses. Exactly the shape a user's direct grants take, which is what lets one editor
	// and one merge serve both.
	AccesosGrants []coreTypes.AccesoGrantRecord `cb:"6"`
	Status        int8                          `json:"ss" cb:"ss,7"`
	Updated       int32                         `json:"upd" cb:"upd,8"`
	Version       int16                         `json:"ver" cb:"ver,9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
}

type ProfileTable struct {
	db.TableStruct[ProfileTable, Profile]
	ID          db.Col[*ProfileTable, int32]
	CompanyID   db.Col[*ProfileTable, int32]
	Name        db.Col[*ProfileTable, string]
	Description db.Col[*ProfileTable, string]

	AccesosGrants db.Col[*ProfileTable, []coreTypes.AccesoGrantRecord]
	Status        db.Col[*ProfileTable, int8]
	Updated       db.Col[*ProfileTable, int32]
	Version       db.Col[*ProfileTable, int16]
}

func (e ProfileTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           23,
		Name:         "profiles",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// The profiles list (GET.perfiles) syncs by delta from the primary database only; the cloud
		// mirror keeps receiving the writes but serves no read, and TypeDelta has no mirror slot.
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
