package types

import "github.com/ivanjoz/genix-orm/scylla"

type GalleryImage struct {
	scylla.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   int32  `db:"empresa_id,pk"`
	Image       string `db:"image,pk"`
	ImageID     int32  `json:",omitempty" db:"image_id"`
	Description string `db:"description"`
	Status      int8   `json:"ss,omitempty" db:"status,view"`
	Updated     int32  `json:"upd,omitempty" db:"updated,view.1"`
}

type GalleryImageTable struct {
	scylla.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   scylla.Col[GalleryImageTable, int32]
	Image       scylla.Col[GalleryImageTable, string]
	ImageID     scylla.Col[GalleryImageTable, int32]
	Description scylla.Col[GalleryImageTable, string]
	Status      scylla.Col[GalleryImageTable, int8]
	Updated     scylla.Col[GalleryImageTable, int32]
}

func (e GalleryImageTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "gallery_images",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.Image),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
