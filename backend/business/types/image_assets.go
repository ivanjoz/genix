package types

import "app/db"

type ImageAsset struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
	ID          int32    `json:",omitempty"`
	CategoryID  int16    `json:",omitempty" db:"category_id"`
	Description string   `json:",omitempty"`
	Objects     []string `json:",omitempty" db:",list"`
	Bigrams     []int8   `json:",omitempty" db:",list"`
	Updated     int32    `json:"upd,omitempty"`
}

type ImageAssetTable struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
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
		Partition: e.CategoryID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			// Serve global frontend deltas without scanning every category partition.
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
