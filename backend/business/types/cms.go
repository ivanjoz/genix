package types

import "app/db"

type GalleryImage struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   int32  `db:"empresa_id,pk"`
	Image       string `db:"image,pk"`
	ImageID     int32  `json:",omitempty" db:"image_id"`
	Description string `db:"description"`
	Status      int8   `json:"ss,omitempty" db:"status,view"`
	Updated     int32  `json:"upd,omitempty" db:"updated,view.1"`
}

type GalleryImageTable struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   db.Col[GalleryImageTable, int32]
	Image       db.Col[GalleryImageTable, string]
	ImageID     db.Col[GalleryImageTable, int32]
	Description db.Col[GalleryImageTable, string]
	Status      db.Col[GalleryImageTable, int8]
	Updated     db.Col[GalleryImageTable, int32]
}

func (e GalleryImageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        15,
		Name:      "gallery_images",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.Image),
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.Status)},
			{Type: db.TypeView, Keys: db.Cols(e.Updated)},
		},
	}
}
