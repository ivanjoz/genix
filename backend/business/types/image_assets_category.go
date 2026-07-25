package types

import "github.com/ivanjoz/genix-orm/scylla"

type ImageAssetCategory struct {
	scylla.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID int32  `json:"-"`
	ID      int16  `json:",omitempty"`
	Name    string `json:",omitempty"`
	Updated int32  `json:"upd,omitempty"`
	MaxID   int32  `json:"-" db:"max_id"`
}

type ImageAssetCategoryTable struct {
	scylla.TableStruct[ImageAssetCategoryTable, ImageAssetCategory]
	GroupID scylla.Col[ImageAssetCategoryTable, int32]
	ID      scylla.Col[ImageAssetCategoryTable, int16]
	Name    scylla.Col[ImageAssetCategoryTable, string]
	Updated scylla.Col[ImageAssetCategoryTable, int32]
	MaxID   scylla.Col[ImageAssetCategoryTable, int32]
}

func (e ImageAssetCategoryTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "image_assets_category",
		Partition:    e.GroupID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Category names are stable repository slugs used to resolve existing IDs.
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.Name)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
