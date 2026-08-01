package types

import "app/db"

type GalleryImage struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID      int32  `db:"empresa_id,pk"`
	Image          string `db:"image,pk"`
	ImageID        int32  `json:",omitempty" db:"image_id"`
	Description    string `db:"description"`
	Status         int8   `json:"ss,omitempty" db:"status"`
	Updated        int32  `json:"upd,omitempty" db:"updated"`
	UpdatedVersion int32  `json:"upv,omitempty"`
}

type GalleryImageTable struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID      db.Col[GalleryImageTable, int32]
	Image          db.Col[GalleryImageTable, string]
	ImageID        db.Col[GalleryImageTable, int32]
	Description    db.Col[GalleryImageTable, string]
	Status         db.Col[GalleryImageTable, int8]
	Updated        db.Col[GalleryImageTable, int32]
	UpdatedVersion db.Col[GalleryImageTable, int32]
}

func (e GalleryImageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        15,
		Name:      "gallery_images",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.Image),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
