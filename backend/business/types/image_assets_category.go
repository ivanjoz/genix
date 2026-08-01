package types

import "app/db"

type ImageAssetCategory struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID        int32  `json:"-"`
	ID             int16  `json:",omitempty"`
	Name           string `json:",omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedVersion int32  `json:"upv,omitempty"`
	MaxID          int32  `json:"-" db:"max_id"`
}

type ImageAssetCategoryTable struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID        db.Col[ImageAssetCategoryTable, int32]
	ID             db.Col[ImageAssetCategoryTable, int16]
	Name           db.Col[ImageAssetCategoryTable, string]
	Updated        db.Col[ImageAssetCategoryTable, int32]
	UpdatedVersion db.Col[ImageAssetCategoryTable, int32]
	MaxID          db.Col[ImageAssetCategoryTable, int32]
}

func (e ImageAssetCategoryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           17,
		Name:         "image_assets_category",
		Partition:    e.GroupID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// Category names are stable repository slugs used to resolve existing IDs.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.Name)},
			// Keyless: this sync filters nothing but its watermark. Read with Delta(upv).
			{Type: db.TypeDelta},
		},
	}
}
