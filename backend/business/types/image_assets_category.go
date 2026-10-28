package types

import "app/db"

type ImageAssetCategory struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID int32  `json:"-" cb:"-"`
	ID      int16  `cb:"1"`
	Name    string `cb:"2"`
	Updated int32  `json:"upd" cb:"upd,3"`
	Version int16  `json:"ver" cb:"ver,4"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,5"`
	MaxID    int32   `json:"-" cb:"-"`
}

type ImageAssetCategoryTable struct {
	db.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID db.Col[*ImageAssetCategoryTable, int32]
	ID      db.Col[*ImageAssetCategoryTable, int16]
	Name    db.Col[*ImageAssetCategoryTable, string]
	Updated db.Col[*ImageAssetCategoryTable, int32]
	Version db.Col[*ImageAssetCategoryTable, int16]
	MaxID   db.Col[*ImageAssetCategoryTable, int32]
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
			// Keyless: one log per partition and no sync filter. Read with Delta(cursor).
			{Type: db.TypeDelta},
		},
	}
}
