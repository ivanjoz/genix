package types

import "app/db"

type GalleryImage struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   int32  `db:"empresa_id,pk" cb:"1"`
	Image       string `db:",pk" cb:"2"`
	ImageID     int32  `cb:"3"`
	Description string `cb:"4"`
	Status      int8   `json:"ss" cb:"ss,5"`
	Updated     int32  `json:"upd" cb:"upd,6"`
}

type GalleryImageTable struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   db.Col[*GalleryImageTable, int32]
	Image       db.Col[*GalleryImageTable, string]
	ImageID     db.Col[*GalleryImageTable, int32]
	Description db.Col[*GalleryImageTable, string]
	Status      db.Col[*GalleryImageTable, int8]
	Updated     db.Col[*GalleryImageTable, int32]
}

func (e GalleryImageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        15,
		Name:      "gallery_images",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.Image),
		// The key is a string, which a delta log can't name, so the route answers its whole
		// active list with the replace Metadata (core.WithDeltaReplace).
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.Status)},
		},
	}
}
