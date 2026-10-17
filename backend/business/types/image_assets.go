package types

import "app/db"

// ImageConfigDigitFull is the last digit of an imageID for the standard product upload
// (base x6 + x4 + x2). Exported because the id counter lives in `business` while the product
// uploader lives in `production`, and both encode the same digit.
const ImageConfigDigitFull = 7

type ImageAsset struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
	GroupID            int32    `json:"-"`
	ID                 int32    `json:",omitempty"`
	CategoryID         int16    `json:",omitempty" db:"category_id"`
	Description        string   `json:",omitempty"`            // English description (IMAGES_LIST.md).
	SpanishDescription string   `json:",omitempty"`            // Spanish description (IMAGES_LIST.ES.md).
	Keywords           string   `json:",omitempty"`            // Deduplicated English words; feeds the TextSearch AI search.
	SpanishKeywords    []string `json:",omitempty" db:",list"` // Spanish object list; feeds the bigram local search.
	Bigrams            []int8   `json:",omitempty" db:",list"` // Encoded from the Spanish keywords for frontend local search.
	Ratio              float32  `json:",omitempty"`            // Aspect ratio = width/height (1.0=1:1, 1.777=16:9, 0.75=3:4). 0 ⇒ unknown, treated as 1:1 by find_image.
	Updated            int32    `json:"upd,omitempty"`
	Version            int16    `json:"ver,omitempty"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt,omitempty"`
}

type ImageAssetTable struct {
	db.TableStruct[ImageAssetTable, ImageAsset]
	GroupID            db.Col[*ImageAssetTable, int32]
	ID                 db.Col[*ImageAssetTable, int32]
	CategoryID         db.Col[*ImageAssetTable, int16]
	Description        db.Col[*ImageAssetTable, string]
	SpanishDescription db.Col[*ImageAssetTable, string]
	Keywords           db.Col[*ImageAssetTable, string]
	SpanishKeywords    db.ColSlice[*ImageAssetTable, string]
	Bigrams            db.ColSlice[*ImageAssetTable, int8]
	Ratio              db.Col[*ImageAssetTable, float32]
	Updated            db.Col[*ImageAssetTable, int32]
	Version            db.Col[*ImageAssetTable, int16]
}

func (e ImageAssetTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        16,
		Name:      "image_assets",
		Partition: e.GroupID,
		// Keywords holds the deduplicated English text the AI search queries. No status
		// column: every image is searchable.
		TextSearch: &db.TextSearch{Columns: db.Cols(e.Keywords)},
		Keys:       db.Cols(e.ID),
		Indexes: []db.Index{
			// Keyless: one log per partition and no sync filter. Read with Delta(cursor).
			{Type: db.TypeDelta},
		},
	}
}

const ImageAssetCategoryGroupID int32 = 1
