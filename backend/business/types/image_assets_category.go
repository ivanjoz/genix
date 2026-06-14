package types

import "app/db"

type ImageAssetCategory struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID int8   `json:"-"`
	ID      int16  `json:",omitempty"`
	Name    string `json:",omitempty"`
	Updated int32  `json:"upd,omitempty"`
	MaxID   int32  `json:"-" db:"max_id"`
}

type ImageAssetCategoryTable struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID db.Col[ImageAssetCategoryTable, int8]
	ID      db.Col[ImageAssetCategoryTable, int16]
	Name    db.Col[ImageAssetCategoryTable, string]
	Updated db.Col[ImageAssetCategoryTable, int32]
	MaxID   db.Col[ImageAssetCategoryTable, int32]
}

func (e ImageAssetCategoryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "image_assets_category",
		Partition:    e.GroupID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Category names are stable repository slugs used to resolve existing IDs.
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.Name}},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
