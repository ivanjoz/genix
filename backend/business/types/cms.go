package types

import "app/db"

type GalleryImage struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID int32 `db:",pk" cb:"1"`
	// ImageID is the company-unique ID reserved by GET.image-id-counter (final digit 4 = gallery).
	ImageID int32 `db:",pk" cb:"3"`
	// Image is the CDN base name, "<companyID>_<imageID>".
	Image       string `cb:"2"`
	Description string `cb:"4"`
	Status      int8   `json:"ss" cb:"ss,5"`
	Updated     int32  `json:"upd" cb:"upd,6"`
	Version     int16  `json:"ver" cb:"ver,7"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,8"`
}

type GalleryImageTable struct {
	db.TableStruct[GalleryImageTable, GalleryImage]
	CompanyID   db.Col[*GalleryImageTable, int32]
	Image       db.Col[*GalleryImageTable, string]
	ImageID     db.Col[*GalleryImageTable, int32]
	Description db.Col[*GalleryImageTable, string]
	Status      db.Col[*GalleryImageTable, int8]
	Updated     db.Col[*GalleryImageTable, int32]
	Version     db.Col[*GalleryImageTable, int16]
}

func (e GalleryImageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        15,
		Name:      "gallery_images",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ImageID),
		// GET.gallery-images syncs the active images by delta; a deleted image (ss=0) arrives as a
		// removed ID.
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
