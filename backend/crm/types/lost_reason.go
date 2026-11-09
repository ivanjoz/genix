package types

import "app/db"

// CrmLostReason is the configurable list an opportunity picks from when it is closed as lost.
type CrmLostReason struct {
	db.TableStruct[CrmLostReasonTable, CrmLostReason]
	CompanyID int32  `cb:"1"`
	ID        int16  `cb:"2"`
	Name      string `cb:"3"`
	Status    int8   `json:"ss" cb:"ss,4"`
	Updated   int32  `json:"upd" cb:"upd,5"`
	Version   int16  `json:"ver" cb:"ver,6"`
	UpdatedBy int32  `cb:"7"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,8"`
}

type CrmLostReasonTable struct {
	db.TableStruct[CrmLostReasonTable, CrmLostReason]
	CompanyID db.Col[*CrmLostReasonTable, int32]
	ID        db.Col[*CrmLostReasonTable, int16]
	Name      db.Col[*CrmLostReasonTable, string]
	Status    db.Col[*CrmLostReasonTable, int8]
	Updated   db.Col[*CrmLostReasonTable, int32]
	Version   db.Col[*CrmLostReasonTable, int16]
	UpdatedBy db.Col[*CrmLostReasonTable, int32]
}

func (e CrmLostReasonTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          69,
		Name:        "crm_lost_reasons",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}
