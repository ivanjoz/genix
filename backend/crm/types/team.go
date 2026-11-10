package types

import "app/db"

// CrmTeam groups sellers so goals, forecasts and scorecards can be rolled up (plan §3.1).
// A user belongs to at most one active team, so a team roll-up never counts a seller twice.
type CrmTeam struct {
	db.TableStruct[CrmTeamTable, CrmTeam]
	CompanyID     int32   `cb:"1"`
	ID            int16   `cb:"2"`
	Name          string  `cb:"3"`
	LeaderUserID  int32   `cb:"4"`
	MemberUserIDs []int32 `cb:"5"`
	Status        int8    `json:"ss" cb:"ss,6"`
	Updated       int32   `json:"upd" cb:"upd,7"`
	Version       int16   `json:"ver" cb:"ver,8"`
	UpdatedBy     int32   `cb:"9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
}

type CrmTeamTable struct {
	db.TableStruct[CrmTeamTable, CrmTeam]
	CompanyID     db.Col[*CrmTeamTable, int32]
	ID            db.Col[*CrmTeamTable, int16]
	Name          db.Col[*CrmTeamTable, string]
	LeaderUserID  db.Col[*CrmTeamTable, int32]
	MemberUserIDs db.ColSlice[*CrmTeamTable, int32]
	Status        db.Col[*CrmTeamTable, int8]
	Updated       db.Col[*CrmTeamTable, int32]
	Version       db.Col[*CrmTeamTable, int16]
	UpdatedBy     db.Col[*CrmTeamTable, int32]
}

func (e CrmTeamTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          76,
		Name:        "crm_teams",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		// A handful of teams per company: the whole table syncs.
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}
