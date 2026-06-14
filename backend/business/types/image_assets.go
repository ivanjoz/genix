package types

import "app/db"

type ImageAsset struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
	GroupID     int8     `json:"-"`
	ID          int32    `json:",omitempty"`
	CategoryID  int16    `json:",omitempty" db:"category_id"`
	Description string   `json:",omitempty"`
	Objects     []string `json:",omitempty" db:",list"`
	Bigrams     []int8   `json:",omitempty" db:",list"`
	Updated     int32    `json:"upd,omitempty"`
}

type ImageAssetTable struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
	GroupID     db.Col[ImageAssetTable, int8]
	ID          db.Col[ImageAssetTable, int32]
	CategoryID  db.Col[ImageAssetTable, int16]
	Description db.Col[ImageAssetTable, string]
	Objects     db.ColSlice[ImageAssetTable, string]
	Bigrams     db.ColSlice[ImageAssetTable, int8]
	Updated     db.Col[ImageAssetTable, int32]
}

func (e ImageAssetTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "image_assets",
		Partition: e.GroupID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			// Keep Updated as the first clustering column for global frontend deltas.
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
